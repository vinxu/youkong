package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	// 是否已对该好友当前状态 +1
	HasPlusOned bool `json:"has_plus_oned"`
	// 是否是自己
	IsSelf bool `json:"is_self"`
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
			IsSelf:        isSelf,
		})

		// 最多显示 16 个（包括自己）
		if len(friends) >= 16 {
			break
		}
	}

	// 8. 填充场景+互动数据
	s.enrichFriendsWithScene(ctx, friends)

	// 8.5 填充 +1 状态
	s.enrichPlusOned(ctx, userID, friends)

	// 9. 为缺少 giphy_query 的好友补充英文搜索词（客户端用它快速获取 GIF）
	s.enrichGiphyQueries(friends)

	// 10. 从 Redis 缓存填充 gif_url（客户端写回的 cos_url，秒加载）
	s.resolveGifURLsFromCache(ctx, friends)

	// 11. 计算宫格大小
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

// enrichPlusOned 为好友列表填充：
// 1. has_plus_oned: 当前用户是否已对该好友当前状态 +1
// 2. interaction_count: 该好友当前状态收到的 +1 总数（覆盖旧的全天计数）
func (s *HomeService) enrichPlusOned(ctx context.Context, userID string, friends []FriendGridItem) {
	if s.interactionService == nil || len(friends) == 0 {
		return
	}

	// 收集所有人（含自己）的 ID 及 updatedAt
	allIDs := make([]string, 0, len(friends))
	updatedAtMap := make(map[string]time.Time)
	for _, f := range friends {
		allIDs = append(allIDs, f.UserID)
		if t, err := time.Parse(time.RFC3339, f.UpdatedAt); err == nil {
			updatedAtMap[f.UserID] = t
		}
	}
	if len(allIDs) == 0 {
		return
	}

	// 批量获取我对每个好友最近 +1 时间（不含自己）
	friendIDs := make([]string, 0, len(allIDs))
	for _, id := range allIDs {
		if id != userID {
			friendIDs = append(friendIDs, id)
		}
	}
	var plusOneMap map[string]time.Time
	if len(friendIDs) > 0 {
		plusOneMap, _ = s.interactionService.GetLatestPlusOneMap(ctx, userID, friendIDs)
	}

	// 批量获取所有人自其状态更新后收到的 +1 数
	var earliest time.Time
	for _, t := range updatedAtMap {
		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
	}
	if earliest.IsZero() {
		earliest = time.Now().Add(-24 * time.Hour)
	}
	allPlusOnes, _ := s.interactionService.GetPlusOnesSince(ctx, allIDs, earliest)

	for i := range friends {
		fid := friends[i].UserID
		friendUpdatedAt := updatedAtMap[fid]

		// has_plus_oned: 我最近的 +1 时间 >= 好友状态更新时间（仅好友，不含自己）
		if fid != userID {
			if latestPlusOne, ok := plusOneMap[fid]; ok {
				if !friendUpdatedAt.IsZero() {
					friends[i].HasPlusOned = !latestPlusOne.Before(friendUpdatedAt)
				}
			}
		}

		// interaction_count: 只计算当前状态期间的 +1 数（含自己）
		count := 0
		if interactions, ok := allPlusOnes[fid]; ok && !friendUpdatedAt.IsZero() {
			for _, inter := range interactions {
				if !inter.CreatedAt.Before(friendUpdatedAt) {
					count++
				}
			}
		}
		friends[i].InteractionCount = count
	}
}

// enrichGiphyQueries 为缺少 giphy_query 的好友补充英文搜索词
// 服务器无法直接访问 gif.playa.cn（延迟太高），所以只生成英文关键词，
// 客户端用 giphy_query 调 gif.playa.cn 获取 GIF（英文查询 ~500ms，中文会超时）
func (s *HomeService) enrichGiphyQueries(friends []FriendGridItem) {
	for i := range friends {
		if friends[i].GiphyQuery != "" || friends[i].GifURL != "" || friends[i].NeedsSchedule {
			continue
		}
		if translated := translateStatusToEnglish(friends[i].Status); translated != "" {
			friends[i].GiphyQuery = translated
		}
	}
}


// translateStatusToEnglish 将中文状态翻译为英文 GIF 搜索词
// 基于关键词匹配，覆盖常见活动。未匹配时返回空字符串。
func translateStatusToEnglish(status string) string {
	// 关键词→英文搜索词映射（按优先级排列，先匹配先返回）
	mappings := []struct {
		keywords []string
		english  string
	}{
		{[]string{"睡觉", "睡眠", "入睡", "午睡", "休息"}, "sleeping bed"},
		{[]string{"工作", "办公", "加班", "上班"}, "working office"},
		{[]string{"开会", "会议"}, "meeting office"},
		{[]string{"做饭", "煮饭", "烹饪", "炒菜"}, "cooking kitchen"},
		{[]string{"吃饭", "用餐", "午餐", "晚餐", "早餐", "吃"}, "eating food"},
		{[]string{"烤串", "烧烤", "撸串"}, "barbecue grill"},
		{[]string{"火锅"}, "hotpot dining"},
		{[]string{"咖啡"}, "coffee cafe"},
		{[]string{"喝酒", "酒吧", "喝啤酒"}, "drinking bar"},
		{[]string{"健身", "运动", "锻炼", "撸铁"}, "gym exercise"},
		{[]string{"跑步", "慢跑"}, "running jogging"},
		{[]string{"游泳"}, "swimming pool"},
		{[]string{"瑜伽"}, "yoga stretching"},
		{[]string{"散步", "走路", "遛弯", "逛"}, "walking outdoor"},
		{[]string{"购物", "逛街", "超市", "商场"}, "shopping store"},
		{[]string{"看书", "阅读", "读书", "学习"}, "reading book"},
		{[]string{"打游戏", "游戏", "打机"}, "gaming playing"},
		{[]string{"听音乐", "音乐"}, "listening music headphones"},
		{[]string{"看电影", "电影", "看剧", "追剧"}, "watching movie"},
		{[]string{"刷手机", "玩手机"}, "scrolling phone"},
		{[]string{"开车", "驾驶", "自驾"}, "driving car"},
		{[]string{"坐车", "地铁", "公交", "通勤"}, "commuting transit"},
		{[]string{"飞机", "航班"}, "airplane flying"},
		{[]string{"高铁", "火车"}, "train travel"},
		{[]string{"旅游", "旅行", "出游"}, "traveling vacation"},
		{[]string{"拍照", "摄影"}, "photography camera"},
		{[]string{"画画", "绘画"}, "painting drawing"},
		{[]string{"弹琴", "练琴", "钢琴", "吉他"}, "playing music instrument"},
		{[]string{"唱歌", "KTV", "卡拉OK"}, "singing karaoke"},
		{[]string{"聊天", "谈话", "聚会"}, "chatting friends"},
		{[]string{"遛狗"}, "walking dog"},
		{[]string{"看医生", "看病", "医院", "体检"}, "hospital doctor"},
		{[]string{"带娃", "照顾孩子"}, "parenting baby"},
		{[]string{"打扫", "做家务", "收拾"}, "cleaning house"},
		{[]string{"洗澡", "淋浴"}, "shower bathroom"},
		{[]string{"化妆", "护肤"}, "makeup beauty"},
		{[]string{"寺庙", "拜佛"}, "temple praying"},
		{[]string{"在家"}, "relaxing home"},
		{[]string{"出差"}, "business trip"},
		{[]string{"约会"}, "dating romantic"},
	}

	for _, m := range mappings {
		for _, kw := range m.keywords {
			if containsChinese(status, kw) {
				return m.english
			}
		}
	}
	return ""
}

// containsChinese 检查 status 是否包含关键词
func containsChinese(status, keyword string) bool {
	return len(status) > 0 && len(keyword) > 0 && strings.Contains(status, keyword)
}

// ==================== GIF URL 缓存系统 ====================
// 策略：客户端解析 gif.playa.cn 获得 cos_url 后写回服务器缓存。
// 后续任何客户端加载 Grid 时直接从缓存返回 cos_url，无需再调 gif.playa.cn。

// gifCacheKey 计算 GIF 缓存键（与 iOS/Android 客户端 seed 算法一致）
// seed = md5(friendId + giphyQuery + today)
func gifCacheKey(friendID, giphyQuery string) string {
	today := time.Now().Format("2006-01-02")
	hash := md5.Sum([]byte(friendID + giphyQuery + today))
	seed := hex.EncodeToString(hash[:])
	return "gif:url:" + seed
}

// resolveGifURLsFromCache 从 Redis 缓存填充缺失的 gif_url
func (s *HomeService) resolveGifURLsFromCache(ctx context.Context, friends []FriendGridItem) {
	if s.redisClient == nil {
		return
	}

	for i := range friends {
		// 已有 gif_url 或无 giphy_query → 跳过
		if friends[i].GifURL != "" || friends[i].GiphyQuery == "" || friends[i].NeedsSchedule {
			continue
		}
		cacheKey := gifCacheKey(friends[i].UserID, friends[i].GiphyQuery)
		if cosURL, err := s.redisClient.Get(ctx, cacheKey); err == nil && cosURL != "" {
			friends[i].GifURL = cosURL
			friends[i].UseGif = true
		}
	}
}

// GifCacheItem 客户端写回的 GIF 缓存条目
type GifCacheItem struct {
	FriendID   string `json:"friend_id"`
	GiphyQuery string `json:"query"`
	CosURL     string `json:"cos_url"`
}

// FetchGifFromProxy 从 gif.playa.cn 代理获取 GIF cos_url
// 返回 COS URL 或空字符串（失败时）
func FetchGifFromProxy(query, seed string) string {
	encoded := url.QueryEscape(query)
	reqURL := fmt.Sprintf("https://gif.playa.cn/api/giphy?q=%s&seed=%s", encoded, seed)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		fmt.Printf("[GifProxy] 请求失败: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var result struct {
		Result struct {
			CosURL string `json:"cos_url"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Result.CosURL == "" {
		fmt.Printf("[GifProxy] 解析失败: query=%s body=%s\n", query, string(body[:min(len(body), 100)]))
		return ""
	}
	return result.Result.CosURL
}

// ResolveAndCacheGifURL 异步解析 GIF 并回填到 schedule + analysis cache + redis gif cache
// 在任何状态保存后调用（go s.homeService.ResolveAndCacheGifURL(...)）
func (s *HomeService) ResolveAndCacheGifURL(userID, giphyQuery string) {
	if giphyQuery == "" {
		return
	}

	// 计算 seed（与客户端一致）
	today := time.Now().Format("2006-01-02")
	hash := md5.Sum([]byte(userID + giphyQuery + today))
	seed := hex.EncodeToString(hash[:])

	cosURL := FetchGifFromProxy(giphyQuery, seed)
	if cosURL == "" {
		return
	}
	fmt.Printf("[GifResolve] 成功 user=%s query=%s → %s\n", userID, giphyQuery, cosURL[:60])

	ctx := context.Background()

	// 1. 更新 gif redis 缓存
	cacheKey := "gif:url:" + seed
	if s.redisClient != nil {
		s.redisClient.Set(ctx, cacheKey, cosURL, 25*time.Hour)
	}

	// 2. 更新 schedule item 的 gif_url
	if s.scheduleRepo != nil {
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, userID, today); err == nil && schedule != nil {
			currentTime := now.Format("15:04")
			for i := range schedule.Items {
				item := &schedule.Items[i]
				if item.GiphyQuery == giphyQuery && item.GifURL == "" {
					item.GifURL = cosURL
					_ = s.scheduleRepo.Update(ctx, schedule)
					break
				}
				// 也匹配当前时段
				if isTimeInSlot(currentTime, item.StartTime, item.EndTime) && item.GifURL == "" {
					item.GifURL = cosURL
					_ = s.scheduleRepo.Update(ctx, schedule)
					break
				}
			}
		}
	}

	// 3. 更新 analysis cache 的 gif_url
	if s.memoryRepo != nil {
		if existing, err := s.memoryRepo.GetAnalysisCache(ctx, userID); err == nil && existing != nil {
			if existing.LifeStatus.GifURL == "" {
				existing.LifeStatus.GifURL = cosURL
				existing.LifeStatus.UseGif = true
				_ = s.memoryRepo.SaveAnalysisCache(ctx, userID, existing)
			}
		}
	}
}

// isTimeInSlot 检查时间是否在时段内
func isTimeInSlot(current, start, end string) bool {
	if end <= start { // 跨午夜
		return current >= start || current < end
	}
	return current >= start && current < end
}

// CacheGifURLs 批量缓存客户端解析的 GIF cos_url（写回接口）
func (s *HomeService) CacheGifURLs(ctx context.Context, items []GifCacheItem) int {
	if s.redisClient == nil {
		return 0
	}
	cached := 0
	for _, item := range items {
		if item.FriendID == "" || item.GiphyQuery == "" || item.CosURL == "" {
			continue
		}
		cacheKey := gifCacheKey(item.FriendID, item.GiphyQuery)
		// TTL 25h：seed 含当日日期，跨天自动失效
		if err := s.redisClient.Set(ctx, cacheKey, item.CosURL, 25*time.Hour); err == nil {
			cached++
		}
	}
	if cached > 0 {
		fmt.Printf("[GifCache] 写入 %d 条 GIF 缓存\n", cached)
	}
	return cached
}
