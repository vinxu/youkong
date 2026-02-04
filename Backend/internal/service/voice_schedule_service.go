package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"youkong/internal/model"
	"youkong/internal/pkg/asr"
	"youkong/internal/pkg/llm"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

// 对话历史限制
const (
	maxConversationTurns = 10  // 最多保留10轮对话
	maxTurnLength        = 200 // 每轮最多200字符
)

const (
	// Redis key 前缀
	keyVoiceSession = "voice_session:%s"
	// 会话过期时间
	voiceSessionTTL = 10 * time.Minute
)

// VoiceScheduleService 语音时刻表服务
type VoiceScheduleService struct {
	scheduleRepo       *repository.ScheduleRepository
	memoryRepo         *repository.MemoryRepository
	userProfileService *UserProfileService
	redisClient        *tencent.RedisClient
	asrClient          *asr.AliyunASRClient
	llmClient          *llm.OpenRouterClient
}

// NewVoiceScheduleService 创建语音时刻表服务
func NewVoiceScheduleService(
	scheduleRepo *repository.ScheduleRepository,
	memoryRepo *repository.MemoryRepository,
	userProfileService *UserProfileService,
	redisClient *tencent.RedisClient,
	asrClient *asr.AliyunASRClient,
	llmClient *llm.OpenRouterClient,
) *VoiceScheduleService {
	return &VoiceScheduleService{
		scheduleRepo:       scheduleRepo,
		memoryRepo:         memoryRepo,
		userProfileService: userProfileService,
		redisClient:        redisClient,
		asrClient:          asrClient,
		llmClient:          llmClient,
	}
}

// ProcessVoiceInput 处理语音输入（SSE 流式）
func (s *VoiceScheduleService) ProcessVoiceInput(
	ctx context.Context,
	userID string,
	audioData []byte,
	audioFormat string,
	callback func(event *model.VoiceScheduleEvent),
) (*model.VoiceScheduleSession, error) {
	// 1. 创建会话
	sessionID := uuid.New().String()
	session := &model.VoiceScheduleSession{
		UserID:              userID,
		SessionID:           sessionID,
		State:               "initial",
		TranscriptHistory:   []string{},
		ConversationHistory: []model.ConversationTurn{},
		CreatedAt:           time.Now(),
	}

	// 发送会话开始事件
	callback(&model.VoiceScheduleEvent{
		Type:      model.VSEventSessionStart,
		SessionID: sessionID,
	})

	// 2. 语音识别
	s.sendProgress(callback, model.ProgressRecognizing, "正在识别语音...")

	var transcript string
	var err error

	if s.asrClient != nil && s.asrClient.IsConfigured() {
		// 获取 Token
		token, tokenErr := s.asrClient.GetToken(ctx)
		if tokenErr != nil {
			fmt.Printf("[VoiceSchedule] 获取 ASR Token 失败: %v, 使用模拟识别\n", tokenErr)
			transcript = s.mockASR(audioData)
		} else {
			transcript, err = s.asrClient.RecognizeSpeechWithToken(ctx, audioData, audioFormat, token)
			if err != nil {
				fmt.Printf("[VoiceSchedule] 语音识别失败: %v, 使用模拟识别\n", err)
				transcript = s.mockASR(audioData)
			}
		}
	} else {
		// ASR 未配置，使用模拟
		fmt.Println("[VoiceSchedule] ASR 未配置，使用模拟识别")
		transcript = s.mockASR(audioData)
	}

	// 发送识别结果
	callback(&model.VoiceScheduleEvent{
		Type: model.VSEventTranscript,
		Text: transcript,
	})

	session.TranscriptHistory = append(session.TranscriptHistory, transcript)
	s.addConversationTurn(session, "user", transcript)

	// 3. 加载用户上下文（Plan Mode 核心）
	s.sendProgress(callback, model.ProgressLoadingContext, "正在了解你的情况...")

	userCtx := s.BuildUserContext(ctx, userID)

	// 发送上下文相关的进度反馈
	if userCtx.DeviceData != nil && userCtx.DeviceData.CurrentEvent != "" {
		s.sendProgress(callback, model.ProgressCheckingCalendar, "查看今日日历...")
		s.sendProgressDetail(callback, fmt.Sprintf("发现正在进行「%s」", userCtx.DeviceData.CurrentEvent))
	} else if userCtx.DeviceData != nil && userCtx.DeviceData.NextEvent != "" {
		s.sendProgress(callback, model.ProgressCheckingCalendar, "查看今日日历...")
		s.sendProgressDetail(callback, fmt.Sprintf("%d分钟后有「%s」",
			userCtx.DeviceData.NextEventIn, userCtx.DeviceData.NextEvent))
	}

	if userCtx.CoreMemory != nil && userCtx.CoreMemory.BehaviorInsights != "" {
		s.sendProgress(callback, model.ProgressCheckingHistory, "回顾你的习惯...")
		s.sendProgressDetail(callback, truncateString(userCtx.CoreMemory.BehaviorInsights, 50))
	}

	if len(userCtx.TodaySchedules) > 0 {
		s.sendProgressDetail(callback, fmt.Sprintf("今日已有%d个行程", len(userCtx.TodaySchedules)))
	}

	// 压缩上下文
	compressedCtx := s.CompressUserContext(ctx, userCtx)
	session.UserContext = compressedCtx

	// 4. LLM 分析
	s.sendProgress(callback, model.ProgressAnalyzing, "分析你的意图...")

	result, err := s.analyzeWithLLMAndContext(ctx, userID, transcript, session, compressedCtx)
	if err != nil {
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventError,
			Message: "分析失败: " + err.Error(),
		})
		return nil, err
	}

	// 5. 根据 LLM 结果更新会话并发送事件
	s.handleLLMResult(ctx, session, result, callback)

	// 6. 保存会话到 Redis
	if err := s.saveSession(ctx, session); err != nil {
		fmt.Printf("[VoiceSchedule] 保存会话失败: %v\n", err)
	}

	return session, nil
}

// ProcessInteraction 处理后续交互
func (s *VoiceScheduleService) ProcessInteraction(
	ctx context.Context,
	userID string,
	req *model.VoiceScheduleInteractionRequest,
	audioData []byte, // 如果是语音输入
	callback func(event *model.VoiceScheduleEvent),
) error {
	// 1. 获取会话
	session, err := s.getSession(ctx, req.SessionID)
	if err != nil {
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventError,
			Message: "会话已失效",
		})
		return err
	}

	// 验证用户
	if session.UserID != userID {
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventError,
			Message: "无权操作此会话",
		})
		return fmt.Errorf("user mismatch")
	}

	// 2. 处理不同动作
	switch req.Action {
	case model.VSActionAnswer:
		return s.handleAnswerAction(ctx, session, req.Data, callback)

	case model.VSActionVoice, model.VSActionSupplement:
		return s.handleVoiceAction(ctx, session, audioData, callback)

	case model.VSActionConfirm:
		// 如果带有可见性设置
		if req.Data != nil && req.Data.Visibility != "" {
			return s.handleConfirmWithVisibility(ctx, session, req.Data.Visibility, req.Data.CircleIDs, callback)
		}
		return s.handleConfirmAction(ctx, session, callback)

	case model.VSActionCancel:
		return s.handleCancelAction(ctx, session, callback)

	default:
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventError,
			Message: "未知操作",
		})
		return fmt.Errorf("unknown action: %s", req.Action)
	}
}

// handleAnswerAction 处理回答澄清问题
func (s *VoiceScheduleService) handleAnswerAction(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	data *model.VoiceScheduleInteractionData,
	callback func(event *model.VoiceScheduleEvent),
) error {
	if data == nil || len(data.Answers) == 0 {
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventError,
			Message: "请提供回答",
		})
		return fmt.Errorf("no answers provided")
	}

	// 将回答转换为文本
	var answerText strings.Builder
	for _, answer := range data.Answers {
		answerText.WriteString(answer)
		answerText.WriteString(" ")
	}

	session.TranscriptHistory = append(session.TranscriptHistory, "回答: "+answerText.String())

	// 重新分析
	callback(&model.VoiceScheduleEvent{
		Type:   model.VSEventThinking,
		Status: "AI 正在重新分析...",
	})

	// 构建完整上下文
	fullTranscript := strings.Join(session.TranscriptHistory, "\n")
	result, err := s.analyzeWithLLM(ctx, session.UserID, fullTranscript, session)
	if err != nil {
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventError,
			Message: "分析失败",
		})
		return err
	}

	s.handleLLMResult(ctx, session, result, callback)

	return s.saveSession(ctx, session)
}

// handleVoiceAction 处理语音输入（补充或回答）
func (s *VoiceScheduleService) handleVoiceAction(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	audioData []byte,
	callback func(event *model.VoiceScheduleEvent),
) error {
	// 语音识别
	callback(&model.VoiceScheduleEvent{
		Type:   model.VSEventRecognizing,
		Status: "正在识别语音...",
	})

	var transcript string
	var err error

	if s.asrClient != nil && s.asrClient.IsConfigured() {
		token, _ := s.asrClient.GetToken(ctx)
		transcript, err = s.asrClient.RecognizeSpeechWithToken(ctx, audioData, "m4a", token)
		if err != nil {
			transcript = s.mockASR(audioData)
		}
	} else {
		transcript = s.mockASR(audioData)
	}

	callback(&model.VoiceScheduleEvent{
		Type: model.VSEventTranscript,
		Text: transcript,
	})

	session.TranscriptHistory = append(session.TranscriptHistory, transcript)

	// 重新分析
	callback(&model.VoiceScheduleEvent{
		Type:   model.VSEventThinking,
		Status: "AI 正在分析...",
	})

	fullTranscript := strings.Join(session.TranscriptHistory, "\n")
	result, err := s.analyzeWithLLM(ctx, session.UserID, fullTranscript, session)
	if err != nil {
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventError,
			Message: "分析失败",
		})
		return err
	}

	s.handleLLMResult(ctx, session, result, callback)

	return s.saveSession(ctx, session)
}

// handleConfirmAction 处理确认
func (s *VoiceScheduleService) handleConfirmAction(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	callback func(event *model.VoiceScheduleEvent),
) error {
	// 保存时刻表到数据库
	if len(session.CurrentSchedule) > 0 {
		// 设置可见性（使用会话中的设置，或默认为所有好友可见）
		visibility := session.Visibility
		if visibility == "" {
			visibility = model.VisibilityAllFriends
		}

		schedule := &model.StatusSchedule{
			UserID:       session.UserID,
			ScheduleDate: time.Now(),
			Items:        session.CurrentSchedule,
			CurrentIndex: 0,
			Status:       model.ScheduleStatusActive,
			Visibility:   visibility,
			CircleIDs:    session.CircleIDs,
		}

		// 先取消用户之前的活跃时刻表
		_ = s.scheduleRepo.CancelUserActiveSchedules(ctx, session.UserID)

		// 创建新时刻表
		if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
			callback(&model.VoiceScheduleEvent{
				Type:    model.VSEventError,
				Message: "保存失败",
			})
			return err
		}

		// 保存到状态记忆
		s.saveToMemory(ctx, session)
	} else if session.CurrentStatusGuess != nil {
		// 保存当前状态猜测到记忆
		s.saveStatusGuessToMemory(ctx, session)
	}

	// 删除会话
	s.deleteSession(ctx, session.SessionID)

	callback(&model.VoiceScheduleEvent{
		Type:    model.VSEventConfirmed,
		Message: "已保存",
	})

	return nil
}

// handleConfirmWithVisibility 处理带可见性的确认
func (s *VoiceScheduleService) handleConfirmWithVisibility(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	visibility model.ScheduleVisibility,
	circleIDs []string,
	callback func(event *model.VoiceScheduleEvent),
) error {
	// 设置可见性
	session.Visibility = visibility
	session.CircleIDs = circleIDs

	return s.handleConfirmAction(ctx, session, callback)
}

// handleCancelAction 处理取消
func (s *VoiceScheduleService) handleCancelAction(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	callback func(event *model.VoiceScheduleEvent),
) error {
	// 删除会话
	s.deleteSession(ctx, session.SessionID)
	return nil
}

// analyzeWithLLM 使用 LLM 分析语音内容（旧版，保持兼容）
func (s *VoiceScheduleService) analyzeWithLLM(
	ctx context.Context,
	userID string,
	transcript string,
	session *model.VoiceScheduleSession,
) (*model.LLMVoiceAnalysisResult, error) {
	if s.llmClient == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}

	// 获取用户画像
	var profileType string
	if s.userProfileService != nil {
		profileType, _ = s.userProfileService.GetSimpleProfileType(userID)
	}

	// 获取最近的状态记忆
	var recentMemory []*model.UserStatusMemory
	if s.memoryRepo != nil {
		recentMemory, _ = s.memoryRepo.GetRecentUserStatusMemory(ctx, userID, 10)
	}

	// 构建 Prompt
	prompt := s.buildAnalysisPrompt(transcript, profileType, recentMemory, session)

	// 调用 LLM（不使用思考模式，更快响应）
	response, err := s.llmClient.Chat(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// 解析 JSON 响应
	return s.parseLLMResponse(response)
}

// analyzeWithLLMAndContext 使用 LLM 分析语音内容（Plan Mode 增强版，支持自适应思考）
func (s *VoiceScheduleService) analyzeWithLLMAndContext(
	ctx context.Context,
	userID string,
	transcript string,
	session *model.VoiceScheduleSession,
	compressedCtx *model.CompressedUserContext,
) (*model.LLMVoiceAnalysisResult, error) {
	if s.llmClient == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}

	// 自适应思考模式：判断是否需要深度思考
	needThinking := s.shouldUseThinking(transcript, compressedCtx, session)

	// 构建 Plan Mode 风格的 Prompt
	prompt := s.buildPlanModePrompt(transcript, compressedCtx, session)

	var response string
	var err error

	if needThinking {
		// 复杂场景：使用思考模式（如果 LLM 支持）
		response, err = s.llmClient.Chat(ctx, prompt)
	} else {
		// 简单场景：快速响应
		response, err = s.llmClient.Chat(ctx, prompt)
	}

	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// 解析 JSON 响应
	result, err := s.parseLLMResponse(response)
	if err != nil {
		return nil, err
	}

	// 标记是否使用了深度思考
	result.NeedThinking = needThinking

	return result, nil
}

// shouldUseThinking 判断是否需要深度思考
// 返回 true 表示需要深度分析，false 表示可以快速响应
func (s *VoiceScheduleService) shouldUseThinking(
	transcript string,
	compressedCtx *model.CompressedUserContext,
	session *model.VoiceScheduleSession,
) bool {
	// 1. 时间模糊词检测
	fuzzyTimeWords := []string{"待会儿", "待会", "一会儿", "一会", "过会儿", "过会",
		"稍后", "晚点", "早点", "差不多", "大概", "左右"}
	for _, word := range fuzzyTimeWords {
		if strings.Contains(transcript, word) {
			return true
		}
	}

	// 2. 需要推理的词检测（依赖关系）
	inferenceWords := []string{"完了", "之后", "然后", "接着", "完", "后"}
	for _, word := range inferenceWords {
		if strings.Contains(transcript, word) {
			// "开完会去吃饭" 需要知道会议结束时间
			return true
		}
	}

	// 3. 检查是否有已有行程（可能有冲突）
	if len(session.CurrentSchedule) > 0 {
		return true
	}

	// 4. 检查今日行程（可能有冲突）
	if compressedCtx != nil && compressedCtx.TodayScheduleSummary != "" {
		// 有已有行程，需要检查冲突
		return true
	}

	// 5. 信息不完整检测（没有具体时间）
	hasConcreteTime := false
	timePatterns := []string{"点", "时", "分", ":", "："}
	for _, pattern := range timePatterns {
		if strings.Contains(transcript, pattern) {
			hasConcreteTime = true
			break
		}
	}

	// 如果提到了行程但没有具体时间，需要深度思考
	scheduleWords := []string{"开会", "吃饭", "午餐", "晚餐", "会议", "约", "见面", "去"}
	hasScheduleIntent := false
	for _, word := range scheduleWords {
		if strings.Contains(transcript, word) {
			hasScheduleIntent = true
			break
		}
	}
	if hasScheduleIntent && !hasConcreteTime {
		return true
	}

	// 6. 简单明确的场景：不需要深度思考
	// - "我在吃饭" - 当前状态
	// - "下午2点到4点开会" - 时间明确
	// - "取消下午的安排" - 意图明确

	simplePatterns := []string{"我在", "我正在", "现在", "取消"}
	for _, pattern := range simplePatterns {
		if strings.HasPrefix(transcript, pattern) {
			return false
		}
	}

	// 默认：简单响应
	return false
}

// parseLLMResponse 解析 LLM 响应
func (s *VoiceScheduleService) parseLLMResponse(response string) (*model.LLMVoiceAnalysisResult, error) {
	var result model.LLMVoiceAnalysisResult

	// 清理响应
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		fmt.Printf("[VoiceSchedule] LLM 响应解析失败: %v, response: %s\n", err, response)
		// 返回默认的猜测结果
		return &model.LLMVoiceAnalysisResult{
			Action: "guess",
			CurrentStatus: &model.CurrentStatusGuess{
				Emoji:  "🤔",
				Status: "状态未知",
				Reason: "无法理解您的输入",
			},
		}, nil
	}

	return &result, nil
}

// buildPlanModePrompt 构建 Plan Mode 风格的 Prompt
func (s *VoiceScheduleService) buildPlanModePrompt(
	transcript string,
	compressedCtx *model.CompressedUserContext,
	session *model.VoiceScheduleSession,
) string {
	now := time.Now()
	weekday := getChineseWeekday(now.Weekday())

	// 构建上下文部分
	contextParts := []string{}

	if compressedCtx.ProfileSummary != "" {
		contextParts = append(contextParts, fmt.Sprintf("用户画像: %s", compressedCtx.ProfileSummary))
	}
	if compressedCtx.TodayScheduleSummary != "" {
		contextParts = append(contextParts, fmt.Sprintf("今日行程: %s", compressedCtx.TodayScheduleSummary))
	}
	if compressedCtx.BehaviorInsights != "" {
		contextParts = append(contextParts, fmt.Sprintf("历史习惯: %s", compressedCtx.BehaviorInsights))
	}
	if compressedCtx.TimePatterns != "" {
		contextParts = append(contextParts, fmt.Sprintf("时间规律: %s", compressedCtx.TimePatterns))
	}
	if compressedCtx.DeviceStateSummary != "" {
		contextParts = append(contextParts, fmt.Sprintf("当前状态: %s", compressedCtx.DeviceStateSummary))
	}

	contextStr := strings.Join(contextParts, "\n")
	if contextStr == "" {
		contextStr = "暂无上下文信息"
	}

	// 构建会话历史
	historyStr := ""
	if len(session.ConversationHistory) > 1 {
		historyParts := []string{}
		for _, turn := range session.ConversationHistory[:len(session.ConversationHistory)-1] {
			role := "用户"
			if turn.Role == "assistant" {
				role = "AI"
			}
			historyParts = append(historyParts, fmt.Sprintf("%s: %s", role, turn.Content))
		}
		historyStr = fmt.Sprintf("\n## 对话历史\n%s\n", strings.Join(historyParts, "\n"))
	}

	// 构建当前时刻表上下文（如果有修改意图）
	scheduleContext := ""
	if len(session.CurrentSchedule) > 0 {
		scheduleLines := []string{}
		for _, item := range session.CurrentSchedule {
			scheduleLines = append(scheduleLines, fmt.Sprintf("- %s-%s %s %s", item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
		scheduleContext = fmt.Sprintf("\n## 当前时刻表（用户可能在修改）\n%s\n", strings.Join(scheduleLines, "\n"))
	}

	return fmt.Sprintf(`你是一个智能生活助手，帮助用户规划状态时刻表。你的工作方式类似于一个贴心的秘书：
先理解用户的意图，结合上下文信息，然后给出建议。

## 当前时间
%s %02d:%02d

## 用户上下文
%s
%s%s
## 用户最新语音
"%s"

## 任务
分析用户输入，返回 JSON 格式结果。

### 输出格式

1. 如果可以生成时刻表：
{
  "action": "create",
  "schedule": [
    {"start_time": "12:00", "end_time": "13:00", "emoji": "🏨🍚", "status": "在酒店午餐"}
  ],
  "reasoning": ["推理依据1", "推理依据2"]
}

2. 如果需要修改已有时刻表：
{
  "action": "modify",
  "schedule": [更新后的完整时刻表],
  "reasoning": ["修改原因"]
}

3. 如果需要澄清信息：
{
  "action": "clarify",
  "questions": [
    {"id": "q1", "question": "开会是几点开始？", "options": ["14:00", "15:00", "其他"], "allow_voice": true}
  ]
}

4. 如果无行程信息，猜测当前状态：
{
  "action": "guess",
  "current_status": {"emoji": "🏠📱", "status": "在家刷手机"},
  "reasoning": ["位置在家", "当前时间是休息时段"]
}

## 规则
1. emoji 用 1-2 个表达场所+行为
2. status 描述 2-6 字
3. 时刻表按时间顺序排列
4. 充分利用上下文（用户画像、历史习惯、设备状态）
5. reasoning 要说明推理依据，让用户理解你的判断
6. 如果有已有行程，注意检查时间冲突
7. 只返回 JSON，不要解释`, weekday, now.Hour(), now.Minute(), contextStr, scheduleContext, historyStr, transcript)
}

// buildAnalysisPrompt 构建 LLM 分析 Prompt
func (s *VoiceScheduleService) buildAnalysisPrompt(
	transcript string,
	profileType string,
	recentMemory []*model.UserStatusMemory,
	session *model.VoiceScheduleSession,
) string {
	now := time.Now()
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	weekday := weekdays[now.Weekday()]

	// 构建记忆上下文
	memoryContext := ""
	if len(recentMemory) > 0 {
		var memoryLines []string
		for _, m := range recentMemory {
			memoryLines = append(memoryLines, fmt.Sprintf("- %s %s", m.Emoji, m.Status))
		}
		memoryContext = fmt.Sprintf("\n## 用户历史状态记忆\n%s\n", strings.Join(memoryLines, "\n"))
	}

	// 构建会话上下文
	sessionContext := ""
	if len(session.CurrentSchedule) > 0 {
		var scheduleLines []string
		for _, item := range session.CurrentSchedule {
			scheduleLines = append(scheduleLines, fmt.Sprintf("- %s-%s %s %s", item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
		sessionContext = fmt.Sprintf("\n## 当前会话上下文（用户可能在修改）\n当前时刻表：\n%s\n", strings.Join(scheduleLines, "\n"))
	}

	// 用户角色
	profileName := "普通用户"
	if profileType != "" {
		profileName = model.GetProfileTypeName(profileType)
	}

	return fmt.Sprintf(`你是一个生活状态分析助手。用户通过语音告诉你他的当前状态或行程安排。
用户可能会继续说话来修改之前的安排。

## 用户信息
- 角色: %s
- 当前时间: %s %02d:%02d
%s%s
## 用户语音内容
"%s"

## 任务
分析用户输入，返回 JSON 格式结果：

### 1. 新建行程（首次输入或明确新行程）
{
  "action": "create",
  "schedule": [
    {"start_time": "12:00", "end_time": "13:00", "emoji": "🏨🍚", "status": "在酒店午餐"},
    {"start_time": "13:00", "end_time": "14:00", "emoji": "😴", "status": "午休"}
  ]
}

### 2. 修改行程（"把午休改成开会"）
{
  "action": "modify",
  "schedule": [更新后的完整时刻表]
}

### 3. 取消行程（"取消下午的安排"）
{
  "action": "cancel",
  "cancelled_items": ["13:00-14:00"],
  "schedule": [剩余的时刻表]
}

### 4. 无行程信息（用户只是描述当前状态）
{
  "action": "guess",
  "current_status": {"emoji": "🏠📱", "status": "在家刷手机"},
  "reason": "您的输入没有明确的行程信息"
}

### 5. 需要澄清（信息不完整时）
{
  "action": "clarify",
  "questions": [
    {
      "id": "q1",
      "question": "开会是几点开始？",
      "options": ["14:00", "15:00", "16:00"],
      "allow_voice": true
    }
  ],
  "partial_schedule": [
    {"start_time": "?", "end_time": "?", "emoji": "💼", "status": "开会"}
  ]
}

## 要求
1. emoji 用 1-2 个表达场所+行为
2. status 描述 2-6 字
3. 时刻表按时间顺序排列
4. 理解修改意图（"改成"、"换成"、"取消"、"不要了"等）
5. 信息不完整时，主动提问（时间、地点不明确时）
6. 问题选项要简洁，最多 3-4 个
7. 只返回 JSON，不要解释`, profileName, weekday, now.Hour(), now.Minute(), memoryContext, sessionContext, transcript)
}

// handleLLMResult 处理 LLM 分析结果
func (s *VoiceScheduleService) handleLLMResult(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	result *model.LLMVoiceAnalysisResult,
	callback func(event *model.VoiceScheduleEvent),
) {
	// 保存推理依据
	session.LastReasoning = result.Reasoning

	// 添加 AI 回复到对话历史
	if result.Thinking != "" {
		s.addConversationTurn(session, "assistant", result.Thinking)
	}

	switch result.Action {
	case "create", "modify":
		session.CurrentSchedule = result.Schedule
		session.State = "schedule_ready"
		callback(&model.VoiceScheduleEvent{
			Type:      model.VSEventSchedule,
			Items:     result.Schedule,
			Reasoning: result.Reasoning,
		})

	case "cancel":
		session.CurrentSchedule = result.Schedule
		session.State = "schedule_ready"
		callback(&model.VoiceScheduleEvent{
			Type:      model.VSEventSchedule,
			Items:     result.Schedule,
			Reasoning: result.Reasoning,
		})

	case "guess":
		session.CurrentStatusGuess = result.CurrentStatus
		session.State = "status_guess"
		callback(&model.VoiceScheduleEvent{
			Type:       model.VSEventCurrentStatus,
			Emoji:      result.CurrentStatus.Emoji,
			StatusText: result.CurrentStatus.Status,
			Reason:     result.Reason,
			Reasoning:  result.Reasoning,
		})

	case "clarify":
		session.PendingQuestions = result.Questions
		session.PartialSchedule = result.PartialSchedule
		session.State = "clarifying"
		callback(&model.VoiceScheduleEvent{
			Type:      model.VSEventClarify,
			Questions: result.Questions,
			Items:     result.PartialSchedule,
		})
	}
}

// saveToMemory 保存时刻表到状态记忆
func (s *VoiceScheduleService) saveToMemory(ctx context.Context, session *model.VoiceScheduleSession) {
	if s.memoryRepo == nil {
		return
	}

	for _, item := range session.CurrentSchedule {
		memory := &model.UserStatusMemory{
			UserID:    session.UserID,
			Emoji:     item.Emoji,
			Status:    item.Status,
			CreatedAt: time.Now(),
		}
		_ = s.memoryRepo.SaveUserStatusMemory(ctx, memory)
	}
}

// saveStatusGuessToMemory 保存状态猜测到记忆
func (s *VoiceScheduleService) saveStatusGuessToMemory(ctx context.Context, session *model.VoiceScheduleSession) {
	if s.memoryRepo == nil || session.CurrentStatusGuess == nil {
		return
	}

	memory := &model.UserStatusMemory{
		UserID:    session.UserID,
		Emoji:     session.CurrentStatusGuess.Emoji,
		Status:    session.CurrentStatusGuess.Status,
		CreatedAt: time.Now(),
	}
	_ = s.memoryRepo.SaveUserStatusMemory(ctx, memory)
}

// saveSession 保存会话到 Redis
func (s *VoiceScheduleService) saveSession(ctx context.Context, session *model.VoiceScheduleSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	key := fmt.Sprintf(keyVoiceSession, session.SessionID)
	return s.redisClient.Set(ctx, key, data, voiceSessionTTL)
}

// getSession 从 Redis 获取会话
func (s *VoiceScheduleService) getSession(ctx context.Context, sessionID string) (*model.VoiceScheduleSession, error) {
	key := fmt.Sprintf(keyVoiceSession, sessionID)
	data, err := s.redisClient.GetBytes(ctx, key)
	if err != nil {
		return nil, err
	}

	var session model.VoiceScheduleSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// deleteSession 删除会话
func (s *VoiceScheduleService) deleteSession(ctx context.Context, sessionID string) {
	key := fmt.Sprintf(keyVoiceSession, sessionID)
	_ = s.redisClient.Del(ctx, key)
}

// mockASR 模拟语音识别（开发测试用）
func (s *VoiceScheduleService) mockASR(audioData []byte) string {
	// 根据音频大小模拟不同的识别结果
	size := len(audioData)
	if size < 10000 {
		return "我现在在家休息"
	} else if size < 50000 {
		return "下午两点开会，五点去吃饭"
	} else {
		return "我今天下午两点要开会，然后五点钟去吃晚饭，晚上八点回家"
	}
}

// GetScheduleRepo 获取 Repository（用于定时任务）
func (s *VoiceScheduleService) GetScheduleRepo() *repository.ScheduleRepository {
	return s.scheduleRepo
}

// ========== Plan Mode 上下文构建 ==========

// BuildUserContext 构建用户完整上下文
func (s *VoiceScheduleService) BuildUserContext(ctx context.Context, userID string) *model.UserContext {
	userCtx := &model.UserContext{}
	now := time.Now()

	// 设置当前时间信息
	userCtx.CurrentTime = now.Format("15:04")
	userCtx.Weekday = getChineseWeekday(now.Weekday())

	// 1. 获取用户画像
	if s.userProfileService != nil {
		profile, err := s.userProfileService.GetProfile(userID)
		if err == nil && profile != nil {
			userCtx.Profile = profile
		}
	}

	// 2. 获取今日已有行程
	if s.scheduleRepo != nil {
		schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, userID, now)
		if err == nil && schedule != nil {
			userCtx.TodaySchedules = []model.ScheduleItem(schedule.Items)
		}
	}

	// 3. 获取最近的状态记忆
	if s.memoryRepo != nil {
		memories, err := s.memoryRepo.GetRecentUserStatusMemory(ctx, userID, 20)
		if err == nil && len(memories) > 0 {
			recentStatuses := make([]model.StatusMemoryItem, 0, len(memories))
			for _, m := range memories {
				recentStatuses = append(recentStatuses, model.StatusMemoryItem{
					Emoji:     m.Emoji,
					Status:    m.Status,
					Timestamp: m.CreatedAt.Format("01-02 15:04"),
				})
			}
			userCtx.RecentStatuses = recentStatuses
		}

		// 获取核心记忆
		coreMemory, err := s.memoryRepo.GetCoreMemory(ctx, userID)
		if err == nil && coreMemory != nil {
			userCtx.CoreMemory = coreMemory
		}
	}

	// 4. 获取最近的设备数据（从 Redis 缓存）
	userCtx.DeviceData = s.getLatestDeviceData(ctx, userID)

	return userCtx
}

// getLatestDeviceData 从 Redis 获取最近的设备数据
func (s *VoiceScheduleService) getLatestDeviceData(ctx context.Context, userID string) *model.DeviceContextData {
	if s.redisClient == nil {
		return nil
	}

	// 尝试从 Redis 获取用户实时状态
	key := fmt.Sprintf("user:realtime:%s", userID)
	data, err := s.redisClient.GetBytes(ctx, key)
	if err != nil || len(data) == 0 {
		return nil
	}

	var status model.UserRealtimeStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil
	}

	deviceData := &model.DeviceContextData{
		CurrentPlace: string(status.Location.PlaceType),
		AtPlaceSince: status.Location.AtPlaceSinceMinutes,
	}

	// 屏幕状态
	if status.Screen.IsActive {
		deviceData.IsScreenActive = true
		deviceData.ActivityType = string(status.Screen.ActivityType)
	}

	return deviceData
}

// CompressUserContext 压缩用户上下文（控制在 ~1000 字符内）
func (s *VoiceScheduleService) CompressUserContext(ctx context.Context, userCtx *model.UserContext) *model.CompressedUserContext {
	compressed := &model.CompressedUserContext{}

	// 1. 用户画像压缩：提取关键信息
	if userCtx.Profile != nil {
		profileType := model.GetProfileTypeName(string(userCtx.Profile.OccupationType))
		workSchedule := ""
		switch userCtx.Profile.WorkSchedule {
		case model.WorkScheduleRegular9to5:
			workSchedule = "朝九晚五"
		case model.WorkScheduleFlexible:
			workSchedule = "弹性工作"
		case model.WorkScheduleShift:
			workSchedule = "倒班制"
		}
		if workSchedule != "" {
			compressed.ProfileSummary = fmt.Sprintf("%s，%s", profileType, workSchedule)
		} else {
			compressed.ProfileSummary = profileType
		}
	}

	// 2. 今日行程压缩：只保留时间+状态
	if len(userCtx.TodaySchedules) > 0 {
		summaries := make([]string, 0, 3)
		for i, s := range userCtx.TodaySchedules {
			if i >= 3 {
				break
			}
			summaries = append(summaries, fmt.Sprintf("%s %s", s.StartTime, s.Status))
		}
		compressed.TodayScheduleSummary = fmt.Sprintf("已有%d个行程：%s",
			len(userCtx.TodaySchedules), strings.Join(summaries, "、"))
	}

	// 3. 历史规律：复用 CoreMemory 的 LLM 提炼结果
	if userCtx.CoreMemory != nil {
		compressed.BehaviorInsights = truncateString(userCtx.CoreMemory.BehaviorInsights, 100)
		compressed.TimePatterns = truncateString(userCtx.CoreMemory.TimePatterns, 100)
		compressed.LocationPrefs = truncateString(userCtx.CoreMemory.LocationPreferences, 100)
	}

	// 4. 设备状态压缩
	if userCtx.DeviceData != nil {
		parts := make([]string, 0, 3)

		// 位置
		if userCtx.DeviceData.CurrentPlace != "" {
			placeNames := map[string]string{
				"home":    "在家",
				"work":    "在公司",
				"leisure": "在休闲场所",
				"transit": "在路上",
			}
			if name, ok := placeNames[userCtx.DeviceData.CurrentPlace]; ok {
				if userCtx.DeviceData.AtPlaceSince > 0 {
					parts = append(parts, fmt.Sprintf("%s已%d分钟", name, userCtx.DeviceData.AtPlaceSince))
				} else {
					parts = append(parts, name)
				}
			}
		}

		// 运动状态
		if userCtx.DeviceData.IsMoving {
			parts = append(parts, "正在移动")
		}

		// 日历事件
		if userCtx.DeviceData.CurrentEvent != "" {
			parts = append(parts, fmt.Sprintf("正在「%s」", truncateString(userCtx.DeviceData.CurrentEvent, 10)))
		} else if userCtx.DeviceData.NextEvent != "" && userCtx.DeviceData.NextEventIn > 0 {
			parts = append(parts, fmt.Sprintf("%d分钟后有「%s」",
				userCtx.DeviceData.NextEventIn, truncateString(userCtx.DeviceData.NextEvent, 10)))
		}

		if len(parts) > 0 {
			compressed.DeviceStateSummary = strings.Join(parts, "，")
		}
	}

	return compressed
}

// CompressConversationHistory 压缩对话历史
func (s *VoiceScheduleService) CompressConversationHistory(history []model.ConversationTurn) []model.ConversationTurn {
	if len(history) <= maxConversationTurns {
		return history
	}

	// 保留：第1轮（初始意图）+ 最后5轮（近期上下文）
	compressed := make([]model.ConversationTurn, 0, 7)
	compressed = append(compressed, history[0]) // 首轮

	// 中间部分生成摘要
	middleCount := len(history) - 6
	compressed = append(compressed, model.ConversationTurn{
		Role:    "system",
		Content: fmt.Sprintf("[省略中间 %d 轮对话...]", middleCount),
	})

	// 最后5轮
	compressed = append(compressed, history[len(history)-5:]...)

	return compressed
}

// addConversationTurn 添加对话轮次
func (s *VoiceScheduleService) addConversationTurn(session *model.VoiceScheduleSession, role, content string) {
	turn := model.ConversationTurn{
		Role:    role,
		Content: truncateString(content, maxTurnLength),
		Time:    time.Now().Format("15:04:05"),
	}
	session.ConversationHistory = append(session.ConversationHistory, turn)

	// 压缩历史
	if len(session.ConversationHistory) > maxConversationTurns {
		session.ConversationHistory = s.CompressConversationHistory(session.ConversationHistory)
	}
}

// ========== 辅助函数 ==========

// getChineseWeekday 获取中文星期
func getChineseWeekday(weekday time.Weekday) string {
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return weekdays[weekday]
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen-3]) + "..."
	}
	return s
}

// sendProgress 发送过程反馈事件
func (s *VoiceScheduleService) sendProgress(callback func(event *model.VoiceScheduleEvent), action model.ProgressAction, message string) {
	callback(&model.VoiceScheduleEvent{
		Type:    model.VSEventProgress,
		Action:  string(action),
		Message: message,
	})
}

// sendProgressDetail 发送过程反馈详情
func (s *VoiceScheduleService) sendProgressDetail(callback func(event *model.VoiceScheduleEvent), detail string) {
	callback(&model.VoiceScheduleEvent{
		Type:   model.VSEventProgress,
		Detail: detail,
	})
}
