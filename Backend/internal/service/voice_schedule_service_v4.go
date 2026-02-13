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
	v4MaxLoops         = 10                   // 最大循环次数
	v4SessionTTL       = 10 * time.Minute     // 会话过期时间
	v4SessionKeyPrefix = "v4_voice_session:"  // Redis key 前缀
	v4SessionLockPrefix = "v4_session_lock:"  // Session 锁前缀
	v4SessionLockTTL    = 60 * time.Second    // 锁超时（防死锁）
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

	// ========== 好友相关依赖 ==========
	friendshipService   *FriendshipService
	conversationService *ConversationService
	agentService        *AgentService

	// ========== 扩展功能依赖 ==========
	contactService *ContactService
	bookingService *BookingService
}

// SetBookingService 设置预约服务（延迟注入，避免循环依赖）
func (s *VoiceScheduleServiceV4) SetBookingService(bs *BookingService) {
	s.bookingService = bs
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

// getToolsForSession 根据 session 状态动态选择工具集
// 基础 8 工具始终加载，其余工具按条件加载（LLM 看不到的工具不会误选）
func (s *VoiceScheduleServiceV4) getToolsForSession(session *model.V4Session) []*agent.Tool {
	// 推断模式：使用推断专用版 set_status（优化描述引导具体输出）
	if session.InferenceMode {
		return []*agent.Tool{agent.V4InferenceSetStatusTool()}
	}

	tools := agent.V4BaseTools() // 基础 8 个

	// 条件加载：有安排时暴露删除工具
	if len(session.CurrentSchedule) > 0 || len(session.TomorrowSchedule) > 0 || len(session.TargetDateSchedule) > 0 {
		tools = append(tools, agent.V4DeleteScheduleTool())
	}

	// 条件加载：有好友上下文时暴露删除好友工具
	if session.LastQueriedFriend != nil {
		tools = append(tools, agent.V4RemoveFriendTool())
	}

	// 条件加载：有通讯录数据时（已有逻辑）
	if len(session.ContactHashes) > 0 || len(session.MatchedContacts) > 0 {
		tools = append(tools, agent.V4ContactTools()...) // match_contacts, add_friend
	}

	return tools
}

// ProcessTextInput 处理文本输入（公共入口，带 session 锁）
// sensorData 可选，一键生成时传入传感器数据，注入到系统提示词
func (s *VoiceScheduleServiceV4) ProcessTextInput(
	ctx context.Context,
	userID string,
	sessionID string,
	text string,
	sensorData *model.ExtendedStatusReportRequest,
	callback func(event *model.V4Event),
) (*model.V4Session, error) {
	// 安全化 callback
	if callback == nil {
		callback = func(event *model.V4Event) {}
	}
	// 获取 session 锁（防止同一 session 并发处理导致状态丢失）
	if sessionID != "" && !s.acquireSessionLock(ctx, sessionID) {
		callback(&model.V4Event{
			Type:    model.V4EventTypeChat,
			Message: "正在处理上一条消息，请稍后再试",
		})
		return nil, fmt.Errorf("session %s is locked", sessionID)
	}
	if sessionID != "" {
		defer s.releaseSessionLock(ctx, sessionID)
	}
	return s.processTextInputInternal(ctx, userID, sessionID, text, sensorData, callback)
}

// InferStatus V4 推断模式入口（替代 V3 StatusInferenceAgent）
// 创建临时 session，只暴露 set_status 工具，强制 LLM 根据数据推断状态
func (s *VoiceScheduleServiceV4) InferStatus(
	ctx context.Context,
	userID string,
	sensorData *model.ExtendedStatusReportRequest,
	inferenceContext string,
	callback func(event *model.V4Event),
) (*model.InferenceResponse, error) {
	if s.llmAdapter == nil {
		return nil, fmt.Errorf("LLM 未配置")
	}

	// 安全化 callback：nil 转为 noop，避免下游 nil pointer panic
	if callback == nil {
		callback = func(event *model.V4Event) {}
	}

	// 缓存传感器数据
	if sensorData != nil {
		s.cacheSensorDataForInference(ctx, userID, sensorData)
	}

	// 创建临时推断 session（不持久化到 Redis）
	session := &model.V4Session{
		UserID:           userID,
		SessionID:        fmt.Sprintf("infer_%s_%d", userID, time.Now().UnixMilli()),
		CreatedAt:        time.Now(),
		InferenceMode:    true,
		InferenceContext: inferenceContext,
		SensorData:       sensorData,
	}

	// 加载今日安排到 session（供系统提示词展示）
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, userID, today); err == nil && schedule != nil {
		session.CurrentSchedule = schedule.Items
	}

	// 添加推断请求作为用户消息
	session.AddMessage("user", "请根据上述数据推断我此刻的状态")

	// 运行 V4 主循环（只有 set_status 工具可用）
	// executeUpdateCurrentStatus 会将结果保存到 session.InferenceResult
	if err := s.processLoopWithTools(ctx, session, callback); err != nil {
		return nil, fmt.Errorf("推断循环失败: %w", err)
	}

	// 从 session 获取推断结果
	inferResult := session.InferenceResult
	if inferResult == nil {
		// LLM 没有调用 set_status，返回默认结果
		inferResult = &model.CurrentStatusInference{
			Emoji:      "🤔",
			Activity:   "状态未知",
			Confidence: "low",
			Reasoning:  "推断循环未产出结果",
			InferredAt: time.Now().UnixMilli(),
			Source:     model.InferenceSourceAIOneclick,
		}
	}

	// 设置来源标记
	inferResult.Source = model.InferenceSourceAIOneclick

	// GIF 翻译
	s.enrichInferenceWithGif(ctx, inferResult)

	// 注意：一键生成推断不设置推断锁、不保存推断历史
	// 等用户通过 status-feedback 确认发布后，由 SaveStatusFeedback 处理入库

	return &model.InferenceResponse{
		Phase:  model.InferencePhaseCompleted,
		Result: inferResult,
	}, nil
}

// processTextInputInternal 处理文本输入（内部方法，调用方已持有锁）
func (s *VoiceScheduleServiceV4) processTextInputInternal(
	ctx context.Context,
	userID string,
	sessionID string,
	text string,
	sensorData *model.ExtendedStatusReportRequest,
	callback func(event *model.V4Event),
) (*model.V4Session, error) {
	// 1. 获取或创建会话
	session, isNew := s.getOrCreateSession(ctx, userID, sessionID)

	// 注入传感器数据（一键生成场景）
	if sensorData != nil {
		session.SensorData = sensorData
	}

	// 加载 V3 近期推断状态（让 LLM 感知冲突）
	if s.redisClient != nil {
		prevKey := fmt.Sprintf("infer:prev:%s", userID)
		if prevData, err := s.redisClient.Get(ctx, prevKey); err == nil && prevData != "" {
			var prevInfer model.CurrentStatusInference
			if json.Unmarshal([]byte(prevData), &prevInfer) == nil && prevInfer.Activity != "" {
				session.RecentInference = fmt.Sprintf("%s%s", prevInfer.Emoji, prevInfer.Activity)
			}
		}
	}

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

	// 2. 每次请求都刷新时刻表（用户可能在两次对话间修改了安排）
	s.loadCurrentSchedule(ctx, session)

	// 3. 添加用户消息
	session.AddMessage("user", text)

	// 3.5 Layer 1: 时间预处理（注入事实，非规则）
	hint := agent.ParseTemporalHints(text, time.Now())
	session.TemporalAnnotation = agent.FormatHintAnnotation(hint)

	// 3.6 按需加载目标日期安排（如果用户提到了非今天/明天的日期）
	if hint.DateMatch != nil {
		s.loadTargetDateSchedule(ctx, session, hint.DateMatch.ResolvedDate)
	}

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

		// 2. 动态选择工具集
		tools := s.getToolsForSession(session)

		// 3. 调用 LLM（启用 thinking mode，精确决策低温度）
		// 推断模式是后台任务，给更多思考空间让模型自主推理
		thinkingBudget := 2048
		if session.InferenceMode {
			thinkingBudget = 6144
		}
		response, err := s.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
			Messages:       messages,
			Tools:          tools,
			Temperature:    0.3,
			EnableThinking: true,
			ThinkingBudget: thinkingBudget,
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
const v4TokenBudget = 12000

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

	// few-shot 示例已禁用：测试表明会干扰 confirm 场景的准确率
	// messages = append(messages, s.buildFewShotExamples()...)

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

	// ========== 角色定义 + 行为原则 ==========
	sb.WriteString(`你是「有空」的 AI 助手，帮用户管理日程、查询好友、发送消息。

行为原则：
- 先理解用户意图，再选择工具。不确定时用文字问清楚，不要猜测
- 回复简洁自然，像朋友聊天
- 工具返回 notices 时，在回复中自然提及（不忽略也不过度强调）
- 【今日安排】是快照，涉及用户当前实时状态时，用 view_schedule 获取最新数据

上下文确认规则：
- 用户说"好的/保存/发吧" + 有⚠️待确认内容 → confirm
- 用户说"好的" + 无⚠️ → 闲聊，不调用工具
- "不对/再改改" + 有⚠️ → 不调用工具，问用户想怎么改
- 有【最近查询的好友】+ "他/她" → 用该好友ID
- 有⚠️待确认消息 + "改成XX" → draft_message 重新生成
- 意图不完整（仅代词、缺关键信息）→ 不调用工具，先问清楚
- 有⚠️待确认删除 + "确认/删吧" → confirm
- 有⚠️待确认删除 + "算了/不删了" → 不调用工具

时间参考：
- "等会/一会儿"≈当前时间+30min，"中午"≈12:00，"下午"≈14:00，"晚上"≈19:00
- 跨午夜活动：用 start_time > end_time 表示（如 22:00→02:00 = 当晚到次日凌晨），保持同一天日期
- 跨午夜后的后续活动：如果用户描述了凌晨之后的安排（如"工作到凌晨2点，然后睡到8点"），把后续活动也放在同一个 plan_activities 中，用正常时间表示（如 02:00→08:00）。系统会自动将其归入次日
- 关键：不要因为涉及凌晨就丢弃任何活动。完整列出用户提到的每一个活动段
- 凌晨时段（"凌晨0点到2点"）：start_time="00:00" end_time="02:00"，这是正常时段不是跨午夜

有空标记：
- "有空/我有空/标记有空" = 设置时段的有空状态 → plan_activities(available=true) 或 set_status(available=true)
- "什么时候有空/哪些时段有空" = 查询有空时段 → view_schedule

`)

	// ========== 当前时间（增强版：含时段标记 + 7天日历）==========
	todayStr := now.Format("2006-01-02")
	period := v4GetTimePeriod(now.Hour())
	sb.WriteString(fmt.Sprintf("【时间】%s %s %02d:%02d | 当前时段=%s\n",
		todayStr, weekdays[now.Weekday()], now.Hour(), now.Minute(), period))
	// 7 天日历参考：让 LLM 知道"后天"/"下周X"对应的具体日期
	sb.WriteString("日历: ")
	dayLabels := []string{"今天", "明天", "后天"}
	for i := 0; i < 7; i++ {
		d := now.AddDate(0, 0, i)
		dateStr := d.Format("01-02")
		wd := weekdays[d.Weekday()]
		if i < len(dayLabels) {
			sb.WriteString(fmt.Sprintf("%s=%s(%s)", dayLabels[i], dateStr, wd))
		} else {
			sb.WriteString(fmt.Sprintf("%s(%s)", dateStr, wd))
		}
		if i < 6 {
			sb.WriteString(" ")
		}
	}
	sb.WriteString("\n\n")

	// ========== 时间预处理注解（Layer 1: 事实注入）==========
	if session.TemporalAnnotation != "" {
		sb.WriteString(session.TemporalAnnotation + "\n\n")
	}

	// ========== 今日安排 ==========
	if len(session.CurrentSchedule) > 0 {
		sb.WriteString(fmt.Sprintf("【今日安排】(%d个时段，会话开始时快照，可能已过时)\n", len(session.CurrentSchedule)))
		for _, item := range session.CurrentSchedule {
			availMark := ""
			if item.Highlight {
				availMark = " ✅有空"
			}
			sb.WriteString(fmt.Sprintf("%s-%s %s%s%s\n", item.StartTime, item.EndTime, item.Emoji, item.Status, availMark))
		}
		sb.WriteString("\n")
	}

	// ========== 明日安排 ==========
	if len(session.TomorrowSchedule) > 0 {
		sb.WriteString(fmt.Sprintf("【明日安排】(%d个时段) ", len(session.TomorrowSchedule)))
		for i, item := range session.TomorrowSchedule {
			if i > 0 {
				sb.WriteString(" | ")
			}
			availMark := ""
			if item.Highlight {
				availMark = " ✅有空"
			}
			sb.WriteString(fmt.Sprintf("%s-%s %s%s%s", item.StartTime, item.EndTime, item.Emoji, item.Status, availMark))
		}
		sb.WriteString("\n\n")
	}

	// ========== 目标日期安排（按需加载）==========
	if len(session.TargetDateSchedule) > 0 && session.TargetDateLabel != "" {
		sb.WriteString(fmt.Sprintf("【%s 安排】(%d个时段)\n", session.TargetDateLabel, len(session.TargetDateSchedule)))
		for _, item := range session.TargetDateSchedule {
			availMark := ""
			if item.Highlight {
				availMark = " ✅有空"
			}
			sb.WriteString(fmt.Sprintf("%s-%s %s%s%s\n", item.StartTime, item.EndTime, item.Emoji, item.Status, availMark))
		}
		sb.WriteString("\n")
	}

	// ========== 待确认的时刻表 ==========
	if session.HasPendingSchedule() {
		for _, dateStr := range session.PendingScheduleDates() {
			items := session.PendingSchedules[dateStr]
			sb.WriteString(fmt.Sprintf("【⚠️待确认时刻表 %s】", dateStr))
			for i, item := range items {
				if i > 0 {
					sb.WriteString(" | ")
				}
				sb.WriteString(fmt.Sprintf("%s-%s %s%s", item.StartTime, item.EndTime, item.Emoji, item.Status))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("用户确认后调用 confirm 保存，要修改则调用 plan_activities 重新生成\n\n")
	}

	// ========== 待确认的消息/邀请 ==========
	if session.HasPendingMessage() {
		sb.WriteString(fmt.Sprintf("【⚠️待确认消息】发送给 %s (ID: %s): \"%s\"\n用户确认后调用 confirm，修改内容调用 draft_message(friend_id=%s)\n\n",
			session.PendingMessage.FriendName, session.PendingMessage.FriendID,
			session.PendingMessage.Message, session.PendingMessage.FriendID))
	}
	if session.HasPendingInvite() {
		sb.WriteString(fmt.Sprintf("【⚠️待确认邀请】邀请 %s %s %s-%s %s\n用户确认后调用 confirm\n\n",
			session.PendingInvite.FriendName, session.PendingInvite.Date,
			session.PendingInvite.StartTime, session.PendingInvite.EndTime,
			session.PendingInvite.Activity))
	}

	// ========== 待确认的删除操作 ==========
	if session.HasPendingDeletion() {
		pd := session.PendingDeletion
		if pd.Type == "schedule" && len(pd.Entries) > 0 {
			sb.WriteString(fmt.Sprintf("【⚠️待确认删除】共%d个安排待删除：\n", pd.TotalDeletedCount()))
			for _, entry := range pd.Entries {
				sb.WriteString(fmt.Sprintf("  %s:", entry.Date))
				for _, item := range entry.DeletedItems {
					sb.WriteString(fmt.Sprintf(" %s-%s %s%s |", item.StartTime, item.EndTime, item.Emoji, item.Status))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("用户确认后调用 confirm 执行删除，说'算了/不删了'则不操作\n\n")
		} else if pd.Type == "friend" {
			sb.WriteString(fmt.Sprintf("【⚠️待确认删除好友】将删除好友 %s (ID: %s)\n用户确认后调用 confirm 执行删除\n\n",
				pd.FriendName, pd.FriendID))
		}
	}

	// ========== 推断模式（V4 替代 V3 StatusInferenceAgent）==========
	// 设计原则：prompt 引导思考方向，不硬编码规则；推理在 thinking tokens 中完成
	if session.InferenceMode {
		sb.WriteString(`【推断模式】
你是「有空」状态推断引擎。分析实时设备信号，推断用户此刻的具体活动，然后调用 set_status 工具。不要输出文字给用户。

# 你的思考过程应该覆盖：
1. **信号优先级**：实时设备信号（屏幕、位置、运动、日历）是此刻状态的直接证据，优先级最高。时刻表和历史记录是辅助参考。AI 推测与实时信号矛盾时忽略。
2. **具体化**：用户状态应该是具体的日常活动（2-4字），不要使用"娱乐中""在家休息""居家休闲"等笼统描述。想象你是用户的朋友，你会怎样用最简短的口语描述他现在在做什么？比如"刷手机""工作""睡觉""刷剧""聊微信""开车上班"。
3. **时间语境**：工作日和周末的同一信号可能意味着不同活动。深夜、清晨、午间有各自的合理推断。
4. **is_available 含义**：表示用户此刻是否方便被打扰/有空社交。工作、开会、通勤、睡觉、运动等通常为 false；在家休闲、空闲、午休等通常为 true。
5. **用户修正**：修正历史中纠正过的推断不能再犯。

# status 格式
- 2-4字，口语化动词短语。如：刷手机、工作、睡觉、刷剧、聊微信、开车上班、午休、散步
- 不要加地点前缀（如"在家""居家"）和修饰词（如"专注""深夜"），这些信息已在 emoji 中体现

# emoji 选择
用 1-3 个 emoji 组合反映地点+活动，例如 📱🏠 表示在家用手机、💼💻 表示在办公。

`)
		if session.InferenceContext != "" {
			sb.WriteString("【多维上下文数据】\n")
			sb.WriteString(session.InferenceContext)
			sb.WriteString("\n\n")
		}
		if session.SensorData != nil {
			sb.WriteString("【实时设备数据】\n")
			sb.WriteString(formatSensorData(session.SensorData))
			sb.WriteString("\n\n")
		}
		return sb.String()
	}

	// ========== AI 推断状态（V3 近期结果）==========
	if session.RecentInference != "" {
		sb.WriteString(fmt.Sprintf("【当前 AI 推断状态】%s（用户可能想修改）\n\n", session.RecentInference))
	}

	// ========== 传感器数据（一键生成时注入）==========
	if session.SensorData != nil {
		sb.WriteString("【设备数据】（用户请求推断状态时的实时数据）\n")
		sb.WriteString(formatSensorData(session.SensorData))
		sb.WriteString("\n请结合设备数据和对话上下文，用 set_status 设置最合适的状态。\n\n")
	}

	return sb.String()
}

// formatSensorData 将传感器数据格式化为自然语言
func formatSensorData(data *model.ExtendedStatusReportRequest) string {
	var sb strings.Builder

	// 屏幕（最关键的推断信号）
	if data.Screen != nil {
		s := data.Screen
		if s.IsActive {
			sb.WriteString(fmt.Sprintf("- screen: is_active=true, activity_type=%s, session_duration_minutes=%d\n",
				s.ActivityType, s.SessionDurationMinutes))
		} else {
			sb.WriteString(fmt.Sprintf("- screen: is_active=false, last_active_minutes_ago=%d\n",
				s.LastActiveMinutesAgo))
		}
	}

	// 位置（place_type 由客户端猜测，不一定准确）
	if data.ExtendedLocation != nil {
		loc := data.ExtendedLocation
		sb.WriteString(fmt.Sprintf("- location: place_type=%s (client_guess, may be inaccurate), at_place_since_minutes=%d",
			loc.PlaceType, loc.AtPlaceSinceMinutes))
		if loc.PlaceName != "" {
			sb.WriteString(fmt.Sprintf(", place_name=%s", loc.PlaceName))
		}
		sb.WriteString("\n")
	} else if data.Location != nil {
		loc := data.Location
		sb.WriteString(fmt.Sprintf("- location: place_type=%s (client_guess, may be inaccurate), at_place_since_minutes=%d\n",
			loc.PlaceType, loc.AtPlaceSinceMinutes))
	}

	// 运动
	if data.Movement != nil {
		m := data.Movement
		if m.IsMoving {
			sb.WriteString(fmt.Sprintf("- movement: is_moving=true, activity=%s", m.MovementType))
			if m.MovingDirection != "" {
				sb.WriteString(fmt.Sprintf(", direction=%s", m.MovingDirection))
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString(fmt.Sprintf("- movement: is_moving=false, stationary_minutes=%d\n",
				m.StationaryMinutes))
		}
	}

	// 日历
	if data.Calendar != nil {
		cal := data.Calendar
		if cal.HasCurrentEvent {
			sb.WriteString(fmt.Sprintf("- calendar: has_current_event=true, title=%s, ends_in_minutes=%d\n",
				cal.CurrentEventTitle, cal.EventEndMinutes))
		} else if cal.NextEventInMinutes > 0 {
			sb.WriteString(fmt.Sprintf("- calendar: has_current_event=false, next_event_in_minutes=%d\n",
				cal.NextEventInMinutes))
		}
		if cal.TodayRemainingCount > 0 {
			sb.WriteString(fmt.Sprintf("- calendar: today_remaining_events=%d\n", cal.TodayRemainingCount))
		}
	}

	// 电池
	if data.Battery != nil {
		b := data.Battery
		sb.WriteString(fmt.Sprintf("- battery: level=%d%%, is_charging=%v\n",
			b.BatteryLevel, b.IsCharging))
	}

	// 模式
	if data.Mode != nil {
		m := data.Mode
		if m.IsFocusModeOn || m.IsLowPowerMode {
			sb.WriteString(fmt.Sprintf("- mode: focus_mode=%v, low_power=%v\n",
				m.IsFocusModeOn, m.IsLowPowerMode))
		}
	}

	// 连接
	if data.Connection != nil {
		c := data.Connection
		sb.WriteString(fmt.Sprintf("- connection: network=%s, headphones=%v\n",
			c.NetworkType, c.IsHeadphonesConnected))
	}

	return sb.String()
}

// buildFewShotExamples 构建 few-shot 示例消息（校准口语化输入的工具选择）
// 只展示 user → tool_call 映射，不包含 tool result/assistant 回复，避免 awaiting_approval 误导
func (s *VoiceScheduleServiceV4) buildFewShotExamples() []agent.AgentMessage {
	today := time.Now().Format("2006-01-02")

	return []agent.AgentMessage{
		// 示例1: 口语化活动 → plan_activities（不是 set_status）
		agent.NewUserMessage("等会去搓一顿"),
		agent.NewAssistantMessageWithToolCall("plan_activities",
			fmt.Sprintf(`{"date":"%s","activities":[{"start_time":"12:00","end_time":"13:30","emoji":"🍜","status":"吃饭"}]}`, today)),
		agent.NewToolResultMessage("fewshot_plan_activities", "plan_activities", `{"message":"已保存"}`),
		agent.NewAssistantMessage("帮你安排好了吃饭时间 🍜"),

		// 示例2: 纯状态 → set_status（不是 plan_activities）
		agent.NewUserMessage("在加班"),
		agent.NewAssistantMessageWithToolCall("set_status",
			`{"emoji":"💼","status":"加班中"}`),
		agent.NewToolResultMessage("fewshot_set_status", "set_status", `{"message":"已更新"}`),
		agent.NewAssistantMessage("已更新你的状态为 💼加班中"),

		// 示例3: 模糊时间 → plan_activities（合理推断）
		agent.NewUserMessage("中午出去吃个饭"),
		agent.NewAssistantMessageWithToolCall("plan_activities",
			fmt.Sprintf(`{"date":"%s","activities":[{"start_time":"12:00","end_time":"13:00","emoji":"🍽️","status":"午饭"}]}`, today)),
		agent.NewToolResultMessage("fewshot_plan_activities", "plan_activities", `{"message":"已保存"}`),
		agent.NewAssistantMessage("安排好了中午的午饭时间 🍽️"),
	}
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

		// 添加工具结果到会话（裁剪过长结果防止上下文膨胀）
		resultJSON, _ := json.Marshal(result)
		session.AddToolResult(tc.ID, toolName, trimToolResult(string(resultJSON), 2000))

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
		"view_schedule":             "正在查询时刻表...",
		"plan_activities":           "正在生成时刻表预览...",
		"set_status":                "正在更新当前状态...",
		"confirm":                   "正在确认...",
		"update_preference":         "正在更新偏好设置...",
		"find_friends":              "正在查询好友...",
		"draft_message":             "正在生成消息预览...",
		"draft_invite":              "正在生成日程邀请...",
		"delete_schedule":           "正在查找要删除的安排...",
		"remove_friend":             "正在准备删除好友...",
		"match_contacts":            "正在匹配通讯录...",
		"add_friend":                "正在添加好友...",
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
	case "view_schedule":
		return s.executeGetSchedule(ctx, session, args, callback)

	case "plan_activities":
		return s.executeUpdateSchedule(ctx, session, args, callback)

	case "delete_schedule":
		return s.executeDeleteSchedule(ctx, session, args, callback)

	case "set_status":
		return s.executeUpdateCurrentStatus(ctx, session, args, callback)

	case "confirm":
		return s.executeConfirm(ctx, session, args, callback)

	case "update_preference":
		return s.executeUpdatePreference(ctx, session, args, callback)

	// ========== 好友相关工具 ==========
	case "find_friends":
		return s.executeQueryFriends(ctx, session, args, callback)

	case "draft_message":
		return s.executeSendMessage(ctx, session, args, callback)

	case "draft_invite":
		return s.executeCreateScheduleInvite(ctx, session, args, callback)

	case "remove_friend":
		return s.executeRemoveFriend(ctx, session, args, callback)

	// ========== 扩展工具 ==========
	case "match_contacts":
		return s.executeMatchContacts(ctx, session, args, callback)

	case "add_friend":
		return s.executeAddFriend(ctx, session, args, callback)

	default:
		return map[string]string{"error": "未知工具: " + toolName}, false
	}
}

// executeGetSchedule 执行获取时刻表（view_schedule）
func (s *VoiceScheduleServiceV4) executeGetSchedule(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	// 解析日期（date 现在是必填参数）
	date, err := agent.ParseDate(args)
	if err != nil {
		date = time.Now()
	}

	// 获取时刻表：先查 active，再查所有（包含 completed）合并展示
	schedules, err := s.scheduleRepo.GetAllByUserAndDate(ctx, session.UserID, date)
	if err != nil || len(schedules) == 0 {
		return map[string]interface{}{
			"message": "暂无时刻表",
			"date":    date.Format("2006-01-02"),
			"notices": []model.TimeNotice{},
		}, false
	}

	// 合并所有时刻表的 items（按 start_time 去重，保留最新的）
	seen := make(map[string]bool)
	var allItems model.ScheduleItems
	for _, schedule := range schedules {
		for _, item := range schedule.Items {
			key := item.StartTime + "-" + item.EndTime
			if !seen[key] {
				seen[key] = true
				allItems = append(allItems, item)
			}
		}
	}

	if len(allItems) == 0 {
		return map[string]interface{}{
			"message": "暂无时刻表",
			"date":    date.Format("2006-01-02"),
			"notices": []model.TimeNotice{},
		}, false
	}

	// 转换为返回格式
	items := make([]map[string]interface{}, 0, len(allItems))
	for _, item := range allItems {
		items = append(items, map[string]interface{}{
			"start_time": item.StartTime,
			"end_time":   item.EndTime,
			"emoji":      item.Emoji,
			"status":     item.Status,
			"available":  item.Highlight,
		})
	}

	// 时间注解：标注 ongoing/soon 项目（不标 past，避免信息过载）
	notices := s.analyzeViewNotices(allItems, date)

	return map[string]interface{}{
		"date":    date.Format("2006-01-02"),
		"items":   items,
		"count":   len(items),
		"notices": notices,
	}, false
}

// executeConfirm 统一确认工具（合并原 save_schedule + confirm_send）
// 根据 session 状态自动判断要确认的类型
func (s *VoiceScheduleServiceV4) executeConfirm(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	if session.HasPendingSchedule() {
		return s.executeSaveSchedule(ctx, session, args, callback)
	}
	if session.HasPendingMessage() || session.HasPendingInvite() {
		return s.executeConfirmSend(ctx, session, args, callback)
	}
	if session.HasPendingDeletion() {
		return s.executeConfirmDeletion(ctx, session, args, callback)
	}
	return map[string]string{"error": "没有待确认的内容"}, false
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

	// Layer 3: 参数后验证
	isToday := date.Format("2006-01-02") == time.Now().Format("2006-01-02")
	valErrors, valWarnings := agent.ValidateScheduleParams(items, time.Now(), isToday)
	if len(valErrors) > 0 {
		return map[string]interface{}{
			"error":   true,
			"message": "参数错误: " + strings.Join(valErrors, "; "),
			"retry":   true,
		}, false // breakLoop=false → LLM 重试
	}

	// 转换为 model.ScheduleItem
	newItems := make([]model.ScheduleItem, len(items))
	for i, item := range items {
		newItems[i] = model.ScheduleItem{
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			Emoji:     item.Emoji,
			Status:    item.Status,
			Highlight: item.Available != nil && *item.Available,
		}
	}
	// 保存 Available nil 信息用于 modify 时继承（items 切片与 newItems 索引对应）
	availableNils := make([]bool, len(items))
	for i, item := range items {
		availableNils[i] = item.Available == nil
	}

	// 获取现有时刻表（包含 completed）
	var existingItems []model.ScheduleItem
	todayDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	existingSchedule, _ := s.scheduleRepo.GetLatestByUserAndDate(ctx, session.UserID, todayDate)
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
		// 修改模式前：对 Available==nil 的新条目，从被替换的旧条目继承 Highlight
		for i, newItem := range newItems {
			if availableNils[i] {
				for _, existItem := range existingItems {
					if s.isTimeOverlap(existItem.StartTime, existItem.EndTime, newItem.StartTime, newItem.EndTime) {
						newItems[i].Highlight = existItem.Highlight
						break
					}
				}
			}
		}
		// 修改模式（默认）：智能合并，更新匹配时间段的条目
		finalItems = s.mergeScheduleItems(existingItems, newItems, true)
		fmt.Printf("[V4] 时刻表操作=modify, 现有=%d, 修改=%d, 合并后=%d\n",
			len(existingItems), len(newItems), len(finalItems))
	}

	// 检测跨午夜后的次日溢出条目
	todayFinalItems, nextDayItems := s.detectNextDaySpillover(finalItems)

	// 按时间排序
	todayFinalItems = s.sortScheduleItems(todayFinalItems)

	// 时间分析：生成时间注解（Phase 7: 工具层时间感知）
	notices := s.analyzeTimeContext(todayFinalItems, newItems, existingItems, date)

	// Layer 3: 将验证警告附加到 notices
	for _, w := range valWarnings {
		notices = append(notices, model.TimeNotice{Type: "warning", Detail: w, ItemIdx: -1})
	}

	// 保存到 session 的 PendingSchedules（支持多日累积）
	dateStr := date.Format("2006-01-02")
	if session.PendingSchedules == nil {
		session.PendingSchedules = make(map[string][]model.ScheduleItem)
	}
	session.PendingSchedules[dateStr] = todayFinalItems

	// 处理次日溢出条目
	if len(nextDayItems) > 0 {
		nextDay := date.AddDate(0, 0, 1)
		nextDayItems = s.sortScheduleItems(nextDayItems)

		// 获取次日现有时刻表并合并
		nextDayDate := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, nextDay.Location())
		existingNextDay, _ := s.scheduleRepo.GetLatestByUserAndDate(ctx, session.UserID, nextDayDate)
		if existingNextDay != nil && len(existingNextDay.Items) > 0 {
			nextDayItems = s.mergeScheduleItems(existingNextDay.Items, nextDayItems, true)
			nextDayItems = s.sortScheduleItems(nextDayItems)
		}

		nextDayStr := nextDay.Format("2006-01-02")
		session.PendingSchedules[nextDayStr] = nextDayItems

		fmt.Printf("[V4] 检测到跨午夜次日溢出: %d 条 → %s\n", len(nextDayItems), nextDay.Format("01-02"))
	}

	// 发送预览事件
	callback(&model.V4Event{
		Type:    model.V4EventTypeSchedulePreview,
		Items:   todayFinalItems,
		Date:    date.Format("2006-01-02"),
		Message: fmt.Sprintf("已生成 %s 的时刻表预览", date.Format("01月02日")),
	})

	// 如果有次日条目，也发送次日预览
	if len(nextDayItems) > 0 {
		nextDay := date.AddDate(0, 0, 1)
		callback(&model.V4Event{
			Type:    model.V4EventTypeSchedulePreview,
			Items:   nextDayItems,
			Date:    nextDay.Format("2006-01-02"),
			Message: fmt.Sprintf("已生成 %s 的时刻表预览（跨午夜后续）", nextDay.Format("01月02日")),
		})
	}

	totalItems := len(todayFinalItems) + len(nextDayItems)
	return map[string]interface{}{
		"message":           "时刻表预览已生成，等待用户确认",
		"date":              date.Format("2006-01-02"),
		"items_count":       totalItems,
		"operation":         operation,
		"awaiting_approval": true,
		"notices":           notices,
	}, true // breakLoop: 停止循环，等待用户确认
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
				Highlight: existItem.Highlight,
			})
		}

		// 后段：newItem.EndTime ~ existItem.EndTime
		if newItem.EndTime < existItem.EndTime {
			splitResult = append(splitResult, model.ScheduleItem{
				StartTime: newItem.EndTime,
				EndTime:   existItem.EndTime,
				Emoji:     existItem.Emoji,
				Status:    existItem.Status,
				Highlight: existItem.Highlight,
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

// detectNextDaySpillover 检测跨午夜后的次日溢出条目
// 当 LLM 在同一个 plan_activities 中返回了跨午夜活动和凌晨后续活动时，
// 自动将后续活动拆分到次日。
// 例：[{09:00-18:00 上班}, {22:00-02:00 加班}, {02:00-08:00 睡觉}]
// → todayItems=[{09:00-18:00 上班}, {22:00-02:00 加班}], nextDayItems=[{02:00-08:00 睡觉}]
func (s *VoiceScheduleServiceV4) detectNextDaySpillover(items []model.ScheduleItem) (todayItems, nextDayItems []model.ScheduleItem) {
	isCrossMN := func(start, end string) bool { return start > end }

	// 找到第一个跨午夜条目
	crossMidnightIdx := -1
	for i, item := range items {
		if isCrossMN(item.StartTime, item.EndTime) {
			crossMidnightIdx = i
			break
		}
	}

	if crossMidnightIdx < 0 {
		return items, nil // 无跨午夜，全部留在当天
	}

	// 跨午夜条目及之前的条目留在当天
	todayItems = append(todayItems, items[:crossMidnightIdx+1]...)

	// 跨午夜之后的条目：start_time < 12:00 且非跨午夜的视为次日
	for _, item := range items[crossMidnightIdx+1:] {
		if !isCrossMN(item.StartTime, item.EndTime) && item.StartTime < "12:00" {
			nextDayItems = append(nextDayItems, item)
		} else {
			todayItems = append(todayItems, item)
		}
	}

	return todayItems, nextDayItems
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
	highlight, _ := args["available"].(bool) // 默认 false

	if emoji == "" || status == "" {
		return map[string]string{"error": "emoji 和 status 不能为空"}, false
	}

	// 推断模式（一键生成）：只保存结果到 session，不入库
	// 用户确认发布后，通过 status-feedback 接口才真正写入时刻表和分析缓存
	if session.InferenceMode {
		session.InferenceResult = &model.CurrentStatusInference{
			Emoji:       emoji,
			Activity:    status,
			IsAvailable: highlight,
			Confidence:  "high",
			InferredAt:  time.Now().UnixMilli(),
		}
		fmt.Printf("[V4] 推断模式：结果暂存 session（未入库），等待用户确认发布 user=%s %s%s\n",
			session.UserID, emoji, status)

		callback(&model.V4Event{
			Type:    model.V4EventTypeStatusUpdated,
			Emoji:   emoji,
			Status:  status,
			Message: "推断完成，等待确认发布",
		})

		return map[string]interface{}{
			"message": "推断完成，等待用户确认发布",
			"emoji":   emoji,
			"status":  status,
		}, false
	}

	// === 以下为非推断模式（用户主动设置 / 语音设置）：立即入库 ===

	// 检查 V3 推断锁：用户主动设置时覆盖 AI 推断
	var prevInferEmoji, prevInferActivity string
	if s.redisClient != nil {
		prevKey := fmt.Sprintf("infer:prev:%s", session.UserID)
		if prevData, err := s.redisClient.Get(ctx, prevKey); err == nil && prevData != "" {
			var prevInfer model.CurrentStatusInference
			if json.Unmarshal([]byte(prevData), &prevInfer) == nil {
				prevInferEmoji = prevInfer.Emoji
				prevInferActivity = prevInfer.Activity
				fmt.Printf("[V4] 检测到 V3 推断: %s%s，用户覆盖为: %s%s\n",
					prevInferEmoji, prevInferActivity, emoji, status)
			}
		}
		// 清除推断锁，用户主动设置优先
		lockKey := fmt.Sprintf("infer:lock:%s", session.UserID)
		s.redisClient.Del(ctx, lockKey)
	}

	// 更新当前状态到时刻表（当前时段）
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 获取或创建今日时刻表（优先 active，其次 completed，最后新建）
	schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, session.UserID, today)
	if err != nil || schedule == nil {
		// 没有 active 的，尝试复用最新的 completed
		schedule, err = s.scheduleRepo.GetLatestByUserAndDate(ctx, session.UserID, today)
		if err != nil || schedule == nil {
			schedule = &model.StatusSchedule{
				UserID:       session.UserID,
				ScheduleDate: today,
				Items:        model.ScheduleItems{},
				Status:       model.ScheduleStatusActive,
				Visibility:   model.VisibilityAllFriends,
			}
		} else {
			// 复用已有时刻表，重新激活
			schedule.Status = model.ScheduleStatusActive
		}
	}

	// 计算当前时段
	startTime, endTime := getCurrentTimeSlot(now)

	// 添加或更新当前时段（匹配 startTime 相同或时间段重叠的 item）
	found := false
	for i, item := range schedule.Items {
		if item.StartTime == startTime || s.isTimeOverlap(item.StartTime, item.EndTime, startTime, endTime) {
			schedule.Items[i].StartTime = startTime
			schedule.Items[i].EndTime = endTime
			schedule.Items[i].Emoji = emoji
			schedule.Items[i].Status = status
			schedule.Items[i].Highlight = highlight
			schedule.Items[i].Executed = false // 重置 executed 状态
			schedule.Items[i].IsAIGuess = false
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
			Highlight: highlight,
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

	{
		// 同步更新首页分析缓存，让首页 grid 立即显示新状态
		analysisResult := &model.AnalysisResult{
			LifeStatus: model.LifeStatus{
				Emoji: emoji,
				Label: status,
			},
			Availability: model.AvailabilityAnalysis{
				Status: "有空",
			},
			IsAIGuess: false, // 用户主动设置，非 AI 猜测
		}
		if cacheErr := s.memoryRepo.SaveAnalysisCache(ctx, session.UserID, analysisResult); cacheErr != nil {
			fmt.Printf("[V4] 更新分析缓存失败: %v\n", cacheErr)
		}
	}

	// 构造消息（若覆盖了 V3 推断，告知用户）
	msg := "当前状态已更新"
	if prevInferActivity != "" {
		msg = fmt.Sprintf("已将状态从 AI 推断的「%s%s」更新为「%s%s」",
			prevInferEmoji, prevInferActivity, emoji, status)
	}

	// 发送状态更新事件
	callback(&model.V4Event{
		Type:    model.V4EventTypeStatusUpdated,
		Emoji:   emoji,
		Status:  status,
		Message: msg,
	})

	return map[string]interface{}{
		"message": msg,
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

	// 遍历所有待确认日期，逐个保存
	savedDates := session.PendingScheduleDates()
	for _, dateStr := range savedDates {
		items := session.PendingSchedules[dateStr]
		if len(items) == 0 {
			continue
		}

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			fmt.Printf("[V4] 跳过无效日期 %s: %v\n", dateStr, err)
			continue
		}
		dayDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

		schedule := &model.StatusSchedule{
			UserID:       session.UserID,
			ScheduleDate: dayDate,
			Items:        model.ScheduleItems(items),
			Status:       model.ScheduleStatusActive,
			Visibility:   visibility,
		}

		existing, _ := s.scheduleRepo.GetLatestByUserAndDate(ctx, session.UserID, dayDate)
		if existing != nil {
			schedule.ID = existing.ID
			if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
				return map[string]string{"error": fmt.Sprintf("保存 %s 失败: %s", dateStr, err.Error())}, false
			}
		} else {
			if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
				return map[string]string{"error": fmt.Sprintf("保存 %s 失败: %s", dateStr, err.Error())}, false
			}
		}

		callback(&model.V4Event{
			Type:    model.V4EventTypeScheduleSaved,
			Items:   items,
			Date:    dateStr,
			Message: fmt.Sprintf("%s 的时刻表已保存", dateStr),
		})
		fmt.Printf("[V4] 保存时刻表 %s: %d 条\n", dateStr, len(items))
	}

	// 更新会话状态（CurrentSchedule 保留最后一天的）
	if len(savedDates) > 0 {
		lastDate := savedDates[len(savedDates)-1]
		session.CurrentSchedule = session.PendingSchedules[lastDate]
	}
	session.ClearPendingSchedule()

	return map[string]interface{}{
		"message":     "时刻表已保存",
		"date":        time.Now().Format("2006-01-02"),
		"saved_dates": savedDates,
	}, true // 结束循环
}

// executeDeleteSchedule 执行删除日程条目（生成删除预览）
// 参数对齐工具定义：clear_all(bool), start_time/end_time(string), date(string, 支持逗号分隔多日)
func (s *VoiceScheduleServiceV4) executeDeleteSchedule(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	clearAll, _ := args["clear_all"].(bool)
	startTime, _ := args["start_time"].(string)
	endTime, _ := args["end_time"].(string)

	// 解析日期（支持逗号分隔多日）
	dateStr, _ := args["date"].(string)
	var dates []time.Time
	if dateStr == "" {
		now := time.Now()
		dates = append(dates, time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()))
	} else {
		for _, ds := range strings.Split(dateStr, ",") {
			ds = strings.TrimSpace(ds)
			if ds == "" {
				continue
			}
			t, err := time.Parse("2006-01-02", ds)
			if err != nil {
				continue
			}
			dates = append(dates, t)
		}
		if len(dates) == 0 {
			now := time.Now()
			dates = append(dates, time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()))
		}
	}

	// 对每个日期生成删除条目
	var entries []model.V4DeletionEntry
	for _, date := range dates {
		ds := date.Format("2006-01-02")
		schedule, err := s.scheduleRepo.GetLatestByUserAndDate(ctx, session.UserID, date)
		if err != nil || schedule == nil || len(schedule.Items) == 0 {
			fmt.Printf("[V4] delete_schedule: %s 没有安排\n", ds)
			continue
		}

		var deletedItems, remainingItems []model.ScheduleItem
		if clearAll {
			deletedItems = schedule.Items
		} else if startTime != "" && endTime != "" {
			// 精确时间匹配
			for _, item := range schedule.Items {
				if item.StartTime == startTime && item.EndTime == endTime {
					deletedItems = append(deletedItems, item)
				} else {
					remainingItems = append(remainingItems, item)
				}
			}
		} else if startTime != "" {
			// 只匹配 start_time
			for _, item := range schedule.Items {
				if item.StartTime == startTime {
					deletedItems = append(deletedItems, item)
				} else {
					remainingItems = append(remainingItems, item)
				}
			}
		} else {
			// 无具体条件 → 视为清空
			deletedItems = schedule.Items
		}

		if len(deletedItems) > 0 {
			entries = append(entries, model.V4DeletionEntry{
				Date:           ds,
				DeletedItems:   deletedItems,
				RemainingItems: remainingItems,
			})
		}
	}

	if len(entries) == 0 {
		return map[string]interface{}{
			"message": "没有找到可删除的安排",
		}, false
	}

	// 保存到 PendingDeletion（每次全新，逗号分隔 date 已包含所有目标日期）
	session.PendingDeletion = &model.V4PendingDeletion{
		Type:    "schedule",
		Entries: entries,
	}

	totalDeleted := session.PendingDeletion.TotalDeletedCount()

	// 构建预览消息
	var previewMsg string
	if len(session.PendingDeletion.Entries) == 1 {
		e := session.PendingDeletion.Entries[0]
		previewMsg = fmt.Sprintf("将删除 %s 的%d个安排，等待确认", e.Date, len(e.DeletedItems))
	} else {
		parts := make([]string, 0, len(session.PendingDeletion.Entries))
		for _, e := range session.PendingDeletion.Entries {
			parts = append(parts, fmt.Sprintf("%s %d个", e.Date, len(e.DeletedItems)))
		}
		previewMsg = fmt.Sprintf("将删除 %s 安排（共%d个），等待确认", strings.Join(parts, "、"), totalDeleted)
	}

	// 发送删除预览事件
	callback(&model.V4Event{
		Type:            model.V4EventTypeDeletePreview,
		PendingDeletion: session.PendingDeletion,
		Message:         previewMsg,
	})

	return map[string]interface{}{
		"message":           previewMsg,
		"total_deleted":     totalDeleted,
		"dates":             len(session.PendingDeletion.Entries),
		"awaiting_approval": true,
	}, true // breakLoop: 停止循环，等待用户确认
}

// executeRemoveFriend 执行删除好友（生成删除预览）
func (s *VoiceScheduleServiceV4) executeRemoveFriend(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	friendID, _ := args["friend_id"].(string)
	friendName, _ := args["friend_name"].(string)

	if friendID == "" || friendName == "" {
		return map[string]string{"error": "friend_id 和 friend_name 不能为空"}, false
	}

	// 保存到 PendingDeletion
	session.PendingDeletion = &model.V4PendingDeletion{
		Type:       "friend",
		FriendID:   friendID,
		FriendName: friendName,
	}

	// 发送删除预览事件
	callback(&model.V4Event{
		Type:            model.V4EventTypeDeletePreview,
		PendingDeletion: session.PendingDeletion,
		Message:         fmt.Sprintf("将删除好友 %s，等待确认", friendName),
	})

	return map[string]interface{}{
		"message":           fmt.Sprintf("将删除好友 %s，这是不可逆操作。等待用户确认", friendName),
		"friend_name":       friendName,
		"awaiting_approval": true,
	}, true // breakLoop: 停止循环，等待用户确认
}

// executeConfirmDeletion 执行确认删除操作（分发到具体类型）
func (s *VoiceScheduleServiceV4) executeConfirmDeletion(
	ctx context.Context,
	session *model.V4Session,
	args map[string]interface{},
	callback func(event *model.V4Event),
) (interface{}, bool) {
	pd := session.PendingDeletion
	if pd == nil {
		return map[string]string{"error": "没有待确认的删除操作"}, false
	}

	switch pd.Type {
	case "schedule":
		return s.confirmScheduleDeletion(ctx, session, callback)
	case "friend":
		return s.confirmFriendRemoval(ctx, session, callback)
	default:
		return map[string]string{"error": "未知的删除类型: " + pd.Type}, false
	}
}

// confirmScheduleDeletion 确认删除日程条目（支持多日）
func (s *VoiceScheduleServiceV4) confirmScheduleDeletion(
	ctx context.Context,
	session *model.V4Session,
	callback func(event *model.V4Event),
) (interface{}, bool) {
	pd := session.PendingDeletion
	if len(pd.Entries) == 0 {
		session.ClearPendingDeletion()
		return map[string]string{"error": "没有待删除的条目"}, false
	}

	totalDeleted := 0
	todayStr := time.Now().Format("2006-01-02")
	tomorrowStr := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	for _, entry := range pd.Entries {
		date, err := time.Parse("2006-01-02", entry.Date)
		if err != nil {
			fmt.Printf("[V4] confirmScheduleDeletion: 日期解析失败 %s\n", entry.Date)
			continue
		}
		dateTime := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

		if len(entry.RemainingItems) == 0 {
			// 全部删除 → 取消整个时刻表
			if err := s.scheduleRepo.CancelUserSchedulesByDate(ctx, session.UserID, dateTime); err != nil {
				fmt.Printf("[V4] confirmScheduleDeletion: 删除 %s 失败: %v\n", entry.Date, err)
				continue
			}
			fmt.Printf("[V4] confirmScheduleDeletion: 已清空 %s 全部安排\n", entry.Date)
		} else {
			// 部分删除 → 用 RemainingItems 覆盖
			schedule, err := s.scheduleRepo.GetLatestByUserAndDate(ctx, session.UserID, dateTime)
			if err != nil || schedule == nil {
				fmt.Printf("[V4] confirmScheduleDeletion: 获取 %s 时刻表失败\n", entry.Date)
				continue
			}
			schedule.Items = model.ScheduleItems(entry.RemainingItems)
			if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
				fmt.Printf("[V4] confirmScheduleDeletion: 更新 %s 时刻表失败: %v\n", entry.Date, err)
				continue
			}
			fmt.Printf("[V4] confirmScheduleDeletion: 已删除 %s 的%d个安排，剩余%d个\n", entry.Date, len(entry.DeletedItems), len(entry.RemainingItems))
		}

		totalDeleted += len(entry.DeletedItems)

		// 更新 session 缓存
		if entry.Date == todayStr {
			session.CurrentSchedule = entry.RemainingItems
		} else if entry.Date == tomorrowStr {
			session.TomorrowSchedule = entry.RemainingItems
		}
	}

	// 发送一个聚合删除成功事件（客户端只处理一个事件）
	var msg string
	if len(pd.Entries) == 1 {
		msg = fmt.Sprintf("已删除 %s 的%d个安排", pd.Entries[0].Date, totalDeleted)
	} else {
		parts := make([]string, 0, len(pd.Entries))
		for _, e := range pd.Entries {
			parts = append(parts, fmt.Sprintf("%s %d个", e.Date, len(e.DeletedItems)))
		}
		msg = fmt.Sprintf("已删除 %s 安排（共%d个）", strings.Join(parts, "、"), totalDeleted)
	}

	callback(&model.V4Event{
		Type:         model.V4EventTypeScheduleDeleted,
		DeletedCount: totalDeleted,
		Message:      msg,
	})

	session.ClearPendingDeletion()

	return map[string]interface{}{
		"message":       msg,
		"deleted_count": totalDeleted,
		"dates":         len(pd.Entries),
	}, true // 结束循环
}

// confirmFriendRemoval 确认删除好友
func (s *VoiceScheduleServiceV4) confirmFriendRemoval(
	ctx context.Context,
	session *model.V4Session,
	callback func(event *model.V4Event),
) (interface{}, bool) {
	if s.friendshipService == nil {
		session.ClearPendingDeletion()
		return map[string]string{"error": "好友服务未配置"}, false
	}

	pd := session.PendingDeletion

	// 执行删除好友
	if err := s.friendshipService.RemoveFriend(ctx, session.UserID, pd.FriendID); err != nil {
		return map[string]string{"error": "删除好友失败: " + err.Error()}, false
	}

	// 发送好友已删除事件
	callback(&model.V4Event{
		Type:    model.V4EventTypeFriendRemoved,
		Message: fmt.Sprintf("已删除好友 %s", pd.FriendName),
		SentTo:  pd.FriendName,
	})

	// 清理状态
	session.ClearPendingDeletion()
	// 如果最近查询的好友就是被删的，清理掉
	if session.LastQueriedFriend != nil && session.LastQueriedFriend.ID == pd.FriendID {
		session.LastQueriedFriend = nil
	}

	return map[string]interface{}{
		"message":     fmt.Sprintf("已删除好友 %s", pd.FriendName),
		"friend_name": pd.FriendName,
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

	// 保存 session
	if err := s.redisClient.Set(ctx, key, string(data), v4SessionTTL); err != nil {
		return err
	}

	// 副作用：保存用户上下文摘要（供状态推断使用）
	s.saveUserContextSummary(ctx, session)

	return nil
}

// saveUserContextSummary 保存用户对话上下文摘要到 Redis（供推断使用，TTL 30 分钟）
func (s *VoiceScheduleServiceV4) saveUserContextSummary(ctx context.Context, session *model.V4Session) {
	if s.redisClient == nil || session == nil || len(session.Messages) == 0 {
		return
	}

	summary := extractSessionSummary(session)
	if summary == "" {
		return
	}

	key := fmt.Sprintf("v4:user_context:%s", session.UserID)
	s.redisClient.Set(ctx, key, summary, 30*time.Minute)
}

// extractSessionSummary 从 V4 session 提取关键信息摘要（不调用 LLM）
func extractSessionSummary(session *model.V4Session) string {
	var sb strings.Builder

	// 提取最近 3 条用户消息
	userMsgs := 0
	for i := len(session.Messages) - 1; i >= 0 && userMsgs < 3; i-- {
		msg := session.Messages[i]
		if msg.Role == "user" {
			content := msg.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			sb.WriteString(fmt.Sprintf("- 用户: %s\n", content))
			userMsgs++
		}
	}

	// 提取使用过的工具
	toolSet := make(map[string]bool)
	for _, msg := range session.Messages {
		if msg.Role == "tool" && msg.Name != "" {
			toolSet[msg.Name] = true
		}
	}
	if len(toolSet) > 0 {
		var tools []string
		for t := range toolSet {
			tools = append(tools, t)
		}
		sb.WriteString(fmt.Sprintf("- 使用工具: %s\n", strings.Join(tools, ", ")))
	}

	// 提取 pending 操作
	if session.HasPendingSchedule() {
		sb.WriteString("- 待确认: 已生成时刻表待用户确认\n")
	}
	if session.HasPendingMessage() {
		sb.WriteString("- 待确认: 已草拟消息待用户确认\n")
	}
	if session.HasPendingInvite() {
		sb.WriteString("- 待确认: 已草拟邀请待用户确认\n")
	}
	if session.HasPendingDeletion() {
		sb.WriteString("- 待确认: 已生成删除预览待用户确认\n")
	}

	// 最后的 AI 回复
	for i := len(session.Messages) - 1; i >= 0; i-- {
		msg := session.Messages[i]
		if msg.Role == "assistant" && msg.Content != "" {
			content := msg.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			sb.WriteString(fmt.Sprintf("- AI回复: %s\n", content))
			break
		}
	}

	return sb.String()
}

// acquireSessionLock 获取 session 分布式锁，防止同一 session 并发处理
// 返回 true 表示获取成功，调用方必须 defer releaseSessionLock
func (s *VoiceScheduleServiceV4) acquireSessionLock(ctx context.Context, sessionID string) bool {
	if s.redisClient == nil || sessionID == "" {
		return true // 无 Redis 或无 session 时不锁
	}
	key := v4SessionLockPrefix + sessionID
	acquired, err := s.redisClient.SetNX(ctx, key, "1", v4SessionLockTTL)
	if err != nil {
		fmt.Printf("[V4] 获取 session 锁失败: %v\n", err)
		return true // 出错时不阻塞（降级处理）
	}
	return acquired
}

// releaseSessionLock 释放 session 锁
func (s *VoiceScheduleServiceV4) releaseSessionLock(ctx context.Context, sessionID string) {
	if s.redisClient == nil || sessionID == "" {
		return
	}
	key := v4SessionLockPrefix + sessionID
	_ = s.redisClient.Del(ctx, key)
}

// loadCurrentSchedule 加载用户当前时刻表（合并所有非取消的时刻表）
func (s *VoiceScheduleServiceV4) loadCurrentSchedule(ctx context.Context, session *model.V4Session) {
	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	// 合并今日所有时刻表（包括 completed 的）
	schedules, err := s.scheduleRepo.GetAllByUserAndDate(ctx, session.UserID, todayDate)
	if err == nil && len(schedules) > 0 {
		seen := make(map[string]bool)
		var merged model.ScheduleItems
		for _, sch := range schedules {
			for _, item := range sch.Items {
				key := item.StartTime + "-" + item.EndTime
				if !seen[key] {
					seen[key] = true
					merged = append(merged, item)
				}
			}
		}
		if len(merged) > 0 {
			session.CurrentSchedule = merged
			fmt.Printf("[V4] 加载用户时刻表: %d 个时段（合并 %d 个时刻表）\n", len(merged), len(schedules))
		}
	}

	// 加载明日时刻表
	tomorrowDate := todayDate.AddDate(0, 0, 1)
	tomorrowSchedules, err := s.scheduleRepo.GetAllByUserAndDate(ctx, session.UserID, tomorrowDate)
	if err == nil && len(tomorrowSchedules) > 0 {
		seen := make(map[string]bool)
		var merged model.ScheduleItems
		for _, sch := range tomorrowSchedules {
			for _, item := range sch.Items {
				key := item.StartTime + "-" + item.EndTime
				if !seen[key] {
					seen[key] = true
					merged = append(merged, item)
				}
			}
		}
		if len(merged) > 0 {
			session.TomorrowSchedule = merged
			fmt.Printf("[V4] 加载明日时刻表: %d 个时段\n", len(merged))
		}
	}
}

// loadTargetDateSchedule 按需加载目标日期的安排
// 如果目标日期是今天或明天，跳过（已在 loadCurrentSchedule 中加载）
func (s *VoiceScheduleServiceV4) loadTargetDateSchedule(ctx context.Context, session *model.V4Session, targetDate time.Time) {
	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	tomorrowDate := todayDate.AddDate(0, 0, 1)
	target := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, today.Location())

	if target.Equal(todayDate) || target.Equal(tomorrowDate) {
		return
	}

	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	schedules, err := s.scheduleRepo.GetAllByUserAndDate(ctx, session.UserID, target)
	if err == nil && len(schedules) > 0 {
		seen := make(map[string]bool)
		var merged model.ScheduleItems
		for _, sch := range schedules {
			for _, item := range sch.Items {
				key := item.StartTime + "-" + item.EndTime
				if !seen[key] {
					seen[key] = true
					merged = append(merged, item)
				}
			}
		}
		if len(merged) > 0 {
			session.TargetDateSchedule = merged
			fmt.Printf("[V4] 加载目标日期时刻表(%s): %d 个时段\n", target.Format("2006-01-02"), len(merged))
		}
	}

	session.TargetDateLabel = target.Format("2006-01-02") + "(" + weekdays[target.Weekday()] + ")"
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

// ========== 时间感知分析（Phase 7: 工具层时间感知）==========

// analyzeTimeContext 分析 plan_activities 返回值的时间上下文
// 只对新增项目做检测，不标注已有项目（避免"已过去墙"问题）
func (s *VoiceScheduleServiceV4) analyzeTimeContext(
	finalItems []model.ScheduleItem,
	newItems []model.ScheduleItem,
	existingItems []model.ScheduleItem,
	targetDate time.Time,
) []model.TimeNotice {
	notices := []model.TimeNotice{}
	now := time.Now()
	isToday := targetDate.Format("2006-01-02") == now.Format("2006-01-02")
	currentTime := now.Format("15:04")

	// 1. 冲突检测：新项目 vs 已有项目
	for _, newItem := range newItems {
		for _, exist := range existingItems {
			if s.isTimeOverlap(newItem.StartTime, newItem.EndTime, exist.StartTime, exist.EndTime) {
				notices = append(notices, model.TimeNotice{
					Type: "conflict",
					Detail: fmt.Sprintf("%s-%s %s 与已有 %s-%s %s%s 时间重叠",
						newItem.StartTime, newItem.EndTime, newItem.Status,
						exist.StartTime, exist.EndTime, exist.Emoji, exist.Status),
					ItemIdx: -1,
				})
			}
		}
	}

	// 2. 仅对今天的新增项目做 past/ongoing/soon 检测
	if isToday {
		for _, newItem := range newItems {
			if newItem.EndTime <= currentTime {
				notices = append(notices, model.TimeNotice{
					Type:    "past",
					Detail:  fmt.Sprintf("%s-%s 已过去（当前 %s）", newItem.StartTime, newItem.EndTime, currentTime),
					ItemIdx: -1,
				})
			} else if newItem.StartTime <= currentTime && newItem.EndTime > currentTime {
				notices = append(notices, model.TimeNotice{
					Type:    "ongoing",
					Detail:  fmt.Sprintf("%s-%s %s 正在进行中", newItem.StartTime, newItem.EndTime, newItem.Status),
					ItemIdx: -1,
				})
			}
		}
	}

	return prioritizeNotices(notices, 3)
}

// analyzeViewNotices 分析 view_schedule 返回值的时间上下文
// 只标注 ongoing/soon，不标 past（避免信息过载）
func (s *VoiceScheduleServiceV4) analyzeViewNotices(
	items []model.ScheduleItem,
	targetDate time.Time,
) []model.TimeNotice {
	notices := []model.TimeNotice{}
	now := time.Now()
	isToday := targetDate.Format("2006-01-02") == now.Format("2006-01-02")

	if !isToday {
		return notices
	}

	currentTime := now.Format("15:04")
	for i, item := range items {
		if item.StartTime <= currentTime && item.EndTime > currentTime {
			notices = append(notices, model.TimeNotice{
				Type:    "ongoing",
				Detail:  fmt.Sprintf("%s-%s %s%s 正在进行中", item.StartTime, item.EndTime, item.Emoji, item.Status),
				ItemIdx: i,
			})
		} else if item.StartTime > currentTime {
			// 计算距离开始的分钟数
			startParts := strings.SplitN(item.StartTime, ":", 2)
			if len(startParts) == 2 {
				startHour := 0
				startMin := 0
				fmt.Sscanf(startParts[0], "%d", &startHour)
				fmt.Sscanf(startParts[1], "%d", &startMin)
				minutesUntil := (startHour-now.Hour())*60 + (startMin - now.Minute())
				if minutesUntil > 0 && minutesUntil <= 30 {
					notices = append(notices, model.TimeNotice{
						Type:    "soon",
						Detail:  fmt.Sprintf("%s-%s %s%s 将在%d分钟后开始", item.StartTime, item.EndTime, item.Emoji, item.Status, minutesUntil),
						ItemIdx: i,
					})
				}
			}
		}
	}

	return prioritizeNotices(notices, 3)
}

// prioritizeNotices 限制最多 maxN 条，优先级: conflict > ongoing > soon > past
func prioritizeNotices(notices []model.TimeNotice, maxN int) []model.TimeNotice {
	if len(notices) <= maxN {
		return notices
	}

	priority := map[string]int{"conflict": 0, "ongoing": 1, "soon": 2, "past": 3}
	// 简单冒泡排序按优先级
	for i := 0; i < len(notices)-1; i++ {
		for j := 0; j < len(notices)-i-1; j++ {
			pi, pj := priority[notices[j].Type], priority[notices[j+1].Type]
			if pi > pj {
				notices[j], notices[j+1] = notices[j+1], notices[j]
			}
		}
	}

	return notices[:maxN]
}

// trimToolResult 裁剪工具结果，防止大返回值膨胀上下文
func trimToolResult(result string, maxLen int) string {
	if len(result) <= maxLen {
		return result
	}
	return result[:maxLen] + "...(已截断)"
}

// v4GetTimePeriod 根据小时返回当前时段名称（中文标签）
func v4GetTimePeriod(hour int) string {
	switch {
	case hour < 6:
		return "凌晨"
	case hour < 9:
		return "早晨"
	case hour < 12:
		return "上午"
	case hour < 14:
		return "中午"
	case hour < 18:
		return "下午"
	case hour < 22:
		return "晚上"
	default:
		return "深夜"
	}
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
	// 0. 预先获取 session 锁（在 ASR 之前锁住，防止并发请求操作同一 session）
	if sessionID != "" && !s.acquireSessionLock(ctx, sessionID) {
		callback(&model.V4Event{
			Type:    model.V4EventTypeChat,
			Message: "正在处理上一条消息，请稍后再试",
		})
		return nil, fmt.Errorf("session %s is locked", sessionID)
	}
	if sessionID != "" {
		defer s.releaseSessionLock(ctx, sessionID)
	}

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

	// 2. 使用文本处理流程（skipLock=true，因为已在本方法获取了锁）
	return s.processTextInputInternal(ctx, userID, sessionID, transcript, nil, callback)
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
	}, true // breakLoop: 停止循环，等待用户确认
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
	}, true // breakLoop: 停止循环，等待用户确认
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

// [已删除] executeQueryDeviceData, executeQueryMemory, executeUpdateMemory, executeGetBehaviorSuggestion
// Phase 5: 设备数据通过系统提示词注入，记忆通过 buildSystemPromptWithMemory 注入，行为建议是 LLM 自身能力

// ========== 推断模式：实现 StatusInferrer 接口 ==========

// InferWithAgent 实现 job.StatusInferrer 接口，供 StatusScheduler 自动推断使用
// sensorData 可为 nil，此时从 Redis 缓存读取
func (s *VoiceScheduleServiceV4) InferWithAgent(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest) (*model.CurrentStatusInference, error) {
	// 检查推断锁（防止自动推断过于频繁地覆盖用户主动设置）
	// 注：一键生成直接调用 InferStatus，不经过此方法，不受锁限制
	if s.redisClient != nil {
		lockKey := fmt.Sprintf("infer:lock:%s", userID)
		if locked, _ := s.redisClient.Get(ctx, lockKey); locked != "" {
			return &model.CurrentStatusInference{
				Emoji:      "⏳",
				Activity:   "状态保持中",
				Confidence: "low",
				Reasoning:  "推断锁生效中，跳过本次自动推断",
				InferredAt: time.Now().UnixMilli(),
			}, nil
		}
	}

	// 聚合推断上下文（复用 V3 的 PreGatherContext）
	deps := &agent.InferenceToolDeps{
		RedisClient:        s.redisClient,
		MemoryRepo:         s.memoryRepo,
		ScheduleRepo:       s.scheduleRepo,
		UserProfileService: s.userProfileService,
		UserID:             userID,
	}
	inferCtx := agent.PreGatherContext(ctx, deps)
	inferenceContext := agent.FormatContextForPrompt(inferCtx)

	// 调用 V4 推断
	resp, err := s.InferStatus(ctx, userID, sensorData, inferenceContext, nil)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

// GatherInferenceContext 聚合推断上下文并格式化为文本
func (s *VoiceScheduleServiceV4) GatherInferenceContext(ctx context.Context, userID string) string {
	deps := &agent.InferenceToolDeps{
		RedisClient:        s.redisClient,
		MemoryRepo:         s.memoryRepo,
		ScheduleRepo:       s.scheduleRepo,
		UserProfileService: s.userProfileService,
		UserID:             userID,
	}
	inferCtx := agent.PreGatherContext(ctx, deps)
	return agent.FormatContextForPrompt(inferCtx)
}

// ========== 推断模式辅助方法 ==========

// enrichInferenceWithGif 为推断结果生成 GIF 搜索词
func (s *VoiceScheduleServiceV4) enrichInferenceWithGif(ctx context.Context, result *model.CurrentStatusInference) {
	if result == nil || result.Activity == "" {
		return
	}

	// 先查缓存
	cacheKey := fmt.Sprintf("giphy:query:%s", result.Activity)
	if s.redisClient != nil {
		if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil && cached != "" {
			result.GiphyQuery = cached
			return
		}
	}

	// 用 LLM 翻译中文活动为英文 Giphy 搜索词
	if s.llmAdapter == nil {
		return
	}

	prompt := fmt.Sprintf(`You are generating a Giphy GIF search query. Given a user's current activity, output 2-4 English keywords that would find a relevant, visually matching animated GIF on Giphy.

Rules:
- Focus on the ACTION or SCENE, not greetings or time of day
- Use concrete visual words (e.g. "sleeping cozy bed" not "chill at home")
- Prefer verbs and objects that show up well in animations
- Do NOT include words like "good morning", "hello", "hi" etc.
- Only output the search query, nothing else

Activity: %s`, result.Activity)

	query, err := s.llmAdapter.Chat(ctx, prompt)
	if err != nil {
		fmt.Printf("[V4 Infer] GIF 翻译失败: %v\n", err)
		return
	}

	query = strings.TrimSpace(query)
	query = strings.Trim(query, "\"'`")

	if query != "" {
		result.GiphyQuery = query
		// 缓存翻译结果
		if s.redisClient != nil {
			s.redisClient.Set(ctx, cacheKey, query, 24*time.Hour)
		}
	}
}

// setInferenceLock 设置推断锁（防止 StatusScheduler 立即覆盖用户/AI 的推断）
func (s *VoiceScheduleServiceV4) setInferenceLock(ctx context.Context, userID string) {
	if s.redisClient == nil {
		return
	}
	key := fmt.Sprintf("infer:lock:%s", userID)
	s.redisClient.Set(ctx, key, "1", 10*time.Minute)
}

// savePrevInference 保存推断结果（供下次推断参考时间连续性）
func (s *VoiceScheduleServiceV4) savePrevInference(ctx context.Context, userID string, result *model.CurrentStatusInference) {
	if s.redisClient == nil || result == nil {
		return
	}
	key := fmt.Sprintf("infer:prev:%s", userID)
	data, err := json.Marshal(result)
	if err != nil {
		return
	}
	s.redisClient.Set(ctx, key, data, 2*time.Hour)
}

// cacheSensorDataForInference 缓存传感器数据到 Redis（推断模式用）
func (s *VoiceScheduleServiceV4) cacheSensorDataForInference(ctx context.Context, userID string, data *model.ExtendedStatusReportRequest) {
	if s.redisClient == nil || data == nil {
		return
	}
	extKey := fmt.Sprintf("agent:extended:%s", userID)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	s.redisClient.Set(ctx, extKey, jsonData, 30*time.Minute)
}
