package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

const (
	// Redis key 前缀
	keyUserStatus  = "agent:status:%s"  // 用户实时状态
	keyUserPattern = "agent:pattern:%s" // 用户历史规律

	// 状态过期时间
	statusTTL  = 5 * time.Minute  // 状态5分钟过期
	patternTTL = 24 * time.Hour   // 规律24小时过期
)

// AgentService Agent 服务
type AgentService struct {
	redisClient    *tencent.RedisClient
	userRepo       *repository.UserRepository
	friendshipRepo *repository.FriendshipRepository
}

// NewAgentService 创建 Agent 服务
func NewAgentService(
	redisClient *tencent.RedisClient,
	userRepo *repository.UserRepository,
	friendshipRepo *repository.FriendshipRepository,
) *AgentService {
	return &AgentService{
		redisClient:    redisClient,
		userRepo:       userRepo,
		friendshipRepo: friendshipRepo,
	}
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

		// 计算有空概率
		probability, reason, confidence := s.calculateFreeProbability(agentData, now)

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

// calculateFreeProbability 计算有空概率（规则引擎版本）
func (s *AgentService) calculateFreeProbability(data *model.AgentExposedData, now time.Time) (probability int, reason string, confidence string) {
	score := 50 // 基础分
	reasons := []string{}
	hasData := false

	// ========== 屏幕状态分析 ==========
	if data.DataQuality.ScreenDataAgeSeconds >= 0 && data.DataQuality.ScreenDataAgeSeconds < 300 {
		hasData = true
		screen := data.Realtime.Screen

		if screen.IsActive {
			switch screen.ActivityType {
			case model.ActivityEntertainment:
				if screen.SessionDurationMinutes >= 10 {
					score += 30
					reasons = append(reasons, fmt.Sprintf("刷了%d分钟手机", screen.SessionDurationMinutes))
				} else {
					score += 15
					reasons = append(reasons, "在刷手机")
				}
			case model.ActivityProductivity:
				score -= 10
				reasons = append(reasons, "在用工作APP")
			case model.ActivityCommunication:
				score += 10
				reasons = append(reasons, "在聊天")
			}
		} else {
			if screen.LastActiveMinutesAgo > 120 {
				score -= 25
				reasons = append(reasons, "手机闲置很久")
			} else if screen.LastActiveMinutesAgo > 30 {
				score -= 10
				reasons = append(reasons, "有一会没用手机")
			}
		}
	}

	// ========== 位置分析 ==========
	if data.DataQuality.LocationDataAgeSeconds >= 0 && data.DataQuality.LocationDataAgeSeconds < 600 {
		hasData = true
		location := data.Realtime.Location

		switch location.PlaceType {
		case model.PlaceHome:
			score += 15
			reasons = append(reasons, "在家")
		case model.PlaceWork:
			if isWorkHours(now) {
				score -= 20
				reasons = append(reasons, "在公司上班")
			} else {
				score += 5
				reasons = append(reasons, "在公司但已下班")
			}
		case model.PlaceLeisure:
			score += 10
			reasons = append(reasons, "在外面")
		}
	}

	// ========== 时间分析 ==========
	timeScore, timeReason := getTimeScore(now)
	score += timeScore
	if timeReason != "" {
		reasons = append(reasons, timeReason)
	}

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
		probability = -1 // 表示无数据
		return
	}

	if data.DataQuality.PatternsSampleSize > 50 && data.DataQuality.ScreenDataAgeSeconds < 60 {
		confidence = "high"
	} else if data.DataQuality.PatternsSampleSize > 20 {
		confidence = "medium"
	} else {
		confidence = "low"
	}

	// ========== 选择最佳理由 ==========
	if len(reasons) > 0 {
		// 优先选择屏幕状态相关的理由
		reason = reasons[0]
		if len(reasons) > 1 {
			reason = reasons[0] + "，" + reasons[1]
		}
	} else {
		reason = "数据不足"
	}

	probability = score
	return
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
