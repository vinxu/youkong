package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
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
	sessionID string, // 可选，如果提供则尝试恢复会话
	audioData []byte,
	audioFormat string,
	callback func(event *model.VoiceScheduleEvent),
) (*model.VoiceScheduleSession, error) {
	var session *model.VoiceScheduleSession
	var isNewSession bool

	// 1. 如果提供了 sessionID，尝试恢复现有会话
	if sessionID != "" {
		existingSession, err := s.getSession(ctx, sessionID)
		if err == nil && existingSession != nil && existingSession.UserID == userID {
			session = existingSession
			isNewSession = false
			fmt.Printf("[VoiceSchedule] 恢复现有会话: %s, TargetDate=%s\n", sessionID, session.TargetDate.Format("2006-01-02"))
		}
	}

	// 2. 如果没有现有会话，创建新会话
	if session == nil {
		sessionID = uuid.New().String()
		session = &model.VoiceScheduleSession{
			UserID:              userID,
			SessionID:           sessionID,
			State:               "initial",
			TranscriptHistory:   []string{},
			ConversationHistory: []model.ConversationTurn{},
			CreatedAt:           time.Now(),
		}
		isNewSession = true
	}

	// 1.1 加载用户当前的活跃时刻表（关键：修改时需要知道已有的行程）
	// 注意：如果 session 有 TargetDate，加载那个日期的时刻表；否则加载今天的
	scheduleDate := time.Now()
	if !session.TargetDate.IsZero() {
		scheduleDate = session.TargetDate
	}
	existingSchedule, loadErr := s.scheduleRepo.GetActiveByUserAndDate(ctx, userID, scheduleDate)
	if loadErr == nil && existingSchedule != nil && len(existingSchedule.Items) > 0 {
		session.CurrentSchedule = existingSchedule.Items
		fmt.Printf("[VoiceSchedule] 加载用户已有时刻表: %d 个时段 (日期: %s)\n", len(existingSchedule.Items), scheduleDate.Format("2006-01-02"))
	} else if isNewSession {
		fmt.Println("[VoiceSchedule] 用户无已有时刻表")
	}

	// 发送会话开始事件
	callback(&model.VoiceScheduleEvent{
		Type:      model.VSEventSessionStart,
		SessionID: sessionID,
	})

	// 2. 语音识别（带超时处理）
	s.sendProgress(callback, model.ProgressRecognizing, "正在识别语音...")

	var transcript string
	var err error

	// ASR 超时设置：30秒
	asrCtx, asrCancel := context.WithTimeout(ctx, 30*time.Second)
	defer asrCancel()

	if s.asrClient != nil && s.asrClient.IsConfigured() {
		// 获取 Token
		token, tokenErr := s.asrClient.GetToken(asrCtx)
		if tokenErr != nil {
			fmt.Printf("[VoiceSchedule] 获取 ASR Token 失败: %v, 使用模拟识别\n", tokenErr)
			transcript = s.mockASR(audioData)
		} else {
			transcript, err = s.asrClient.RecognizeSpeechWithToken(asrCtx, audioData, audioFormat, token)
			if err != nil {
				if asrCtx.Err() == context.DeadlineExceeded {
					fmt.Println("[VoiceSchedule] 语音识别超时（30秒），使用模拟识别")
					callback(&model.VoiceScheduleEvent{
						Type:    model.VSEventError,
						Message: "语音识别超时，请重试",
					})
					return session, fmt.Errorf("ASR timeout")
				}
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
		s.sendProgressDetail(callback, truncateStr(userCtx.CoreMemory.BehaviorInsights, 50))
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

// ProcessTextInput 处理文本输入（测试用，跳过语音识别）
func (s *VoiceScheduleService) ProcessTextInput(
	ctx context.Context,
	userID string,
	sessionID string,
	text string,
	callback func(event *model.VoiceScheduleEvent),
) (*model.VoiceScheduleSession, error) {
	var session *model.VoiceScheduleSession
	var isNewSession bool

	// 如果提供了 sessionID，尝试获取现有会话
	if sessionID != "" {
		existingSession, err := s.getSession(ctx, sessionID)
		if err == nil && existingSession != nil && existingSession.UserID == userID {
			session = existingSession
			isNewSession = false
		}
	}

	// 如果没有现有会话，创建新会话
	if session == nil {
		sessionID = uuid.New().String()
		session = &model.VoiceScheduleSession{
			UserID:              userID,
			SessionID:           sessionID,
			State:               "initial",
			TranscriptHistory:   []string{},
			ConversationHistory: []model.ConversationTurn{},
			CreatedAt:           time.Now(),
		}
		isNewSession = true

		// 加载用户当前的活跃时刻表（关键：修改时需要知道已有的行程）
		existingSchedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, userID, time.Now())
		if err == nil && existingSchedule != nil && len(existingSchedule.Items) > 0 {
			session.CurrentSchedule = existingSchedule.Items
			fmt.Printf("[VoiceSchedule] 加载用户已有时刻表: %d 个时段\n", len(existingSchedule.Items))
		}

		// 发送会话开始事件
		callback(&model.VoiceScheduleEvent{
			Type:      model.VSEventSessionStart,
			SessionID: sessionID,
		})
	}

	// 发送识别结果（直接使用文本）
	callback(&model.VoiceScheduleEvent{
		Type: model.VSEventTranscript,
		Text: text,
	})

	session.TranscriptHistory = append(session.TranscriptHistory, text)
	s.addConversationTurn(session, "user", text)

	// 加载用户上下文（仅新会话需要）
	if isNewSession {
		s.sendProgress(callback, model.ProgressLoadingContext, "正在了解你的情况...")
		userCtx := s.BuildUserContext(ctx, userID)
		compressedCtx := s.CompressUserContext(ctx, userCtx)
		session.UserContext = compressedCtx
	}

	// LLM 分析
	s.sendProgress(callback, model.ProgressAnalyzing, "分析你的意图...")

	result, err := s.analyzeWithLLMAndContext(ctx, userID, text, session, session.UserContext)
	if err != nil {
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventError,
			Message: "分析失败: " + err.Error(),
		})
		return nil, err
	}

	// 处理 LLM 结果
	s.handleLLMResult(ctx, session, result, callback)

	// 保存会话
	if err := s.saveSession(ctx, session); err != nil {
		fmt.Printf("[VoiceScheduleText] 保存会话失败: %v\n", err)
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

	answerStr := strings.TrimSpace(answerText.String())
	session.TranscriptHistory = append(session.TranscriptHistory, "回答: "+answerStr)
	s.addConversationTurn(session, "user", answerStr)

	// 重新分析
	callback(&model.VoiceScheduleEvent{
		Type:   model.VSEventThinking,
		Status: "AI 正在重新分析...",
	})

	// 使用增强版分析（带完整上下文）
	result, err := s.analyzeWithLLMAndContext(ctx, session.UserID, answerStr, session, session.UserContext)
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
	s.addConversationTurn(session, "user", transcript)

	// 重新分析
	callback(&model.VoiceScheduleEvent{
		Type:   model.VSEventThinking,
		Status: "AI 正在分析...",
	})

	// 使用增强版分析（带完整上下文和时间推理规则）
	result, err := s.analyzeWithLLMAndContext(ctx, session.UserID, transcript, session, session.UserContext)
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

		// 确定时刻表日期（支持明天、后天等）
		scheduleDate := time.Now()
		if !session.TargetDate.IsZero() {
			scheduleDate = session.TargetDate
			fmt.Printf("[VoiceSchedule] 保存到指定日期: %s\n", scheduleDate.Format("2006-01-02"))
		}

		schedule := &model.StatusSchedule{
			UserID:       session.UserID,
			ScheduleDate: scheduleDate,
			Items:        session.CurrentSchedule,
			CurrentIndex: 0,
			Status:       model.ScheduleStatusActive,
			Visibility:   visibility,
			CircleIDs:    session.CircleIDs,
		}

		// 先取消用户该日期的活跃时刻表（而不是所有时刻表）
		_ = s.scheduleRepo.CancelUserSchedulesByDate(ctx, session.UserID, scheduleDate)

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

		// 格式化日期描述
		dateDesc := "今天"
		today := time.Now().Truncate(24 * time.Hour)
		if scheduleDate.Truncate(24*time.Hour).Equal(today.AddDate(0, 0, 1)) {
			dateDesc = "明天"
		} else if scheduleDate.Truncate(24*time.Hour).Equal(today.AddDate(0, 0, 2)) {
			dateDesc = "后天"
		} else if !scheduleDate.Truncate(24*time.Hour).Equal(today) {
			dateDesc = scheduleDate.Format("1月2日")
		}

		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventConfirmed,
			Message: fmt.Sprintf("%s的时刻表已保存", dateDesc),
		})

		// 删除会话
		s.deleteSession(ctx, session.SessionID)
		return nil
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

	// 如果 compressedCtx 为空，尝试从 session 获取
	if compressedCtx == nil {
		compressedCtx = session.UserContext
	}
	// 如果还是空，创建一个空的
	if compressedCtx == nil {
		compressedCtx = &model.CompressedUserContext{}
	}

	// 自适应思考模式：判断是否需要深度思考
	needThinking := s.shouldUseThinking(transcript, compressedCtx, session)

	// 构建 Prompt
	var prompt string
	if needThinking {
		// 复杂场景：添加额外的推理指令
		prompt = s.buildPlanModePrompt(transcript, compressedCtx, session)
		prompt = s.addDeepThinkingInstructions(prompt, transcript, session)
	} else {
		// 简单场景：标准 prompt
		prompt = s.buildPlanModePrompt(transcript, compressedCtx, session)
	}

	// LLM 超时设置：思考模式180秒，非思考模式60秒
	var llmTimeout time.Duration
	if needThinking {
		llmTimeout = 180 * time.Second // 思考模式：3分钟
		fmt.Println("[VoiceSchedule] 使用深度思考模式，超时180秒")
	} else {
		llmTimeout = 60 * time.Second // 普通模式：1分钟
	}
	llmCtx, llmCancel := context.WithTimeout(ctx, llmTimeout)
	defer llmCancel()

	// 使用 JSON Object 模式，确保输出有效 JSON
	// 注意：qwen-max 支持 JSON Object 模式
	messages := []llm.ChatMessage{
		{Role: "user", Content: prompt},
	}
	response, err := s.llmClient.ChatWithOptions(llmCtx, messages, &llm.ChatOptions{
		EnableJSONMode: true, // 启用 JSON Object 模式
		Temperature:    0.7,  // 略低的温度，提高结构化输出的稳定性
	})
	if err != nil {
		if llmCtx.Err() == context.DeadlineExceeded {
			fmt.Printf("[VoiceSchedule] LLM 调用超时（%v）\n", llmTimeout)
			return nil, fmt.Errorf("LLM timeout: 分析超时，请重试")
		}
		fmt.Printf("[VoiceSchedule] LLM 调用失败: %v\n", err)
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// 打印 LLM 响应（截断到 500 字符）
	responsePreview := response
	if len(responsePreview) > 500 {
		responsePreview = responsePreview[:500] + "..."
	}
	fmt.Printf("[VoiceSchedule] LLM 响应: %s\n", responsePreview)

	// 解析 JSON 响应
	result, err := s.parseLLMResponse(response)
	if err != nil {
		return nil, err
	}

	// 打印解析结果
	fmt.Printf("[VoiceSchedule] 解析结果: action=%s, schedule=%d项, message=%s\n",
		result.Action, len(result.Schedule), truncateStr(result.Message, 50))

	// 标记是否使用了深度思考
	result.NeedThinking = needThinking

	return result, nil
}

// addDeepThinkingInstructions 为复杂场景添加深度思考指令
func (s *VoiceScheduleService) addDeepThinkingInstructions(basePrompt, transcript string, session *model.VoiceScheduleSession) string {
	// 分析用户输入的复杂性
	instructions := []string{}

	// 检测睡眠相关
	sleepWords := []string{"睡觉", "睡眠", "入睡", "睡"}
	for _, word := range sleepWords {
		if strings.Contains(transcript, word) {
			instructions = append(instructions, "【深度分析-睡眠】用户提到睡眠，请推理睡眠结束时间（参考时间推理规则）")
			break
		}
	}

	// 检测修改意图
	modifyWords := []string{"改", "换成", "改成", "变成", "调整", "修改"}
	if len(session.CurrentSchedule) > 0 {
		for _, word := range modifyWords {
			if strings.Contains(transcript, word) {
				instructions = append(instructions, fmt.Sprintf("【深度分析-修改】用户要修改已有时刻表，当前有%d个时段，请保留未被修改的时段", len(session.CurrentSchedule)))
				break
			}
		}
	}

	// 检测时间模糊
	fuzzyWords := []string{"待会", "一会", "稍后", "晚点", "大概", "左右"}
	for _, word := range fuzzyWords {
		if strings.Contains(transcript, word) {
			instructions = append(instructions, "【深度分析-时间】用户使用模糊时间表达，请根据当前时间和上下文推断具体时间")
			break
		}
	}

	// 检测连续活动
	sequenceWords := []string{"然后", "接着", "之后", "完了", "完"}
	for _, word := range sequenceWords {
		if strings.Contains(transcript, word) {
			instructions = append(instructions, "【深度分析-衔接】用户描述连续活动，请确保时间衔接正确")
			break
		}
	}

	if len(instructions) == 0 {
		return basePrompt
	}

	return basePrompt + "\n\n## 深度分析要求\n" + strings.Join(instructions, "\n")
}

// shouldUseThinking 判断是否需要深度思考
// 返回 true 表示需要深度分析，false 表示可以快速响应
func (s *VoiceScheduleService) shouldUseThinking(
	transcript string,
	compressedCtx *model.CompressedUserContext,
	session *model.VoiceScheduleSession,
) bool {
	// 1. 睡眠相关（需要推理结束时间）
	sleepWords := []string{"睡觉", "睡眠", "入睡", "睡了", "睡"}
	for _, word := range sleepWords {
		if strings.Contains(transcript, word) {
			return true
		}
	}

	// 2. 时间模糊词检测
	fuzzyTimeWords := []string{"待会儿", "待会", "一会儿", "一会", "过会儿", "过会",
		"稍后", "晚点", "早点", "差不多", "大概", "左右"}
	for _, word := range fuzzyTimeWords {
		if strings.Contains(transcript, word) {
			return true
		}
	}

	// 3. 需要推理的词检测（依赖关系）
	inferenceWords := []string{"完了", "之后", "然后", "接着", "完", "后"}
	for _, word := range inferenceWords {
		if strings.Contains(transcript, word) {
			// "开完会去吃饭" 需要知道会议结束时间
			return true
		}
	}

	// 4. 修改相关词汇（需要保留上下文）
	modifyWords := []string{"改", "换成", "改成", "变成", "调整", "修改", "删掉", "去掉", "不要"}
	for _, word := range modifyWords {
		if strings.Contains(transcript, word) {
			return true
		}
	}

	// 5. 检查是否有已有行程（可能有冲突或修改）
	if len(session.CurrentSchedule) > 0 {
		return true
	}

	// 6. 检查今日行程（可能有冲突）
	if compressedCtx != nil && compressedCtx.TodayScheduleSummary != "" {
		return true
	}

	// 7. 信息不完整检测（没有具体时间）
	hasConcreteTime := false
	timePatterns := []string{"点", "时", "分", ":", "："}
	for _, pattern := range timePatterns {
		if strings.Contains(transcript, pattern) {
			hasConcreteTime = true
			break
		}
	}

	// 如果提到了行程但没有具体时间，需要深度思考
	scheduleWords := []string{"开会", "吃饭", "午餐", "晚餐", "会议", "约", "见面", "去", "健身", "运动", "看电影"}
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

	// 8. 简单明确的场景：不需要深度思考
	// - "我在吃饭" - 当前状态描述
	// - "取消" - 简单操作

	simplePatterns := []string{"我在", "我正在", "现在"}
	for _, pattern := range simplePatterns {
		if strings.HasPrefix(transcript, pattern) {
			return false
		}
	}

	// 默认：如果有具体时间且没有复杂词汇，不需要深度思考
	return !hasConcreteTime
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

	// 构建会话历史（包含修改链摘要）
	historyStr := ""
	if len(session.ConversationHistory) > 1 {
		historyParts := []string{}
		for _, turn := range session.ConversationHistory[:len(session.ConversationHistory)-1] {
			role := "用户"
			if turn.Role == "assistant" {
				role = "AI"
			} else if turn.Role == "system" {
				role = "系统"
			}
			historyParts = append(historyParts, fmt.Sprintf("%s: %s", role, turn.Content))
		}
		historyStr = fmt.Sprintf("\n## 对话历史\n%s\n", strings.Join(historyParts, "\n"))
	}

	// 构建当前时刻表上下文（关键：修改时必须基于此）
	scheduleContext := ""
	if len(session.CurrentSchedule) > 0 {
		scheduleLines := []string{}
		for _, item := range session.CurrentSchedule {
			executedMark := ""
			if item.Executed {
				executedMark = " [已执行]"
			}
			scheduleLines = append(scheduleLines, fmt.Sprintf("- %s-%s %s %s%s", item.StartTime, item.EndTime, item.Emoji, item.Status, executedMark))
		}
		scheduleContext = fmt.Sprintf("\n## 当前完整时刻表【修改/删除时必须基于此表】\n%s\n【注意】修改时保留未提及的时段；删除时移除指定时段\n", strings.Join(scheduleLines, "\n"))
		fmt.Printf("[VoiceSchedule] 传递给LLM的时刻表:\n%s\n", scheduleContext)
	} else {
		fmt.Println("[VoiceSchedule] 用户无已有时刻表")
	}

	// 计算明天的日期
	tomorrow := now.AddDate(0, 0, 1)
	dayAfterTomorrow := now.AddDate(0, 0, 2)

	// 会话已确定的目标日期（用于多轮对话保持一致）
	sessionTargetDateHint := ""
	todayDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if !session.TargetDate.IsZero() {
		if session.TargetDate.Equal(todayDate.AddDate(0, 0, 1)) {
			sessionTargetDateHint = fmt.Sprintf("\n\n## 【重要】当前会话已确定的目标日期\n本次对话正在讨论【明天 %s】的时刻表，除非用户明确说要改成其他日期，否则继续使用 target_date=\"%s\"\n", session.TargetDate.Format("2006-01-02"), session.TargetDate.Format("2006-01-02"))
			fmt.Printf("[VoiceSchedule] 会话目标日期: 明天 (%s)\n", session.TargetDate.Format("2006-01-02"))
		} else if session.TargetDate.Equal(todayDate.AddDate(0, 0, 2)) {
			sessionTargetDateHint = fmt.Sprintf("\n\n## 【重要】当前会话已确定的目标日期\n本次对话正在讨论【后天 %s】的时刻表，除非用户明确说要改成其他日期，否则继续使用 target_date=\"%s\"\n", session.TargetDate.Format("2006-01-02"), session.TargetDate.Format("2006-01-02"))
			fmt.Printf("[VoiceSchedule] 会话目标日期: 后天 (%s)\n", session.TargetDate.Format("2006-01-02"))
		} else if !session.TargetDate.Equal(todayDate) {
			sessionTargetDateHint = fmt.Sprintf("\n\n## 【重要】当前会话已确定的目标日期\n本次对话正在讨论【%s】的时刻表，除非用户明确说要改成其他日期，否则继续使用 target_date=\"%s\"\n", session.TargetDate.Format("1月2日"), session.TargetDate.Format("2006-01-02"))
			fmt.Printf("[VoiceSchedule] 会话目标日期: %s\n", session.TargetDate.Format("2006-01-02"))
		}
	}

	return fmt.Sprintf(`你是一个智能生活助手，帮助用户规划状态时刻表。你需要像一个聪明的秘书一样理解用户的真实意图，进行合理推理。

## 当前时间
%s %s %02d:%02d
明天是 %s
后天是 %s

## 用户上下文
%s
%s%s
## 用户最新输入
"%s"

## 日期识别规则【最重要】

### 今日参考日期（直接复制使用）
- **今天** = %s（%s）
- **明天** = %s
- **后天** = %s

### 核心规则
1. target_date 必须是 YYYY-MM-DD 格式
2. 用户说"今天" → 直接使用 **%s**
3. 用户说"明天" → 直接使用 **%s**
4. 用户说"后天" → 直接使用 **%s**
5. 用户说"今天晚上"/"今晚" → 使用 **%s**
6. 用户没说日期 → 使用会话当前目标日期

### 【极其重要】用户说"今天"时
如果用户说了"今天"/"今晚"/"今天的"等词，**必须**返回 target_date="%s"，不要返回其他日期！

### 日期切换规则
- 用户说"今天" → target_date="%s"
- 用户说"明天" → target_date="%s"
- 即使会话已有目标日期，用户明确说了不同日期，也必须覆盖
%s
### 跨天时刻表
- 用户说"明天的行程"时，返回的是明天整天的安排
- "明天全天外出"意味着：明天从起床到某个时间都在外面
- "明天晚上八点回来"意味着：明天20:00之前都在外面，20:00之后在家

## 时间推理规则【必须严格遵守】

### 1. 睡眠推理【核心规则】
- "X点睡觉"/"X点入睡"表示【开始】睡觉的时间
- **时间范围处理**：当用户说"A-B点睡觉"时，意思是"在A到B点之间入睡"
  - 【必须】取范围的【结束时间B】作为入睡时间（取较晚的那个）
  - 原因：用户表达的是入睡时间区间，取末尾更准确
  - 示例：
    - "1-2点睡觉" → start_time="02:00"（取范围末尾2点）
    - "11-12点睡" → start_time="00:00"（12点=凌晨0点）
    - "10点到11点睡觉" → start_time="23:00"（取11点）
- 睡眠结束时间推断：
  - 凌晨(0:00-4:00)入睡 → 睡到 09:00
  - 深夜(22:00-24:00)入睡 → 睡到第二天 09:00
- 完整示例：
  - "1-2点睡觉" → {"start_time":"02:00","end_time":"09:00","emoji":"😴","status":"睡觉"}
  - "12点睡" → {"start_time":"00:00","end_time":"09:00","emoji":"😴","status":"睡觉"}
  - "晚上11点睡觉" → {"start_time":"23:00","end_time":"09:00","emoji":"😴","status":"睡觉"}

### 2. 时间歧义处理
- 基础规则：
  - "1点-6点" → 凌晨（01:00-06:00）
  - "7点-11点" → 早上（07:00-11:00）
  - "12点" → 中午（12:00），除非上下文明确是凌晨
  - "1点-5点"（下午语境）→ 下午（13:00-17:00）
- 明确词汇：
  - "下午X点"/"晚上X点" → 12小时制转24小时制
  - "凌晨X点" → 00:00-06:00
- **活动类型影响时间推断**【重要】：
  - 健身/运动/跑步 → 通常是下午或晚上（6点=18:00，7点=19:00）
  - 晚餐/晚饭/吃饭（无午餐标记）→ 通常是傍晚（6点=18:00，7点=19:00）
  - 洗澡 → 通常是晚上（跟在健身后）
  - 约会/看电影/聚餐 → 通常是晚上
  - 示例："7点健身" → 19:00-20:30（不是07:00）
  - 示例："6点吃饭" → 18:00-19:00（晚餐时间）

### 3. 活动时长常识
- 未指定结束时间时的默认时长：
  - 会议/开会：2小时
  - 午餐/晚餐/吃饭：1小时
  - 健身/运动：1.5小时
  - 午休/小憩：1小时
  - 看电影：2.5小时
  - 逛街/购物：2-3小时
  - 外出/出门：如果说"全天"则从09:00到指定返回时间

### 4. 修改时保留上下文【极其重要】
- 用户修改某个时段时，【必须保留】其他未提及的时段
- 例如：原有 [09:00-12:00 工作, 12:00-13:00 午餐, 14:00-18:00 工作]
- 用户说"把下午改成开会"
- 正确结果：[09:00-12:00 工作, 12:00-13:00 午餐, 14:00-18:00 开会]
- 错误结果：[14:00-18:00 开会]（丢失了其他时段）

### 5. 替换整个时刻表（"覆盖"/"重新安排"/"按新计划"）
- 当用户说"覆盖掉"/"重新安排"/"按我新的计划"/"全部改成"时
- 使用 action="replace"，表示完全替换旧时刻表
- 示例：用户说"明天全天外出，晚上八点回来"
  - 正确：{"action":"replace","target_date":"tomorrow","schedule":[{"start_time":"09:00","end_time":"20:00","emoji":"🚶","status":"外出"}]}

### 6. 修正场景（"不对/错了/应该是"）
- 当用户说"不对/错了/应该是X点"时，表示【完全替换】之前的时间
- 不需要保留被替换的原时间段
- 示例：
  - 原有：14:00-16:00 开会
  - 用户说："不对，是3点开会"
  - 正确结果：15:00-17:00 开会（完全替换，不保留14-15点）

### 7. 时间衔接
- 连续活动自动衔接：
  - "开完会去吃饭" → 会议结束时间 = 吃饭开始时间
  - "然后"/"接着"/"之后" → 前一活动结束 = 后一活动开始

## 输出格式（必须返回有效的 JSON 格式）

1. 创建/修改时刻表（有具体行程信息时）：
{
  "action": "create",
  "target_date": "2026-02-04",
  "date_reasoning": "用户没有提及具体日期，默认使用今天",
  "schedule": [
    {"start_time": "09:00", "end_time": "12:00", "emoji": "💼", "status": "工作"},
    {"start_time": "12:00", "end_time": "13:00", "emoji": "🍚", "status": "午餐"}
  ],
  "reasoning": ["推理过程1", "推理过程2"]
}

2. 替换整个时刻表（用户说"覆盖"/"重新"/"按新计划"）：
{
  "action": "replace",
  "target_date": "2026-02-05",
  "date_reasoning": "用户说'明天'，今天是2026-02-04",
  "schedule": [全新的时刻表],
  "reasoning": ["用户要求完全替换原有时刻表"]
}

3. 查询已有时刻表（"调出"/"查看"/"显示"某天的时刻表）：
{
  "action": "query",
  "target_date": "2026-02-05",
  "date_reasoning": "用户说'明天的行程'，今天是2026-02-04，所以是2026-02-05"
}

4. 需要澄清（信息确实不足时才问，能推理就不要问）：
{
  "action": "clarify",
  "target_date": "2026-02-04",
  "date_reasoning": "默认今天",
  "questions": [
    {"id": "q1", "question": "会议几点开始？", "options": ["14:00", "15:00", "其他"], "allow_voice": true}
  ]
}

5. 描述当前状态（仅当用户明确说"我在..."/"我正在..."/"现在在..."时）：
{
  "action": "guess",
  "target_date": "2026-02-04",
  "date_reasoning": "当前状态默认今天",
  "current_status": {"emoji": "🏠📱", "status": "在家休息"},
  "reasoning": ["用户描述了当前状态"]
}

6. 删除已有行程（用户说"去掉..."/"删掉..."/"取消..."某个行程）：
{
  "action": "create",
  "target_date": "2026-02-04",
  "date_reasoning": "修改当前时刻表，使用今天日期",
  "schedule": [保留的其他时段],
  "reasoning": ["根据用户要求删除了XX时段，保留了其他时段"]
}

7. 聊天/闲聊/问答（用户不是在说行程，而是在聊天或问问题）：
{
  "action": "chat",
  "message": "你的回复内容",
  "reasoning": ["这是一个闲聊/问答，不是时刻表操作"]
}

## 意图识别规则【最重要】

### 用户反馈优先处理【最高优先级】
如果用户表达了以下任何一种情况，**必须**使用 action="chat" 回应，不要执行时刻表操作：
- 不满/抱怨："你错了"/"不对"/"又..."/"一直..."
- 要求讨论："先讨论"/"先确认"/"先商量"/"不要急着执行"
- 日期纠正："我说的是今天"/"不是明天"/"你搞错日期了"

回应示例：
{
  "action": "chat",
  "message": "抱歉给您带来困扰。让我们先确认一下您的需求：您是想修改今天的时刻表对吗？请告诉我具体想怎么调整。"
}

### 区分"聊天"和"时刻表操作"
- 聊天/问答场景（使用 action="chat"）：
  - 打招呼："你好"/"嗨"/"在吗"
  - 问问题："现在几点"/"今天星期几"/"天气怎么样"
  - 闲聊："无聊"/"陪我聊聊"/"你是谁"
  - 感谢/道别："谢谢"/"拜拜"/"好的"
  - 询问功能："你能做什么"/"怎么用"
  - **用户反馈/投诉**："你错了"/"不对"/"又..."

- 时刻表操作场景（使用其他 action）：
  - 明确说行程："下午开会"/"明天出差"
  - 修改请求："把下午改成..."/"加一个..."
  - 查询请求："调出明天的时刻表"/"看看我今天的安排"

### 关键判断原则
- **用户反馈优先**：用户表达不满时，先道歉确认，不要继续执行
- **默认是聊天**：如果用户输入不明确包含时间+活动，优先当作聊天处理
- **不要强行创建时刻表**：用户说"你好"不需要创建时刻表
- **保持友好对话**：用简洁自然的语言回复

## 规则
1. emoji 用 1-2 个表达场所+行为
2. status 描述 2-6 字
3. 时刻表按时间顺序排列，时间不能重叠
4. 【最重要】区分聊天和时刻表操作！不确定时用 chat 回复
5. 【重要】只有当用户说"我在..."/"我正在..."时才返回 guess 类型
6. 如果用户输入不明确或没有具体行程，用 chat 友好回复，不要 clarify
7. reasoning 要说明推理依据
8. 修改时【必须】输出完整时刻表，包含所有时段（包括未修改的）
9. 删除时也要输出修改后的完整时刻表
10. 只返回 JSON，不要其他内容
11. 【重要】如果用户提到"明天"/"后天"，必须在返回中包含 target_date 和 date_reasoning 字段！
12. 【重要】target_date 必须是 YYYY-MM-DD 格式，date_reasoning 解释推理过程`,
		weekday, now.Format("2006-01-02"), now.Hour(), now.Minute(),
		tomorrow.Format("2006-01-02"), dayAfterTomorrow.Format("2006-01-02"),
		contextStr, scheduleContext, historyStr, transcript,
		now.Format("2006-01-02"), weekday, tomorrow.Format("2006-01-02"), dayAfterTomorrow.Format("2006-01-02"),
		now.Format("2006-01-02"), tomorrow.Format("2006-01-02"), dayAfterTomorrow.Format("2006-01-02"), now.Format("2006-01-02"),
		now.Format("2006-01-02"), now.Format("2006-01-02"), tomorrow.Format("2006-01-02"),
		sessionTargetDateHint)
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

// smartParseDateFromInput 智能日期解析器（第二层防守）
// 不是简单的关键字匹配，而是理解语义
// 返回值: (解析出的日期, 推理原因, 是否解析成功)
func (s *VoiceScheduleService) smartParseDateFromInput(input string, currentDate time.Time) (time.Time, string, bool) {
	input = strings.ToLower(input)
	today := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, currentDate.Location())

	// 1. 尝试提取 YYYY-MM-DD 格式
	dateRegex := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
	if match := dateRegex.FindString(input); match != "" {
		if parsed, err := time.Parse("2006-01-02", match); err == nil {
			return parsed, fmt.Sprintf("从输入中提取到日期: %s", match), true
		}
	}

	// 2. 尝试提取 X月X日 格式
	monthDayRegex := regexp.MustCompile(`(\d{1,2})月(\d{1,2})[日号]`)
	if match := monthDayRegex.FindStringSubmatch(input); len(match) == 3 {
		month, _ := strconv.Atoi(match[1])
		day, _ := strconv.Atoi(match[2])
		year := currentDate.Year()
		// 如果日期已过，假设是明年
		targetDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, currentDate.Location())
		if targetDate.Before(today) {
			targetDate = targetDate.AddDate(1, 0, 0)
		}
		return targetDate, fmt.Sprintf("解析 %d月%d日 为 %s", month, day, targetDate.Format("2006-01-02")), true
	}

	// 3. 语义理解相对日期（这不是简单关键字，而是理解整句话的意图）
	// 注意：这里的顺序很重要，先检查更长的短语
	relativePatterns := []struct {
		patterns []string
		offset   int
		name     string
	}{
		{[]string{"大后天", "大后日"}, 3, "大后天"},
		{[]string{"后天", "后日", "后儿"}, 2, "后天"},
		{[]string{"明天", "明日", "明儿", "第二天"}, 1, "明天"},
		{[]string{"今天", "今日", "今儿", "当天", "本日"}, 0, "今天"},
	}

	for _, rp := range relativePatterns {
		for _, pattern := range rp.patterns {
			if strings.Contains(input, pattern) {
				targetDate := today.AddDate(0, 0, rp.offset)
				return targetDate, fmt.Sprintf("识别到'%s'，转换为 %s", rp.name, targetDate.Format("2006-01-02")), true
			}
		}
	}

	// 4. 星期几的解析
	weekdayPatterns := map[string]time.Weekday{
		"周一": time.Monday, "星期一": time.Monday,
		"周二": time.Tuesday, "星期二": time.Tuesday,
		"周三": time.Wednesday, "星期三": time.Wednesday,
		"周四": time.Thursday, "星期四": time.Thursday,
		"周五": time.Friday, "星期五": time.Friday,
		"周六": time.Saturday, "星期六": time.Saturday,
		"周日": time.Sunday, "星期日": time.Sunday, "周天": time.Sunday,
	}

	isNextWeek := strings.Contains(input, "下周") || strings.Contains(input, "下星期")
	for pattern, targetWeekday := range weekdayPatterns {
		if strings.Contains(input, pattern) {
			// 计算到目标星期几的天数
			daysUntil := int(targetWeekday - currentDate.Weekday())
			if daysUntil <= 0 {
				daysUntil += 7
			}
			if isNextWeek {
				daysUntil += 7
			}
			targetDate := today.AddDate(0, 0, daysUntil)
			return targetDate, fmt.Sprintf("识别到'%s'，转换为 %s", pattern, targetDate.Format("2006-01-02")), true
		}
	}

	// 5. 未能解析
	return time.Time{}, "", false
}

// sanitizeLLMDateString 清理 LLM 返回的畸形日期字符串
// LLM 有时会返回 Unicode 字符如 ½（分数）、Ⅵ（罗马数字）代替正常数字
func sanitizeLLMDateString(dateStr string) string {
	// Unicode 分数符号到数字的映射
	fractionMap := map[rune]string{
		'½': "5", // 有时 LLM 会把 6 错误输出为 ½
		'⅓': "3",
		'⅔': "6",
		'¼': "4",
		'¾': "7",
		'⅕': "5",
		'⅖': "4",
		'⅗': "6",
		'⅘': "8",
		'⅙': "6",
		'⅚': "8",
		'⅐': "7",
		'⅛': "8",
		'⅜': "3",
		'⅝': "6",
		'⅞': "8",
		'⅑': "9",
		'⅒': "1",
	}

	// 罗马数字到阿拉伯数字的映射（单个字符形式）
	romanMap := map[rune]string{
		'Ⅰ': "1", 'ⅰ': "1",
		'Ⅱ': "2", 'ⅱ': "2",
		'Ⅲ': "3", 'ⅲ': "3",
		'Ⅳ': "4", 'ⅳ': "4",
		'Ⅴ': "5", 'ⅴ': "5",
		'Ⅵ': "6", 'ⅵ': "6",
		'Ⅶ': "7", 'ⅶ': "7",
		'Ⅷ': "8", 'ⅷ': "8",
		'Ⅸ': "9", 'ⅸ': "9",
		'Ⅹ': "10", 'ⅹ': "10",
		'Ⅺ': "11", 'ⅺ': "11",
		'Ⅻ': "12", 'ⅻ': "12",
	}

	// 全角数字到半角数字的映射
	fullwidthMap := map[rune]string{
		'０': "0", '１': "1", '２': "2", '３': "3", '４': "4",
		'５': "5", '６': "6", '７': "7", '８': "8", '９': "9",
	}

	result := strings.Builder{}
	for _, r := range dateStr {
		if replacement, ok := fractionMap[r]; ok {
			result.WriteString(replacement)
			fmt.Printf("[VoiceSchedule] 清理畸形日期: 将 '%c' 替换为 '%s'\n", r, replacement)
		} else if replacement, ok := romanMap[r]; ok {
			result.WriteString(replacement)
			fmt.Printf("[VoiceSchedule] 清理畸形日期: 将罗马数字 '%c' 替换为 '%s'\n", r, replacement)
		} else if replacement, ok := fullwidthMap[r]; ok {
			result.WriteString(replacement)
		} else {
			result.WriteRune(r)
		}
	}

	sanitized := result.String()
	if sanitized != dateStr {
		fmt.Printf("[VoiceSchedule] 日期清理: '%s' -> '%s'\n", dateStr, sanitized)
	}
	return sanitized
}

// sanitizeLLMTimeString 清理 LLM 返回的畸形时间字符串
// 处理时间格式如 "0 9:00" -> "09:00", "13:½" -> "13:00", "1˜9:00" -> "19:00"
func sanitizeLLMTimeString(timeStr string) string {
	if timeStr == "" {
		return timeStr
	}

	// 先使用通用的字符清理（复用 sanitizeLLMDateString 的逻辑）
	result := sanitizeLLMDateString(timeStr)

	// 额外处理时间特有的问题

	// 0. 先将不间断空格（U+00A0）转为普通空格
	result = strings.ReplaceAll(result, "\u00A0", " ")

	// 1. 移除数字之间的空格（"0 9:00" -> "09:00"）
	// 使用正则：数字+空格+数字 -> 数字数字
	spacePattern := regexp.MustCompile(`(\d)\s+(\d)`)
	result = spacePattern.ReplaceAllString(result, "$1$2")

	// 2. 移除波浪号和其他特殊符号（"1˜9:00" -> "19:00"）
	specialChars := []string{"˜", "~", "～", "‾", "⁓"}
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "")
	}

	// 3. 标准化时间格式（确保是 HH:MM 格式）
	// 如果只有一位小时数，补零
	timePattern := regexp.MustCompile(`^(\d):(\d{2})$`)
	if timePattern.MatchString(result) {
		result = "0" + result
	}

	// 4. 处理可能的中文冒号
	result = strings.ReplaceAll(result, "：", ":")

	if result != timeStr {
		fmt.Printf("[VoiceSchedule] 时间清理: '%s' -> '%s'\n", timeStr, result)
	}

	return result
}

// sanitizeScheduleItems 净化时刻表项目中的时间字段
func sanitizeScheduleItems(items []model.ScheduleItem) []model.ScheduleItem {
	if len(items) == 0 {
		return items
	}

	sanitized := make([]model.ScheduleItem, len(items))
	for i, item := range items {
		sanitized[i] = model.ScheduleItem{
			StartTime: sanitizeLLMTimeString(item.StartTime),
			EndTime:   sanitizeLLMTimeString(item.EndTime),
			Emoji:     item.Emoji,
			Status:    item.Status,
			Executed:  item.Executed,
		}
	}
	return sanitized
}

// parseTargetDate 解析目标日期
func (s *VoiceScheduleService) parseTargetDate(targetDateStr string) time.Time {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if targetDateStr == "" {
		return today
	}

	// 先清理可能的畸形字符
	sanitizedDate := sanitizeLLMDateString(targetDateStr)

	switch sanitizedDate {
	case "tomorrow":
		return today.AddDate(0, 0, 1)
	case "day_after_tomorrow":
		return today.AddDate(0, 0, 2)
	case "today":
		return today
	default:
		// 尝试解析 YYYY-MM-DD 格式
		if parsed, err := time.Parse("2006-01-02", sanitizedDate); err == nil {
			return parsed
		}
		// 如果清理后仍无法解析，打印警告
		fmt.Printf("[VoiceSchedule] 警告：无法解析日期 '%s' (清理后: '%s')，使用今天\n", targetDateStr, sanitizedDate)
		return today
	}
}

// handleLLMResult 处理 LLM 分析结果
func (s *VoiceScheduleService) handleLLMResult(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	result *model.LLMVoiceAnalysisResult,
	callback func(event *model.VoiceScheduleEvent),
) {
	// 【重要】净化 LLM 返回的时刻表中的畸形时间字符
	// LLM 有时会返回 "0 9:00"、"13:½"、"1˜9:00" 等畸形格式
	if len(result.Schedule) > 0 {
		result.Schedule = sanitizeScheduleItems(result.Schedule)
	}
	if len(result.PartialSchedule) > 0 {
		result.PartialSchedule = sanitizeScheduleItems(result.PartialSchedule)
	}

	// 保存推理依据
	session.LastReasoning = result.Reasoning

	// 【分层防守】日期解析
	var targetDate time.Time
	var dateReason string
	now := time.Now()

	// 第一层：LLM 返回了日期，尝试解析
	if result.TargetDate != "" {
		parsed := s.parseTargetDate(result.TargetDate)
		if !parsed.IsZero() {
			targetDate = parsed
			dateReason = result.DateReasoning
			if dateReason == "" {
				dateReason = fmt.Sprintf("LLM 返回日期: %s", result.TargetDate)
			}
			fmt.Printf("[VoiceSchedule] 第一层(LLM): 日期=%s, 原因=%s\n", targetDate.Format("2006-01-02"), dateReason)
		}
	}

	// 第二层：智能日期解析器
	if targetDate.IsZero() && len(session.TranscriptHistory) > 0 {
		latestTranscript := session.TranscriptHistory[len(session.TranscriptHistory)-1]
		parsed, reason, ok := s.smartParseDateFromInput(latestTranscript, now)
		if ok {
			targetDate = parsed
			dateReason = fmt.Sprintf("[智能解析] %s", reason)
			fmt.Printf("[VoiceSchedule] 第二层(智能解析): 日期=%s, 原因=%s (from: %s)\n",
				targetDate.Format("2006-01-02"), reason, truncateStr(latestTranscript, 30))
		}
	}

	// 第三层：保持会话原有日期
	if targetDate.IsZero() && !session.TargetDate.IsZero() {
		targetDate = session.TargetDate
		dateReason = "保持会话原有日期"
		fmt.Printf("[VoiceSchedule] 第三层(会话保持): 日期=%s\n", targetDate.Format("2006-01-02"))
	}

	// 第四层：默认今天
	if targetDate.IsZero() {
		targetDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		dateReason = "默认使用今天"
		fmt.Printf("[VoiceSchedule] 第四层(默认): 日期=%s\n", targetDate.Format("2006-01-02"))
	}

	// 更新 result.TargetDate 为 YYYY-MM-DD 格式，供后续使用
	result.TargetDate = targetDate.Format("2006-01-02")
	result.DateReasoning = dateReason

	// 打印最终日期推理信息
	fmt.Printf("[VoiceSchedule] 最终日期: %s (reason: %s)\n", targetDate.Format("2006-01-02"), dateReason)

	// 处理目标日期：如果目标日期变了，需要重新加载该日期的时刻表
	if session.TargetDate.IsZero() || !session.TargetDate.Equal(targetDate) {
		session.TargetDate = targetDate
		fmt.Printf("[VoiceSchedule] 会话目标日期更新: %s\n", session.TargetDate.Format("2006-01-02"))

		// 加载目标日期的时刻表（如果不是 query 动作，因为 query 会自己加载）
		if result.Action != "query" {
			existingSchedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, session.UserID, targetDate)
			if err == nil && existingSchedule != nil && len(existingSchedule.Items) > 0 {
				// 如果 LLM 没有返回时刻表，使用数据库中的
				if len(result.Schedule) == 0 {
					session.CurrentSchedule = existingSchedule.Items
					fmt.Printf("[VoiceSchedule] 加载%s的时刻表: %d 个时段\n", result.TargetDate, len(existingSchedule.Items))
				}
			} else {
				fmt.Printf("[VoiceSchedule] %s暂无时刻表\n", result.TargetDate)
			}
		}
	}

	// 添加 AI 回复到对话历史
	if result.Thinking != "" {
		s.addConversationTurn(session, "assistant", result.Thinking)
	}

	fmt.Printf("[VoiceSchedule] handleLLMResult: action=%s\n", result.Action)

	switch result.Action {
	case "create", "modify":
		session.CurrentSchedule = result.Schedule
		session.State = "schedule_ready"
		callback(&model.VoiceScheduleEvent{
			Type:      model.VSEventSchedule,
			Items:     result.Schedule,
			Reasoning: result.Reasoning,
		})

	case "replace":
		// 完全替换时刻表（用户说"覆盖"/"重新安排"）
		session.CurrentSchedule = result.Schedule
		session.State = "schedule_ready"
		callback(&model.VoiceScheduleEvent{
			Type:      model.VSEventSchedule,
			Items:     result.Schedule,
			Reasoning: append([]string{"完全替换原有时刻表"}, result.Reasoning...),
		})

	case "query":
		// 查询指定日期的时刻表（targetDate 已在分层防守中解析）
		existingSchedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, session.UserID, targetDate)
		if err != nil || existingSchedule == nil || len(existingSchedule.Items) == 0 {
			// 没有找到该日期的时刻表，直接告知用户"没有"
			callback(&model.VoiceScheduleEvent{
				Type:    model.VSEventChat,
				Message: fmt.Sprintf("%s没有行程安排。", targetDate.Format("1月2日")),
			})
		} else {
			// 找到了，展示给用户（设置 IsQuery=true，前端不显示确认按钮）
			session.CurrentSchedule = existingSchedule.Items
			session.State = "schedule_ready"
			callback(&model.VoiceScheduleEvent{
				Type:      model.VSEventSchedule,
				Items:     existingSchedule.Items,
				Reasoning: []string{fmt.Sprintf("这是%s的时刻表", targetDate.Format("1月2日"))},
				IsQuery:   true, // 标记为查询模式，前端不显示确认按钮
			})
		}

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

	case "chat":
		// 聊天/问答回复（非时刻表操作）
		fmt.Printf("[VoiceSchedule] 发送 chat 事件: message=%s\n", truncateStr(result.Message, 50))
		session.State = "idle" // 保持 idle 状态，可以继续对话
		callback(&model.VoiceScheduleEvent{
			Type:      model.VSEventChat,
			Message:   result.Message,
			Reasoning: result.Reasoning,
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
		compressed.BehaviorInsights = truncateStr(userCtx.CoreMemory.BehaviorInsights, 100)
		compressed.TimePatterns = truncateStr(userCtx.CoreMemory.TimePatterns, 100)
		compressed.LocationPrefs = truncateStr(userCtx.CoreMemory.LocationPreferences, 100)
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
			parts = append(parts, fmt.Sprintf("正在「%s」", truncateStr(userCtx.DeviceData.CurrentEvent, 10)))
		} else if userCtx.DeviceData.NextEvent != "" && userCtx.DeviceData.NextEventIn > 0 {
			parts = append(parts, fmt.Sprintf("%d分钟后有「%s」",
				userCtx.DeviceData.NextEventIn, truncateStr(userCtx.DeviceData.NextEvent, 10)))
		}

		if len(parts) > 0 {
			compressed.DeviceStateSummary = strings.Join(parts, "，")
		}
	}

	return compressed
}

// CompressConversationHistory 压缩对话历史（保留关键信息）
func (s *VoiceScheduleService) CompressConversationHistory(history []model.ConversationTurn) []model.ConversationTurn {
	if len(history) <= maxConversationTurns {
		return history
	}

	// 智能压缩策略：
	// 1. 保留第1轮（用户初始意图）
	// 2. 生成中间对话的智能摘要（提取用户的所有修改请求）
	// 3. 保留最后4轮（最近上下文）

	compressed := make([]model.ConversationTurn, 0, 7)
	compressed = append(compressed, history[0]) // 首轮（用户初始输入）

	// 提取中间对话中的用户请求摘要
	middleTurns := history[1 : len(history)-4]
	userRequests := []string{}
	for _, turn := range middleTurns {
		if turn.Role == "user" {
			userRequests = append(userRequests, turn.Content)
		}
	}

	// 生成有意义的摘要
	summaryContent := ""
	if len(userRequests) > 0 {
		summaryContent = fmt.Sprintf("[历史修改记录] 用户依次进行了以下操作：%s", strings.Join(userRequests, " → "))
	} else {
		summaryContent = fmt.Sprintf("[省略中间 %d 轮对话]", len(middleTurns))
	}

	compressed = append(compressed, model.ConversationTurn{
		Role:    "system",
		Content: summaryContent,
	})

	// 最后4轮
	compressed = append(compressed, history[len(history)-4:]...)

	return compressed
}

// addConversationTurn 添加对话轮次
func (s *VoiceScheduleService) addConversationTurn(session *model.VoiceScheduleSession, role, content string) {
	turn := model.ConversationTurn{
		Role:    role,
		Content: truncateStr(content, maxTurnLength),
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

// truncateStr 截断字符串（避免与其他 service 冲突）
func truncateStr(s string, maxLen int) string {
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

// ========== 多阶段对话状态机 ==========

// PhaseManager 阶段管理器
type PhaseManager struct {
	service           *VoiceScheduleService
	understandingHandler *UnderstandingHandler
	discussingHandler    *DiscussingHandler
	planningHandler      *PlanningHandler
	approvalHandler      *ApprovalHandler
	executionHandler     *ExecutionHandler
	idleHandler          *IdleHandler
}

// NewPhaseManager 创建阶段管理器
func (s *VoiceScheduleService) NewPhaseManager() *PhaseManager {
	return &PhaseManager{
		service:              s,
		understandingHandler: NewUnderstandingHandler(s.llmClient),
		discussingHandler:    NewDiscussingHandler(s.llmClient),
		planningHandler:      NewPlanningHandler(s.llmClient),
		approvalHandler:      NewApprovalHandler(),
		executionHandler:     NewExecutionHandler(s.scheduleRepo),
		idleHandler:          NewIdleHandler(s.llmClient),
	}
}

// ProcessWithStateMachine 使用状态机处理输入
func (s *VoiceScheduleService) ProcessWithStateMachine(
	ctx context.Context,
	userID string,
	sessionID string,
	input string,
	callback func(event *model.VoiceScheduleEvent),
) (*model.VoiceScheduleSession, error) {
	pm := s.NewPhaseManager()

	// 获取或创建会话
	var session *model.VoiceScheduleSession
	var isNewSession bool

	if sessionID != "" {
		existingSession, err := s.getSession(ctx, sessionID)
		if err == nil && existingSession != nil && existingSession.UserID == userID {
			session = existingSession
			isNewSession = false
		}
	}

	if session == nil {
		sessionID = uuid.New().String()
		session = &model.VoiceScheduleSession{
			UserID:              userID,
			SessionID:           sessionID,
			State:               "initial",
			Phase:               model.PhaseUnderstanding, // 初始阶段
			TranscriptHistory:   []string{},
			ConversationHistory: []model.ConversationTurn{},
			PhaseHistory:        []model.PhaseTransition{},
			CreatedAt:           time.Now(),
		}
		isNewSession = true

		// 加载用户已有时刻表
		existingSchedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, userID, time.Now())
		if err == nil && existingSchedule != nil && len(existingSchedule.Items) > 0 {
			session.CurrentSchedule = existingSchedule.Items
			fmt.Printf("[StateMachine] 加载用户已有时刻表: %d 个时段\n", len(existingSchedule.Items))
		}

		// 发送会话开始事件
		callback(&model.VoiceScheduleEvent{
			Type:      model.VSEventSessionStart,
			SessionID: sessionID,
		})
	}

	// 发送识别结果
	callback(&model.VoiceScheduleEvent{
		Type: model.VSEventTranscript,
		Text: input,
	})

	// 记录用户输入
	session.TranscriptHistory = append(session.TranscriptHistory, input)
	s.addConversationTurn(session, "user", input)

	// 加载用户上下文（仅新会话）
	if isNewSession {
		s.sendProgress(callback, model.ProgressLoadingContext, "正在了解你的情况...")
		userCtx := s.BuildUserContext(ctx, userID)
		session.UserContext = s.CompressUserContext(ctx, userCtx)
	}

	// 使用状态机处理
	err := pm.Process(ctx, session, input, callback)
	if err != nil {
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventError,
			Message: "处理失败: " + err.Error(),
		})
		return nil, err
	}

	// 保存会话
	if err := s.saveSession(ctx, session); err != nil {
		fmt.Printf("[StateMachine] 保存会话失败: %v\n", err)
	}

	return session, nil
}

// Process 状态机主处理流程
func (pm *PhaseManager) Process(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	input string,
	callback EventCallback,
) error {
	// 获取当前阶段（如果为空，默认为 Understanding）
	currentPhase := session.Phase
	if currentPhase == "" {
		currentPhase = model.PhaseUnderstanding
	}

	fmt.Printf("[StateMachine] 当前阶段: %s, 输入: %s\n", currentPhase, truncateStr(input, 50))

	// 根据当前阶段处理
	var err error
	var nextPhase model.ConversationPhase

	switch currentPhase {
	case model.PhaseUnderstanding:
		err = pm.understandingHandler.Handle(ctx, session, input, callback)
		if err == nil {
			nextPhase = pm.understandingHandler.NextPhase(session)
		}

	case model.PhaseDiscussing:
		err = pm.discussingHandler.Handle(ctx, session, input, callback)
		if err == nil {
			nextPhase = pm.discussingHandler.NextPhase(session)
		}

	case model.PhasePlanning:
		err = pm.planningHandler.Handle(ctx, session, input, callback)
		if err == nil {
			nextPhase = pm.planningHandler.NextPhase(session)
		}

	case model.PhaseApproval:
		// 先分析用户决策
		decision := pm.approvalHandler.analyzeUserDecision(input)

		switch decision {
		case "approve":
			// 执行保存
			nextPhase = model.PhaseExecution
			err = pm.executionHandler.Handle(ctx, session, input, callback)
			if err == nil {
				nextPhase = pm.executionHandler.NextPhase(session)
			}
		case "reject":
			// 取消，返回 Idle
			nextPhase = model.PhaseCompleted
			callback(&model.VoiceScheduleEvent{
				Type:    model.VSEventChat,
				Message: "好的，已取消。",
			})
		case "modify":
			// 用户要修改，回到讨论阶段
			nextPhase = model.PhaseDiscussing
			err = pm.discussingHandler.Handle(ctx, session, input, callback)
		default:
			// 不确定，继续等待
			err = pm.approvalHandler.Handle(ctx, session, input, callback)
			nextPhase = model.PhaseApproval
		}

	case model.PhaseExecution:
		err = pm.executionHandler.Handle(ctx, session, input, callback)
		if err == nil {
			nextPhase = model.PhaseCompleted
		}

	case model.PhaseIdle:
		err = pm.idleHandler.Handle(ctx, session, input, callback)
		nextPhase = model.PhaseIdle

	case model.PhaseCompleted:
		// 完成状态，可以开始新的对话
		session.Phase = model.PhaseUnderstanding
		err = pm.understandingHandler.Handle(ctx, session, input, callback)
		if err == nil {
			nextPhase = pm.understandingHandler.NextPhase(session)
		}

	default:
		// 默认进入理解阶段
		err = pm.understandingHandler.Handle(ctx, session, input, callback)
		if err == nil {
			nextPhase = pm.understandingHandler.NextPhase(session)
		}
	}

	if err != nil {
		return err
	}

	// 自动推进到下一阶段
	if nextPhase != currentPhase && nextPhase != "" {
		return pm.transitionAndProcess(ctx, session, input, currentPhase, nextPhase, callback)
	}

	return nil
}

// transitionAndProcess 转换阶段并继续处理
func (pm *PhaseManager) transitionAndProcess(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	input string,
	fromPhase model.ConversationPhase,
	toPhase model.ConversationPhase,
	callback EventCallback,
) error {
	// 记录阶段转换
	session.PhaseHistory = append(session.PhaseHistory, model.PhaseTransition{
		From:      fromPhase,
		To:        toPhase,
		Reason:    fmt.Sprintf("从 %s 转换到 %s", fromPhase, toPhase),
		Timestamp: time.Now(),
	})
	session.Phase = toPhase

	fmt.Printf("[StateMachine] 阶段转换: %s -> %s\n", fromPhase, toPhase)

	// 某些阶段需要自动继续处理
	switch toPhase {
	case model.PhasePlanning:
		// 进入规划阶段后自动生成计划
		return pm.planningHandler.Handle(ctx, session, input, callback)

	case model.PhaseApproval:
		// 进入审批阶段后发送审批提示
		callback(&model.VoiceScheduleEvent{
			Type:       model.VSEventApprovalPrompt,
			Message:    "这是生成的时刻表，确认后将保存。说「确定」保存，或告诉我要修改的内容。",
			DraftPlan:  session.DraftPlan,
			Items:      session.DraftPlan.Schedule,
			CanApprove: true,
		})
		return nil

	case model.PhaseExecution:
		// 进入执行阶段后自动保存
		return pm.executionHandler.Handle(ctx, session, input, callback)

	case model.PhaseDiscussing:
		// 进入讨论阶段，等待用户输入
		// 生成初始澄清问题
		if session.IntentSummary != nil && len(session.IntentSummary.AmbiguityReasons) > 0 {
			clarifications := []model.ClarificationItem{}
			for i, reason := range session.IntentSummary.AmbiguityReasons {
				clarifications = append(clarifications, model.ClarificationItem{
					ID:       fmt.Sprintf("q%d", i+1),
					Question: reason,
					Reason:   "需要确认",
					Answered: false,
				})
			}
			session.Clarifications = clarifications

			callback(&model.VoiceScheduleEvent{
				Type:           model.VSEventDiscussion,
				Message:        "有几个问题想确认一下：",
				Clarifications: clarifications,
			})
		}
		return nil

	case model.PhaseIdle:
		// 闲置状态，直接处理聊天
		return pm.idleHandler.Handle(ctx, session, input, callback)

	case model.PhaseCompleted:
		// 完成状态，可以删除会话或保持
		return nil

	default:
		return nil
	}
}

// ProcessTextInputStateMachine 使用状态机处理文本输入（供外部调用）
func (s *VoiceScheduleService) ProcessTextInputStateMachine(
	ctx context.Context,
	userID string,
	sessionID string,
	text string,
	callback func(event *model.VoiceScheduleEvent),
) (*model.VoiceScheduleSession, error) {
	return s.ProcessWithStateMachine(ctx, userID, sessionID, text, callback)
}
