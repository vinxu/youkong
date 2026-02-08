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
	scheduleRepo   *repository.ScheduleRepository
	redisClient    *tencent.RedisClient
}

// NewHomeService 创建首页服务
func NewHomeService(friendshipRepo *repository.FriendshipRepository, userRepo *repository.UserRepository, memoryRepo *repository.MemoryRepository, scheduleRepo *repository.ScheduleRepository, redisClient *tencent.RedisClient) *HomeService {
	return &HomeService{
		friendshipRepo: friendshipRepo,
		userRepo:       userRepo,
		memoryRepo:     memoryRepo,
		scheduleRepo:   scheduleRepo,
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
	City         string `json:"city,omitempty"`       // 城市名称（如"上海"、"北京"）
	IsAvailable  bool   `json:"is_available"`         // 是否有空（用于高亮显示）
	GifURL       string `json:"gif_url,omitempty"`    // GIF 动图 URL（仅有空时有值）
	GiphyQuery   string `json:"giphy_query,omitempty"` // Giphy 搜索词（客户端可自行搜索）
	UseGif       bool   `json:"use_gif"`              // 是否使用 GIF 显示模式
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

	// 6. 批量获取用户的城市显示设置
	showCityMap := s.getUserShowCitySettings(ctx, friendIDs)

	// 7. 批量检查时刻表 highlight 状态（当前时段是否标记为"有空"）
	highlightMap := s.getUserHighlightStatus(ctx, friendIDs)

	// 8. 构建宫格数据
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

		// 时刻表 highlight 优先：用户手动设置的"有空"状态实时生效
		if hl, ok := highlightMap[fid]; ok {
			isAvailable = hl
		}

		// 只有用户开启了城市显示才返回城市信息
		city := ""
		if showCityMap[fid] {
			city = cityMap[fid]
		}

		// 获取 GIF 信息
		gifURL := ""
		giphyQuery := ""
		useGif := false
		if analysis != nil {
			gifURL = analysis.LifeStatus.GifURL
			giphyQuery = analysis.LifeStatus.GiphyQuery
			useGif = analysis.LifeStatus.UseGif
		}

		friends = append(friends, FriendGridItem{
			UserID:       user.ID,
			Nickname:     user.Nickname,
			Avatar:       user.Avatar,
			Emoji:        emoji,
			Status:       status,
			UpdatedAt:    updatedAt.Format(time.RFC3339),
			RelativeTime: formatRelativeTime(updatedAt),
			City:         city,
			IsAvailable:  isAvailable,
			GifURL:       gifURL,
			GiphyQuery:   giphyQuery,
			UseGif:       useGif,
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

// getUserShowCitySettings 批量获取用户的城市显示设置
func (s *HomeService) getUserShowCitySettings(ctx context.Context, userIDs []string) map[string]bool {
	showCityMap := make(map[string]bool)

	// 默认所有用户都显示城市
	for _, userID := range userIDs {
		showCityMap[userID] = true
	}

	if s.scheduleRepo == nil {
		return showCityMap
	}

	// 获取每个用户的偏好设置
	for _, userID := range userIDs {
		pref, err := s.scheduleRepo.GetUserPreference(ctx, userID)
		if err != nil {
			// 获取失败时默认显示城市
			continue
		}
		showCityMap[userID] = pref.ShowCity
	}

	return showCityMap
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

// getUserHighlightStatus 检查每个用户当前时段的时刻表是否有 highlight=true 的条目
// 返回 map[userID]bool，仅包含有明确 highlight 设置的用户
func (s *HomeService) getUserHighlightStatus(ctx context.Context, userIDs []string) map[string]bool {
	result := make(map[string]bool)
	if s.scheduleRepo == nil {
		return result
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	currentTime := now.Format("15:04")

	for _, uid := range userIDs {
		schedule, err := s.scheduleRepo.GetLatestByUserAndDate(ctx, uid, today)
		if err != nil || schedule == nil {
			continue
		}

		// 检查当前时段是否有 highlight 条目
		for _, item := range schedule.Items {
			if item.StartTime <= currentTime && currentTime < item.EndTime {
				// 找到当前时段的条目，用它的 highlight 值
				result[uid] = item.Highlight
				break
			}
		}
	}

	return result
}
