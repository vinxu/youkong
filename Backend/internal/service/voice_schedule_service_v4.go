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
) *VoiceScheduleServiceV4 {
	svc := &VoiceScheduleServiceV4{
		scheduleRepo:       scheduleRepo,
		memoryRepo:         memoryRepo,
		memoryDocRepo:      memoryDocRepo,
		userProfileService: userProfileService,
		redisClient:        redisClient,
		asrClient:          asrClient,
		tools:              agent.V4ScheduleTools(),
	}

	// 创建 LLM 适配器
	if llmAPIKey != "" {
		svc.llmAdapter = agent.NewLLMAdapter(&agent.LLMAdapterConfig{
			APIKey: llmAPIKey,
			Model:  "qwen-plus", // 使用 qwen-plus，平衡速度和质量
		})

		// 创建记忆学习服务
		svc.memoryLearningService = NewMemoryLearningService(memoryDocRepo, svc.llmAdapter)
	}

	return svc
}

// ProcessTextInput 处理文本输入（主入口）
// 优化版：快速路径 + 阶段暴露 + 减少循环
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

	// 4. 快速路径判断：简单闲聊不需要工具
	needTools := s.needsToolCalling(text, session)

	// 5. 发送阶段：理解请求
	callback(&model.V4Event{
		Type:    model.V4EventTypePhase,
		Phase:   "understanding",
		Message: "正在理解你的请求...",
	})

	// 6. 主循环（优化版）
	var err error
	if needTools {
		err = s.processLoopWithTools(ctx, session, callback)
	} else {
		err = s.processSimpleChat(ctx, session, callback)
	}

	if err != nil {
		callback(&model.V4Event{
			Type:    model.V4EventTypeError,
			Message: err.Error(),
		})
		return session, err
	}

	// 7. 保存会话
	if saveErr := s.saveSession(ctx, session); saveErr != nil {
		fmt.Printf("[V4] 保存会话失败: %v\n", saveErr)
	}

	// 8. 异步触发记忆学习（会话结束或消息数达到阈值时）
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

// needsToolCalling 快速判断是否需要工具调用
// 用于减少不必要的工具传递，加速简单闲聊场景
func (s *VoiceScheduleServiceV4) needsToolCalling(text string, session *model.V4Session) bool {
	// 简单闲聊模式（不需要工具）
	chatPatterns := []string{
		"你好", "您好", "嗨", "hi", "hello",
		"谢谢", "感谢", "多谢", "thanks",
		"再见", "拜拜", "bye",
		"你是谁", "你能做什么", "帮助",
		"天气", "今天天气", "明天天气",
		"😊", "😄", "👍", "🙏",
	}

	textLower := strings.ToLower(strings.TrimSpace(text))

	// 检查是否匹配简单闲聊
	for _, pattern := range chatPatterns {
		if textLower == pattern || strings.HasPrefix(textLower, pattern) {
			fmt.Printf("[V4] 快速路径: 简单闲聊 '%s'\n", text)
			return false
		}
	}

	// 需要工具的模式
	toolPatterns := []string{
		// 时刻表操作
		"安排", "时刻", "日程", "计划", "行程",
		"几点", "什么时候", "上午", "下午", "晚上", "明天", "后天", "周",
		"开会", "会议", "健身", "吃饭", "睡觉", "工作", "休息",
		// 状态更新
		"现在", "正在", "我在", "在做",
		"累", "忙", "闲", "有空", "没空",
		// 查询操作
		"看看", "查看", "调出", "显示", "列出",
		// 确认操作
		"好的", "可以", "确认", "保存", "行", "ok", "没问题",
		// 修改操作
		"改", "换", "调整", "取消", "删除",
	}

	for _, pattern := range toolPatterns {
		if strings.Contains(textLower, pattern) {
			fmt.Printf("[V4] 需要工具: 匹配 '%s'\n", pattern)
			return true
		}
	}

	// 如果有待确认的时刻表，可能需要处理确认
	if session.HasPendingSchedule() {
		fmt.Printf("[V4] 需要工具: 有待确认时刻表\n")
		return true
	}

	// 默认：较长文本可能需要工具处理
	if len(text) > 10 {
		fmt.Printf("[V4] 需要工具: 文本较长\n")
		return true
	}

	fmt.Printf("[V4] 快速路径: 默认闲聊\n")
	return false
}

// processSimpleChat 处理简单闲聊（不需要工具）
func (s *VoiceScheduleServiceV4) processSimpleChat(
	ctx context.Context,
	session *model.V4Session,
	callback func(event *model.V4Event),
) error {
	// 发送阶段：生成回复
	callback(&model.V4Event{
		Type:    model.V4EventTypePhase,
		Phase:   "generating",
		Message: "正在生成回复...",
	})

	// 构建消息（简化版 Prompt，不传工具）
	messages := s.buildMessagesSimple(session)

	// 调用 LLM（不传工具，更快）
	response, err := s.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
		Messages:    messages,
		Tools:       nil, // 不传工具
		Temperature: 0.7,
	})
	if err != nil {
		return fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 发送回复
	if response.Content != "" {
		callback(&model.V4Event{
			Type:    model.V4EventTypeChat,
			Message: response.Content,
		})
		session.AddMessage("assistant", response.Content)
	}

	return nil
}

// buildMessagesSimple 构建简化版消息（用于闲聊）
func (s *VoiceScheduleServiceV4) buildMessagesSimple(session *model.V4Session) []agent.AgentMessage {
	messages := []agent.AgentMessage{
		agent.NewSystemMessage("你是一个友好的时刻表助手。简洁友好地回复用户。"),
	}

	// 只添加最近 3 轮对话
	recentMessages := session.GetRecentMessages(3)
	for _, msg := range recentMessages {
		messages = append(messages, s.convertV4Message(msg))
	}

	return messages
}

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

		// 2. 调用 LLM
		response, err := s.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
			Messages:    messages,
			Tools:       s.tools,
			Temperature: 0.7,
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

// buildMessages 构建 LLM 消息列表
func (s *VoiceScheduleServiceV4) buildMessages(session *model.V4Session) []agent.AgentMessage {
	// 构建系统提示词（包含用户记忆）
	systemPrompt := s.buildSystemPromptWithMemory(context.Background(), session)

	messages := []agent.AgentMessage{
		agent.NewSystemMessage(systemPrompt),
	}

	// 添加会话历史（最多保留最近 10 轮）
	recentMessages := session.GetRecentMessages(10)
	for _, msg := range recentMessages {
		messages = append(messages, s.convertV4Message(msg))
	}

	return messages
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

// buildSystemPrompt 构建精简的系统提示词（优化版 ~800 字符）
// 优化：减少 token 数量，加快首 token 延迟
func (s *VoiceScheduleServiceV4) buildSystemPrompt(session *model.V4Session) string {
	now := time.Now()
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	var sb strings.Builder

	// ========== 角色 + 时间（合并） ==========
	sb.WriteString("你是时刻表助手，像朋友聊天一样帮用户管理日程。\n\n")

	// 当前时间
	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	sb.WriteString(fmt.Sprintf("【时间】%s %s %02d:%02d | 今天=%s 明天=%s\n\n",
		today, weekdays[now.Weekday()], now.Hour(), now.Minute(), today, tomorrow))

	// ========== 用户当前时刻表（精简格式）==========
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
		sb.WriteString("【⚠️待确认】")
		for i, item := range session.PendingSchedule {
			if i > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(fmt.Sprintf("%s-%s %s%s", item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
		sb.WriteString(" (日期:" + session.PendingDate.Format("01-02") + ")\n\n")
	}

	// ========== 工具决策（精简版）==========
	sb.WriteString(`【工具选择】
- 查看安排/调出来 → get_schedule
- 新建日程 → update_schedule(operation="create")
- 修改某时段 → update_schedule(operation="modify") ⚠️修改时只传变更的条目，系统会自动保留其他时段
- 替换全部 → update_schedule(operation="replace")
- 我现在很累/在工作 → update_current_status
- 好的/确认/保存 → save_schedule（有待确认时）

【示例】
Q:"看看今天安排" → get_schedule(date="` + today + `")
Q:"下午3点开会" → update_schedule(operation="create",items=[{start:"15:00",end:"17:00",emoji:"💼",status:"开会"}])
Q:"把12点到1点改成去医院" → update_schedule(operation="modify",items=[{start:"12:00",end:"13:00",emoji:"🏥",status:"去医院"}])
Q:"好的" → save_schedule()
Q:"我在加班" → update_current_status(emoji="💼",status="加班中")

【规则】回复简洁，emoji匹配活动(💼工作 🍽️吃饭 😴睡觉 🏃运动 🏥医院)，时间模糊时先确认
`)

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
		"get_schedule":          "正在查询时刻表...",
		"update_schedule":       "正在生成时刻表预览...",
		"update_current_status": "正在更新当前状态...",
		"save_schedule":         "正在保存时刻表...",
		"update_preference":     "正在更新偏好设置...",
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
// replaceOverlap: true=更新重叠时段, false=保留现有（仅添加不冲突的）
func (s *VoiceScheduleServiceV4) mergeScheduleItems(existing, new []model.ScheduleItem, replaceOverlap bool) []model.ScheduleItem {
	if len(existing) == 0 {
		return new
	}
	if len(new) == 0 {
		return existing
	}

	// 创建结果切片
	result := make([]model.ScheduleItem, 0, len(existing)+len(new))

	// 遍历现有条目，检查是否被新条目覆盖
	for _, existItem := range existing {
		replaced := false
		for _, newItem := range new {
			if s.isTimeOverlap(existItem.StartTime, existItem.EndTime, newItem.StartTime, newItem.EndTime) {
				if replaceOverlap {
					// 修改模式：用新条目替换重叠的旧条目
					replaced = true
					break
				}
				// 添加模式：跳过冲突的新条目（保留现有）
			}
		}
		if !replaced {
			result = append(result, existItem)
		}
	}

	// 添加新条目
	for _, newItem := range new {
		// 检查是否与现有结果冲突
		conflict := false
		if !replaceOverlap {
			for _, existItem := range result {
				if s.isTimeOverlap(existItem.StartTime, existItem.EndTime, newItem.StartTime, newItem.EndTime) {
					conflict = true
					break
				}
			}
		}
		if !conflict {
			result = append(result, newItem)
		}
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
