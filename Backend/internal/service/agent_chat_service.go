package service

import (
	"context"
	"fmt"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
	"youkong/internal/pkg/asr"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

// AgentChatService Agent 聊天服务
type AgentChatService struct {
	executor       *agent.AgentExecutor
	toolRegistry   *agent.ToolRegistry
	redisClient    *tencent.RedisClient
	userRepo       *repository.UserRepository
	friendshipRepo *repository.FriendshipRepository
	memoryRepo     *repository.MemoryRepository
	scheduleRepo   *repository.ScheduleRepository
	agentService   *AgentService
	memoryService  *MemoryService
	asrClient      *asr.AliyunASRClient // 语音识别客户端
}

// NewAgentChatService 创建 Agent 聊天服务
func NewAgentChatService(
	llmAPIKey string,
	llmModel string,
	redisClient *tencent.RedisClient,
	userRepo *repository.UserRepository,
	friendshipRepo *repository.FriendshipRepository,
	memoryRepo *repository.MemoryRepository,
	scheduleRepo *repository.ScheduleRepository,
	agentService *AgentService,
	memoryService *MemoryService,
	asrClient *asr.AliyunASRClient,
) *AgentChatService {
	// 创建 LLM 适配器
	llmAdapter := agent.NewLLMAdapter(&agent.LLMAdapterConfig{
		APIKey: llmAPIKey,
		Model:  llmModel,
	})

	// 创建工具注册中心
	toolRegistry := agent.NewToolRegistry()

	// 创建执行器
	executor := agent.NewAgentExecutor(
		llmAdapter,
		toolRegistry,
		redisClient,
		&agent.ExecutorConfig{
			TokenBudget:      8000,
			MaxIterations:    10,
			SummaryThreshold: 6000,
			SessionTTL:       30 * time.Minute,
		},
	)

	svc := &AgentChatService{
		executor:       executor,
		toolRegistry:   toolRegistry,
		redisClient:    redisClient,
		userRepo:       userRepo,
		friendshipRepo: friendshipRepo,
		memoryRepo:     memoryRepo,
		scheduleRepo:   scheduleRepo,
		agentService:   agentService,
		memoryService:  memoryService,
		asrClient:      asrClient,
	}

	return svc
}

// RegisterTools 注册工具（需要在服务创建后调用）
func (s *AgentChatService) RegisterTools(userID string) {
	// 清空现有工具
	s.toolRegistry.Clear()

	// 注册内置工具
	agent.RegisterBuiltinTools(s.toolRegistry, &agent.BuiltinToolDeps{
		CurrentUserID:            userID,
		GetFriendStatusFunc:      s.getFriendStatus,
		GetUserMemoryFunc:        s.getUserMemory,
		GetTodayScheduleFunc:     s.getTodaySchedule,
		SearchUsersFunc:          s.searchUsers,
		CreateStatusScheduleFunc: s.createStatusSchedule,
		GetFriendListFunc:        s.getFriendList,
	})
}

// AgentChatRequest Agent 聊天请求
type AgentChatRequest struct {
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message" binding:"required"`
}

// Chat 执行 Agent 聊天
func (s *AgentChatService) Chat(ctx context.Context, userID string, req *AgentChatRequest, callback agent.StreamCallback) (*agent.AgentResponse, error) {
	// 注册工具（每次请求更新当前用户）
	s.RegisterTools(userID)

	// 获取或创建会话
	session, isNew, err := s.executor.GetOrCreateSession(ctx, req.SessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}

	fmt.Printf("[AgentChat] userID=%s sessionID=%s isNew=%v message=%s\n", userID, session.SessionID, isNew, req.Message)

	// 构建用户上下文
	agentCtx := s.buildAgentContext(ctx, userID)

	// 执行
	opts := &agent.ExecuteOptions{
		Context:        agentCtx,
		SystemPrompt:   s.getSystemPrompt(),
		StreamCallback: callback,
		Temperature:    0.7,
	}

	response, err := s.executor.ExecuteAndSave(ctx, session, req.Message, opts)
	if err != nil {
		return nil, err
	}

	// 发送完成事件
	if callback != nil {
		callback(agent.NewResponseEvent(response))
		callback(agent.NewDoneEvent())
	}

	return response, nil
}

// GetSessionID 获取新的会话 ID（如果需要创建新会话）
func (s *AgentChatService) GetSessionID(ctx context.Context, userID string) string {
	session := agent.NewSession(userID, nil)
	return session.SessionID
}

// ========== 私有方法 ==========

// getSystemPrompt 获取系统提示词
func (s *AgentChatService) getSystemPrompt() string {
	return `你是「有空」应用的智能助手，帮助用户管理社交状态和约人。像一个贴心的私人助理，理解用户的自然表达。

## 你的能力（工具）
1. get_friend_status - 查看好友的状态和有空概率
2. query_calendar - 查询今天的日程安排
3. create_status_schedule - 创建状态时刻表
4. search_users - 搜索用户
5. get_user_memory - 获取用户行为规律
6. get_friend_list - 获取好友列表
7. get_current_time - 获取当前时间

## 意图识别（决定是否使用工具）

### 需要调用工具的场景：
- "帮我看看谁有空" → get_friend_status
- "今天有什么安排" → query_calendar
- "我今天上午工作下午休息" → create_status_schedule
- "设置一下今天的状态" → create_status_schedule
- "小明有空吗" → get_friend_status

### 不需要工具，直接聊天的场景：
- "你好"、"早上好" → 直接问候回复
- "谢谢"、"好的" → 直接礼貌回应
- "在干嘛"、"你是谁" → 直接介绍自己
- 问天气、问时间 → 可以用 get_current_time 或直接回复

## 时刻表创建规则（重要）

### 1. 时间推理
- 用户说"X点睡觉"，表示开始睡觉时间
  - 凌晨(0-4点)入睡 → 睡到早上 9:00
  - 晚上(22-24点)入睡 → 第二天早上 9:00
- "下午3点"明确为 15:00
- "1点-6点"默认为凌晨（除非上下文说"下午"）

### 2. 活动时长常识
- 会议：1-2 小时
- 午餐/晚餐：1 小时
- 健身：1-1.5 小时
- 午休：30 分钟 - 1 小时

### 3. 修改时保留上下文
修改时刻表时，必须保留用户未提及的时段：
- 原有：[9-12工作, 12-13午餐, 14-18工作]
- 用户说"把下午改成开会"
- 结果：[9-12工作, 12-13午餐, 14-18开会]

### 4. 时刻表格式
每条记录包含：
- start_time: HH:MM 格式
- end_time: HH:MM 格式
- emoji: 一个表情符号
- status: 简短状态描述

## 回复风格
- 简洁、口语化、友好
- 适当使用 emoji
- 2-3 句话以内
- 不要过度解释

## 决策流程
1. 理解用户意图
2. 判断是否需要工具
3. 如需要，调用工具获取信息
4. 基于工具结果组织回复
5. 简洁友好地回答用户`
}

// buildAgentContext 构建用户上下文
func (s *AgentChatService) buildAgentContext(ctx context.Context, userID string) *agent.AgentContext {
	agentCtx := agent.NewAgentContextFromTime(userID)

	// 获取用户信息
	if user, err := s.userRepo.GetByID(ctx, userID); err == nil && user != nil {
		agentCtx.Nickname = user.Nickname
	}

	// 获取核心记忆
	if s.memoryService != nil {
		if memoryResp, err := s.memoryService.GetCoreMemory(ctx, userID); err == nil && memoryResp != nil {
			agentCtx.CoreMemory = &model.CoreMemory{
				BehaviorInsights:    memoryResp.BehaviorInsights,
				TimePatterns:        memoryResp.TimePatterns,
				LocationPreferences: memoryResp.LocationPreferences,
				SocialTendency:      memoryResp.SocialTendency,
			}
		}
	}

	// 获取今日行程
	if schedules, err := s.getTodaySchedule(ctx, userID); err == nil && len(schedules) > 0 {
		items := make([]model.ScheduleItem, len(schedules))
		for i, s := range schedules {
			items[i] = model.ScheduleItem{
				StartTime: s.StartTime,
				EndTime:   s.EndTime,
				Emoji:     s.Emoji,
				Status:    s.Status,
			}
		}
		agentCtx.TodaySchedules = items
	}

	return agentCtx
}

// ========== 工具实现函数 ==========

func (s *AgentChatService) getFriendStatus(ctx context.Context, userID string, friendIDs []string) ([]agent.FriendStatusInfo, error) {
	if s.agentService == nil {
		return nil, fmt.Errorf("agent service not available")
	}

	// 获取好友有空概率
	result, err := s.agentService.GetFriendsFreeProbability(ctx, userID)
	if err != nil {
		return nil, err
	}

	if result == nil || len(result.Friends) == 0 {
		return []agent.FriendStatusInfo{}, nil
	}

	// 过滤指定好友
	friendIDSet := make(map[string]bool)
	for _, id := range friendIDs {
		friendIDSet[id] = true
	}

	var statuses []agent.FriendStatusInfo
	for _, f := range result.Friends {
		// 如果指定了好友 ID，则过滤
		if len(friendIDs) > 0 && !friendIDSet[f.FriendID] {
			continue
		}

		statuses = append(statuses, agent.FriendStatusInfo{
			FriendID:    f.FriendID,
			Name:        f.Name,
			Avatar:      f.Avatar,
			Probability: f.Probability,
			Confidence:  f.Confidence,
			Reason:      f.Reason,
			UpdatedAt:   f.UpdatedAt,
		})
	}

	return statuses, nil
}

func (s *AgentChatService) getUserMemory(ctx context.Context, userID string) (*agent.UserMemoryInfo, error) {
	if s.memoryService == nil {
		return nil, fmt.Errorf("memory service not available")
	}

	memory, err := s.memoryService.GetCoreMemory(ctx, userID)
	if err != nil {
		return nil, err
	}

	if memory == nil {
		return nil, nil
	}

	return &agent.UserMemoryInfo{
		BehaviorInsights:    memory.BehaviorInsights,
		TimePatterns:        memory.TimePatterns,
		LocationPreferences: memory.LocationPreferences,
		SocialTendency:      memory.SocialTendency,
	}, nil
}

func (s *AgentChatService) getTodaySchedule(ctx context.Context, userID string) ([]agent.ScheduleItemInfo, error) {
	if s.scheduleRepo == nil {
		return nil, nil
	}

	schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, userID, time.Now())
	if err != nil {
		return nil, err
	}

	if schedule == nil || len(schedule.Items) == 0 {
		return []agent.ScheduleItemInfo{}, nil
	}

	items := make([]agent.ScheduleItemInfo, len(schedule.Items))
	for i, item := range schedule.Items {
		items[i] = agent.ScheduleItemInfo{
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			Emoji:     item.Emoji,
			Status:    item.Status,
		}
	}

	return items, nil
}

func (s *AgentChatService) searchUsers(ctx context.Context, keyword string, limit int) ([]agent.UserInfo, error) {
	if s.userRepo == nil {
		return nil, fmt.Errorf("user repo not available")
	}

	users, err := s.userRepo.SearchByNickname(ctx, keyword, limit)
	if err != nil {
		return nil, err
	}

	result := make([]agent.UserInfo, len(users))
	for i, u := range users {
		result[i] = agent.UserInfo{
			ID:       u.ID,
			Nickname: u.Nickname,
			Avatar:   u.GetAvatar(),
		}
	}

	return result, nil
}

func (s *AgentChatService) createStatusSchedule(ctx context.Context, userID string, items []agent.ScheduleItemInfo, visibility string) error {
	if s.scheduleRepo == nil {
		return fmt.Errorf("schedule repo not available")
	}

	// 先取消用户当天的所有活动时刻表
	_ = s.scheduleRepo.CancelUserActiveSchedules(ctx, userID)

	// 转换为数据库模型
	scheduleItems := make(model.ScheduleItems, len(items))
	for i, item := range items {
		scheduleItems[i] = model.ScheduleItem{
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			Emoji:     item.Emoji,
			Status:    item.Status,
			Executed:  false,
		}
	}

	schedule := &model.StatusSchedule{
		UserID:       userID,
		ScheduleDate: time.Now(),
		Items:        scheduleItems,
		CurrentIndex: 0,
		Status:       model.ScheduleStatusActive,
		Visibility:   model.ScheduleVisibility(visibility),
	}

	return s.scheduleRepo.Create(ctx, schedule)
}

func (s *AgentChatService) getFriendList(ctx context.Context, userID string) ([]agent.FriendInfo, error) {
	if s.friendshipRepo == nil {
		return nil, fmt.Errorf("friendship repo not available")
	}

	friendships, err := s.friendshipRepo.GetFriendsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(friendships) == 0 {
		return []agent.FriendInfo{}, nil
	}

	// 获取好友用户信息
	friendIDs := make([]string, len(friendships))
	for i, f := range friendships {
		friendIDs[i] = f.FriendID
	}

	users, err := s.userRepo.GetByIDs(ctx, friendIDs)
	if err != nil {
		return nil, err
	}

	userMap := make(map[string]*model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]agent.FriendInfo, 0, len(friendships))
	for _, f := range friendships {
		if user, ok := userMap[f.FriendID]; ok {
			result = append(result, agent.FriendInfo{
				FriendID: f.FriendID,
				Nickname: user.Nickname,
				Avatar:   user.GetAvatar(),
			})
		}
	}

	return result, nil
}

// ========== 语音聊天支持 ==========

// AgentVoiceChatRequest Agent 语音聊天请求
type AgentVoiceChatRequest struct {
	SessionID   string `json:"session_id,omitempty"`
	AudioData   []byte `json:"-"`
	AudioFormat string `json:"audio_format"` // m4a, wav, mp3
}

// VoiceChat 执行 Agent 语音聊天（语音转文字 + Tool Use Loop）
func (s *AgentChatService) VoiceChat(ctx context.Context, userID string, req *AgentVoiceChatRequest, callback agent.StreamCallback) (*agent.AgentResponse, string, error) {
	// 1. 语音转文字
	transcript, err := s.transcribeAudio(ctx, req.AudioData, req.AudioFormat, callback)
	if err != nil {
		return nil, "", fmt.Errorf("语音识别失败: %w", err)
	}

	if transcript == "" {
		return nil, "", fmt.Errorf("未识别到语音内容")
	}

	fmt.Printf("[AgentVoiceChat] user=%s session=%s transcript=%s\n", userID, req.SessionID, transcript)

	// 2. 发送转写结果事件
	if callback != nil {
		callback(&agent.AgentStreamEvent{
			Type:    agent.EventTranscript,
			Content: transcript,
		})
	}

	// 3. 调用文字聊天
	textReq := &AgentChatRequest{
		SessionID: req.SessionID,
		Message:   transcript,
	}

	response, err := s.Chat(ctx, userID, textReq, callback)
	return response, transcript, err
}

// transcribeAudio 语音转文字
func (s *AgentChatService) transcribeAudio(ctx context.Context, audioData []byte, format string, callback agent.StreamCallback) (string, error) {
	// 发送识别中事件
	if callback != nil {
		callback(&agent.AgentStreamEvent{
			Type:    agent.EventRecognizing,
			Content: "识别中...",
		})
	}

	// 检查 ASR 客户端
	if s.asrClient == nil || !s.asrClient.IsConfigured() {
		// 使用 mock 数据
		return s.mockASR(audioData), nil
	}

	// 获取 Token
	token, err := s.asrClient.GetToken(ctx)
	if err != nil {
		fmt.Printf("[AgentVoiceChat] 获取 ASR Token 失败: %v, 使用 mock\n", err)
		return s.mockASR(audioData), nil
	}

	// 调用 ASR
	transcript, err := s.asrClient.RecognizeSpeechWithToken(ctx, audioData, format, token)
	if err != nil {
		fmt.Printf("[AgentVoiceChat] ASR 识别失败: %v, 使用 mock\n", err)
		return s.mockASR(audioData), nil
	}

	return transcript, nil
}

// mockASR 模拟语音识别（开发测试用）
func (s *AgentChatService) mockASR(audioData []byte) string {
	size := len(audioData)
	if size < 10000 {
		return "你好"
	} else if size < 50000 {
		return "今天天气怎么样"
	} else if size < 100000 {
		return "帮我看看谁有空"
	}
	return "我想设置一下今天的状态，上午工作，下午开会，晚上休息"
}
