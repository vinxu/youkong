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
	scheduleRepo       *repository.ScheduleRepository
	memoryRepo         *repository.MemoryRepository
	userProfileService *UserProfileService
	redisClient        *tencent.RedisClient
	llmAdapter         *agent.LLMAdapter

	// V4 工具集
	tools []*agent.Tool
}

// NewVoiceScheduleServiceV4 创建 V4 版本语音时刻表服务
func NewVoiceScheduleServiceV4(
	scheduleRepo *repository.ScheduleRepository,
	memoryRepo *repository.MemoryRepository,
	userProfileService *UserProfileService,
	redisClient *tencent.RedisClient,
	llmAPIKey string,
) *VoiceScheduleServiceV4 {
	svc := &VoiceScheduleServiceV4{
		scheduleRepo:       scheduleRepo,
		memoryRepo:         memoryRepo,
		userProfileService: userProfileService,
		redisClient:        redisClient,
		tools:              agent.V4ScheduleTools(),
	}

	// 创建 LLM 适配器
	if llmAPIKey != "" {
		svc.llmAdapter = agent.NewLLMAdapter(&agent.LLMAdapterConfig{
			APIKey: llmAPIKey,
			Model:  "qwen-max", // 使用 qwen-max，不需要思考模式
		})
	}

	return svc
}

// ProcessTextInput 处理文本输入（主入口）
// 核心设计：简单的 while 循环，让 LLM 自主决定何时调用工具
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

	// 4. 发送思考状态
	callback(&model.V4Event{
		Type:    model.V4EventTypeThinking,
		Message: "正在思考...",
	})

	// 5. 主循环（类似 Claude CLI 架构）
	err := s.processLoop(ctx, session, callback)
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

	return session, nil
}

// processLoop 主处理循环
func (s *VoiceScheduleServiceV4) processLoop(
	ctx context.Context,
	session *model.V4Session,
	callback func(event *model.V4Event),
) error {
	for i := 0; i < v4MaxLoops; i++ {
		fmt.Printf("[V4] 循环 %d/%d\n", i+1, v4MaxLoops)

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
					Type:    model.V4EventTypeChat,
					Message: response.Content,
				})
				session.AddMessage("assistant", response.Content)
			}
			break
		}

		// 4. 执行工具调用
		shouldBreak := s.executeToolCalls(ctx, session, response.ToolCalls, callback)

		// 添加 assistant 消息（包含工具调用）
		session.AddAssistantMessageWithToolCalls(response.Content, s.convertToolCalls(response.ToolCalls))

		if shouldBreak {
			break
		}
	}

	return nil
}

// buildMessages 构建 LLM 消息列表
func (s *VoiceScheduleServiceV4) buildMessages(session *model.V4Session) []agent.AgentMessage {
	messages := []agent.AgentMessage{
		agent.NewSystemMessage(s.buildSystemPrompt(session)),
	}

	// 添加会话历史（最多保留最近 10 轮）
	recentMessages := session.GetRecentMessages(10)
	for _, msg := range recentMessages {
		messages = append(messages, s.convertV4Message(msg))
	}

	return messages
}

// buildSystemPrompt 构建简化的系统提示词（~500 字符）
func (s *VoiceScheduleServiceV4) buildSystemPrompt(session *model.V4Session) string {
	now := time.Now()
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	var sb strings.Builder
	sb.WriteString("你是时刻表助手，帮助用户管理每天的状态时刻表。\n\n")

	// 当前时间
	sb.WriteString(fmt.Sprintf("当前时间：%s %s %02d:%02d\n\n",
		now.Format("2006-01-02"), weekdays[now.Weekday()], now.Hour(), now.Minute()))

	// 用户当前时刻表
	if len(session.CurrentSchedule) > 0 {
		sb.WriteString("用户当前时刻表：\n")
		for _, item := range session.CurrentSchedule {
			sb.WriteString(fmt.Sprintf("- %s-%s %s %s\n",
				item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("用户当前时刻表：（暂无）\n\n")
	}

	// 待确认的时刻表
	if session.HasPendingSchedule() {
		sb.WriteString("待确认的时刻表：\n")
		for _, item := range session.PendingSchedule {
			sb.WriteString(fmt.Sprintf("- %s-%s %s %s\n",
				item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
		sb.WriteString("\n")
	}

	// 简单的使用说明
	sb.WriteString(`你可以：
1. 直接回复用户的问题或闲聊
2. 调用工具来查看、修改、保存时刻表

回复要求：简洁友好，像朋友聊天。不要主动展示完整时刻表，除非用户明确要求"查看"或"看看"。
`)

	return sb.String()
}

// executeToolCalls 执行工具调用
// 返回 true 表示应该结束循环
func (s *VoiceScheduleServiceV4) executeToolCalls(
	ctx context.Context,
	session *model.V4Session,
	toolCalls []agent.ToolCall,
	callback func(event *model.V4Event),
) bool {
	shouldBreak := false

	for _, tc := range toolCalls {
		fmt.Printf("[V4] 执行工具: %s\n", tc.Function.Name)

		// 解析参数
		args, err := tc.ParseArguments()
		if err != nil {
			fmt.Printf("[V4] 解析参数失败: %v\n", err)
			session.AddToolResult(tc.ID, tc.Function.Name, fmt.Sprintf(`{"error": "%s"}`, err.Error()))
			continue
		}

		// 执行工具
		result, breakLoop := s.executeTool(ctx, session, tc.Function.Name, args, callback)

		// 添加工具结果到会话
		resultJSON, _ := json.Marshal(result)
		session.AddToolResult(tc.ID, tc.Function.Name, string(resultJSON))

		if breakLoop {
			shouldBreak = true
		}
	}

	return shouldBreak
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

	// 获取时刻表
	schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, session.UserID, date)
	if err != nil || schedule == nil || len(schedule.Items) == 0 {
		return map[string]interface{}{
			"message": "暂无时刻表",
			"date":    date.Format("2006-01-02"),
		}, false
	}

	// 转换为返回格式
	items := make([]map[string]string, len(schedule.Items))
	for i, item := range schedule.Items {
		items[i] = map[string]string{
			"start_time": item.StartTime,
			"end_time":   item.EndTime,
			"emoji":      item.Emoji,
			"status":     item.Status,
		}
	}

	return map[string]interface{}{
		"date":  date.Format("2006-01-02"),
		"items": items,
		"count": len(items),
	}, false
}

// executeUpdateSchedule 执行更新时刻表（生成预览）
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

	// 解析时刻表条目
	items, err := agent.ParseScheduleItems(args)
	if err != nil {
		return map[string]string{"error": err.Error()}, false
	}

	// 转换为 model.ScheduleItem
	scheduleItems := make([]model.ScheduleItem, len(items))
	for i, item := range items {
		scheduleItems[i] = model.ScheduleItem{
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			Emoji:     item.Emoji,
			Status:    item.Status,
		}
	}

	// 保存到 session 的 PendingSchedule
	session.PendingSchedule = scheduleItems
	session.PendingDate = date

	// 发送预览事件
	callback(&model.V4Event{
		Type:    model.V4EventTypeSchedulePreview,
		Items:   scheduleItems,
		Date:    date.Format("2006-01-02"),
		Message: fmt.Sprintf("已生成 %s 的时刻表预览", date.Format("01月02日")),
	})

	return map[string]interface{}{
		"message":           "时刻表预览已生成，等待用户确认",
		"date":              date.Format("2006-01-02"),
		"items_count":       len(items),
		"awaiting_approval": true,
	}, false
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
