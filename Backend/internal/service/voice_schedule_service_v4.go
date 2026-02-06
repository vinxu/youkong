package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
	"youkong/internal/pkg/asr"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

// V4 架构常量
const (
	v4MaxLoops        = 5                    // 最大循环次数
	v4SessionTTL      = 10 * time.Minute     // 会话过期时间
	v4SessionKeyPrefix = "v4_voice_session:" // Redis key 前缀
)

// VoiceScheduleServiceV4 V4 版本语音时刻表服务
// 设计理念：像 Claude CLI 一样，用简单的 while 循环 + 工具调用，不使用复杂的状态机
type VoiceScheduleServiceV4 struct {
	scheduleRepo          *repository.ScheduleRepository
	memoryRepo            *repository.MemoryRepository
	memoryDocRepo         *repository.UserMemoryDocumentRepository // 用户记忆文档
	memoryLearningService *MemoryLearningService                   // 记忆学习服务
	userProfileService    *UserProfileService
	redisClient           *tencent.RedisClient
	llmAdapter            *agent.LLMAdapter
	asrClient             *asr.AliyunASRClient // 语音识别客户端

	// V4 工具集
	tools []*agent.Tool

	// ========== 好友相关依赖 ==========
	friendshipService   *FriendshipService
	conversationService *ConversationService
	agentService        *AgentService

	// ========== 扩展功能依赖 ==========
	contactService *ContactService
}

// NewVoiceScheduleServiceV4 创建 V4 版本语音时刻表服务
func NewVoiceScheduleServiceV4(
	scheduleRepo *repository.ScheduleRepository,
	memoryRepo *repository.MemoryRepository,
	memoryDocRepo *repository.UserMemoryDocumentRepository,
	userProfileService *UserProfileService,
	redisClient *tencent.RedisClient,
	asrClient *asr.AliyunASRClient,
	llmAPIKey string,
	llmModel string,
	friendshipService *FriendshipService,
	conversationService *ConversationService,
	agentService *AgentService,
	contactService *ContactService,
) *VoiceScheduleServiceV4 {
	svc := &VoiceScheduleServiceV4{
		scheduleRepo:        scheduleRepo,
		memoryRepo:          memoryRepo,
		memoryDocRepo:       memoryDocRepo,
		userProfileService:  userProfileService,
		redisClient:         redisClient,
		asrClient:           asrClient,
		tools:               agent.V4AllTools(), // 所有工具（时刻表+好友+扩展）
		friendshipService:   friendshipService,
		conversationService: conversationService,
		agentService:        agentService,
		contactService:      contactService,
	}

	// 创建 LLM 适配器
	if llmAPIKey != "" {
		svc.llmAdapter = agent.NewLLMAdapter(&agent.LLMAdapterConfig{
			APIKey: llmAPIKey,
			Model:  llmModel, // 使用外部传入的模型配置
		})

		// 创建记忆学习服务
		svc.memoryLearningService = NewMemoryLearningService(memoryDocRepo, svc.llmAdapter)
	}

	return svc
}

// ProcessTextInput 处理文本输入（主入口）
// 架构原则：所有输入统一走 Agent 循环，LLM 自主决定是否需要工具
func (s *VoiceScheduleServiceV4) ProcessTextInput(
	ctx context.Context,
	userID string,
	sessionID string,
	text string,
	callback func(event *model.V4Event),
) (*model.V4Session, error) {
	// 1. 获取或创建会话
	session, isNew := s.getOrCreateSession(ctx, userID, sessionID)

	// 发送会话开始事件
	if isNew {
		callback(&model.V4Event{
			Type:      model.V4EventTypeSessionStart,
			SessionID: session.SessionID,
		})
	}

	// 发送识别结果
	callback(&model.V4Event{
		Type:    model.V4EventTypeTranscript,
		Message: text,
	})

	// 2. 加载用户时刻表（如果是新会话）
	if isNew || len(session.CurrentSchedule) == 0 {
		s.loadCurrentSchedule(ctx, session)
	}

	// 3. 添加用户消息
	session.AddMessage("user", text)

	// 4. 发送阶段：理解请求
	callback(&model.V4Event{
		Type:    model.V4EventTypePhase,
		Phase:   "understanding",
		Message: "正在理解你的请求...",
	})

	// 5. 统一走 Agent 循环（LLM 自主决定是否调用工具）
	err := s.processLoopWithTools(ctx, session, callback)

	if err != nil {
		callback(&model.V4Event{
			Type:    model.V4EventTypeError,
			Message: err.Error(),
		})
		return session, err
	}

	// 6. 保存会话
	if saveErr := s.saveSession(ctx, session); saveErr != nil {
		fmt.Printf("[V4] 保存会话失败: %v\n", saveErr)
	}

	// 7. 异步触发记忆学习（会话结束或消息数达到阈值时）
	if s.memoryLearningService != nil && len(session.Messages) >= 4 {
		go func() {
			learnCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if learnErr := s.memoryLearningService.LearnFromSession(learnCtx, session); learnErr != nil {
				fmt.Printf("[V4] 记忆学习失败: %v\n", learnErr)
			}
		}()
	}

	return session, nil
}

// [已删除] needsToolCalling, processSimpleChat, buildMessagesSimple
// 架构改进：消除代码层的语义决策，所有输入统一走 Agent 循环
// LLM 自主判断是否需要调用工具——不需要时自然不调用，不需要代码层替它做判断

// processLoopWithTools 带工具的主处理循环（优化版）
func (s *VoiceScheduleServiceV4) processLoopWithTools(
	ctx context.Context,
	session *model.V4Session,
	callback func(event *model.V4Event),
) error {
	for i := 0; i < v4MaxLoops; i++ {
		loopNum := i + 1
		fmt.Printf("[V4] 循环 %d/%d\n", loopNum, v4MaxLoops)

		// 发送阶段：调用 AI
		if i == 0 {
			callback(&model.V4Event{
				Type:    model.V4EventTypePhase,
				Phase:   "thinking",
				Message: "正在分析...",
				Loop:    loopNum,
			})
		} else {
			callback(&model.V4Event{
				Type:    model.V4EventTypePhase,
				Phase:   "continuing",
				Message: "继续处理...",
				Loop:    loopNum,
			})
		}

		// 1. 构建消息列表
		messages := s.buildMessages(session)

		// 2. 调用 LLM（启用 thinking mode，精确决策低温度）
		response, err := s.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
			Messages:       messages,
			Tools:          s.tools,
			Temperature:    0.3,
			EnableThinking: true,
			ThinkingBudget: 2048,
		})
		if err != nil {
			return fmt.Errorf("LLM 调用失败: %w", err)
		}

		// 3. 处理响应
		if len(response.ToolCalls) == 0 {
			// 无工具调用 → 发送文本响应并结束
			if response.Content != "" {
				callback(&model.V4Event{
					Type:    model.V4EventTypePhase,
					Phase:   "complete",
					Message: "处理完成",
				})
				callback(&model.V4Event{
					Type:    model.V4EventTypeChat,
					Message: response.Content,
				})
				session.AddMessage("assistant", response.Content)
			}
			break
		}

		// 4. 先添加 assistant 消息（包含 tool_calls）
		session.AddAssistantMessageWithToolCalls(response.Content, s.convertToolCalls(response.ToolCalls))

		// 5. 执行工具调用（会添加 tool 结果到消息历史）
		shouldBreak, hasContent := s.executeToolCallsWithPhase(ctx, session, response, callback)

		// 优化：如果工具执行后 LLM 已经返回了文本内容，直接结束
		if hasContent && response.Content != "" {
			callback(&model.V4Event{
				Type:    model.V4EventTypeChat,
				Message: response.Content,
			})
			session.AddMessage("assistant", response.Content)
			break
		}

		if shouldBreak {
			break
		}
	}

	return nil
}

// v4TokenBudget 上下文 Token 预算
// 超过此阈值时触发语义压缩，而非粗暴截断
const v4TokenBudget = 6000

// v4KeepLastN 压缩时保留最近的消息数量
const v4KeepLastN = 8

// buildMessages 构建 LLM 消息列表
// 架构原则：使用语义压缩替代 Last-N 截断，保留关键决策和用户偏好
func (s *VoiceScheduleServiceV4) buildMessages(session *model.V4Session) []agent.AgentMessage {
	// 构建系统提示词（包含用户记忆）
	systemPrompt := s.buildSystemPromptWithMemory(context.Background(), session)

	messages := []agent.AgentMessage{
		agent.NewSystemMessage(systemPrompt),
	}

	// 如果有历史摘要，注入为上下文
	if session.Summary != "" {
		messages = append(messages, agent.NewSystemMessage("[历史对话摘要]\n"+session.Summary))
	}

	// 转换所有 V4Message 为 AgentMessage
	allMessages := make([]agent.AgentMessage, 0, len(session.Messages))
	for _, msg := range session.Messages {
		allMessages = append(allMessages, s.convertV4Message(msg))
	}

	// 估算总 token 数
	totalTokens := 0
	for i := range allMessages {
		totalTokens += allMessages[i].EstimateTokens()
	}
	// 加上系统提示词的 token
	for i := range messages {
		totalTokens += messages[i].EstimateTokens()
	}

	if totalTokens <= v4TokenBudget || len(allMessages) <= v4KeepLastN {
		// 预算内或消息太少，使用全部消息
		messages = append(messages, allMessages...)
	} else {
		// 超出预算，触发语义压缩
		s.compressV4Context(session, allMessages)

		// 压缩后取最近消息
		recentStart := len(allMessages) - v4KeepLastN
		if recentStart < 0 {
			recentStart = 0
		}
		messages = append(messages, allMessages[recentStart:]...)
	}

	return messages
}

// compressV4Context 压缩 V4 会话上下文
// 将较早的消息通过 LLM 压缩为摘要，保留在 session.Summary 中
func (s *VoiceScheduleServiceV4) compressV4Context(session *model.V4Session, allMessages []agent.AgentMessage) {
	if s.llmAdapter == nil {
		return
	}

	// 计算需要压缩的消息范围（除去最近保留的）
	compressEnd := len(allMessages) - v4KeepLastN
	if compressEnd <= 0 {
		return
	}

	toCompress := allMessages[:compressEnd]

	// 使用 LLM 生成摘要
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	summary, err := s.llmAdapter.GenerateSummary(ctx, toCompress, session.Summary)
	if err != nil {
		fmt.Printf("[V4] 上下文压缩失败: %v\n", err)
		return
	}

	// 更新会话摘要
	session.Summary = summary

	// 清理已压缩的消息，只保留最近的
	if compressEnd < len(session.Messages) {
		session.Messages = session.Messages[compressEnd:]
	}

	fmt.Printf("[V4] 上下文压缩完成: 压缩了 %d 条消息, 摘要长度 %d 字符\n", compressEnd, len(summary))
}

// buildSystemPromptWithMemory 构建包含用户记忆的系统提示词
func (s *VoiceScheduleServiceV4) buildSystemPromptWithMemory(ctx context.Context, session *model.V4Session) string {
	basePrompt := s.buildSystemPrompt(session)

	// 加载用户记忆
	if s.memoryLearningService == nil {
		return basePrompt
	}

	memoryContext, err := s.memoryLearningService.GetMemoryContext(ctx, session.UserID)
	if err != nil {
		fmt.Printf("[V4] 加载用户记忆失败: %v\n", err)
		return basePrompt
	}

	if memoryContext == "" {
		return basePrompt
	}

	// 注入用户记忆到系统提示词
	return fmt.Sprintf("%s\n【用户记忆】\n%s\n", basePrompt, memoryContext)
}

// buildSystemPrompt 构建系统提示词
// 架构原则：描述行为原则，不枚举输入→输出映射
// LLM 根据工具定义自行推理何时使用哪个工具
func (s *VoiceScheduleServiceV4) buildSystemPrompt(session *model.V4Session) string {
	now := time.Now()
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	var sb strings.Builder

	// ========== 角色定义 ==========
	sb.WriteString(`你是「有空」的 AI 助手。你有一组工具可以帮用户管理日程、查询好友状态、发送消息。

行为原则：
- 先理解用户真正想做什么，再决定用什么工具
- 如果用户的请求模糊或有多种解读，先用文字问清楚
- 查询类操作可以直接执行
- 修改类操作（创建/修改日程）会生成预览等用户确认
- 发消息/约人前，先用 query_friends 找到好友
- 回复简洁自然，像朋友聊天，emoji 匹配活动
- 时间模糊时先确认，不要猜测
- 用户只是闲聊时，不需要调用任何工具，直接回复即可

【重要区分】
- 用户描述当前状态（"我在加班"、"好累"、"到公司了"）→ update_current_status（即时生效，不需确认）
- 用户规划时间安排（"下午3点开会"、"明天去健身"）→ update_schedule（生成预览等确认）
- 用户给出精确时间+活动，即使多条也应一次性调用 update_schedule（如"10点瑜伽课，2点兼职，7点聚餐"→一次 update_schedule 包含3个items）
- 修改时刻表时只传变更的条目，operation="modify"，系统会自动保留其他时段
- 用户说"给他/她发消息"时，使用上一次 query_friends 查到的好友
- save_schedule 和 confirm_send 是互斥的：save_schedule 保存日程，confirm_send 发送消息/邀请。根据待确认内容类型选择正确的工具

`)

	// ========== 当前时间 ==========
	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	sb.WriteString(fmt.Sprintf("【时间】%s %s %02d:%02d | 今天=%s 明天=%s\n\n",
		today, weekdays[now.Weekday()], now.Hour(), now.Minute(), today, tomorrow))

	// ========== 用户当前时刻表 ==========
	if len(session.CurrentSchedule) > 0 {
		sb.WriteString("【当前时刻表】")
		for i, item := range session.CurrentSchedule {
			if i > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(fmt.Sprintf("%s-%s %s%s", item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
		sb.WriteString("\n\n")
	}

	// ========== 待确认的时刻表 ==========
	if session.HasPendingSchedule() {
		sb.WriteString("【⚠️待确认时刻表】")
		for i, item := range session.PendingSchedule {
			if i > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(fmt.Sprintf("%s-%s %s%s", item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
		sb.WriteString(" (日期:" + session.PendingDate.Format("01-02") + ")")
		sb.WriteString("\n用户确认后调用 save_schedule 保存，要修改则调用 update_schedule 重新生成\n\n")
	}

	// ========== 待确认的消息/邀请 ==========
	if session.HasPendingMessage() {
		sb.WriteString(fmt.Sprintf("【⚠️待确认消息】发送给 %s: \"%s\"\n用户确认后调用 confirm_send\n\n",
			session.PendingMessage.FriendName, session.PendingMessage.Message))
	}
	if session.HasPendingInvite() {
		sb.WriteString(fmt.Sprintf("【⚠️待确认邀请】邀请 %s %s %s-%s %s\n用户确认后调用 confirm_send\n\n",
			session.PendingInvite.FriendName, session.PendingInvite.Date,
			session.PendingInvite.StartTime, session.PendingInvite.EndTime,
			session.PendingInvite.Activity))
	}

	return sb.String()
}

// executeToolCallsWithPhase 执行工具调用（带阶段暴露）
// 返回 (shouldBreak, hasContent)
func (s *VoiceScheduleServiceV4) executeToolCallsWithPhase(
	ctx context.Context,
	session *model.V4Session,
	response *agent.LLMResponse,
	callback func(event *model.V4Event),
) (bool, bool) {
	shouldBreak := false
	hasContent := response.Content != ""

	for _, tc := range response.ToolCalls {
		toolName := tc.Function.Name
		fmt.Printf("[V4] 执行工具: %s\n", toolName)

		// 发送工具开始事件
		callback(&model.V4Event{
			Type:     model.V4EventTypeToolStart,
			ToolName: toolName,
			Message:  s.getToolDescription(toolName),
		})

		// 解析参数
		args, err := tc.ParseArguments()
		if err != nil {
			fmt.Printf("[V4] 解析参数失败: %v\n", err)
			session.AddToolResult(tc.ID, toolName, fmt.Sprintf(`{"error": "%s"}`, err.Error()))

			// 发送工具失败事件
			callback(&model.V4Event{
				Type:     model.V4EventTypeToolEnd,
				ToolName: toolName,
				Message:  "参数解析失败",
			})
			continue
		}

		// 执行工具
		result, breakLoop := s.executeTool(ctx, session, toolName, args, callback)

		// 添加工具结果到会话
		resultJSON, _ := json.Marshal(result)
		session.AddToolResult(tc.ID, toolName, string(resultJSON))

		// 发送工具完成事件
		callback(&model.V4Event{
			Type:     model.V4EventTypeToolEnd,
			ToolName: toolName,
			Message:  "执行完成",
		})

		if breakLoop {
			shouldBreak = true
		}
	}

	return shouldBreak, hasContent
}

// getToolDescription 获取工具的友好描述
func (s *VoiceScheduleServiceV4) getToolDescription(toolName string) string {
	descriptions := map[string]string{
		"get_schedule":              "正在查询时刻表...",
		"update_schedule":           "正在生成时刻表预览...",
		"update_current_status":     "正在更新当前状态...",
		"save_schedule":             "正在保存时刻表...",
		"update_preference":         "正在更新偏好设置...",
		"query_friends":             "正在查询好友...",
		"send_message":              "正在生成消息预览...",
		"create_schedule_invite":    "正在生成日程邀请...",
		"confirm_send":              "正在发送...",
		"match_contacts":            "正在匹配通讯录...",
		"add_friend":                "正在添加好友...",
		"query_device_data":         "正在查询设备数据...",
		"query_memory":              "正在查询记忆...",
		"update_memory":             "正在更新记忆...",
		"get_behavior_suggestion":   "正在生成建议...",
	}

	if desc, ok := descriptions[toolName]; ok {
		return desc
	}
	return "正在处理..."
}

// executeTool 执行单个工具
func (s *VoiceScheduleServiceV4) executeTool(
	ctx context.Context,
	session *model.V4Session,
	toolName string,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	switch toolName {
	case "get_schedule":
		return s.executeGetSchedule(ctx, session, args, callback)

	case "update_schedule":
		return s.executeUpdateSchedule(ctx, session, args, callback)

	case "update_current_status":
		return s.executeUpdateCurrentStatus(ctx, session, args, callback)

	case "save_schedule":
		return s.executeSaveSchedule(ctx, session, args, callback)

	case "update_preference":
		return s.executeUpdatePreference(ctx, session, args, callback)

	// ========== 好友相关工具 ==========
	case "query_friends":
		return s.executeQueryFriends(ctx, session, args, callback)

	case "send_message":
		return s.executeSendMessage(ctx, session, args, callback)

	case "create_schedule_invite":
		return s.executeCreateScheduleInvite(ctx, session, args, callback)

	case "confirm_send":
		return s.executeConfirmSend(ctx, session, args, callback)

	// ========== 扩展工具 ==========
	case "match_contacts":
		return s.executeMatchContacts(ctx, session, args, callback)

	case "add_friend":
		return s.executeAddFriend(ctx, session, args, callback)

	case "query_device_data":
		return s.executeQueryDeviceData(ctx, session, args, callback)

	case "query_memory":
		return s.executeQueryMemory(ctx, session, args, callback)

	case "update_memory":
		return s.executeUpdateMemory(ctx, session, args, callback)

	case "get_behavior_suggestion":
		return s.executeGetBehaviorSuggestion(ctx, session, args, callback)

	default:
		return map[string]string{"error": "未知工具: " + toolName}, false
	}
}

// executeGetSchedule 执行获取时刻表
func (s *VoiceScheduleServiceV4) executeGetSchedule(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	// 解析日期
	date, err := agent.ParseDate(args)
	if err != nil {
		date = time.Now()
	}

	// 解析时间范围过滤参数
	startTimeFilter, _ := args["start_time"].(string)
	endTimeFilter, _ := args["end_time"].(string)

	// 获取时刻表
	schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, session.UserID, date)
	if err != nil || schedule == nil || len(schedule.Items) == 0 {
		return map[string]interface{}{
			"message": "暂无时刻表",
			"date":    date.Format("2006-01-02"),
		}, false
	}

	// 转换为返回格式，同时应用时间范围过滤
	filteredItems := make([]map[string]string, 0, len(schedule.Items))
	for _, item := range schedule.Items {
		// 应用时间范围过滤
		if startTimeFilter != "" && item.EndTime <= startTimeFilter {
			continue // 时段在过滤开始时间之前结束，跳过
		}
		if endTimeFilter != "" && item.StartTime >= endTimeFilter {
			continue // 时段在过滤结束时间之后开始，跳过
		}

		filteredItems = append(filteredItems, map[string]string{
			"start_time": item.StartTime,
			"end_time":   item.EndTime,
			"emoji":      item.Emoji,
			"status":     item.Status,
		})
	}

	if len(filteredItems) == 0 {
		msg := "该时间段暂无安排"
		if startTimeFilter != "" || endTimeFilter != "" {
			msg = fmt.Sprintf("在%s", date.Format("01月02日"))
			if startTimeFilter != "" {
				msg += " " + startTimeFilter
			}
			if endTimeFilter != "" {
				msg += "到" + endTimeFilter
			}
			msg += "期间暂无安排"
		}
		return map[string]interface{}{
			"message": msg,
			"date":    date.Format("2006-01-02"),
		}, false
	}

	return map[string]interface{}{
		"date":  date.Format("2006-01-02"),
		"items": filteredItems,
		"count": len(filteredItems),
	}, false
}

// executeUpdateSchedule 执行更新时刻表（生成预览）
// 支持三种操作模式：create（添加）、modify（修改）、replace（替换）
func (s *VoiceScheduleServiceV4) executeUpdateSchedule(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	// 解析日期
	date, err := agent.ParseDate(args)
	if err != nil {
		date = time.Now()
	}

	// 解析操作类型（默认 modify）
	operation := "modify"
	if op, ok := args["operation"].(string); ok && op != "" {
		operation = op
	}

	// 解析时刻表条目
	items, err := agent.ParseScheduleItems(args)
	if err != nil {
		return map[string]string{"error": err.Error()}, false
	}

	// 转换为 model.ScheduleItem
	newItems := make([]model.ScheduleItem, len(items))
	for i, item := range items {
		newItems[i] = model.ScheduleItem{
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			Emoji:     item.Emoji,
			Status:    item.Status,
		}
	}

	// 获取现有时刻表
	var existingItems []model.ScheduleItem
	todayDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	existingSchedule, _ := s.scheduleRepo.GetActiveByUserAndDate(ctx, session.UserID, todayDate)
	if existingSchedule != nil {
		existingItems = existingSchedule.Items
	}

	// 根据操作类型合并时刻表
	var finalItems []model.ScheduleItem
	switch operation {
	case "replace":
		// 替换模式：直接用新条目替换全部
		finalItems = newItems
		fmt.Printf("[V4] 时刻表操作=replace, 新条目=%d\n", len(newItems))

	case "create":
		// 添加模式：保留现有，添加新条目
		finalItems = s.mergeScheduleItems(existingItems, newItems, false)
		fmt.Printf("[V4] 时刻表操作=create, 现有=%d, 新增=%d, 合并后=%d\n",
			len(existingItems), len(newItems), len(finalItems))

	case "modify":
		fallthrough
	default:
		// 修改模式（默认）：智能合并，更新匹配时间段的条目
		finalItems = s.mergeScheduleItems(existingItems, newItems, true)
		fmt.Printf("[V4] 时刻表操作=modify, 现有=%d, 修改=%d, 合并后=%d\n",
			len(existingItems), len(newItems), len(finalItems))
	}

	// 按时间排序
	finalItems = s.sortScheduleItems(finalItems)

	// 保存到 session 的 PendingSchedule
	session.PendingSchedule = finalItems
	session.PendingDate = date

	// 发送预览事件
	callback(&model.V4Event{
		Type:    model.V4EventTypeSchedulePreview,
		Items:   finalItems,
		Date:    date.Format("2006-01-02"),
		Message: fmt.Sprintf("已生成 %s 的时刻表预览", date.Format("01月02日")),
	})

	return map[string]interface{}{
		"message":           "时刻表预览已生成，等待用户确认",
		"date":              date.Format("2006-01-02"),
		"items_count":       len(finalItems),
		"operation":         operation,
		"awaiting_approval": true,
	}, false
}

// mergeScheduleItems 合并时刻表条目
// replaceOverlap: true=更新重叠时段（修改模式）, false=插入并拆分重叠时段（添加模式）
func (s *VoiceScheduleServiceV4) mergeScheduleItems(existing, new []model.ScheduleItem, replaceOverlap bool) []model.ScheduleItem {
	if len(existing) == 0 {
		return new
	}
	if len(new) == 0 {
		return existing
	}

	// 创建结果切片
	result := make([]model.ScheduleItem, 0, len(existing)+len(new))

	if replaceOverlap {
		// 修改模式：用新条目替换时间匹配的旧条目
		for _, existItem := range existing {
			replaced := false
			for _, newItem := range new {
				if s.isTimeOverlap(existItem.StartTime, existItem.EndTime, newItem.StartTime, newItem.EndTime) {
					replaced = true
					break
				}
			}
			if !replaced {
				result = append(result, existItem)
			}
		}
		// 添加所有新条目
		result = append(result, new...)
	} else {
		// 添加模式：插入新时段，拆分被覆盖的现有时段
		for _, existItem := range existing {
			// 检查这个现有条目是否与任何新条目重叠
			splitItems := s.splitExistingItem(existItem, new)
			result = append(result, splitItems...)
		}
		// 添加所有新条目
		result = append(result, new...)
	}

	return result
}

// splitExistingItem 将现有条目按新条目拆分
// 如果新条目在现有条目时间范围内，将现有条目拆分为前后两段（不含新条目时间）
func (s *VoiceScheduleServiceV4) splitExistingItem(existItem model.ScheduleItem, newItems []model.ScheduleItem) []model.ScheduleItem {
	result := []model.ScheduleItem{existItem}

	for _, newItem := range newItems {
		if !s.isTimeOverlap(existItem.StartTime, existItem.EndTime, newItem.StartTime, newItem.EndTime) {
			continue
		}

		// 有重叠，需要拆分
		var splitResult []model.ScheduleItem

		// 前段：existItem.StartTime ~ newItem.StartTime
		if existItem.StartTime < newItem.StartTime {
			splitResult = append(splitResult, model.ScheduleItem{
				StartTime: existItem.StartTime,
				EndTime:   newItem.StartTime,
				Emoji:     existItem.Emoji,
				Status:    existItem.Status,
			})
		}

		// 后段：newItem.EndTime ~ existItem.EndTime
		if newItem.EndTime < existItem.EndTime {
			splitResult = append(splitResult, model.ScheduleItem{
				StartTime: newItem.EndTime,
				EndTime:   existItem.EndTime,
				Emoji:     existItem.Emoji,
				Status:    existItem.Status,
			})
		}

		// 如果拆分产生了结果，替换原条目
		if len(splitResult) > 0 {
			return splitResult
		}
		// 新条目完全覆盖现有条目，返回空
		return []model.ScheduleItem{}
	}

	return result
}

// isTimeOverlap 检查两个时间段是否重叠
func (s *VoiceScheduleServiceV4) isTimeOverlap(start1, end1, start2, end2 string) bool {
	// 时间格式 HH:MM
	// 如果 start1 < end2 && start2 < end1，则重叠
	return start1 < end2 && start2 < end1
}

// sortScheduleItems 按开始时间排序
func (s *VoiceScheduleServiceV4) sortScheduleItems(items []model.ScheduleItem) []model.ScheduleItem {
	// 简单冒泡排序
	n := len(items)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if items[j].StartTime > items[j+1].StartTime {
				items[j], items[j+1] = items[j+1], items[j]
			}
		}
	}
	return items
}

// executeUpdateCurrentStatus 执行更新当前状态
func (s *VoiceScheduleServiceV4) executeUpdateCurrentStatus(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	emoji, _ := args["emoji"].(string)
	status, _ := args["status"].(string)

	if emoji == "" || status == "" {
		return map[string]string{"error": "emoji 和 status 不能为空"}, false
	}

	// 更新当前状态到时刻表（当前时段）
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 获取或创建今日时刻表
	schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, session.UserID, today)
	if err != nil || schedule == nil {
		schedule = &model.StatusSchedule{
			UserID:       session.UserID,
			ScheduleDate: today,
			Items:        model.ScheduleItems{},
			Status:       model.ScheduleStatusActive,
			Visibility:   model.VisibilityAllFriends,
		}
	}

	// 计算当前时段
	startTime, endTime := getCurrentTimeSlot(now)

	// 添加或更新当前时段
	found := false
	for i, item := range schedule.Items {
		if item.StartTime == startTime {
			schedule.Items[i].Emoji = emoji
			schedule.Items[i].Status = status
			found = true
			break
		}
	}
	if !found {
		schedule.Items = append(schedule.Items, model.ScheduleItem{
			StartTime: startTime,
			EndTime:   endTime,
			Emoji:     emoji,
			Status:    status,
		})
	}

	// 保存时刻表
	var saveErr error
	if schedule.ID == 0 {
		saveErr = s.scheduleRepo.Create(ctx, schedule)
	} else {
		saveErr = s.scheduleRepo.Update(ctx, schedule)
	}

	if saveErr != nil {
		return map[string]string{"error": "保存失败: " + saveErr.Error()}, false
	}

	// 发送状态更新事件
	callback(&model.V4Event{
		Type:    model.V4EventTypeStatusUpdated,
		Emoji:   emoji,
		Status:  status,
		Message: "当前状态已更新",
	})

	return map[string]interface{}{
		"message": "当前状态已更新",
		"emoji":   emoji,
		"status":  status,
	}, true // 结束循环
}

// executeSaveSchedule 执行保存时刻表
func (s *VoiceScheduleServiceV4) executeSaveSchedule(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	// 检查是否有待确认的时刻表
	if !session.HasPendingSchedule() {
		return map[string]string{"error": "没有待保存的时刻表"}, false
	}

	// 解析可见性
	visibility := model.VisibilityAllFriends
	if v, ok := args["visibility"].(string); ok && v != "" {
		visibility = model.ScheduleVisibility(v)
	}

	// 保存时刻表
	date := session.PendingDate
	if date.IsZero() {
		date = time.Now()
	}
	today := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	schedule := &model.StatusSchedule{
		UserID:       session.UserID,
		ScheduleDate: today,
		Items:        model.ScheduleItems(session.PendingSchedule),
		Status:       model.ScheduleStatusActive,
		Visibility:   visibility,
	}

	// 检查是否已存在
	existing, _ := s.scheduleRepo.GetActiveByUserAndDate(ctx, session.UserID, today)
	if existing != nil {
		schedule.ID = existing.ID
		if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
			return map[string]string{"error": "保存失败: " + err.Error()}, false
		}
	} else {
		if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
			return map[string]string{"error": "保存失败: " + err.Error()}, false
		}
	}

	// 发送保存成功事件
	callback(&model.V4Event{
		Type:    model.V4EventTypeScheduleSaved,
		Items:   session.PendingSchedule,
		Date:    today.Format("2006-01-02"),
		Message: "时刻表已保存",
	})

	// 更新会话状态
	session.CurrentSchedule = session.PendingSchedule
	session.ClearPendingSchedule()

	return map[string]interface{}{
		"message": "时刻表已保存",
		"date":    today.Format("2006-01-02"),
	}, true // 结束循环
}

// executeUpdatePreference 执行更新用户偏好设置
func (s *VoiceScheduleServiceV4) executeUpdatePreference(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	// 获取或创建偏好设置
	pref, err := s.scheduleRepo.GetUserPreference(ctx, session.UserID)
	if err != nil {
		// 用户没有偏好设置，创建新的
		pref = &model.UserSchedulePreference{
			UserID:            session.UserID,
			HidePastEvents:    false,
			DefaultVisibility: model.VisibilityAllFriends,
		}
	}

	// 更新偏好设置
	updated := false
	if hidePast, ok := args["hide_past_events"].(bool); ok {
		pref.HidePastEvents = hidePast
		updated = true
	}
	if visibility, ok := args["default_visibility"].(string); ok && visibility != "" {
		pref.DefaultVisibility = model.ScheduleVisibility(visibility)
		updated = true
	}

	if !updated {
		return map[string]string{"error": "没有提供任何偏好设置参数"}, false
	}

	// 保存偏好设置
	if err := s.scheduleRepo.UpsertUserPreference(ctx, pref); err != nil {
		return map[string]string{"error": "保存偏好设置失败: " + err.Error()}, false
	}

	// 发送更新成功事件
	callback(&model.V4Event{
		Type:    model.V4EventTypePreferenceUpdated,
		Message: "偏好设置已更新",
	})

	return map[string]interface{}{
		"message":          "偏好设置已更新",
		"hide_past_events": pref.HidePastEvents,
	}, true // 结束循环
}

// ========== 会话管理 ==========

// getOrCreateSession 获取或创建会话
func (s *VoiceScheduleServiceV4) getOrCreateSession(
	ctx context.Context,
	userID string,
	sessionID string,
) (*model.V4Session, bool) {
	// 尝试恢复现有会话
	if sessionID != "" {
		session, err := s.getSession(ctx, sessionID)
		if err == nil && session != nil && session.UserID == userID {
			return session, false
		}
	}

	// 创建新会话
	newSessionID := uuid.New().String()
	session := model.NewV4Session(userID, newSessionID)
	return session, true
}

// getSession 从 Redis 获取会话
func (s *VoiceScheduleServiceV4) getSession(ctx context.Context, sessionID string) (*model.V4Session, error) {
	if s.redisClient == nil {
		return nil, fmt.Errorf("Redis 未配置")
	}

	key := v4SessionKeyPrefix + sessionID
	data, err := s.redisClient.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var session model.V4Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// saveSession 保存会话到 Redis
func (s *VoiceScheduleServiceV4) saveSession(ctx context.Context, session *model.V4Session) error {
	if s.redisClient == nil {
		return fmt.Errorf("Redis 未配置")
	}

	key := v4SessionKeyPrefix + session.SessionID
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return s.redisClient.Set(ctx, key, string(data), v4SessionTTL)
}

// loadCurrentSchedule 加载用户当前时刻表
func (s *VoiceScheduleServiceV4) loadCurrentSchedule(ctx context.Context, session *model.V4Session) {
	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, session.UserID, todayDate)
	if err == nil && schedule != nil && len(schedule.Items) > 0 {
		session.CurrentSchedule = schedule.Items
		fmt.Printf("[V4] 加载用户时刻表: %d 个时段\n", len(schedule.Items))
	}
}

// ========== 辅助函数 ==========

// convertV4Message 将 V4Message 转换为 AgentMessage
func (s *VoiceScheduleServiceV4) convertV4Message(msg model.V4Message) agent.AgentMessage {
	am := agent.AgentMessage{
		Role:       msg.Role,
		Content:    msg.Content,
		ToolCallID: msg.ToolCallID,
		Name:       msg.Name,
	}

	// 转换 ToolCalls
	if len(msg.ToolCalls) > 0 {
		am.ToolCalls = make([]agent.ToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			am.ToolCalls[i] = agent.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: agent.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	return am
}

// convertToolCalls 将 agent.ToolCall 转换为 model.V4ToolCall
func (s *VoiceScheduleServiceV4) convertToolCalls(toolCalls []agent.ToolCall) []model.V4ToolCall {
	result := make([]model.V4ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = model.V4ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: model.V4ToolFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return result
}

// getCurrentTimeSlot 复用 agent.go 中的函数
func getCurrentTimeSlot(now time.Time) (string, string) {
	hour := now.Hour()
	minute := now.Minute()

	// 向下取整到30分钟
	if minute >= 30 {
		minute = 30
	} else {
		minute = 0
	}

	start := fmt.Sprintf("%02d:%02d", hour, minute)

	// 结束时间 +30 分钟
	endTime := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()).Add(30 * time.Minute)
	end := endTime.Format("15:04")

	return start, end
}

// ProcessVoiceInput 处理语音输入（V4 版本）
func (s *VoiceScheduleServiceV4) ProcessVoiceInput(
	ctx context.Context,
	userID string,
	sessionID string,
	audioData []byte,
	audioFormat string,
	callback func(event *model.V4Event),
) (*model.V4Session, error) {
	// 1. 语音识别
	if s.asrClient == nil || !s.asrClient.IsConfigured() {
		callback(&model.V4Event{
			Type:    model.V4EventTypeError,
			Message: "语音识别服务未配置",
		})
		return nil, fmt.Errorf("语音识别服务未配置")
	}

	// 获取 ASR token
	asrCtx, asrCancel := context.WithTimeout(ctx, 30*time.Second)
	defer asrCancel()

	token, err := s.asrClient.GetToken(asrCtx)
	if err != nil {
		callback(&model.V4Event{
			Type:    model.V4EventTypeError,
			Message: "获取语音识别 token 失败",
		})
		return nil, fmt.Errorf("获取 ASR token 失败: %w", err)
	}

	// 识别语音
	transcript, err := s.asrClient.RecognizeSpeechWithToken(asrCtx, audioData, audioFormat, token)
	if err != nil {
		callback(&model.V4Event{
			Type:    model.V4EventTypeError,
			Message: "语音识别失败",
		})
		return nil, fmt.Errorf("语音识别失败: %w", err)
	}

	if transcript == "" {
		callback(&model.V4Event{
			Type:    model.V4EventTypeError,
			Message: "未识别到语音内容",
		})
		return nil, fmt.Errorf("未识别到语音内容")
	}

	fmt.Printf("[V4] 语音识别结果: %s\n", transcript)

	// 2. 使用文本处理流程
	return s.ProcessTextInput(ctx, userID, sessionID, transcript, callback)
}

// ========== 好友相关工具执行函数 ==========

// executeQueryFriends 执行查询好友
func (s *VoiceScheduleServiceV4) executeQueryFriends(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	// 检查依赖
	if s.friendshipService == nil {
		return map[string]string{"error": "好友服务未配置"}, false
	}

	// 解析参数
	filterType, _ := args["filter_type"].(string)
	filterValue, _ := args["filter_value"].(string)
	limit := 10
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 50 {
			limit = 50
		}
	}

	// 获取好友列表
	friends, err := s.friendshipService.GetFriends(ctx, session.UserID)
	if err != nil {
		return map[string]string{"error": "获取好友列表失败: " + err.Error()}, false
	}

	// 提取好友 ID 列表
	friendIDs := make([]string, len(friends))
	for i, f := range friends {
		friendIDs[i] = f.User.ID
	}

	// 使用 memoryRepo.GetAnalysisCacheByUserIDs 批量获取好友状态
	// 这个方法从 user_analysis_cache 表获取数据，包含：
	// - availability_status: 有空/忙碌/可能有空
	// - availability_probability: 0-100
	// - life_status_emoji, life_status_label
	analysisCaches, err := s.memoryRepo.GetAnalysisCacheByUserIDs(ctx, friendIDs)
	if err != nil {
		fmt.Printf("[V4] 获取好友状态缓存失败: %v\n", err)
		analysisCaches = make(map[string]*model.AnalysisResult)
	}

	// 合并好友数据
	var result []model.V4FriendInfo
	for _, f := range friends {
		info := model.V4FriendInfo{
			ID:          f.User.ID,
			Name:        f.User.Nickname,
			Avatar:      f.User.Avatar,
			Probability: -1,
			Confidence:  "low",
		}

		// 从 user_analysis_cache 获取状态数据
		if analysis, ok := analysisCaches[f.User.ID]; ok && analysis != nil {
			// 有空概率
			info.Probability = analysis.Availability.Probability
			info.Confidence = analysis.Availability.Confidence

			// 生活状态（emoji + label 作为状态描述）
			if analysis.LifeStatus.Emoji != "" {
				info.Emoji = analysis.LifeStatus.Emoji
			}
			if analysis.LifeStatus.Label != "" {
				info.Status = analysis.LifeStatus.Label
			}

			// 有空状态文本（用于筛选）
			info.AvailabilityStatus = analysis.Availability.Status
		}

		// 获取好友位置（城市）- 仍然使用 agentService 获取实时位置
		if s.agentService != nil {
			status, _ := s.agentService.GetUserStatus(ctx, f.User.ID)
			if status != nil && status.City != "" {
				info.City = status.City
			}
		}

		result = append(result, info)
	}

	// 应用筛选
	var filtered []model.V4FriendInfo
	filterDesc := ""

	switch filterType {
	case "available":
		for _, f := range result {
			if filterValue == "free" {
				// 使用 availability_status 字段判断是否有空
				// "有空" 状态或概率 >= 60%
				if f.AvailabilityStatus == "有空" || f.Probability >= 60 {
					filtered = append(filtered, f)
				}
			} else if filterValue == "busy" {
				// "忙碌" 状态或概率 < 40%
				if f.AvailabilityStatus == "忙碌" || (f.Probability >= 0 && f.Probability < 40) {
					filtered = append(filtered, f)
				}
			} else if filterValue == "all" || filterValue == "" {
				filtered = append(filtered, f)
			}
		}
		if filterValue == "free" {
			filterDesc = "有空的好友"
		} else if filterValue == "busy" {
			filterDesc = "忙碌的好友"
		} else {
			filterDesc = "全部好友"
		}

	case "status":
		// 按生活状态筛选（匹配 life_status_label）
		for _, f := range result {
			if f.Status != "" && strings.Contains(f.Status, filterValue) {
				filtered = append(filtered, f)
			}
		}
		filterDesc = fmt.Sprintf("状态包含\"%s\"的好友", filterValue)

	case "location":
		for _, f := range result {
			if f.City != "" && strings.Contains(f.City, filterValue) {
				filtered = append(filtered, f)
			}
		}
		filterDesc = fmt.Sprintf("在%s的好友", filterValue)

	case "name":
		for _, f := range result {
			if strings.Contains(f.Name, filterValue) {
				filtered = append(filtered, f)
			}
		}
		filterDesc = fmt.Sprintf("名字包含\"%s\"的好友", filterValue)
		// 如果只找到一个好友，保存到 session 方便后续使用
		if len(filtered) == 1 {
			session.LastQueriedFriend = &filtered[0]
		}

	default: // "all" 或其他
		filtered = result
		filterDesc = "全部好友"
	}

	// 限制返回数量
	total := len(filtered)
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	// 发送好友结果事件
	callback(&model.V4Event{
		Type:          model.V4EventTypeFriendsResult,
		Friends:       filtered,
		Total:         total,
		FilterApplied: filterDesc,
		Message:       fmt.Sprintf("找到 %d 位%s", total, filterDesc),
	})

	return map[string]interface{}{
		"friends":        filtered,
		"total":          total,
		"filter_applied": filterDesc,
	}, false
}

// executeSendMessage 执行发送消息（生成预览）
func (s *VoiceScheduleServiceV4) executeSendMessage(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	// 解析参数
	friendID, _ := args["friend_id"].(string)
	friendName, _ := args["friend_name"].(string)
	message, _ := args["message"].(string)

	// 如果没有 friend_id，尝试从 session 中获取
	if friendID == "" && session.LastQueriedFriend != nil {
		friendID = session.LastQueriedFriend.ID
		friendName = session.LastQueriedFriend.Name
	}

	if friendID == "" {
		return map[string]string{"error": "请先指定要发送消息的好友"}, false
	}
	if message == "" {
		return map[string]string{"error": "消息内容不能为空"}, false
	}

	// 保存待发送消息到 session
	session.PendingMessage = &model.V4PendingMessage{
		FriendID:   friendID,
		FriendName: friendName,
		Message:    message,
	}

	// 发送消息预览事件
	callback(&model.V4Event{
		Type:            model.V4EventTypeMessagePreview,
		PendingMessage:  session.PendingMessage,
		Message:         fmt.Sprintf("准备发送消息给 %s: \"%s\"", friendName, message),
		AwaitingConfirm: true,
	})

	return map[string]interface{}{
		"preview": map[string]string{
			"to":      friendName,
			"message": message,
		},
		"awaiting_confirmation": true,
		"message":               "消息预览已生成，确认发送吗？",
	}, false
}

// executeCreateScheduleInvite 执行创建日程邀请（生成预览）
func (s *VoiceScheduleServiceV4) executeCreateScheduleInvite(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	// 解析参数
	friendID, _ := args["friend_id"].(string)
	friendName, _ := args["friend_name"].(string)
	date, _ := args["date"].(string)
	startTime, _ := args["start_time"].(string)
	endTime, _ := args["end_time"].(string)
	activity, _ := args["activity"].(string)
	location, _ := args["location"].(string)
	message, _ := args["message"].(string)

	// 如果没有 friend_id，尝试从 session 中获取
	if friendID == "" && session.LastQueriedFriend != nil {
		friendID = session.LastQueriedFriend.ID
		friendName = session.LastQueriedFriend.Name
	}

	if friendID == "" {
		return map[string]string{"error": "请先指定要邀请的好友"}, false
	}
	if date == "" || startTime == "" || activity == "" {
		return map[string]string{"error": "日期、时间和活动内容不能为空"}, false
	}

	// 如果没有指定结束时间，默认1小时后
	if endTime == "" {
		t, err := time.Parse("15:04", startTime)
		if err == nil {
			endTime = t.Add(1 * time.Hour).Format("15:04")
		} else {
			endTime = startTime // fallback
		}
	}

	// 保存待发送邀请到 session
	session.PendingInvite = &model.V4PendingInvite{
		FriendID:   friendID,
		FriendName: friendName,
		Date:       date,
		StartTime:  startTime,
		EndTime:    endTime,
		Activity:   activity,
		Location:   location,
		Message:    message,
	}

	// 构建时间范围描述
	timeRange := fmt.Sprintf("%s %s-%s", date, startTime, endTime)

	// 发送邀请预览事件
	callback(&model.V4Event{
		Type:            model.V4EventTypeInvitePreview,
		PendingInvite:   session.PendingInvite,
		Message:         fmt.Sprintf("准备邀请 %s %s %s", friendName, timeRange, activity),
		AwaitingConfirm: true,
	})

	return map[string]interface{}{
		"preview": map[string]interface{}{
			"to":         friendName,
			"date":       date,
			"time_range": fmt.Sprintf("%s-%s", startTime, endTime),
			"activity":   activity,
			"location":   location,
		},
		"awaiting_confirmation": true,
		"message":               "邀请预览已生成，确认发送吗？",
	}, false
}

// executeConfirmSend 执行确认发送
func (s *VoiceScheduleServiceV4) executeConfirmSend(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	// 检查是否有待发送的消息
	if session.HasPendingMessage() {
		return s.sendPendingMessage(ctx, session, callback)
	}

	// 检查是否有待发送的邀请
	if session.HasPendingInvite() {
		return s.sendPendingInvite(ctx, session, callback)
	}

	return map[string]string{"error": "没有待发送的消息或邀请"}, false
}

// sendPendingMessage 发送待确认的消息
func (s *VoiceScheduleServiceV4) sendPendingMessage(
	ctx context.Context,
	session *model.V4Session,
	callback func(event *model.V4Event),
) (interface{}, bool) {
	if s.conversationService == nil {
		return map[string]string{"error": "消息服务未配置"}, false
	}

	pending := session.PendingMessage

	// 获取或创建会话
	conv, err := s.conversationService.GetOrCreateConversation(ctx, session.UserID, pending.FriendID)
	if err != nil {
		return map[string]string{"error": "创建会话失败: " + err.Error()}, false
	}

	// 发送消息
	msgResp, err := s.conversationService.SendMessage(ctx, conv.ID, session.UserID, &SendMessageRequest{
		Type:    model.MessageTypeText,
		Content: pending.Message,
	})
	if err != nil {
		return map[string]string{"error": "发送消息失败: " + err.Error()}, false
	}

	// 清除待发送状态
	session.ClearPendingMessage()

	// 发送消息已发送事件
	callback(&model.V4Event{
		Type:      model.V4EventTypeMessageSent,
		MessageID: msgResp.ID,
		SentTo:    pending.FriendName,
		Message:   fmt.Sprintf("消息已发送给 %s", pending.FriendName),
	})

	return map[string]interface{}{
		"success":    true,
		"message_id": msgResp.ID,
		"sent_to":    pending.FriendName,
		"message":    fmt.Sprintf("已发送给 %s", pending.FriendName),
	}, true // 结束循环
}

// sendPendingInvite 发送待确认的日程邀请
func (s *VoiceScheduleServiceV4) sendPendingInvite(
	ctx context.Context,
	session *model.V4Session,
	callback func(event *model.V4Event),
) (interface{}, bool) {
	if s.conversationService == nil {
		return map[string]string{"error": "消息服务未配置"}, false
	}

	pending := session.PendingInvite

	// 获取或创建会话
	conv, err := s.conversationService.GetOrCreateConversation(ctx, session.UserID, pending.FriendID)
	if err != nil {
		return map[string]string{"error": "创建会话失败: " + err.Error()}, false
	}

	// 构建邀请卡片的 metadata
	inviteID := uuid.New().String()
	metadata := map[string]interface{}{
		"invite_id":   inviteID,
		"date":        pending.Date,
		"start_time":  pending.StartTime,
		"end_time":    pending.EndTime,
		"activity":    pending.Activity,
		"location":    pending.Location,
		"status":      "pending",
		"inviter_id":  session.UserID,
	}
	metadataJSON, _ := json.Marshal(metadata)

	// 构建邀请消息文本
	content := fmt.Sprintf("邀请你 %s %s-%s %s", pending.Date, pending.StartTime, pending.EndTime, pending.Activity)
	if pending.Location != "" {
		content += fmt.Sprintf("，地点: %s", pending.Location)
	}
	if pending.Message != "" {
		content += fmt.Sprintf("\n%s", pending.Message)
	}

	// 发送消息
	msgResp, err := s.conversationService.SendMessage(ctx, conv.ID, session.UserID, &SendMessageRequest{
		Type:     model.MessageTypeScheduleInvite,
		Content:  content,
		Metadata: metadataJSON,
	})
	if err != nil {
		return map[string]string{"error": "发送邀请失败: " + err.Error()}, false
	}

	// 清除待发送状态
	session.ClearPendingInvite()

	// 发送邀请已发送事件
	callback(&model.V4Event{
		Type:      model.V4EventTypeInviteSent,
		MessageID: msgResp.ID,
		SentTo:    pending.FriendName,
		Message:   fmt.Sprintf("日程邀请已发送给 %s", pending.FriendName),
	})

	return map[string]interface{}{
		"success":    true,
		"message_id": msgResp.ID,
		"sent_to":    pending.FriendName,
		"message":    fmt.Sprintf("邀请已发送给 %s", pending.FriendName),
	}, true // 结束循环
}

// ========== 扩展工具执行函数 ==========

// executeMatchContacts 执行匹配通讯录
func (s *VoiceScheduleServiceV4) executeMatchContacts(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	// 检查依赖
	if s.contactService == nil {
		return map[string]string{"error": "通讯录服务未配置"}, false
	}

	// 从 session 获取手机号哈希（由客户端上传）
	if len(session.ContactHashes) == 0 {
		return map[string]string{
			"error":   "请先上传通讯录数据",
			"message": "需要客户端先上传手机号哈希列表",
		}, false
	}

	// 调用通讯录匹配服务
	result, err := s.contactService.MatchContacts(ctx, session.UserID, session.ContactHashes)
	if err != nil {
		return map[string]string{"error": "匹配失败: " + err.Error()}, false
	}

	// 缓存匹配结果到 session（供后续 add_friend 使用）
	session.MatchedContacts = result.Matches

	// 统计已是好友和可添加的数量
	alreadyFriends := 0
	canAdd := 0
	for _, m := range result.Matches {
		if m.IsFriend {
			alreadyFriends++
		} else {
			canAdd++
		}
	}

	return map[string]interface{}{
		"matches":         result.Matches,
		"total_found":     result.TotalFound,
		"already_friends": alreadyFriends,
		"can_add":         canAdd,
		"message":         fmt.Sprintf("在通讯录中找到 %d 位用户，其中 %d 位已是好友，%d 位可添加", result.TotalFound, alreadyFriends, canAdd),
	}, false
}

// executeAddFriend 执行添加好友
func (s *VoiceScheduleServiceV4) executeAddFriend(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	mode, _ := args["mode"].(string)

	switch mode {
	case "phone":
		// 通过手机号添加（发送好友请求）
		phone, _ := args["phone"].(string)
		if phone == "" {
			return map[string]string{"error": "请提供手机号"}, false
		}
		message, _ := args["message"].(string)

		// 发送好友请求
		if s.friendshipService == nil {
			return map[string]string{"error": "好友服务未配置"}, false
		}

		result, err := s.friendshipService.SendFriendRequest(ctx, session.UserID, phone, message)
		if err != nil {
			return map[string]string{"error": "添加失败: " + err.Error()}, false
		}

		// 构建返回消息
		statusMsg := ""
		switch result.Status {
		case "PENDING":
			statusMsg = "好友请求已发送，等待对方确认"
		case "ACCEPTED":
			statusMsg = "对方之前也想加你，已自动成为好友"
		case "ALREADY_FRIENDS":
			statusMsg = "已经是好友了"
		case "ALREADY_REQUESTED":
			statusMsg = "已经发送过请求，等待对方确认"
		}

		return map[string]interface{}{
			"request_id": result.RequestID,
			"user":       result.User,
			"status":     result.Status,
			"message":    statusMsg,
		}, false

	case "batch":
		// 批量添加通讯录匹配用户
		if s.contactService == nil {
			return map[string]string{"error": "通讯录服务未配置"}, false
		}

		// 获取要添加的用户ID
		var userIDs []string
		if ids, ok := args["user_ids"].([]interface{}); ok && len(ids) > 0 {
			for _, id := range ids {
				if s, ok := id.(string); ok {
					userIDs = append(userIDs, s)
				}
			}
		} else if len(session.MatchedContacts) > 0 {
			// 使用 session 中的匹配结果
			for _, m := range session.MatchedContacts {
				if !m.IsFriend && m.User != nil {
					userIDs = append(userIDs, m.User.ID)
				}
			}
		}

		if len(userIDs) == 0 {
			return map[string]string{"error": "没有可添加的好友"}, false
		}

		// 批量添加
		result, err := s.contactService.BatchAddFriends(ctx, session.UserID, userIDs)
		if err != nil {
			return map[string]string{"error": "批量添加失败: " + err.Error()}, false
		}

		return map[string]interface{}{
			"added":   result.Added,
			"skipped": result.Skipped,
			"message": fmt.Sprintf("成功添加 %d 位好友，跳过 %d 位（已是好友）", result.Added, result.Skipped),
		}, false

	case "id":
		// 通过用户ID直接添加好友
		if s.friendshipService == nil {
			return map[string]string{"error": "好友服务未配置"}, false
		}

		var userIDs []string
		if ids, ok := args["user_ids"].([]interface{}); ok {
			for _, id := range ids {
				if s, ok := id.(string); ok {
					userIDs = append(userIDs, s)
				}
			}
		}

		if len(userIDs) == 0 {
			return map[string]string{"error": "请提供用户ID"}, false
		}

		// 直接添加好友（双向关系）
		added := 0
		skipped := 0
		for _, userID := range userIDs {
			err := s.friendshipService.AddFriend(ctx, session.UserID, userID, model.FriendshipSourceManual)
			if err != nil {
				fmt.Printf("[V4] 添加好友失败 %s: %v\n", userID, err)
				skipped++
			} else {
				added++
			}
		}

		return map[string]interface{}{
			"added":   added,
			"skipped": skipped,
			"message": fmt.Sprintf("成功添加 %d 位好友", added),
		}, false

	default:
		return map[string]string{"error": "无效的添加模式，支持: phone, batch, id"}, false
	}
}

// executeQueryDeviceData 执行查询设备数据
func (s *VoiceScheduleServiceV4) executeQueryDeviceData(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	dataType, _ := args["data_type"].(string)
	if dataType == "" {
		dataType = "all"
	}

	result := make(map[string]interface{})

	// 从 Redis 获取实时状态
	var realtimeStatus *model.UserRealtimeStatus
	if s.agentService != nil {
		realtimeStatus, _ = s.agentService.GetUserStatus(ctx, session.UserID)
	}

	// 从 user_analysis_cache 获取分析结果
	var analysisResult *model.AnalysisResult
	if s.memoryRepo != nil {
		analysisResult, _ = s.memoryRepo.GetAnalysisCache(ctx, session.UserID)
	}

	// 根据 data_type 组装返回数据
	switch dataType {
	case "location":
		if realtimeStatus != nil {
			result["location"] = map[string]interface{}{
				"place_type":     realtimeStatus.Location.PlaceType,
				"place_name":     "", // TODO: 从反向地理编码获取
				"city":           realtimeStatus.City,
				"at_place_since": realtimeStatus.Location.AtPlaceSinceMinutes,
			}
		} else {
			result["location"] = map[string]string{"message": "暂无位置数据"}
		}

	case "calendar":
		// TODO: 从日历数据收集器获取
		result["calendar"] = map[string]interface{}{
			"has_current_event": false,
			"message":           "暂无日历数据",
		}

	case "screen":
		if realtimeStatus != nil {
			result["screen"] = map[string]interface{}{
				"is_active":        realtimeStatus.Screen.IsActive,
				"activity_type":    realtimeStatus.Screen.ActivityType,
				"session_duration": realtimeStatus.Screen.SessionDurationMinutes,
			}
		} else {
			result["screen"] = map[string]string{"message": "暂无屏幕数据"}
		}

	case "device":
		// TODO: 从设备状态收集器获取
		result["device"] = map[string]interface{}{
			"message": "暂无设备状态数据",
		}

	case "status":
		if analysisResult != nil {
			result["status"] = map[string]interface{}{
				"availability": analysisResult.Availability.Status,
				"probability":  analysisResult.Availability.Probability,
				"emoji":        analysisResult.LifeStatus.Emoji,
				"life_status":  analysisResult.LifeStatus.Label,
			}
		} else {
			result["status"] = map[string]string{"message": "暂无状态分析数据"}
		}

	case "all":
		fallthrough
	default:
		// 返回所有可用数据
		if realtimeStatus != nil {
			result["location"] = map[string]interface{}{
				"place_type":     realtimeStatus.Location.PlaceType,
				"city":           realtimeStatus.City,
				"at_place_since": realtimeStatus.Location.AtPlaceSinceMinutes,
			}
			result["screen"] = map[string]interface{}{
				"is_active":        realtimeStatus.Screen.IsActive,
				"activity_type":    realtimeStatus.Screen.ActivityType,
				"session_duration": realtimeStatus.Screen.SessionDurationMinutes,
			}
		}
		if analysisResult != nil {
			result["status"] = map[string]interface{}{
				"availability": analysisResult.Availability.Status,
				"probability":  analysisResult.Availability.Probability,
				"emoji":        analysisResult.LifeStatus.Emoji,
				"life_status":  analysisResult.LifeStatus.Label,
			}
		}
	}

	if realtimeStatus != nil {
		result["updated_at"] = realtimeStatus.UpdatedAt.Unix()
	}

	return result, false
}

// executeQueryMemory 执行查询记忆
func (s *VoiceScheduleServiceV4) executeQueryMemory(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	memoryType, _ := args["memory_type"].(string)
	if memoryType == "" {
		memoryType = "all"
	}

	result := make(map[string]interface{})

	// 获取用户记忆文档
	var memDoc *model.UserMemoryDocument
	if s.memoryLearningService != nil {
		memDoc, _ = s.memoryLearningService.GetUserMemoryDocument(ctx, session.UserID)
	}

	// 获取核心记忆
	var coreMemory *model.CoreMemory
	if s.memoryRepo != nil {
		coreMemory, _ = s.memoryRepo.GetCoreMemory(ctx, session.UserID)
	}

	switch memoryType {
	case "patterns":
		if memDoc != nil && len(memDoc.SchedulePatterns) > 0 {
			patterns := make([]map[string]interface{}, 0, len(memDoc.SchedulePatterns))
			for _, p := range memDoc.SchedulePatterns {
				patterns = append(patterns, map[string]interface{}{
					"pattern":    p.Pattern,
					"confidence": p.Confidence,
				})
			}
			result["patterns"] = patterns
		} else {
			result["patterns"] = []interface{}{}
			result["message"] = "暂无行为模式记录"
		}

	case "preferences":
		if memDoc != nil {
			result["preferences"] = map[string]interface{}{
				"hide_past_events":   memDoc.Preferences.HidePastEvents,
				"default_view_days":  memDoc.Preferences.DefaultViewDays,
				"preferred_timezone": memDoc.Preferences.PreferredTimezone,
				"language":           memDoc.Preferences.Language,
			}
		} else {
			result["preferences"] = map[string]interface{}{
				"message": "使用默认偏好设置",
			}
		}

	case "facts":
		if memDoc != nil && len(memDoc.KeyFacts) > 0 {
			facts := make([]map[string]interface{}, 0, len(memDoc.KeyFacts))
			for _, f := range memDoc.KeyFacts {
				facts = append(facts, map[string]interface{}{
					"fact":       f.Fact,
					"source":     f.Source,
					"learned_at": f.LearnedAt,
				})
			}
			result["facts"] = facts
		} else {
			result["facts"] = []interface{}{}
			result["message"] = "暂无关键事实记录"
		}

	case "sessions":
		if memDoc != nil && len(memDoc.SessionSummaries) > 0 {
			sessions := make([]map[string]interface{}, 0, len(memDoc.SessionSummaries))
			maxSessions := 3
			for i, s := range memDoc.SessionSummaries {
				if i >= maxSessions {
					break
				}
				sessions = append(sessions, map[string]interface{}{
					"date":    s.Date,
					"summary": s.Summary,
				})
			}
			result["sessions"] = sessions
		} else {
			result["sessions"] = []interface{}{}
			result["message"] = "暂无历史会话记录"
		}

	case "core":
		if coreMemory != nil {
			result["core"] = map[string]interface{}{
				"behavior_insights":    coreMemory.BehaviorInsights,
				"time_patterns":        coreMemory.TimePatterns,
				"location_preferences": coreMemory.LocationPreferences,
				"social_tendency":      coreMemory.SocialTendency,
				"confidence_score":     coreMemory.ConfidenceScore,
			}
		} else {
			result["core"] = map[string]interface{}{
				"message": "暂无核心记忆洞察",
			}
		}

	case "all":
		fallthrough
	default:
		// 返回所有记忆
		if memDoc != nil {
			if len(memDoc.SchedulePatterns) > 0 {
				patterns := make([]map[string]interface{}, 0)
				for _, p := range memDoc.SchedulePatterns {
					patterns = append(patterns, map[string]interface{}{
						"pattern":    p.Pattern,
						"confidence": p.Confidence,
					})
				}
				result["patterns"] = patterns
			}
			result["preferences"] = map[string]interface{}{
				"hide_past_events":   memDoc.Preferences.HidePastEvents,
				"default_view_days":  memDoc.Preferences.DefaultViewDays,
				"preferred_timezone": memDoc.Preferences.PreferredTimezone,
				"language":           memDoc.Preferences.Language,
			}
			if len(memDoc.KeyFacts) > 0 {
				facts := make([]map[string]interface{}, 0)
				for _, f := range memDoc.KeyFacts {
					facts = append(facts, map[string]interface{}{
						"fact":   f.Fact,
						"source": f.Source,
					})
				}
				result["facts"] = facts
			}
		}
		if coreMemory != nil {
			result["core"] = map[string]interface{}{
				"behavior_insights":    coreMemory.BehaviorInsights,
				"time_patterns":        coreMemory.TimePatterns,
				"location_preferences": coreMemory.LocationPreferences,
				"social_tendency":      coreMemory.SocialTendency,
			}
		}
	}

	return result, false
}

// executeUpdateMemory 执行更新记忆
func (s *VoiceScheduleServiceV4) executeUpdateMemory(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	if s.memoryLearningService == nil {
		return map[string]string{"error": "记忆服务未配置"}, false
	}

	updateType, _ := args["update_type"].(string)

	switch updateType {
	case "fact":
		content, _ := args["content"].(string)
		if content == "" {
			return map[string]string{"error": "请提供要记住的内容"}, false
		}

		err := s.memoryLearningService.AddKeyFact(ctx, session.UserID, content, "voice_agent")
		if err != nil {
			return map[string]string{"error": "记录失败: " + err.Error()}, false
		}

		return map[string]interface{}{
			"success":       true,
			"message":       fmt.Sprintf("好的，我记住了：%s", content),
			"updated_field": "key_facts",
		}, false

	case "pattern":
		content, _ := args["content"].(string)
		if content == "" {
			return map[string]string{"error": "请提供行为模式内容"}, false
		}

		confidence := 0.8
		if c, ok := args["confidence"].(float64); ok && c > 0 && c <= 1 {
			confidence = c
		}

		err := s.memoryLearningService.AddSchedulePattern(ctx, session.UserID, content, confidence, session.SessionID)
		if err != nil {
			return map[string]string{"error": "记录失败: " + err.Error()}, false
		}

		return map[string]interface{}{
			"success":       true,
			"message":       fmt.Sprintf("好的，我记住了你的习惯：%s", content),
			"updated_field": "schedule_patterns",
		}, false

	case "preference":
		key, _ := args["preference_key"].(string)
		valueStr, _ := args["preference_value"].(string)

		if key == "" {
			return map[string]string{"error": "请提供偏好键名"}, false
		}

		// 转换值类型
		var value interface{}
		switch key {
		case "hide_past_events":
			value = valueStr == "true"
		case "default_view_days":
			var days int
			fmt.Sscanf(valueStr, "%d", &days)
			value = days
		default:
			value = valueStr
		}

		err := s.memoryLearningService.UpdatePreference(ctx, session.UserID, key, value)
		if err != nil {
			return map[string]string{"error": "更新失败: " + err.Error()}, false
		}

		return map[string]interface{}{
			"success":       true,
			"message":       "偏好设置已更新",
			"updated_field": key,
		}, false

	default:
		return map[string]string{"error": "无效的更新类型，支持: fact, pattern, preference"}, false
	}
}

// executeGetBehaviorSuggestion 执行获取行为建议
func (s *VoiceScheduleServiceV4) executeGetBehaviorSuggestion(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	suggestionContext, _ := args["context"].(string)
	if suggestionContext == "" {
		suggestionContext = "current"
	}
	timeRange, _ := args["time_range"].(string)
	if timeRange == "" {
		timeRange = "now"
	}

	// 保存上下文
	session.LastSuggestionContext = suggestionContext

	// 收集上下文数据
	var deviceData map[string]interface{}
	var memoryData map[string]interface{}
	var friendsData []model.V4FriendInfo

	// 获取设备数据
	deviceResult, _ := s.executeQueryDeviceData(ctx, session, map[string]interface{}{"data_type": "all"}, callback)
	if data, ok := deviceResult.(map[string]interface{}); ok {
		deviceData = data
	}

	// 获取记忆数据
	memoryResult, _ := s.executeQueryMemory(ctx, session, map[string]interface{}{"memory_type": "all"}, callback)
	if data, ok := memoryResult.(map[string]interface{}); ok {
		memoryData = data
	}

	// 如果是社交建议，获取有空的好友
	if suggestionContext == "social" {
		friendsResult, _ := s.executeQueryFriends(ctx, session, map[string]interface{}{
			"filter_type":  "available",
			"filter_value": "free",
			"limit":        5.0,
		}, callback)
		if data, ok := friendsResult.(map[string]interface{}); ok {
			if friends, ok := data["friends"].([]model.V4FriendInfo); ok {
				friendsData = friends
			}
		}
	}

	// 构建建议
	suggestions := make([]map[string]interface{}, 0)

	// 基于设备数据生成建议
	if deviceData != nil {
		if status, ok := deviceData["status"].(map[string]interface{}); ok {
			probability, _ := status["probability"].(int)
			lifeStatus, _ := status["life_status"].(string)

			if probability > 60 {
				suggestions = append(suggestions, map[string]interface{}{
					"type":        "social",
					"title":       "约个朋友",
					"description": "你现在看起来有空，可以约朋友聊聊",
					"reason":      fmt.Sprintf("当前状态: %s, 有空概率 %d%%", lifeStatus, probability),
					"confidence":  "medium",
				})
			}
		}

		if screen, ok := deviceData["screen"].(map[string]interface{}); ok {
			duration, _ := screen["session_duration"].(int)
			activityType, _ := screen["activity_type"].(string)

			if duration > 60 && activityType == "entertainment" {
				suggestions = append(suggestions, map[string]interface{}{
					"type":        "activity",
					"title":       "休息一下",
					"description": "已经娱乐一个多小时了，起来活动活动",
					"reason":      fmt.Sprintf("屏幕使用时长 %d 分钟，活动类型: %s", duration, activityType),
					"confidence":  "high",
				})
			}
		}
	}

	// 基于社交数据生成建议
	if suggestionContext == "social" && len(friendsData) > 0 {
		friend := friendsData[0]
		suggestions = append(suggestions, map[string]interface{}{
			"type":        "social",
			"title":       fmt.Sprintf("可以约 %s", friend.Name),
			"description": fmt.Sprintf("%s 现在%s，有空概率 %d%%", friend.Name, friend.Status, friend.Probability),
			"reason":      "基于好友状态分析",
			"confidence":  friend.Confidence,
		})
	}

	// 如果没有具体建议，给出通用建议
	if len(suggestions) == 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":        "general",
			"title":       "继续当前活动",
			"description": "没有特别的建议，继续你正在做的事情",
			"reason":      "数据不足以给出具体建议",
			"confidence":  "low",
		})
	}

	return map[string]interface{}{
		"suggestions": suggestions,
		"based_on": map[string]interface{}{
			"device_data": deviceData != nil,
			"memory_data": memoryData != nil,
			"friends":     len(friendsData),
			"context":     suggestionContext,
			"time_range":  timeRange,
		},
	}, false
}
