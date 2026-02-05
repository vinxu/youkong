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

// HomeService 首页服务
type HomeService struct {
	friendshipRepo *repository.FriendshipRepository
	userRepo       *repository.UserRepository
	memoryRepo     *repository.MemoryRepository
	redisClient    *tencent.RedisClient
}

// NewHomeService 创建首页服务
func NewHomeService(friendshipRepo *repository.FriendshipRepository, userRepo *repository.UserRepository, memoryRepo *repository.MemoryRepository, redisClient *tencent.RedisClient) *HomeService {
	return &HomeService{
		friendshipRepo: friendshipRepo,
		userRepo:       userRepo,
		memoryRepo:     memoryRepo,
		redisClient:    redisClient,
	}
}

// FriendGridItem 宫格中的好友项
type FriendGridItem struct {
	UserID       string `json:"user_id"`
	Nickname     string `json:"nickname"`
	Avatar       string `json:"avatar,omitempty"`
	Emoji        string `json:"emoji"`
	Status       string `json:"status"`
	UpdatedAt    string `json:"updated_at"`
	RelativeTime string `json:"relative_time"`
	City         string `json:"city,omitempty"` // 城市名称（如"上海"、"北京"）
	IsAvailable  bool   `json:"is_available"`   // 是否有空（用于高亮显示）
}

// GridResponse 宫格响应
type GridResponse struct {
	GridSize int              `json:"grid_size"`
	Friends  []FriendGridItem `json:"friends"`
}

// GetGridData 获取宫格数据
func (s *HomeService) GetGridData(ctx context.Context, userID string) (*GridResponse, error) {
	// 1. 获取所有好友
	friendships, err := s.friendshipRepo.GetFriendsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 2. 提取好友 ID (包含自己)
	friendIDs := make([]string, 0, len(friendships)+1)
	friendIDs = append(friendIDs, userID) // 自己排在第一个
	for _, f := range friendships {
		friendIDs = append(friendIDs, f.FriendID)
	}

	// 3. 批量获取用户信息
	users, err := s.userRepo.GetByIDs(ctx, friendIDs)
	if err != nil {
		return nil, err
	}
	userMap := make(map[string]*model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 4. 批量获取分析缓存
	analysisMap, err := s.memoryRepo.GetAnalysisCacheByUserIDs(ctx, friendIDs)
	if err != nil {
		return nil, err
	}

	// 5. 批量获取城市信息（从 Redis 中的实时状态）
	cityMap := s.getUserCities(ctx, friendIDs)

	// 6. 构建宫格数据
	friends := make([]FriendGridItem, 0, len(friendIDs))
	for _, fid := range friendIDs {
		user := userMap[fid]
		if user == nil {
			continue
		}

		analysis := analysisMap[fid]
		emoji := "🤔"
		status := "未知"
		updatedAt := time.Now()
		isAvailable := false

		if analysis != nil {
			emoji = analysis.LifeStatus.Emoji
			status = analysis.LifeStatus.Label
			updatedAt = analysis.UpdatedAt

			// 如果 emoji 为空，使用默认值
			if emoji == "" {
				emoji = "🤔"
			}
			if status == "" {
				status = "未知"
			}

			// 判断是否有空（Availability.Status == "有空"）
			if analysis.Availability.Status == "有空" {
				isAvailable = true
			}
		}

		friends = append(friends, FriendGridItem{
			UserID:       user.ID,
			Nickname:     user.Nickname,
			Avatar:       user.Avatar,
			Emoji:        emoji,
			Status:       status,
			UpdatedAt:    updatedAt.Format(time.RFC3339),
			RelativeTime: formatRelativeTime(updatedAt),
			City:         cityMap[fid],
			IsAvailable:  isAvailable,
		})

		// 最多显示 16 个（包括自己）
		if len(friends) >= 16 {
			break
		}
	}

	// 7. 计算宫格大小
	gridSize := calculateGridSize(len(friends))

	return &GridResponse{
		GridSize: gridSize,
		Friends:  friends,
	}, nil
}

// calculateGridSize 计算宫格大小
func calculateGridSize(count int) int {
	if count <= 1 {
		return 1
	}
	if count <= 4 {
		return 2
	}
	if count <= 9 {
		return 3
	}
	return 4
}

// getUserCities 批量获取用户的城市信息
func (s *HomeService) getUserCities(ctx context.Context, userIDs []string) map[string]string {
	cityMap := make(map[string]string)
	if s.redisClient == nil {
		return cityMap
	}

	for _, userID := range userIDs {
		key := fmt.Sprintf("agent:status:%s", userID)
		data, err := s.redisClient.GetBytes(ctx, key)
		if err != nil || data == nil {
			continue
		}

		var status model.UserRealtimeStatus
		if err := json.Unmarshal(data, &status); err != nil {
			continue
		}

		// 优先使用 City 字段，其次使用 Location.City
		if status.City != "" {
			cityMap[userID] = status.City
		} else if status.Location.City != "" {
			cityMap[userID] = status.Location.City
		}
	}

	return cityMap
}

// formatRelativeTime 格式化相对时间
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)

	minutes := int(duration.Minutes())
	hours := int(duration.Hours())
	days := int(duration.Hours() / 24)

	if minutes < 1 {
		return "刚刚"
	}
	if minutes < 60 {
		return fmt.Sprintf("%d分钟前", minutes)
	}
	if hours < 24 {
		return fmt.Sprintf("%d小时前", hours)
	}
	if days < 30 {
		return fmt.Sprintf("%d天前", days)
	}
	return "很久以前"
}
