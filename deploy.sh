#!/bin/bash
cd /opt/youkong

# 1. 创建 agent.go
cat > internal/model/agent.go << 'AGENTMODEL'
package model

import "time"

type ActivityType string

const (
	ActivityEntertainment ActivityType = "entertainment"
	ActivityProductivity  ActivityType = "productivity"
	ActivityCommunication ActivityType = "communication"
	ActivityIdle          ActivityType = "idle"
)

type PlaceType string

const (
	PlaceHome    PlaceType = "home"
	PlaceWork    PlaceType = "work"
	PlaceLeisure PlaceType = "leisure"
	PlaceTransit PlaceType = "transit"
	PlaceUnknown PlaceType = "unknown"
)

type ScreenData struct {
	IsActive               bool         `json:"is_active"`
	ActivityType           ActivityType `json:"activity_type"`
	SessionDurationMinutes int          `json:"session_duration_minutes"`
	LastActiveMinutesAgo   int          `json:"last_active_minutes_ago"`
}

type LocationData struct {
	PlaceType           PlaceType `json:"place_type"`
	AtPlaceSinceMinutes int       `json:"at_place_since_minutes"`
}

type UserPatterns struct {
	CurrentHourFreeRate      int `json:"current_hour_free_rate"`
	CurrentWeekdayFreeRate   int `json:"current_weekday_free_rate"`
	AtHomeFreeRate           int `json:"at_home_free_rate"`
	AtWorkAfterHoursFreeRate int `json:"at_work_after_hours_free_rate"`
	AvgResponseTimeMinutes   int `json:"avg_response_time_minutes"`
	ResponseRate             int `json:"response_rate"`
}

type DataQuality struct {
	ScreenDataAgeSeconds   int `json:"screen_data_age_seconds"`
	LocationDataAgeSeconds int `json:"location_data_age_seconds"`
	PatternsSampleSize     int `json:"patterns_sample_size"`
}

type AgentExposedData struct {
	Realtime struct {
		Screen   ScreenData   `json:"screen"`
		Location LocationData `json:"location"`
	} `json:"realtime"`
	Patterns    UserPatterns `json:"patterns"`
	DataQuality DataQuality  `json:"data_quality"`
}

type UserRealtimeStatus struct {
	UserID    string       `json:"user_id"`
	Screen    ScreenData   `json:"screen"`
	Location  LocationData `json:"location"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type StatusReportRequest struct {
	Screen   *ScreenData   `json:"screen"`
	Location *LocationData `json:"location"`
}

type FriendRecommendation struct {
	FriendID    string `json:"friend_id"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar,omitempty"`
	Probability int    `json:"probability"`
	Confidence  string `json:"confidence"`
	Reason      string `json:"reason"`
	Color       string `json:"color"`
	UpdatedAt   int64  `json:"updated_at"`
}

type FreeProbabilityResponse struct {
	Friends     []FriendRecommendation `json:"friends"`
	GeneratedAt int64                  `json:"generated_at"`
}

var (
	ColorVeryLikely = "#22C55E"
	ColorLikely     = "#86EFAC"
	ColorMaybe      = "#FACC15"
	ColorUnlikely   = "#FB923C"
	ColorBusy       = "#EF4444"
	ColorUnknown    = "#9CA3AF"
)

func GetProbabilityColor(probability int) string {
	if probability < 0 {
		return ColorUnknown
	}
	if probability >= 80 {
		return ColorVeryLikely
	}
	if probability >= 60 {
		return ColorLikely
	}
	if probability >= 40 {
		return ColorMaybe
	}
	if probability >= 20 {
		return ColorUnlikely
	}
	return ColorBusy
}
AGENTMODEL

echo "✓ agent.go 已创建"

# 2. 创建 agent_service.go
cat > internal/service/agent_service.go << 'AGENTSERVICE'
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
	keyUserStatus  = "agent:status:%s"
	keyUserPattern = "agent:pattern:%s"
	statusTTL      = 5 * time.Minute
	patternTTL     = 24 * time.Hour
)

type AgentService struct {
	redisClient    *tencent.RedisClient
	userRepo       *repository.UserRepository
	friendshipRepo *repository.FriendshipRepository
}

func NewAgentService(redisClient *tencent.RedisClient, userRepo *repository.UserRepository, friendshipRepo *repository.FriendshipRepository) *AgentService {
	return &AgentService{redisClient: redisClient, userRepo: userRepo, friendshipRepo: friendshipRepo}
}

func (s *AgentService) ReportStatus(ctx context.Context, userID string, req *model.StatusReportRequest) error {
	status := model.UserRealtimeStatus{UserID: userID, UpdatedAt: time.Now()}
	if req.Screen != nil {
		status.Screen = *req.Screen
	}
	if req.Location != nil {
		status.Location = *req.Location
	}
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}
	key := fmt.Sprintf(keyUserStatus, userID)
	return s.redisClient.Set(ctx, key, data, statusTTL)
}

func (s *AgentService) GetUserStatus(ctx context.Context, userID string) (*model.UserRealtimeStatus, error) {
	key := fmt.Sprintf(keyUserStatus, userID)
	data, err := s.redisClient.GetBytes(ctx, key)
	if tencent.IsNil(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var status model.UserRealtimeStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (s *AgentService) GetUserPatterns(ctx context.Context, userID string) (*model.UserPatterns, error) {
	key := fmt.Sprintf(keyUserPattern, userID)
	data, err := s.redisClient.GetBytes(ctx, key)
	if tencent.IsNil(err) {
		return &model.UserPatterns{CurrentHourFreeRate: 50, CurrentWeekdayFreeRate: 50, AtHomeFreeRate: 70, AtWorkAfterHoursFreeRate: 40, AvgResponseTimeMinutes: 10, ResponseRate: 50}, nil
	}
	if err != nil {
		return nil, err
	}
	var patterns model.UserPatterns
	json.Unmarshal(data, &patterns)
	return &patterns, nil
}

func (s *AgentService) GetAgentExposedData(ctx context.Context, userID string) (*model.AgentExposedData, error) {
	status, _ := s.GetUserStatus(ctx, userID)
	patterns, _ := s.GetUserPatterns(ctx, userID)
	data := &model.AgentExposedData{Patterns: *patterns}
	if status != nil {
		data.Realtime.Screen = status.Screen
		data.Realtime.Location = status.Location
		data.DataQuality.ScreenDataAgeSeconds = int(time.Since(status.UpdatedAt).Seconds())
		data.DataQuality.LocationDataAgeSeconds = int(time.Since(status.UpdatedAt).Seconds())
	} else {
		data.DataQuality.ScreenDataAgeSeconds = -1
		data.DataQuality.LocationDataAgeSeconds = -1
	}
	data.DataQuality.PatternsSampleSize = 100
	return data, nil
}

func (s *AgentService) GetFriendsFreeProbability(ctx context.Context, userID string) (*model.FreeProbabilityResponse, error) {
	friends, err := s.friendshipRepo.GetFriendsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	friendIDs := make([]string, len(friends))
	for i, f := range friends {
		friendIDs[i] = f.FriendID
	}
	users, _ := s.userRepo.GetByIDs(ctx, friendIDs)
	userMap := make(map[string]*model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}
	recommendations := make([]model.FriendRecommendation, 0)
	now := time.Now()
	for _, friend := range friends {
		user, ok := userMap[friend.FriendID]
		if !ok {
			continue
		}
		agentData, _ := s.GetAgentExposedData(ctx, friend.FriendID)
		probability, reason, confidence := s.calculateFreeProbability(agentData, now)
		rec := model.FriendRecommendation{FriendID: friend.FriendID, Name: user.Nickname, Avatar: user.Avatar, Probability: probability, Confidence: confidence, Reason: reason, Color: model.GetProbabilityColor(probability), UpdatedAt: now.UnixMilli()}
		recommendations = append(recommendations, rec)
	}
	for i := 0; i < len(recommendations)-1; i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[j].Probability > recommendations[i].Probability {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}
	return &model.FreeProbabilityResponse{Friends: recommendations, GeneratedAt: now.UnixMilli()}, nil
}

func (s *AgentService) calculateFreeProbability(data *model.AgentExposedData, now time.Time) (int, string, string) {
	score := 50
	reasons := []string{}
	hasData := false
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
	if data.DataQuality.LocationDataAgeSeconds >= 0 && data.DataQuality.LocationDataAgeSeconds < 600 {
		hasData = true
		location := data.Realtime.Location
		switch location.PlaceType {
		case model.PlaceHome:
			score += 15
			reasons = append(reasons, "在家")
		case model.PlaceWork:
			weekday := now.Weekday()
			hour := now.Hour()
			isWorkTime := weekday != time.Saturday && weekday != time.Sunday && hour >= 9 && hour < 18
			if isWorkTime {
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
	weekday := now.Weekday()
	hour := now.Hour()
	isWeekend := weekday == time.Saturday || weekday == time.Sunday
	if isWeekend && hour >= 10 && hour <= 22 {
		score += 15
		reasons = append(reasons, "周末")
	} else if !isWeekend {
		if hour >= 18 && hour < 22 {
			score += 20
			if weekday == time.Friday {
				reasons = append(reasons, "周五晚上")
			} else {
				reasons = append(reasons, "下班了")
			}
		} else if hour >= 12 && hour < 14 {
			score += 5
			reasons = append(reasons, "午休时间")
		} else if hour >= 9 && hour < 18 {
			score -= 15
		}
	}
	if data.Patterns.CurrentHourFreeRate > 70 {
		score += 10
	} else if data.Patterns.CurrentHourFreeRate < 30 {
		score -= 10
	}
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	if !hasData {
		return -1, "数据不足", "low"
	}
	confidence := "low"
	if data.DataQuality.PatternsSampleSize > 50 && data.DataQuality.ScreenDataAgeSeconds < 60 {
		confidence = "high"
	} else if data.DataQuality.PatternsSampleSize > 20 {
		confidence = "medium"
	}
	reason := "数据不足"
	if len(reasons) > 0 {
		reason = reasons[0]
		if len(reasons) > 1 {
			reason = reasons[0] + "，" + reasons[1]
		}
	}
	return score, reason, confidence
}
AGENTSERVICE

echo "✓ agent_service.go 已创建"

# 3. 创建 agent handler
cat > internal/handler/agent.go << 'AGENTHANDLER'
package handler

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

type AgentHandler struct {
	agentService *service.AgentService
}

func NewAgentHandler(agentService *service.AgentService) *AgentHandler {
	return &AgentHandler{agentService: agentService}
}

func (h *AgentHandler) ReportStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}
	var req model.StatusReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}
	if err := h.agentService.ReportStatus(c.Request.Context(), userID, &req); err != nil {
		response.InternalError(c, "上报失败")
		return
	}
	response.Success(c, gin.H{"success": true, "next_report_in": 60})
}

func (h *AgentHandler) GetFreeProbability(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}
	result, err := h.agentService.GetFriendsFreeProbability(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}
	response.Success(c, result)
}

func (h *AgentHandler) QueryAgentData(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}
	var req struct {
		ToAgent string `json:"to_agent" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}
	data, err := h.agentService.GetAgentExposedData(c.Request.Context(), req.ToAgent)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}
	if data == nil {
		response.Success(c, gin.H{"available": false, "reason": "用户未授权或无数据"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "成功", "data": data})
}
AGENTHANDLER

echo "✓ agent.go handler 已创建"

# 4. 更新 redis.go - 追加新方法
if ! grep -q "GetBytes" internal/pkg/tencent/redis.go; then
cat >> internal/pkg/tencent/redis.go << 'REDISAPPEND'

func (r *RedisClient) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return r.client.Get(ctx, key).Bytes()
}

func IsNil(err error) bool {
	return err != nil && err.Error() == "redis: nil"
}
REDISAPPEND
echo "✓ redis.go 已更新"
else
echo "✓ redis.go 已包含所需方法"
fi

# 5. 更新 main.go - 添加 agentService 和路由
if ! grep -q "agentService" cmd/server/main.go; then
    # 添加 agentService 初始化
    sed -i 's/friendshipService := service.NewFriendshipService(friendshipRepo, userRepo, invitationRepo, circleRepo)/friendshipService := service.NewFriendshipService(friendshipRepo, userRepo, invitationRepo, circleRepo)\n\tagentService := service.NewAgentService(redisClient, userRepo, friendshipRepo)/' cmd/server/main.go

    # 添加 agentHandler 初始化
    sed -i 's/friendshipHandler := handler.NewFriendshipHandler(friendshipService)/friendshipHandler := handler.NewFriendshipHandler(friendshipService)\n\tagentHandler := handler.NewAgentHandler(agentService)/' cmd/server/main.go

    # 添加路由
    sed -i 's/friends.GET("\/invited-me", friendshipHandler.GetInvitedMe)/friends.GET("\/invited-me", friendshipHandler.GetInvitedMe)\n\t\t\tfriends.GET("\/free-probability", agentHandler.GetFreeProbability)/' cmd/server/main.go

    # 添加 agent 路由组
    sed -i '/friends.GET("\/free-probability", agentHandler.GetFreeProbability)/a\\n\t\t// Agent 模块\n\t\tagent := authorized.Group("/agent")\n\t\t{\n\t\t\tagent.POST("/status", agentHandler.ReportStatus)\n\t\t\tagent.POST("/query", agentHandler.QueryAgentData)\n\t\t}' cmd/server/main.go

    echo "✓ main.go 已更新"
else
    echo "✓ main.go 已包含 agentService"
fi

# 6. 编译
echo ""
echo "正在编译..."
go build -o server_new ./cmd/server/

if [ $? -eq 0 ]; then
    echo "✓ 编译成功"

    # 7. 停止旧服务
    echo "停止旧服务..."
    pkill -f "/opt/youkong/server" || true
    sleep 2

    # 8. 替换并启动
    mv server server.bak 2>/dev/null || true
    mv server_new server
    chmod +x server

    echo "启动新服务..."
    nohup ./server > server.log 2>&1 &
    sleep 3

    # 9. 检查
    if curl -s localhost:8080/health | grep -q "ok"; then
        echo ""
        echo "========================================="
        echo "✓ 部署成功！"
        echo "========================================="
    else
        echo "✗ 启动失败，查看日志:"
        tail -20 /opt/youkong/server.log
    fi
else
    echo "✗ 编译失败"
fi
