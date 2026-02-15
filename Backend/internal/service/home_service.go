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
	friendshipRepo  *repository.FriendshipRepository
	userRepo        *repository.UserRepository
	memoryRepo      *repository.MemoryRepository
	scheduleRepo    *repository.ScheduleRepository
	redisClient     *tencent.RedisClient
	sceneService    *SceneService
	interactionService *InteractionService
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

// SetSceneService 设置场景服务
func (s *HomeService) SetSceneService(ss *SceneService) {
	s.sceneService = ss
}

// SetInteractionService 设置互动服务
func (s *HomeService) SetInteractionService(is *InteractionService) {
	s.interactionService = is
}

// FriendGridItem 宫格中的好友项
type FriendGridItem struct {
	UserID        string `json:"user_id"`
	Nickname      string `json:"nickname"`
	Avatar        string `json:"avatar,omitempty"`
	Emoji         string `json:"emoji"`
	Status        string `json:"status"`
	UpdatedAt     string `json:"updated_at"`
	RelativeTime  string `json:"relative_time"`
	City          string `json:"city,omitempty"`        // 城市名称（如"上海"、"北京"）
	IsAvailable   bool   `json:"is_available"`          // 是否有空（用于高亮显示）
	GifURL        string `json:"gif_url,omitempty"`     // GIF 动图 URL（仅有空时有值）
	GiphyQuery    string `json:"giphy_query,omitempty"` // Giphy 搜索词（客户端可自行搜索）
	UseGif        bool   `json:"use_gif"`               // 是否使用 GIF 显示模式
	NeedsSchedule bool   `json:"needs_schedule"`        // 自己当前无行程，需要设置
	// Rive 动画状态
	RiveState       string                   `json:"rive_state,omitempty"`
	// 像素场景
	ScenePose       string                   `json:"scene_pose,omitempty"`
	SceneArms       string                   `json:"scene_arms,omitempty"`
	SceneExpression string                   `json:"scene_expression,omitempty"`
	SceneProp       string                   `json:"scene_prop,omitempty"`
	SceneSurface    string                   `json:"scene_surface,omitempty"`
	// AI 生成的互动选项
	Interactions    []model.InteractionOption `json:"interactions,omitempty"`
	// 今日互动计数
	InteractionCount int                     `json:"interaction_count"`
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

	// 7. 批量获取时刻表当前时段状态（作为首页展示的权威数据源）
	currentItemMap, hasCurrentStatusMap := s.getUserScheduleStatus(ctx, friendIDs)

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
		gifURL := ""
		giphyQuery := ""
		useGif := false

		// 时间表当前时段优先：确保首页和时间表显示一致
		if schedItem, ok := currentItemMap[fid]; ok {
			emoji = schedItem.Emoji
			status = schedItem.Status
			isAvailable = schedItem.Highlight
			gifURL = schedItem.GifURL
			giphyQuery = schedItem.GiphyQuery
			useGif = gifURL != ""
			// 从 analysis 中补充 updatedAt
			if analysis != nil {
				updatedAt = analysis.UpdatedAt
			}
		} else if analysis != nil {
			// 没有当前时段时，回退到 analysis_cache
			emoji = analysis.LifeStatus.Emoji
			status = analysis.LifeStatus.Label
			updatedAt = analysis.UpdatedAt
			gifURL = analysis.LifeStatus.GifURL
			giphyQuery = analysis.LifeStatus.GiphyQuery
			useGif = analysis.LifeStatus.UseGif

			if analysis.Availability.Status == "有空" {
				isAvailable = true
			}
		}

		// 如果 emoji/status 为空，使用默认值
		if emoji == "" {
			emoji = "🤔"
		}
		if status == "" {
			status = "未知"
		}

		isSelf := fid == userID

		// 好友过滤：此刻没有时刻表条目的不显示（自己始终显示）
		if !isSelf && !hasCurrentStatusMap[fid] {
			continue
		}

		// 只有用户开启了城市显示才返回城市信息
		city := ""
		if showCityMap[fid] {
			city = cityMap[fid]
		}

		// 自己此刻没有时刻表条目时，标记为需要设置行程
		needsSchedule := isSelf && !hasCurrentStatusMap[fid]

		friends = append(friends, FriendGridItem{
			UserID:        user.ID,
			Nickname:      user.Nickname,
			Avatar:        user.Avatar,
			Emoji:         emoji,
			Status:        status,
			UpdatedAt:     updatedAt.Format(time.RFC3339),
			RelativeTime:  formatRelativeTime(updatedAt),
			City:          city,
			IsAvailable:   isAvailable,
			GifURL:        gifURL,
			GiphyQuery:    giphyQuery,
			UseGif:        useGif,
			NeedsSchedule: needsSchedule,
		})

		// 最多显示 16 个（包括自己）
		if len(friends) >= 16 {
			break
		}
	}

	// 8. 填充场景+互动数据
	s.enrichFriendsWithScene(ctx, friends)

	// 9. 计算宫格大小
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

// getUserScheduleStatus 检查每个用户当前时段的时刻表状态
// 返回:
//   - highlightMap: 当前时段 highlight=true 的用户
//   - hasCurrentStatusMap: 此刻有时刻表条目覆盖的用户（"此刻有状态"）
// scheduleStatusInfo 时间表当前时段的状态信息
type scheduleStatusInfo struct {
	Emoji      string
	Status     string
	Highlight  bool
	GifURL     string
	GiphyQuery string
}

func (s *HomeService) getUserScheduleStatus(ctx context.Context, userIDs []string) (currentItemMap map[string]*scheduleStatusInfo, hasCurrentStatusMap map[string]bool) {
	currentItemMap = make(map[string]*scheduleStatusInfo)
	hasCurrentStatusMap = make(map[string]bool)
	if s.scheduleRepo == nil {
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	currentTime := now.Format("15:04")

	for _, uid := range userIDs {
		schedule, err := s.scheduleRepo.GetLatestByUserAndDate(ctx, uid, today)
		if err != nil || schedule == nil {
			continue
		}

		// 检查当前时段是否有条目
		for _, item := range schedule.Items {
			if item.StartTime <= currentTime && currentTime < item.EndTime {
				hasCurrentStatusMap[uid] = true
				currentItemMap[uid] = &scheduleStatusInfo{
					Emoji:      item.Emoji,
					Status:     item.Status,
					Highlight:  item.Highlight,
					GifURL:     item.GifURL,
					GiphyQuery: item.GiphyQuery,
				}
				break
			}
		}
	}

	return
}

// enrichFriendsWithScene 为好友列表填充场景+互动数据
func (s *HomeService) enrichFriendsWithScene(ctx context.Context, friends []FriendGridItem) {
	if len(friends) == 0 {
		return
	}

	// 收集所有好友 ID
	friendIDs := make([]string, 0, len(friends))
	for _, f := range friends {
		friendIDs = append(friendIDs, f.UserID)
	}

	// 批量获取场景缓存
	sceneMap := make(map[string]*model.SceneEnrichment)
	if s.sceneService != nil {
		sceneMap = s.sceneService.GetCachedScenes(ctx, friendIDs)
	}

	// 批量获取今日互动计数
	interactionCountMap := make(map[string]int)
	if s.interactionService != nil {
		if counts, err := s.interactionService.GetTodayCounts(ctx, friendIDs); err == nil {
			interactionCountMap = counts
		}
	}

	// 填充到每个好友项
	for i := range friends {
		uid := friends[i].UserID

		if scene, ok := sceneMap[uid]; ok {
			friends[i].RiveState = scene.RiveState
			friends[i].ScenePose = scene.Scene.Pose
			friends[i].SceneArms = scene.Scene.Arms
			friends[i].SceneExpression = scene.Scene.Expression
			friends[i].SceneProp = scene.Scene.Prop
			friends[i].SceneSurface = scene.Scene.Surface
			friends[i].Interactions = scene.Interactions
		}

		if count, ok := interactionCountMap[uid]; ok {
			friends[i].InteractionCount = count
		}
	}
}
