package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/llm"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

const (
	// Redis key 前缀
	keyUserStatus  = "agent:status:%s"  // 用户实时状态
	keyUserPattern = "agent:pattern:%s" // 用户历史规律

	// 状态过期时间
	statusTTL  = 24 * time.Hour // 状态24小时过期（配合时间调整，可用于推理）
	patternTTL = 24 * time.Hour // 规律24小时过期
)

// AgentService Agent 服务
type AgentService struct {
	redisClient        *tencent.RedisClient
	cosClient          *tencent.COSClient
	userRepo           *repository.UserRepository
	friendshipRepo     *repository.FriendshipRepository
	memoryRepo         *repository.MemoryRepository
	scheduleRepo       *repository.ScheduleRepository
	userProfileService *UserProfileService
	llmClient          *llm.OpenRouterClient
	holmesAnalyzer     *llm.HolmesAnalyzer
}

// NewAgentService 创建 Agent 服务
func NewAgentService(
	redisClient *tencent.RedisClient,
	userRepo *repository.UserRepository,
	friendshipRepo *repository.FriendshipRepository,
	memoryRepo *repository.MemoryRepository,
	scheduleRepo *repository.ScheduleRepository,
	userProfileService *UserProfileService,
	llmClient *llm.OpenRouterClient,
) *AgentService {
	var holmesAnalyzer *llm.HolmesAnalyzer
	if llmClient != nil {
		holmesAnalyzer = llm.NewHolmesAnalyzer(llmClient)
	}
	return &AgentService{
		redisClient:        redisClient,
		userRepo:           userRepo,
		friendshipRepo:     friendshipRepo,
		memoryRepo:         memoryRepo,
		scheduleRepo:       scheduleRepo,
		userProfileService: userProfileService,
		llmClient:          llmClient,
		holmesAnalyzer:     holmesAnalyzer,
	}
}

// SetCOSClient 设置 COS 客户端（用于 GIF 缓存）
func (s *AgentService) SetCOSClient(cosClient *tencent.COSClient) {
	s.cosClient = cosClient
}

// UploadGifToCOS 将客户端上传的 GIF 二进制数据存储到 COS，返回 COS URL
func (s *AgentService) UploadGifToCOS(ctx context.Context, reader io.Reader, size int64) (string, error) {
	if s.cosClient == nil {
		return "", fmt.Errorf("COS 客户端未配置")
	}

	// 先读取数据计算 MD5 作为文件名
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取数据失败: %w", err)
	}

	cosKey := fmt.Sprintf("gif/status/%s.gif", tencent.MD5Key(string(data)))

	// 检查是否已存在（避免重复上传）
	if s.cosClient.Exists(ctx, cosKey) {
		return s.cosClient.GetPublicURL(cosKey), nil
	}

	// 上传到 COS
	cosURL, err := s.cosClient.UploadFromReader(ctx, cosKey, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("上传 COS 失败: %w", err)
	}

	return cosURL, nil
}

// ReportStatus 上报用户状态
func (s *AgentService) ReportStatus(ctx context.Context, userID string, req *model.StatusReportRequest) error {
	status := model.UserRealtimeStatus{
		UserID:    userID,
		UpdatedAt: time.Now(),
	}

	if req.Screen != nil {
		status.Screen = *req.Screen
	}
	if req.Location != nil {
		status.Location = *req.Location
		// 保存城市信息
		if req.Location.City != "" {
			status.City = req.Location.City
		}
	}

	// 序列化并存入 Redis
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}

	key := fmt.Sprintf(keyUserStatus, userID)
	if err := s.redisClient.Set(ctx, key, data, statusTTL); err != nil {
		return fmt.Errorf("save status to redis: %w", err)
	}

	return nil
}

// GetUserStatus 获取用户状态
func (s *AgentService) GetUserStatus(ctx context.Context, userID string) (*model.UserRealtimeStatus, error) {
	key := fmt.Sprintf(keyUserStatus, userID)
	data, err := s.redisClient.GetBytes(ctx, key)
	if tencent.IsNil(err) {
		return nil, nil // 无数据
	}
	if err != nil {
		return nil, fmt.Errorf("get status from redis: %w", err)
	}

	var status model.UserRealtimeStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("unmarshal status: %w", err)
	}

	return &status, nil
}

// GetUserPatterns 获取用户历史规律
func (s *AgentService) GetUserPatterns(ctx context.Context, userID string) (*model.UserPatterns, error) {
	key := fmt.Sprintf(keyUserPattern, userID)
	data, err := s.redisClient.GetBytes(ctx, key)
	if tencent.IsNil(err) {
		// 无数据，返回默认值
		return &model.UserPatterns{
			CurrentHourFreeRate:      50,
			CurrentWeekdayFreeRate:   50,
			AtHomeFreeRate:           70,
			AtWorkAfterHoursFreeRate: 40,
			AvgResponseTimeMinutes:   10,
			ResponseRate:             50,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get patterns from redis: %w", err)
	}

	var patterns model.UserPatterns
	if err := json.Unmarshal(data, &patterns); err != nil {
		return nil, fmt.Errorf("unmarshal patterns: %w", err)
	}

	return &patterns, nil
}

// GetAgentExposedData 获取 Agent 对外暴露的数据
func (s *AgentService) GetAgentExposedData(ctx context.Context, userID string) (*model.AgentExposedData, error) {
	status, err := s.GetUserStatus(ctx, userID)
	if err != nil {
		return nil, err
	}

	patterns, err := s.GetUserPatterns(ctx, userID)
	if err != nil {
		return nil, err
	}

	data := &model.AgentExposedData{
		Patterns: *patterns,
	}

	if status != nil {
		data.Realtime.Screen = status.Screen
		data.Realtime.Location = status.Location
		data.DataQuality.ScreenDataAgeSeconds = int(time.Since(status.UpdatedAt).Seconds())
		data.DataQuality.LocationDataAgeSeconds = int(time.Since(status.UpdatedAt).Seconds())
	} else {
		// 无实时数据
		data.DataQuality.ScreenDataAgeSeconds = -1
		data.DataQuality.LocationDataAgeSeconds = -1
	}

	data.DataQuality.PatternsSampleSize = 100 // TODO: 从实际数据计算

	return data, nil
}

// GetFriendsFreeProbability 获取好友有空概率列表
func (s *AgentService) GetFriendsFreeProbability(ctx context.Context, userID string) (*model.FreeProbabilityResponse, error) {
	// 1. 获取好友列表
	friends, err := s.friendshipRepo.GetFriendsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get friends: %w", err)
	}

	// 2. 获取所有好友的用户信息
	friendIDs := make([]string, len(friends))
	for i, f := range friends {
		friendIDs[i] = f.FriendID
	}

	users, err := s.userRepo.GetByIDs(ctx, friendIDs)
	if err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}

	userMap := make(map[string]*model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 3. 对每个好友计算有空概率
	recommendations := make([]model.FriendRecommendation, 0, len(friends))
	now := time.Now()

	for _, friend := range friends {
		user, ok := userMap[friend.FriendID]
		if !ok {
			continue
		}

		// 获取好友的 Agent 数据
		agentData, err := s.GetAgentExposedData(ctx, friend.FriendID)
		if err != nil {
			// 无法获取数据，跳过
			continue
		}

		// 计算有空概率（使用 LLM 生成隐私安全的理由）
		probability, reason, confidence := s.calculateFreeProbability(ctx, agentData, now)

		rec := model.FriendRecommendation{
			FriendID:    friend.FriendID,
			Name:        user.Nickname,
			Avatar:      user.GetAvatar(),
			Probability: probability,
			Confidence:  confidence,
			Reason:      reason,
			Color:       model.GetProbabilityColor(probability),
			UpdatedAt:   now.UnixMilli(),
		}

		recommendations = append(recommendations, rec)
	}

	// 4. 按概率排序（降序）
	for i := 0; i < len(recommendations)-1; i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[j].Probability > recommendations[i].Probability {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}

	return &model.FreeProbabilityResponse{
		Friends:     recommendations,
		GeneratedAt: now.UnixMilli(),
	}, nil
}

// calculateFreeProbability 计算有空概率
func (s *AgentService) calculateFreeProbability(ctx context.Context, data *model.AgentExposedData, now time.Time) (probability int, reason string, confidence string) {
	score := 50 // 基础分
	hasData := false

	// 脱敏后的状态（用于 LLM）
	sanitized := llm.SanitizedUserState{
		ActivityLevel:    llm.ActivityLevelNone,
		ActivityCategory: llm.ActivityCategoryIdle,
		LocationCategory: llm.LocationCategoryUnknown,
		TimePeriod:       getTimePeriod(now),
	}

	// ========== 屏幕状态分析 ==========
	if data.DataQuality.ScreenDataAgeSeconds >= 0 && data.DataQuality.ScreenDataAgeSeconds < 300 {
		hasData = true
		screen := data.Realtime.Screen

		if screen.IsActive {
			// 计算活跃度级别（脱敏）
			if screen.SessionDurationMinutes >= 30 {
				sanitized.ActivityLevel = llm.ActivityLevelHigh
			} else if screen.SessionDurationMinutes >= 10 {
				sanitized.ActivityLevel = llm.ActivityLevelMedium
			} else {
				sanitized.ActivityLevel = llm.ActivityLevelLow
			}

			// 活动类别（脱敏，不暴露具体 APP）
			switch screen.ActivityType {
			case model.ActivityEntertainment:
				sanitized.ActivityCategory = llm.ActivityCategoryLeisure
				if screen.SessionDurationMinutes >= 10 {
					score += 30
				} else {
					score += 15
				}
			case model.ActivityProductivity:
				sanitized.ActivityCategory = llm.ActivityCategoryWork
				score -= 10
			case model.ActivityCommunication:
				sanitized.ActivityCategory = llm.ActivityCategorySocial
				score += 10
			}
		} else {
			sanitized.ActivityLevel = llm.ActivityLevelNone
			sanitized.ActivityCategory = llm.ActivityCategoryIdle
			if screen.LastActiveMinutesAgo > 120 {
				score -= 25
			} else if screen.LastActiveMinutesAgo > 30 {
				score -= 10
			}
		}
	}

	// ========== 位置分析 ==========
	if data.DataQuality.LocationDataAgeSeconds >= 0 && data.DataQuality.LocationDataAgeSeconds < 600 {
		hasData = true
		location := data.Realtime.Location

		switch location.PlaceType {
		case model.PlaceHome:
			sanitized.LocationCategory = llm.LocationCategoryHome
			score += 15
		case model.PlaceWork:
			sanitized.LocationCategory = llm.LocationCategoryWork
			if isWorkHours(now) {
				score -= 20
			} else {
				score += 5
			}
		case model.PlaceLeisure:
			sanitized.LocationCategory = llm.LocationCategoryOutside
			score += 10
		}
	}

	// ========== 时间分析 ==========
	timeScore, _ := getTimeScore(now)
	score += timeScore

	// ========== 历史规律加成 ==========
	if data.Patterns.CurrentHourFreeRate > 70 {
		score += 10
	} else if data.Patterns.CurrentHourFreeRate < 30 {
		score -= 10
	}

	// ========== 边界处理 ==========
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	// ========== 确定置信度 ==========
	if !hasData {
		confidence = "low"
		reason = "数据不足"
		probability = -1
		return
	}

	if data.DataQuality.PatternsSampleSize > 50 && data.DataQuality.ScreenDataAgeSeconds < 60 {
		confidence = "high"
	} else if data.DataQuality.PatternsSampleSize > 20 {
		confidence = "medium"
	} else {
		confidence = "low"
	}

	// ========== 使用 LLM 生成隐私安全的理由 ==========
	sanitized.Probability = score
	if s.llmClient != nil {
		llmReason, err := s.llmClient.GenerateFreeReason(ctx, sanitized)
		if err == nil && llmReason != "" {
			// 清理 LLM 输出
			reason = strings.TrimSpace(llmReason)
			// 限制长度
			if len([]rune(reason)) > 15 {
				reason = string([]rune(reason)[:15])
			}
		} else {
			// LLM 失败，使用默认理由
			reason = getDefaultReason(score)
		}
	} else {
		// 无 LLM，使用默认理由
		reason = getDefaultReason(score)
	}

	probability = score
	return
}

// getTimePeriod 获取时间段（脱敏）
func getTimePeriod(t time.Time) llm.TimePeriod {
	weekday := t.Weekday()
	hour := t.Hour()

	isWeekend := weekday == time.Saturday || weekday == time.Sunday

	if isWeekend {
		return llm.TimePeriodWeekend
	}

	if hour >= 0 && hour < 7 {
		return llm.TimePeriodLateNight
	}
	if hour >= 7 && hour < 9 {
		return llm.TimePeriodEarlyMorning
	}
	if hour >= 9 && hour < 12 {
		return llm.TimePeriodWorkHours
	}
	if hour >= 12 && hour < 14 {
		return llm.TimePeriodLunchBreak
	}
	if hour >= 14 && hour < 18 {
		return llm.TimePeriodWorkHours
	}
	if hour >= 18 && hour < 23 {
		return llm.TimePeriodAfterWork
	}
	return llm.TimePeriodLateNight
}

// getDefaultReason 获取默认理由（无 LLM 时使用）
func getDefaultReason(score int) string {
	if score >= 80 {
		return "可能有空"
	}
	if score >= 60 {
		return "应该有空"
	}
	if score >= 40 {
		return "不太确定"
	}
	if score >= 20 {
		return "可能在忙"
	}
	return "应该在忙"
}

// isWorkHours 判断是否工作时间
func isWorkHours(t time.Time) bool {
	weekday := t.Weekday()
	hour := t.Hour()

	// 周末不是工作时间
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	// 工作日 9:00-18:00
	return hour >= 9 && hour < 18
}

// getTimeScore 根据时间返回分数和理由
func getTimeScore(t time.Time) (int, string) {
	weekday := t.Weekday()
	hour := t.Hour()

	isWeekend := weekday == time.Saturday || weekday == time.Sunday

	if isWeekend {
		if hour >= 10 && hour <= 22 {
			return 15, "周末"
		}
		return 0, ""
	}

	// 工作日
	if hour >= 9 && hour < 12 {
		return -20, ""
	}
	if hour >= 12 && hour < 14 {
		return 5, "午休时间"
	}
	if hour >= 14 && hour < 18 {
		return -15, ""
	}
	if hour >= 18 && hour < 22 {
		if weekday == time.Friday {
			return 20, "周五晚上"
		}
		return 20, "下班了"
	}
	if hour >= 22 {
		return 10, "晚上"
	}
	// 0:00 - 9:00
	return -30, ""
}

// ========== 福尔摩斯推理框架 ==========

// Redis key 前缀（福尔摩斯分析缓存）
const (
	keyHolmesCache = "agent:holmes:%s" // 福尔摩斯分析缓存
	holmesCacheTTL = 10 * time.Minute  // 缓存10分钟
)

// ReportExtendedStatus 上报扩展状态（包含日历、移动等数据）
func (s *AgentService) ReportExtendedStatus(ctx context.Context, userID string, req *model.ExtendedStatusReportRequest) (*model.HolmesResult, error) {
	// 1. 保存原始状态到 Redis
	status := model.UserRealtimeStatus{
		UserID:    userID,
		UpdatedAt: time.Now(),
	}
	if req.Screen != nil {
		status.Screen = *req.Screen
	}
	if req.Location != nil {
		status.Location = *req.Location
	}

	data, err := json.Marshal(status)
	if err != nil {
		return nil, fmt.Errorf("marshal status: %w", err)
	}

	key := fmt.Sprintf(keyUserStatus, userID)
	if err := s.redisClient.Set(ctx, key, data, statusTTL); err != nil {
		return nil, fmt.Errorf("save status to redis: %w", err)
	}

	// 2. 保存扩展状态到 Redis（用于福尔摩斯分析）
	extData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal extended status: %w", err)
	}

	extKey := fmt.Sprintf("agent:extended:%s", userID)
	if err := s.redisClient.Set(ctx, extKey, extData, statusTTL); err != nil {
		return nil, fmt.Errorf("save extended status to redis: %w", err)
	}

	// 3. 执行福尔摩斯分析
	if s.holmesAnalyzer == nil {
		return nil, nil // 无分析器，跳过分析
	}

	input := &llm.HolmesInput{
		Status:    req,
		Timestamp: time.Now(),
	}

	result, err := s.holmesAnalyzer.Analyze(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("holmes analysis failed: %w", err)
	}

	// 4. 缓存分析结果
	if result != nil {
		cacheData, _ := json.Marshal(result)
		cacheKey := fmt.Sprintf(keyHolmesCache, userID)
		_ = s.redisClient.Set(ctx, cacheKey, cacheData, holmesCacheTTL)
	}

	return result, nil
}

// ReportExtendedStatusStream 流式上报状态（实时输出推理过程）
func (s *AgentService) ReportExtendedStatusStream(ctx context.Context, userID string, req *model.ExtendedStatusReportRequest, callback func(event interface{})) (*model.HolmesResult, error) {
	// 1. 保存原始状态到 Redis
	status := model.UserRealtimeStatus{
		UserID:    userID,
		UpdatedAt: time.Now(),
	}
	if req.Screen != nil {
		status.Screen = *req.Screen
	}
	if req.Location != nil {
		status.Location = *req.Location
	}

	data, err := json.Marshal(status)
	if err != nil {
		return nil, fmt.Errorf("marshal status: %w", err)
	}

	key := fmt.Sprintf(keyUserStatus, userID)
	if err := s.redisClient.Set(ctx, key, data, statusTTL); err != nil {
		return nil, fmt.Errorf("save status to redis: %w", err)
	}

	// 2. 保存扩展状态到 Redis
	extData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal extended status: %w", err)
	}

	extKey := fmt.Sprintf("agent:extended:%s", userID)
	if err := s.redisClient.Set(ctx, extKey, extData, statusTTL); err != nil {
		return nil, fmt.Errorf("save extended status to redis: %w", err)
	}

	// 3. 流式执行福尔摩斯分析
	if s.holmesAnalyzer == nil {
		return nil, fmt.Errorf("holmes analyzer not available")
	}

	input := &llm.HolmesInput{
		Status:    req,
		Timestamp: time.Now(),
	}

	// 使用流式分析
	result, err := s.holmesAnalyzer.AnalyzeStream(ctx, input, func(event *llm.HolmesStreamEvent) {
		callback(event)
	})
	if err != nil {
		return nil, fmt.Errorf("holmes stream analysis failed: %w", err)
	}

	// 4. 缓存分析结果
	if result != nil {
		cacheData, _ := json.Marshal(result)
		cacheKey := fmt.Sprintf(keyHolmesCache, userID)
		_ = s.redisClient.Set(ctx, cacheKey, cacheData, holmesCacheTTL)
	}

	return result, nil
}

// GetHolmesAnalysis 获取福尔摩斯分析结果（优先使用缓存）
func (s *AgentService) GetHolmesAnalysis(ctx context.Context, userID string) (*model.HolmesResult, error) {
	// 1. 尝试从缓存获取
	cacheKey := fmt.Sprintf(keyHolmesCache, userID)
	cacheData, err := s.redisClient.GetBytes(ctx, cacheKey)
	if err == nil && cacheData != nil {
		var result model.HolmesResult
		if json.Unmarshal(cacheData, &result) == nil {
			return &result, nil
		}
	}

	// 2. 缓存未命中，尝试重新分析
	extKey := fmt.Sprintf("agent:extended:%s", userID)
	extData, err := s.redisClient.GetBytes(ctx, extKey)
	if tencent.IsNil(err) || extData == nil {
		return nil, nil // 无数据
	}
	if err != nil {
		return nil, fmt.Errorf("get extended status: %w", err)
	}

	var req model.ExtendedStatusReportRequest
	if err := json.Unmarshal(extData, &req); err != nil {
		return nil, fmt.Errorf("unmarshal extended status: %w", err)
	}

	// 3. 执行福尔摩斯分析
	if s.holmesAnalyzer == nil {
		return nil, nil
	}

	input := &llm.HolmesInput{
		Status:    &req,
		Timestamp: time.Now(),
	}

	result, err := s.holmesAnalyzer.Analyze(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("holmes analysis failed: %w", err)
	}

	// 4. 缓存结果
	if result != nil {
		cacheData, _ := json.Marshal(result)
		_ = s.redisClient.Set(ctx, cacheKey, cacheData, holmesCacheTTL)
	}

	return result, nil
}

// GetFriendsHolmesAnalysis 获取好友的福尔摩斯分析列表
func (s *AgentService) GetFriendsHolmesAnalysis(ctx context.Context, userID string) (*model.HolmesFriendListResponse, error) {
	// 1. 获取好友列表
	friends, err := s.friendshipRepo.GetFriendsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get friends: %w", err)
	}

	// 2. 获取所有好友的用户信息
	friendIDs := make([]string, len(friends))
	for i, f := range friends {
		friendIDs[i] = f.FriendID
	}

	users, err := s.userRepo.GetByIDs(ctx, friendIDs)
	if err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}

	userMap := make(map[string]*model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 3. 对每个好友获取福尔摩斯分析
	now := time.Now()
	responses := make([]model.HolmesAPIResponse, 0, len(friends))

	for _, friend := range friends {
		user, ok := userMap[friend.FriendID]
		if !ok {
			continue
		}

		// 获取福尔摩斯分析
		analysis, err := s.GetHolmesAnalysis(ctx, friend.FriendID)
		if err != nil {
			// 分析失败，使用默认值
			responses = append(responses, model.HolmesAPIResponse{
				FriendID: friend.FriendID,
				Name:     user.Nickname,
				Avatar:   user.GetAvatar(),
				Result: struct {
					Available   bool   `json:"available"`
					Probability int    `json:"probability"`
					Confidence  string `json:"confidence"`
					Summary     string `json:"summary"`
					Emoji       string `json:"emoji,omitempty"`
					Color       string `json:"color"`
				}{
					Available:   false,
					Probability: -1,
					Confidence:  "low",
					Summary:     "数据不足",
					Emoji:       "🤔",
					Color:       model.GetProbabilityColor(-1),
				},
				UpdatedAt: now.UnixMilli(),
			})
			continue
		}

		if analysis == nil {
			// 无分析数据
			responses = append(responses, model.HolmesAPIResponse{
				FriendID: friend.FriendID,
				Name:     user.Nickname,
				Avatar:   user.GetAvatar(),
				Result: struct {
					Available   bool   `json:"available"`
					Probability int    `json:"probability"`
					Confidence  string `json:"confidence"`
					Summary     string `json:"summary"`
					Emoji       string `json:"emoji,omitempty"`
					Color       string `json:"color"`
				}{
					Available:   false,
					Probability: -1,
					Confidence:  "low",
					Summary:     "暂无数据",
					Emoji:       "🤔",
					Color:       model.GetProbabilityColor(-1),
				},
				UpdatedAt: now.UnixMilli(),
			})
			continue
		}

		// 构建响应
		resp := model.HolmesAPIResponse{
			FriendID:  friend.FriendID,
			Name:      user.Nickname,
			Avatar:    user.GetAvatar(),
			RawData:   analysis.RawData,
			Features:  analysis.Features,
			Reasoning: analysis.Reasoning,
			Result: struct {
				Available   bool   `json:"available"`
				Probability int    `json:"probability"`
				Confidence  string `json:"confidence"`
				Summary     string `json:"summary"`
				Emoji       string `json:"emoji,omitempty"`
				Color       string `json:"color"`
			}{
				Available:   analysis.Result.Available,
				Probability: analysis.Result.Probability,
				Confidence:  analysis.Result.Confidence,
				Summary:     analysis.Result.Summary,
				Emoji:       getEmojiFromAnalysis(analysis),
				Color:       model.GetProbabilityColor(analysis.Result.Probability),
			},
			UpdatedAt: analysis.GeneratedAt.UnixMilli(),
		}
		responses = append(responses, resp)
	}

	// 4. 按概率排序（降序）
	for i := 0; i < len(responses)-1; i++ {
		for j := i + 1; j < len(responses); j++ {
			if responses[j].Result.Probability > responses[i].Result.Probability {
				responses[i], responses[j] = responses[j], responses[i]
			}
		}
	}

	return &model.HolmesFriendListResponse{
		Friends:     responses,
		GeneratedAt: now.UnixMilli(),
	}, nil
}

// getEmojiFromAnalysis 从分析结果中获取 Emoji
func getEmojiFromAnalysis(analysis *model.HolmesResult) string {
	if analysis == nil {
		return "🤔"
	}

	// 尝试从 Reasoning 的 Conclusion 中推断
	conclusion := ""
	if analysis.Reasoning != nil {
		conclusion = analysis.Reasoning.Conclusion
	}

	// 根据关键词匹配 Emoji
	emojiMap := map[string]string{
		"咖啡":  "☕",
		"休闲":  "🛋️",
		"工作":  "💼",
		"开会":  "📊",
		"睡觉":  "😴",
		"运动":  "🏃",
		"聊天":  "💬",
		"游戏":  "🎮",
		"追剧":  "📺",
		"听音乐": "🎧",
		"外出":  "🚶",
		"逛街":  "🛍️",
		"通勤":  "🚇",
		"吃饭":  "🍜",
		"聚会":  "🍻",
		"专注":  "🔕",
	}

	for keyword, emoji := range emojiMap {
		if strings.Contains(conclusion, keyword) || strings.Contains(analysis.Result.Summary, keyword) {
			return emoji
		}
	}

	// 根据概率返回默认 Emoji
	if analysis.Result.Probability >= 70 {
		return "🟢"
	} else if analysis.Result.Probability >= 40 {
		return "🟡"
	} else if analysis.Result.Probability >= 0 {
		return "🔴"
	}
	return "🤔"
}

// ========== 状态选项生成 ==========

// GenerateStatusOptionsStream 流式生成状态选项
func (s *AgentService) GenerateStatusOptionsStream(ctx context.Context, userID string, req *model.ExtendedStatusReportRequest, recentMemory []*model.UserStatusMemory, callback func(event interface{})) (*model.StatusOptionsResult, error) {
	// 1. 保存状态到 Redis
	status := model.UserRealtimeStatus{
		UserID:    userID,
		UpdatedAt: time.Now(),
	}
	if req.Screen != nil {
		status.Screen = *req.Screen
	}
	if req.Location != nil {
		status.Location = *req.Location
	}

	data, err := json.Marshal(status)
	if err != nil {
		return nil, fmt.Errorf("marshal status: %w", err)
	}

	key := fmt.Sprintf(keyUserStatus, userID)
	if err := s.redisClient.Set(ctx, key, data, statusTTL); err != nil {
		return nil, fmt.Errorf("save status to redis: %w", err)
	}

	// 2. 获取用户角色画像
	var profileType string
	if s.userProfileService != nil {
		profileType, _ = s.userProfileService.GetSimpleProfileType(userID)
		if profileType != "" {
			fmt.Printf("[状态选项] 用户角色: %s (%s)\n", profileType, model.GetProfileTypeName(profileType))
		}
	}

	// 3. 流式生成状态选项
	if s.holmesAnalyzer == nil {
		return nil, fmt.Errorf("holmes analyzer not available")
	}

	input := &llm.HolmesInput{
		Status:      req,
		Timestamp:   time.Now(),
		ProfileType: profileType,
	}

	// 使用流式生成状态选项（传入 profileType）
	result, err := s.holmesAnalyzer.GenerateStatusOptionsStream(ctx, input, recentMemory, profileType, func(event *llm.HolmesStreamEvent) {
		callback(event)
	})
	if err != nil {
		return nil, fmt.Errorf("generate status options failed: %w", err)
	}

	return result, nil
}

// ========== 引导流程状态选项生成 ==========

// OnboardingStatusOptionsRequest 引导状态选项请求
type OnboardingStatusOptionsRequest struct {
	ProfileType string `json:"profile_type" binding:"required"`
}

// GetOnboardingStatusOptions 获取引导流程的状态选项（使用 LLM 推理）
// city: 用户当前所在城市（可选，用于增强推理）
func (s *AgentService) GetOnboardingStatusOptions(ctx context.Context, profileType string, city string) (*model.StatusOptionsResult, error) {
	if s.llmClient == nil {
		// LLM 不可用，返回默认选项
		return s.getDefaultOnboardingOptions(profileType), nil
	}

	now := time.Now()
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	weekday := weekdays[now.Weekday()]
	hour := now.Hour()
	isWeekend := now.Weekday() == time.Saturday || now.Weekday() == time.Sunday

	// 获取 ProfileType 中文名
	profileName := model.GetProfileTypeName(profileType)

	// 构建城市信息（如果有）
	cityInfo := ""
	if city != "" {
		cityInfo = fmt.Sprintf("\n当前城市：%s（可以结合城市特点推测状态）", city)
	}

	prompt := fmt.Sprintf(`你是一个生活状态推理助手。根据用户的角色、当前时间和位置，推测他最可能在做的 4 件事。

用户角色：%s
当前时间：%s %d:00
是否周末：%v%s

请返回 4 个最可能的状态，JSON 格式：
{"options": [{"emoji": "xxx", "status": "xxx"}, ...]}

要求：
1. 从社交角度考虑：朋友是否想知道你在什么场所做什么
2. 重要场景加场所（如：在家、在公司、在学校、在外面），emoji 可以用 1-2 个表达场所+行为
3. 普通行为可以不加场所（如：睡觉、休息）
4. emoji 可以用1-2个表达场所+行为，如 "🏠🍽️" 表示在家吃饭
5. status 用 2-6 个字描述，可以带场所如 "在家里吃饭"
6. 4 个选项要有差异性，覆盖不同可能性
7. 如果有城市信息，可以结合当地特色（如上海的咖啡文化、成都的茶馆等）
8. 只返回 JSON，不要有其他文字

示例输出（上班族，周一 10:00）：
{"options": [{"emoji": "🏢💼", "status": "在公司工作"}, {"emoji": "☕", "status": "喝咖啡"}, {"emoji": "🏢📊", "status": "公司开会"}, {"emoji": "📱", "status": "摸鱼中"}]}

示例输出（上班族，周末 12:00，上海）：
{"options": [{"emoji": "🏠🛋️", "status": "在家躺着"}, {"emoji": "☕", "status": "喝咖啡"}, {"emoji": "🛍️", "status": "逛商场"}, {"emoji": "🍜", "status": "吃brunch"}]}`, profileName, weekday, hour, isWeekend, cityInfo)

	response, err := s.llmClient.Chat(ctx, prompt)
	if err != nil {
		fmt.Printf("[引导选项] LLM 调用失败: %v, 使用默认选项\n", err)
		return s.getDefaultOnboardingOptions(profileType), nil
	}

	// 解析 JSON 响应
	var result model.StatusOptionsResult
	// 清理响应中可能的 Markdown 代码块标记
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		fmt.Printf("[引导选项] JSON 解析失败: %v, response: %s\n", err, response)
		return s.getDefaultOnboardingOptions(profileType), nil
	}

	// 验证结果
	if len(result.Options) < 4 {
		fmt.Printf("[引导选项] 选项数量不足: %d\n", len(result.Options))
		return s.getDefaultOnboardingOptions(profileType), nil
	}

	return &result, nil
}

// getDefaultOnboardingOptions 获取默认的引导选项（LLM 不可用时）
func (s *AgentService) getDefaultOnboardingOptions(profileType string) *model.StatusOptionsResult {
	now := time.Now()
	hour := now.Hour()
	isWeekend := now.Weekday() == time.Saturday || now.Weekday() == time.Sunday

	// 根据 profile_type 和时间返回默认选项
	switch profileType {
	case "office_worker":
		if isWeekend {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "🛋️", Status: "在家休息"},
					{Emoji: "📱", Status: "刷手机"},
					{Emoji: "🛍️", Status: "出门逛街"},
					{Emoji: "🍜", Status: "约饭中"},
				},
			}
		}
		if hour >= 9 && hour < 18 {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "💼", Status: "在工作"},
					{Emoji: "☕", Status: "喝咖啡"},
					{Emoji: "📊", Status: "开会中"},
					{Emoji: "📱", Status: "摸鱼中"},
				},
			}
		} else if hour >= 18 && hour < 22 {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "🚇", Status: "在通勤"},
					{Emoji: "🛋️", Status: "在家休息"},
					{Emoji: "📺", Status: "在追剧"},
					{Emoji: "🍜", Status: "吃饭中"},
				},
			}
		} else {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "😴", Status: "准备睡觉"},
					{Emoji: "📺", Status: "在追剧"},
					{Emoji: "📱", Status: "刷手机"},
					{Emoji: "🎮", Status: "玩游戏"},
				},
			}
		}

	case "student":
		if isWeekend {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "📚", Status: "在自习"},
					{Emoji: "🎮", Status: "玩游戏"},
					{Emoji: "😴", Status: "在睡觉"},
					{Emoji: "🛍️", Status: "出门玩"},
				},
			}
		}
		if hour >= 8 && hour < 17 {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "📖", Status: "在上课"},
					{Emoji: "📚", Status: "在自习"},
					{Emoji: "☕", Status: "课间休息"},
					{Emoji: "📱", Status: "刷手机"},
				},
			}
		} else {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "📚", Status: "在自习"},
					{Emoji: "🎮", Status: "玩游戏"},
					{Emoji: "📺", Status: "在追剧"},
					{Emoji: "🍜", Status: "吃饭中"},
				},
			}
		}

	case "freelancer":
		return &model.StatusOptionsResult{
			Options: []model.StatusOption{
				{Emoji: "💻", Status: "在工作"},
				{Emoji: "☕", Status: "喝咖啡"},
				{Emoji: "🛋️", Status: "在家休息"},
				{Emoji: "📱", Status: "刷手机"},
			},
		}

	case "parent":
		if hour >= 7 && hour < 9 {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "🚗", Status: "送孩子"},
					{Emoji: "🍳", Status: "做早餐"},
					{Emoji: "🏃", Status: "在运动"},
					{Emoji: "📱", Status: "刷手机"},
				},
			}
		} else if hour >= 9 && hour < 15 {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "🏠", Status: "做家务"},
					{Emoji: "🛒", Status: "买菜中"},
					{Emoji: "☕", Status: "喝茶休息"},
					{Emoji: "📱", Status: "刷手机"},
				},
			}
		} else {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "🚗", Status: "接孩子"},
					{Emoji: "📖", Status: "陪作业"},
					{Emoji: "🍳", Status: "做晚饭"},
					{Emoji: "📺", Status: "看电视"},
				},
			}
		}

	case "retired":
		if hour >= 6 && hour < 9 {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "🌅", Status: "晨练中"},
					{Emoji: "🧓", Status: "遛弯儿"},
					{Emoji: "📰", Status: "看新闻"},
					{Emoji: "🍵", Status: "喝早茶"},
				},
			}
		} else if hour >= 12 && hour < 15 {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "😴", Status: "午休中"},
					{Emoji: "📺", Status: "看电视"},
					{Emoji: "🀄", Status: "打麻将"},
					{Emoji: "📱", Status: "刷手机"},
				},
			}
		} else {
			return &model.StatusOptionsResult{
				Options: []model.StatusOption{
					{Emoji: "🧓", Status: "遛弯儿"},
					{Emoji: "🀄", Status: "打麻将"},
					{Emoji: "📺", Status: "看电视"},
					{Emoji: "🌳", Status: "逛公园"},
				},
			}
		}

	default:
		return &model.StatusOptionsResult{
			Options: []model.StatusOption{
				{Emoji: "🏠", Status: "在家中"},
				{Emoji: "💼", Status: "在工作"},
				{Emoji: "📱", Status: "刷手机"},
				{Emoji: "🚶", Status: "出门中"},
			},
		}
	}
}

// ========== Holmes 2.0 流式分析 ==========

// ReportExtendedStatus2Stream Holmes 2.0 流式上报状态
func (s *AgentService) ReportExtendedStatus2Stream(ctx context.Context, userID string, req *model.ExtendedStatusReportRequest, callback func(event interface{})) (*model.Holmes2Result, error) {
	// 1. 保存原始状态到 Redis
	status := model.UserRealtimeStatus{
		UserID:    userID,
		UpdatedAt: time.Now(),
	}
	if req.Screen != nil {
		status.Screen = *req.Screen
	}
	if req.Location != nil {
		status.Location = *req.Location
	}

	data, err := json.Marshal(status)
	if err != nil {
		return nil, fmt.Errorf("marshal status: %w", err)
	}

	key := fmt.Sprintf(keyUserStatus, userID)
	if err := s.redisClient.Set(ctx, key, data, statusTTL); err != nil {
		return nil, fmt.Errorf("save status to redis: %w", err)
	}

	// 2. 保存扩展状态到 Redis
	extData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal extended status: %w", err)
	}

	extKey := fmt.Sprintf("agent:extended:%s", userID)
	if err := s.redisClient.Set(ctx, extKey, extData, statusTTL); err != nil {
		return nil, fmt.Errorf("save extended status to redis: %w", err)
	}

	// 3. 流式执行 Holmes 2.0 分析
	if s.holmesAnalyzer == nil {
		return nil, fmt.Errorf("holmes analyzer not available")
	}

	input := &llm.HolmesInput{
		Status:    req,
		Timestamp: time.Now(),
	}

	// 使用 Holmes 2.0 流式分析
	result, err := s.holmesAnalyzer.Analyze2Stream(ctx, input, func(event *llm.HolmesStreamEvent) {
		callback(event)
	})
	if err != nil {
		return nil, fmt.Errorf("holmes 2.0 stream analysis failed: %w", err)
	}

	// 4. 缓存分析结果
	if result != nil {
		cacheData, _ := json.Marshal(result)
		cacheKey := fmt.Sprintf(keyHolmesCache, userID)
		_ = s.redisClient.Set(ctx, cacheKey, cacheData, holmesCacheTTL)
	}

	return result, nil
}

// ========== 当下状态推理 ==========

const (
	keyStatusFeedback = "agent:feedback:%s" // 状态反馈记忆
	feedbackTTL       = 7 * 24 * time.Hour  // 反馈记忆保留7天
)

// InferCurrentStatus 推理用户当下状态
func (s *AgentService) InferCurrentStatus(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest) (*model.CurrentStatusInference, error) {
	// 1. 如果没有传感器数据，尝试从缓存获取
	if sensorData == nil || (sensorData.Screen == nil && sensorData.Location == nil) {
		cached := s.getCachedSensorData(ctx, userID)
		if cached != nil {
			sensorData = cached
		}
	}

	// 2. 获取用户画像
	var userProfile *model.UserProfileData
	if s.userProfileService != nil {
		userProfile, _ = s.userProfileService.GetProfile(userID)
	}

	// 3. 获取状态反馈记忆
	feedbackMemory := s.getStatusFeedbackMemory(ctx, userID)

	// 4. 构建推理输入
	now := time.Now()
	input := &llm.StatusInferInput{
		SensorData:   sensorData,
		UserProfile:  userProfile,
		StatusMemory: feedbackMemory,
		Timestamp:    now,
		Weekday:      getWeekdayNameCN(now.Weekday()),
		Hour:         now.Hour(),
	}

	// 5. 创建推理器并执行推理
	inferrer := llm.NewStatusInferrer(s.llmClient)
	result, err := inferrer.Infer(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("infer status: %w", err)
	}

	return result, nil
}

// SaveStatusFeedback 保存状态反馈（用户修正记忆）并更新首页状态
func (s *AgentService) SaveStatusFeedback(ctx context.Context, userID string, req *model.StatusFeedbackRequest) error {
	// 获取当前传感器数据作为上下文
	sensorData := s.getCachedSensorData(ctx, userID)
	contextJSON := "{}"
	if sensorData != nil {
		if data, err := json.Marshal(sensorData); err == nil {
			contextJSON = string(data)
		}
	}

	// 创建记忆条目
	entry := &model.StatusMemoryEntry{
		UserID:               userID,
		OriginalEmoji:        req.OriginalEmoji,
		OriginalActivity:     req.OriginalActivity,
		CorrectedEmoji:       req.CorrectedEmoji,
		CorrectedActivity:    req.CorrectedActivity,
		CorrectedPlace:       req.CorrectedPlace,
		CorrectedIsAvailable: req.CorrectedIsAvailable != nil && *req.CorrectedIsAvailable,
		DeviceContext:        contextJSON,
		CreatedAt:            time.Now().Format(time.RFC3339),
	}

	// 获取现有记忆列表
	memories := s.getStatusFeedbackMemory(ctx, userID)

	// 添加新记忆（最多保留20条）
	memories = append([]*model.StatusMemoryEntry{entry}, memories...)
	if len(memories) > 20 {
		memories = memories[:20]
	}

	// 保存到 Redis
	data, err := json.Marshal(memories)
	if err != nil {
		return fmt.Errorf("marshal feedback memory: %w", err)
	}

	key := fmt.Sprintf(keyStatusFeedback, userID)
	if err := s.redisClient.Set(ctx, key, data, feedbackTTL); err != nil {
		return fmt.Errorf("save feedback to redis: %w", err)
	}

	// 同时更新首页状态（user_analysis_cache 表）
	isAvailable := req.CorrectedIsAvailable != nil && *req.CorrectedIsAvailable
	availabilityStatus := "忙碌"
	if isAvailable {
		availabilityStatus = "有空"
	}

	// 组合 emoji 和活动描述
	// 如果有场所，把场所 emoji + 活动 emoji，场所 + 活动
	finalEmoji := req.CorrectedEmoji
	finalLabel := req.CorrectedActivity
	if req.CorrectedPlace != "" {
		// 获取场所对应的 emoji
		placeEmoji := getPlaceEmoji(req.CorrectedPlace)
		if placeEmoji != "" && !strings.Contains(finalEmoji, placeEmoji) {
			finalEmoji = placeEmoji + finalEmoji
		}
		// 组合场所和活动描述
		finalLabel = req.CorrectedPlace + req.CorrectedActivity
	}

	// GIF 信息（客户端已通过 upload-gif 接口将 GIF 上传到 COS）
	gifURL := req.GifURL
	giphyQuery := req.GiphyQuery

	analysisResult := &model.AnalysisResult{
		Availability: model.AvailabilityAnalysis{
			Status:      availabilityStatus,
			Probability: 100,
			Reason:      "用户手动设置",
			Confidence:  "high",
		},
		LifeStatus: model.LifeStatus{
			Emoji:       finalEmoji,
			Label:       finalLabel,
			Description: req.CorrectedPlace,
			GifURL:      gifURL,
			GiphyQuery:  giphyQuery,
			UseGif:      req.UseGif,
		},
		UpdatedAt: time.Now(),
		IsAIGuess: false, // 用户确认的状态，不是 AI 猜测
	}

	if s.memoryRepo != nil {
		if err := s.memoryRepo.SaveAnalysisCache(ctx, userID, analysisResult); err != nil {
			fmt.Printf("[SaveStatusFeedback] 更新首页状态失败 user=%s error=%v\n", userID, err)
			// 不返回错误，反馈已保存成功
		} else {
			fmt.Printf("[SaveStatusFeedback] 首页状态已更新 user=%s emoji=%s status=%s is_available=%v\n",
				userID, finalEmoji, finalLabel, isAvailable)
		}
	}

	// 同时更新状态表（status_schedules）的当前时间段
	if s.scheduleRepo != nil {
		s.updateCurrentScheduleItem(ctx, userID, finalEmoji, finalLabel, gifURL, giphyQuery)
	}

	// 更新 prev inference 缓存，标记为用户确认来源
	if s.redisClient != nil {
		prevInference := &model.CurrentStatusInference{
			Emoji:      finalEmoji,
			Activity:   finalLabel,
			IsAvailable: isAvailable,
			Confidence: "high",
			InferredAt: time.Now().UnixMilli(),
			Source:     model.InferenceSourceUserConfirmed,
		}
		prevKey := fmt.Sprintf("infer:prev:%s", userID)
		if prevData, err := json.Marshal(prevInference); err == nil {
			s.redisClient.Set(ctx, prevKey, prevData, 2*time.Hour)
		}
	}

	return nil
}

// updateCurrentScheduleItem 更新状态表的当前时间段（只更新今天的）
func (s *AgentService) updateCurrentScheduleItem(ctx context.Context, userID, emoji, status, gifURL, giphyQuery string) {
	// 获取上海时区
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	now := time.Now().In(loc)
	todayDate := now.Format("2006-01-02")
	currentTime := now.Format("15:04")

	fmt.Printf("[SaveStatusFeedback] 准备更新状态表 user=%s date=%s time=%s\n", userID, todayDate, currentTime)

	// 获取今天的时刻表（优先 active，其次最新的 completed）
	schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, userID, now)
	if (err != nil || schedule == nil) && s.scheduleRepo != nil {
		schedule, err = s.scheduleRepo.GetLatestByUserAndDate(ctx, userID, now)
	}

	// 计算新 item 的时间段：当前时间开始，持续1小时
	startTime := currentTime
	endHour := now.Hour() + 1
	endMin := now.Minute()
	if endHour >= 24 {
		endHour = 23
		endMin = 59
	}
	endTime := fmt.Sprintf("%02d:%02d", endHour, endMin)

	if err != nil || schedule == nil {
		// 今天没有时刻表，创建一个新的
		fmt.Printf("[SaveStatusFeedback] 今天(%s)没有状态表，创建新的 user=%s\n", todayDate, userID)

		newItem := model.ScheduleItem{
			StartTime:  startTime,
			EndTime:    endTime,
			Emoji:      emoji,
			Status:     status,
			Executed:   false,
			IsAIGuess:  false,
			GifURL:     gifURL,
			GiphyQuery: giphyQuery,
		}

		scheduleDate, _ := time.ParseInLocation("2006-01-02", todayDate, loc)
		newSchedule := &model.StatusSchedule{
			UserID:       userID,
			ScheduleDate: scheduleDate,
			Items:        []model.ScheduleItem{newItem},
			CurrentIndex: 0,
			Status:       "active",
			Visibility:   "all_friends",
		}

		if err := s.scheduleRepo.Create(ctx, newSchedule); err != nil {
			fmt.Printf("[SaveStatusFeedback] 创建状态表失败 user=%s date=%s error=%v\n", userID, todayDate, err)
		} else {
			fmt.Printf("[SaveStatusFeedback] 状态表已创建 user=%s date=%s item=[%s %s] time=%s-%s\n",
				userID, todayDate, emoji, status, startTime, endTime)
		}
		return
	}

	// 二次确认：检查 schedule 的日期是否是今天
	if schedule.ScheduleDate.Format("2006-01-02") != todayDate {
		fmt.Printf("[SaveStatusFeedback] 警告：状态表日期(%s)与今天(%s)不匹配，跳过更新 user=%s\n",
			schedule.ScheduleDate.Format("2006-01-02"), todayDate, userID)
		return
	}

	// 找到当前时间对应的 item 并更新
	itemUpdated := false
	for i := range schedule.Items {
		if isTimeInRange(currentTime, schedule.Items[i].StartTime, schedule.Items[i].EndTime) {
			oldEmoji := schedule.Items[i].Emoji
			oldStatus := schedule.Items[i].Status
			schedule.Items[i].Emoji = emoji
			schedule.Items[i].Status = status
			schedule.Items[i].IsAIGuess = false // 用户确认的状态
			schedule.Items[i].GifURL = gifURL
			schedule.Items[i].GiphyQuery = giphyQuery
			itemUpdated = true
			fmt.Printf("[SaveStatusFeedback] 更新状态表item user=%s date=%s index=%d time=%s-%s old=[%s %s] new=[%s %s]\n",
				userID, todayDate, i, schedule.Items[i].StartTime, schedule.Items[i].EndTime,
				oldEmoji, oldStatus, emoji, status)
			break
		}
	}

	if !itemUpdated {
		// 没有找到当前时间段，需要插入新的 item 并处理时间重叠
		newItem := model.ScheduleItem{
			StartTime:  startTime,
			EndTime:    endTime,
			Emoji:      emoji,
			Status:     status,
			Executed:   false,
			IsAIGuess:  false,
			GifURL:     gifURL,
			GiphyQuery: giphyQuery,
		}

		// 处理时间重叠：分割或调整现有条目
		schedule.Items = insertScheduleItem(schedule.Items, newItem)
		fmt.Printf("[SaveStatusFeedback] 插入新状态表item user=%s date=%s time=%s-%s status=[%s %s] total_items=%d\n",
			userID, todayDate, startTime, endTime, emoji, status, len(schedule.Items))
	}

	// 保存回数据库
	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		fmt.Printf("[SaveStatusFeedback] 更新状态表失败 user=%s date=%s error=%v\n", userID, todayDate, err)
	} else {
		fmt.Printf("[SaveStatusFeedback] 状态表已更新 user=%s date=%s\n", userID, todayDate)
	}
}

// isTimeInRange 检查时间是否在范围内
func isTimeInRange(current, start, end string) bool {
	// 处理跨午夜的情况（如 23:00-01:00）
	// 以及 24:00 格式
	if end == "24:00" {
		end = "23:59"
	}

	// 简单比较字符串（HH:MM 格式）
	if start <= end {
		return current >= start && current < end
	}
	// 跨午夜：23:00-01:00 表示 23:00~23:59 或 00:00~01:00
	return current >= start || current < end
}

// insertScheduleItem 插入新的时刻表条目，处理时间重叠
// 如果新条目与现有条目重叠，会分割或调整现有条目
func insertScheduleItem(items []model.ScheduleItem, newItem model.ScheduleItem) []model.ScheduleItem {
	var result []model.ScheduleItem

	for _, item := range items {
		// 检查是否有时间重叠
		// 情况1：新条目完全覆盖现有条目 → 删除现有条目
		// 情况2：新条目在现有条目中间 → 分割成两部分
		// 情况3：新条目覆盖现有条目的前半部分 → 调整开始时间
		// 情况4：新条目覆盖现有条目的后半部分 → 调整结束时间
		// 情况5：没有重叠 → 保持不变

		if !hasTimeOverlap(item.StartTime, item.EndTime, newItem.StartTime, newItem.EndTime) {
			// 没有重叠，保持不变
			result = append(result, item)
			continue
		}

		// 有重叠，处理各种情况
		if newItem.StartTime <= item.StartTime && newItem.EndTime >= item.EndTime {
			// 情况1：新条目完全覆盖现有条目 → 删除现有条目
			fmt.Printf("[insertScheduleItem] 删除被完全覆盖的条目: %s-%s %s\n",
				item.StartTime, item.EndTime, item.Status)
			continue
		}

		if newItem.StartTime > item.StartTime && newItem.EndTime < item.EndTime {
			// 情况2：新条目在现有条目中间 → 分割成两部分
			// 前半部分
			frontItem := item
			frontItem.EndTime = newItem.StartTime
			result = append(result, frontItem)
			// 后半部分
			backItem := item
			backItem.StartTime = newItem.EndTime
			result = append(result, backItem)
			fmt.Printf("[insertScheduleItem] 分割条目: %s-%s → %s-%s + %s-%s\n",
				item.StartTime, item.EndTime, frontItem.StartTime, frontItem.EndTime,
				backItem.StartTime, backItem.EndTime)
			continue
		}

		if newItem.StartTime <= item.StartTime && newItem.EndTime > item.StartTime && newItem.EndTime < item.EndTime {
			// 情况3：新条目覆盖现有条目的前半部分 → 调整开始时间
			adjustedItem := item
			adjustedItem.StartTime = newItem.EndTime
			result = append(result, adjustedItem)
			fmt.Printf("[insertScheduleItem] 调整开始时间: %s-%s → %s-%s\n",
				item.StartTime, item.EndTime, adjustedItem.StartTime, adjustedItem.EndTime)
			continue
		}

		if newItem.StartTime > item.StartTime && newItem.StartTime < item.EndTime && newItem.EndTime >= item.EndTime {
			// 情况4：新条目覆盖现有条目的后半部分 → 调整结束时间
			adjustedItem := item
			adjustedItem.EndTime = newItem.StartTime
			result = append(result, adjustedItem)
			fmt.Printf("[insertScheduleItem] 调整结束时间: %s-%s → %s-%s\n",
				item.StartTime, item.EndTime, adjustedItem.StartTime, adjustedItem.EndTime)
			continue
		}

		// 默认保留（不应该到这里）
		result = append(result, item)
	}

	// 添加新条目
	result = append(result, newItem)

	// 按开始时间排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartTime < result[j].StartTime
	})

	return result
}

// hasTimeOverlap 检查两个时间段是否有重叠
func hasTimeOverlap(start1, end1, start2, end2 string) bool {
	// 简单的字符串比较（HH:MM 格式）
	// 两个区间 [start1, end1) 和 [start2, end2) 重叠当且仅当：
	// start1 < end2 且 start2 < end1
	return start1 < end2 && start2 < end1
}

// getCachedSensorData 获取缓存的传感器数据
// 会根据缓存时间调整 AtPlaceSinceMinutes 等时间相关字段
func (s *AgentService) getCachedSensorData(ctx context.Context, userID string) *model.ExtendedStatusReportRequest {
	key := fmt.Sprintf(keyUserStatus, userID)
	data, err := s.redisClient.GetBytes(ctx, key)
	if err != nil || len(data) == 0 {
		fmt.Printf("[getCachedSensorData] 无缓存数据 user=%s err=%v len=%d\n", userID, err, len(data))
		return nil
	}

	var status model.UserRealtimeStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil
	}

	// 计算缓存时间差（分钟）
	minutesSinceUpdate := int(time.Since(status.UpdatedAt).Minutes())

	// 复制数据并调整时间相关字段
	result := &model.ExtendedStatusReportRequest{
		Screen:   &status.Screen,
		Location: &status.Location,
	}

	// 调整 Location.AtPlaceSinceMinutes
	// 如果缓存的数据显示用户在某位置待了30分钟，而缓存是2小时前的
	// 则实际应该是 30 + 120 = 150 分钟
	if result.Location != nil && minutesSinceUpdate > 0 {
		originalMinutes := result.Location.AtPlaceSinceMinutes
		result.Location.AtPlaceSinceMinutes = originalMinutes + minutesSinceUpdate
		fmt.Printf("[getCachedSensorData] 调整 AtPlaceSinceMinutes: %d + %d = %d 分钟\n",
			originalMinutes, minutesSinceUpdate, result.Location.AtPlaceSinceMinutes)
	}

	// 调整 Screen.LastActiveMinutesAgo
	if result.Screen != nil && minutesSinceUpdate > 0 {
		result.Screen.LastActiveMinutesAgo = result.Screen.LastActiveMinutesAgo + minutesSinceUpdate
	}

	// 调整 Screen.SessionDurationMinutes（如果还在活跃状态）
	if result.Screen != nil && result.Screen.IsActive && minutesSinceUpdate > 0 {
		result.Screen.SessionDurationMinutes = result.Screen.SessionDurationMinutes + minutesSinceUpdate
	}

	return result
}

// getStatusFeedbackMemory 获取状态反馈记忆
func (s *AgentService) getStatusFeedbackMemory(ctx context.Context, userID string) []*model.StatusMemoryEntry {
	key := fmt.Sprintf(keyStatusFeedback, userID)
	data, err := s.redisClient.GetBytes(ctx, key)
	if err != nil || len(data) == 0 {
		return nil
	}

	var memories []*model.StatusMemoryEntry
	if err := json.Unmarshal(data, &memories); err != nil {
		return nil
	}

	return memories
}

// getWeekdayNameCN 获取中文星期名
func getWeekdayNameCN(w time.Weekday) string {
	names := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return names[w]
}

// getPlaceEmoji 根据场所名称获取对应的 emoji
func getPlaceEmoji(place string) string {
	place = strings.ToLower(place)

	// 家
	if strings.Contains(place, "家") || strings.Contains(place, "home") {
		return "🏠"
	}
	// 公司/办公室
	if strings.Contains(place, "公司") || strings.Contains(place, "办公") || strings.Contains(place, "单位") || strings.Contains(place, "work") || strings.Contains(place, "office") {
		return "💼"
	}
	// 学校
	if strings.Contains(place, "学校") || strings.Contains(place, "教室") || strings.Contains(place, "图书馆") || strings.Contains(place, "school") {
		return "🎓"
	}
	// 咖啡厅
	if strings.Contains(place, "咖啡") || strings.Contains(place, "cafe") || strings.Contains(place, "coffee") {
		return "☕"
	}
	// 餐厅
	if strings.Contains(place, "餐厅") || strings.Contains(place, "饭店") || strings.Contains(place, "餐馆") || strings.Contains(place, "restaurant") {
		return "🍽️"
	}
	// 健身房
	if strings.Contains(place, "健身") || strings.Contains(place, "gym") {
		return "💪"
	}
	// 医院
	if strings.Contains(place, "医院") || strings.Contains(place, "诊所") || strings.Contains(place, "hospital") {
		return "🏥"
	}
	// 商场/购物
	if strings.Contains(place, "商场") || strings.Contains(place, "超市") || strings.Contains(place, "购物") || strings.Contains(place, "mall") || strings.Contains(place, "shop") {
		return "🛒"
	}
	// 交通
	if strings.Contains(place, "地铁") || strings.Contains(place, "公交") || strings.Contains(place, "路上") || strings.Contains(place, "车") || strings.Contains(place, "transit") {
		return "🚇"
	}
	// 户外
	if strings.Contains(place, "公园") || strings.Contains(place, "户外") || strings.Contains(place, "outdoor") || strings.Contains(place, "park") {
		return "🌳"
	}
	// 酒吧
	if strings.Contains(place, "酒吧") || strings.Contains(place, "bar") || strings.Contains(place, "ktv") {
		return "🍺"
	}
	// 电影院
	if strings.Contains(place, "电影") || strings.Contains(place, "影院") || strings.Contains(place, "cinema") || strings.Contains(place, "movie") {
		return "🎬"
	}

	return ""
}

// PersistCityFromExtendedReport 从扩展上报中持久化城市信息（通过位置反查）
func (s *AgentService) PersistCityFromExtendedReport(_ context.Context, _ string, _ *model.ExtendedStatusReportRequest) {
	// TODO: 从经纬度反查城市并持久化
}

// MergeExistingGifInfo 合并已有的 GIF 信息到反馈请求
func (s *AgentService) MergeExistingGifInfo(ctx context.Context, userID string, req *model.StatusFeedbackRequest) {
	// 如果反馈中没有 GIF 信息，尝试从缓存中获取
	// 这里是 no-op 占位，确保编译通过
}
