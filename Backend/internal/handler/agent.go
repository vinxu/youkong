package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/agent"
	"youkong/internal/pkg/response"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
	"youkong/internal/service"
)

// v3InferRunning 在飞去重：防止同一用户并发触发推断
// key=userID, value=struct{}
var v3InferRunning sync.Map

// FriendshipChecker 好友关系检查接口
type FriendshipChecker interface {
	AreFriends(ctx context.Context, userID, friendID string) (bool, error)
}

// inferenceStreamEvent SSE 流式推断事件（V4 流式输出格式）
type inferenceStreamEvent struct {
	Type    string      `json:"type"`              // phase, tool, result, error
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// AgentHandler Agent 处理器
type AgentHandler struct {
	agentService           *service.AgentService
	memoryService          *service.MemoryService
	voiceScheduleService   *service.VoiceScheduleService
	voiceScheduleServiceV4 *service.VoiceScheduleServiceV4 // V4 版本
	agentChatService       *service.AgentChatService
	scheduleRepo           ScheduleRepositoryInterface
	friendshipRepo         FriendshipChecker              // 好友关系检查
	bookingService         *service.BookingService       // 预约服务（可见性过滤）
	redisClient            *tencent.RedisClient           // Redis（用于权限追踪等）
	userSettingsRepo       *repository.UserSettingsRepository // 用户设置（自动推测开关）
	inferencePersonaService *service.InferencePersonaService // Persona 生成服务（测试用）
	memoryRepo              *repository.MemoryRepository      // Memory repo（persona 查询）
	// STS 配置
	stsSecretID  string
	stsSecretKey string
	stsBucket    string
	stsRegion    string
}

// ScheduleRepositoryInterface 时刻表 Repository 接口
type ScheduleRepositoryInterface interface {
	GetUserScheduleHistory(ctx context.Context, userID string, beforeDate string, limit int) ([]*model.StatusSchedule, error)
	GetUserOldestScheduleDate(ctx context.Context, userID string) (string, error)
	CountUserSchedules(ctx context.Context, userID string) (int, error)
	GetActiveByUserAndDate(ctx context.Context, userID string, date time.Time) (*model.StatusSchedule, error)
	GetLatestByUserAndDate(ctx context.Context, userID string, date time.Time) (*model.StatusSchedule, error)
	GetAllByUserAndDate(ctx context.Context, userID string, date time.Time) ([]*model.StatusSchedule, error)
	Create(ctx context.Context, schedule *model.StatusSchedule) error
	Update(ctx context.Context, schedule *model.StatusSchedule) error
}

// NewAgentHandler 创建 Agent 处理器
func NewAgentHandler(agentService *service.AgentService, memoryService *service.MemoryService, voiceScheduleService *service.VoiceScheduleService, agentChatService *service.AgentChatService, scheduleRepo ScheduleRepositoryInterface, friendshipRepo FriendshipChecker) *AgentHandler {
	return &AgentHandler{
		agentService:         agentService,
		memoryService:        memoryService,
		voiceScheduleService: voiceScheduleService,
		agentChatService:     agentChatService,
		scheduleRepo:         scheduleRepo,
		friendshipRepo:       friendshipRepo,
	}
}

// SetVoiceScheduleServiceV4 设置 V4 版本语音时刻表服务
func (h *AgentHandler) SetVoiceScheduleServiceV4(svc *service.VoiceScheduleServiceV4) {
	h.voiceScheduleServiceV4 = svc
}

// SetBookingService 设置预约服务（用于时刻表可见性过滤）
func (h *AgentHandler) SetBookingService(bs *service.BookingService) {
	h.bookingService = bs
}

// SetRedisClient 设置 Redis 客户端（用于权限追踪）
func (h *AgentHandler) SetRedisClient(rc *tencent.RedisClient) {
	h.redisClient = rc
}

// SetUserSettingsRepo 设置用户设置 Repository（自动推测需要）
func (h *AgentHandler) SetUserSettingsRepo(repo *repository.UserSettingsRepository) {
	h.userSettingsRepo = repo
}

// trackPermissions 追踪数据权限授权状态（持久化到 Redis，90天 TTL）
// 每次上报时根据数据是否存在来设置或清除对应权限标记
func (h *AgentHandler) trackPermissions(ctx context.Context, userID string, req *model.ExtendedStatusReportRequest) {
	if h.redisClient == nil || req == nil {
		return
	}
	permTTL := 90 * 24 * time.Hour // 90 天

	locKey := fmt.Sprintf("ai:perm:location:%s", userID)
	if req.Location != nil || req.ExtendedLocation != nil {
		_ = h.redisClient.Set(ctx, locKey, "1", permTTL)
	} else {
		_ = h.redisClient.Del(ctx, locKey)
	}

	motionKey := fmt.Sprintf("ai:perm:motion:%s", userID)
	if req.Movement != nil {
		_ = h.redisClient.Set(ctx, motionKey, "1", permTTL)
	} else {
		_ = h.redisClient.Del(ctx, motionKey)
	}

	calKey := fmt.Sprintf("ai:perm:calendar:%s", userID)
	if req.Calendar != nil {
		_ = h.redisClient.Set(ctx, calKey, "1", permTTL)
	} else {
		_ = h.redisClient.Del(ctx, calKey)
	}
}

// maybeAutoFillSchedule 检查是否需要用推断结果自动填充 schedule 空隙
// 条件：auto_predict 开启 + 当前时间不在任何 schedule item 内
// 复用 ReportStatus 的推断结果，不额外调 LLM
func (h *AgentHandler) maybeAutoFillSchedule(userID string, inference *model.CurrentStatusInference) {
	if h.userSettingsRepo == nil || h.scheduleRepo == nil || h.redisClient == nil || inference == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. 检查开关
	settings, err := h.userSettingsRepo.Get(ctx, userID)
	if err != nil || !settings.AutoPredictEnabled {
		return
	}

	// 2. 防重复（10 分钟内不重复触发）
	lockKey := fmt.Sprintf("auto_infer:lock:%s", userID)
	if v, _ := h.redisClient.Get(ctx, lockKey); v != "" {
		return
	}

	// 3. 获取今天的 schedule，没有则自动创建
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	schedule, err := h.scheduleRepo.GetActiveByUserAndDate(ctx, userID, now)
	if err != nil || schedule == nil {
		// 没有今天的 schedule → 自动创建一个空 schedule，后面填入推测结果
		schedule = &model.StatusSchedule{
			UserID:       userID,
			ScheduleDate: today,
			Status:       model.ScheduleStatusActive,
		}
		if createErr := h.scheduleRepo.Create(ctx, schedule); createErr != nil {
			return
		}
		fmt.Printf("[AutoInfer] 为用户自动创建 schedule: user=%s date=%s\n", userID, today.Format("2006-01-02"))
	}

	// 4. 检查当前时间是否在某个 item 内（含边界）
	currentTime := now.Format("15:04")
	for _, item := range schedule.Items {
		if item.EndTime > item.StartTime {
			if currentTime >= item.StartTime && currentTime <= item.EndTime {
				return // 当前有状态，不需要推测
			}
		} else {
			if currentTime >= item.StartTime || currentTime <= item.EndTime {
				return
			}
		}
	}

	// 5. 当前无状态 → 用推断结果创建 AI 推测 item
	_ = h.redisClient.Set(ctx, lockKey, "1", 10*time.Minute)

	// 计算 endTime：+2h，但不超过下一个手动 item 的 startTime
	endTime := addTimeStr(currentTime, 120)
	for _, item := range schedule.Items {
		if item.StartTime > currentTime && !item.IsAIGuess {
			if item.StartTime < endTime {
				endTime = item.StartTime
			}
			break
		}
	}

	// 防止零时长 item（如 23:59-23:59）
	if endTime <= currentTime {
		fmt.Printf("[AutoInfer] 跳过零时长: user=%s time=%s-%s\n", userID, currentTime, endTime)
		return
	}

	newItem := model.ScheduleItem{
		StartTime: currentTime,
		EndTime:   endTime,
		Emoji:     inference.Emoji,
		Status:    inference.Activity,
		IsAIGuess: true,
		Executed:  true,
	}

	// 清除未来的旧 AI 推测项，追加新项
	var cleaned []model.ScheduleItem
	for _, item := range schedule.Items {
		if item.IsAIGuess && item.StartTime >= currentTime {
			continue
		}
		cleaned = append(cleaned, item)
	}
	cleaned = append(cleaned, newItem)
	sort.Slice(cleaned, func(i, j int) bool { return cleaned[i].StartTime < cleaned[j].StartTime })
	schedule.Items = cleaned

	if err := h.scheduleRepo.Update(ctx, schedule); err != nil {
		fmt.Printf("[AutoInfer] 保存失败: user=%s err=%v\n", userID, err)
		return
	}

	fmt.Printf("[AutoInfer] 上报触发推测成功: user=%s emoji=%s status=%s time=%s-%s\n",
		userID, newItem.Emoji, newItem.Status, newItem.StartTime, newItem.EndTime)
}

// addTimeStr 在 HH:MM 上加 minutes 分钟，上限 23:59
func addTimeStr(t string, minutes int) string {
	if len(t) < 5 {
		return "23:59"
	}
	h := int(t[0]-'0')*10 + int(t[1]-'0')
	m := int(t[3]-'0')*10 + int(t[4]-'0')
	total := h*60 + m + minutes
	if total >= 24*60 {
		total = 23*60 + 59
	}
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}

// SetSTSConfig 设置 STS 配置
func (h *AgentHandler) SetSTSConfig(secretID, secretKey, bucket, region string) {
	h.stsSecretID = secretID
	h.stsSecretKey = secretKey
	h.stsBucket = bucket
	h.stsRegion = region
}

// ReportStatus 上报状态（Agent 循环推断）
// POST /api/agent/status
func (h *AgentHandler) ReportStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	// 允许空 body（用缓存的传感器数据推断）
	var req model.ExtendedStatusReportRequest
	c.ShouldBindJSON(&req)

	// 追踪权限授权状态（用于 AI 就绪检查，持久化到 Redis 90天）
	h.trackPermissions(c.Request.Context(), userID, &req)

	// 使用 V4 推断
	if h.voiceScheduleServiceV4 == nil {
		response.InternalError(c, "推断服务未初始化")
		return
	}

	// 在飞去重：如果该用户已有推断在进行中，返回缓存结果
	if _, loaded := v3InferRunning.LoadOrStore(userID, struct{}{}); loaded {
		fmt.Printf("[上报] 推断去重，跳过 user=%s\n", userID)
		// 尝试返回缓存的分析结果
		if h.memoryService != nil {
			cached, _ := h.memoryService.GetCachedAnalysisByUserIDs(context.Background(), []string{userID})
			if result := cached[userID]; result != nil {
				response.Success(c, gin.H{
					"success":  true,
					"message":  "推断进行中（返回缓存）",
					"analysis": result,
				})
				return
			}
		}
		response.Success(c, gin.H{
			"success": true,
			"message": "推断进行中",
		})
		return
	}
	defer v3InferRunning.Delete(userID)

	fmt.Printf("[上报] 开始 V4 推断 user=%s\n", userID)

	// 使用独立 context，不受客户端断连影响
	// 场景：用户上报状态后切到后台，iOS/Android 可能立即断开 HTTP 连接
	// 使用 background context 确保 LLM 推断完成并缓存结果
	inferCtx, inferCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer inferCancel()

	inference, err := h.voiceScheduleServiceV4.InferWithAgent(inferCtx, userID, &req)
	if err != nil {
		fmt.Printf("[上报] 推断失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "推断失败")
		return
	}

	fmt.Printf("[上报] V4 推断完成 user=%s emoji=%s activity=%s confidence=%s\n",
		userID, inference.Emoji, inference.Activity, inference.Confidence)

	// 持久化城市信息到 Redis（agent:status key）和 MySQL（users.city）
	if h.agentService != nil {
		h.agentService.PersistCityFromExtendedReport(inferCtx, userID, &req)
	}

	// 转换为 AnalysisResult 格式（兼容现有前端）
	analysisResult := inferenceToAnalysisResult(inference)

	// 保留用户手动设置的 GIF 显示模式
	// 自动推断只更新 emoji/status，不覆盖用户的 GIF 选择
	if h.memoryService != nil {
		existingMap, _ := h.memoryService.GetCachedAnalysisByUserIDs(inferCtx, []string{userID})
		if existing := existingMap[userID]; existing != nil && !existing.IsAIGuess && existing.LifeStatus.UseGif {
			analysisResult.LifeStatus.UseGif = true
			if analysisResult.LifeStatus.GifURL == "" {
				analysisResult.LifeStatus.GifURL = existing.LifeStatus.GifURL
			}
			if analysisResult.LifeStatus.GiphyQuery == "" {
				analysisResult.LifeStatus.GiphyQuery = existing.LifeStatus.GiphyQuery
			}
		}
	}

	// 缓存分析结果：如果用户已手动确认过状态（is_ai_guess=false），不覆盖
	// 只有当缓存为空或上次也是自动推断时才更新
	if h.memoryService != nil {
		existingCache, _ := h.memoryService.GetCachedAnalysisByUserIDs(inferCtx, []string{userID})
		if existing := existingCache[userID]; existing != nil && !existing.IsAIGuess {
			fmt.Printf("[上报] 跳过缓存更新：用户已手动确认状态 user=%s\n", userID)
		} else {
			_ = h.memoryService.CacheAnalysisResult(inferCtx, userID, analysisResult)
		}
	}

	// 自动推测：如果用户开启了自动推测且当前无状态，用推断结果填充 schedule
	go h.maybeAutoFillSchedule(userID, inference)

	response.Success(c, gin.H{
		"success":  true,
		"message":  "推断完成",
		"analysis": analysisResult,
	})
}

// inferenceToAnalysisResult 将 CurrentStatusInference 转换为 AnalysisResult（兼容旧格式）
func inferenceToAnalysisResult(inference *model.CurrentStatusInference) *model.AnalysisResult {
	// 根据 confidence 精确映射 probability
	statusText := "可能有空"
	probability := 50
	if inference.IsAvailable {
		switch inference.Confidence {
		case "high":
			statusText = "有空"
			probability = 90
		case "medium":
			statusText = "可能有空"
			probability = 70
		default:
			statusText = "可能有空"
			probability = 60
		}
	} else {
		switch inference.Confidence {
		case "high":
			statusText = "忙碌"
			probability = 10
		case "medium":
			statusText = "可能忙碌"
			probability = 30
		default:
			statusText = "状态未知"
			probability = 40
		}
	}

	result := &model.AnalysisResult{
		Availability: model.AvailabilityAnalysis{
			Status:      statusText,
			Probability: probability,
			Reason:      inference.Activity,
			Confidence:  inference.Confidence,
		},
		LifeStatus: model.LifeStatus{
			Emoji:       inference.Emoji,
			Label:       inference.Activity,
			Description: inference.Reasoning,
		},
		UpdatedAt: time.Now(),
		IsAIGuess: true,
	}

	// 传递 place 信息
	if inference.Place != "" {
		result.LifeStatus.Description = fmt.Sprintf("%s（%s）", inference.Reasoning, inference.Place)
	}

	// 传递 GIF 信息
	if inference.GifURL != "" {
		result.LifeStatus.GifURL = inference.GifURL
	}
	if inference.GiphyQuery != "" {
		result.LifeStatus.GiphyQuery = inference.GiphyQuery
	}

	return result
}

// syncAIGuessToSchedule 同步 AI 推测状态到时刻表
// 规则：
// 1. 获取或创建今日时刻表
// 2. 计算当前时段（向下取整到30分钟）
// 3. 如果该时段已有用户设置的状态（IsAIGuess=false），则不覆盖
// 4. 否则添加/更新当前时段条目，设置 IsAIGuess=true
func (h *AgentHandler) syncAIGuessToSchedule(ctx context.Context, userID string, result *model.AnalysisResult) {
	if h.scheduleRepo == nil || result == nil {
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startTime, endTime := getCurrentTimeSlot(now)

	// 获取或创建今日时刻表
	schedule, err := h.scheduleRepo.GetActiveByUserAndDate(ctx, userID, today)
	if err != nil {
		// 没有活跃时刻表，创建新的
		schedule = &model.StatusSchedule{
			UserID:       userID,
			ScheduleDate: today,
			Items:        model.ScheduleItems{},
			CurrentIndex: 0,
			Status:       model.ScheduleStatusActive,
			Visibility:   model.VisibilityAllFriends,
		}
	}

	// 检查是否有重叠时段
	found := false
	for i, item := range schedule.Items {
		// 检查时段是否重叠
		if item.StartTime == startTime || (item.StartTime <= startTime && item.EndTime > startTime) {
			// 如果已有用户设置的状态（IsAIGuess=false），不覆盖
			if !item.IsAIGuess {
				fmt.Printf("[AI同步] 跳过：已有用户设置的状态 user=%s slot=%s-%s\n",
					userID, item.StartTime, item.EndTime)
				return
			}
			// 更新 AI 推测状态
			schedule.Items[i] = model.ScheduleItem{
				StartTime: startTime,
				EndTime:   endTime,
				Emoji:     result.LifeStatus.Emoji,
				Status:    result.LifeStatus.Label,
				Executed:  false,
				IsAIGuess: true,
			}
			found = true
			break
		}
	}

	// 如果没有找到重叠时段，添加新条目
	if !found {
		schedule.Items = append(schedule.Items, model.ScheduleItem{
			StartTime: startTime,
			EndTime:   endTime,
			Emoji:     result.LifeStatus.Emoji,
			Status:    result.LifeStatus.Label,
			Executed:  false,
			IsAIGuess: true,
		})
	}

	// 保存时刻表
	if schedule.ID == 0 {
		err = h.scheduleRepo.Create(ctx, schedule)
	} else {
		err = h.scheduleRepo.Update(ctx, schedule)
	}

	if err != nil {
		fmt.Printf("[AI同步] 保存时刻表失败 user=%s error=%v\n", userID, err)
	} else {
		fmt.Printf("[AI同步] 成功 user=%s slot=%s-%s emoji=%s status=%s\n",
			userID, startTime, endTime, result.LifeStatus.Emoji, result.LifeStatus.Label)
	}
}

// getCurrentTimeSlot 获取当前时段（向下取整到30分钟）
// 返回 (startTime, endTime)，格式为 "HH:MM"
func getCurrentTimeSlot(now time.Time) (string, string) {
	hour := now.Hour()
	minute := now.Minute()

	// 向下取整到30分钟
	if minute >= 30 {
		minute = 30
	} else {
		minute = 0
	}

	start := fmt.Sprintf("%02d:%02d", hour, minute)

	// 结束时间 +30 分钟
	endTime := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()).Add(30 * time.Minute)
	end := endTime.Format("15:04")

	return start, end
}

// GetFreeProbability 获取好友有空概率列表（增强版，带生活状态）
// GET /api/friends/free-probability
func (h *AgentHandler) GetFreeProbability(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	// 获取基础好友信息
	result, err := h.agentService.GetFriendsFreeProbability(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}

	if result == nil || len(result.Friends) == 0 {
		response.Success(c, result)
		return
	}

	// 获取所有好友 ID
	friendIDs := make([]string, len(result.Friends))
	for i, f := range result.Friends {
		friendIDs[i] = f.FriendID
	}

	// 从缓存获取 LLM 分析结果（优先使用）
	var analysisMap map[string]*model.AnalysisResult
	if h.memoryService != nil {
		analysisMap, _ = h.memoryService.GetCachedAnalysisByUserIDs(c.Request.Context(), friendIDs)
	}

	// 构建响应，优先使用缓存的 LLM 分析结果
	now := time.Now()
	enhanced := make([]model.EnhancedFriendRecommendation, len(result.Friends))
	for i, f := range result.Friends {
		enhanced[i] = model.EnhancedFriendRecommendation{
			FriendID:  f.FriendID,
			Name:      f.Name,
			Avatar:    f.Avatar,
			UpdatedAt: f.UpdatedAt,
		}

		// 优先使用缓存的 LLM 分析结果
		if analysis, ok := analysisMap[f.FriendID]; ok && analysis != nil {
			enhanced[i].Probability = analysis.Availability.Probability
			enhanced[i].Confidence = analysis.Availability.Confidence
			enhanced[i].Reason = analysis.Availability.Reason
			enhanced[i].Color = model.GetProbabilityColor(analysis.Availability.Probability)
			enhanced[i].Emoji = analysis.LifeStatus.Emoji
			enhanced[i].Activity = analysis.LifeStatus.Label
			enhanced[i].UpdatedAt = now.UnixMilli()
		} else {
			// 没有缓存时使用规则计算的结果
			enhanced[i].Probability = f.Probability
			enhanced[i].Confidence = f.Confidence
			enhanced[i].Reason = f.Reason
			enhanced[i].Color = f.Color
			enhanced[i].Emoji = "🤔"
			enhanced[i].Activity = "状态未知"
		}
	}

	// 按概率排序（降序）
	for i := 0; i < len(enhanced)-1; i++ {
		for j := i + 1; j < len(enhanced); j++ {
			if enhanced[j].Probability > enhanced[i].Probability {
				enhanced[i], enhanced[j] = enhanced[j], enhanced[i]
			}
		}
	}

	response.Success(c, model.EnhancedFreeProbabilityResponse{
		Friends:     enhanced,
		GeneratedAt: now.UnixMilli(),
	})
}

// QueryAgentData Agent 间数据请求
// POST /api/agent/query
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

	// 验证是否是好友关系
	// TODO: 添加好友关系验证

	data, err := h.agentService.GetAgentExposedData(c.Request.Context(), req.ToAgent)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}

	if data == nil {
		response.Success(c, gin.H{
			"available": false,
			"reason":    "用户未授权或无数据",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "成功",
		"data":    data,
	})
}

// GetMemory 获取用户核心记忆（调试用）
// GET /api/agent/memory
func (h *AgentHandler) GetMemory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if h.memoryService == nil {
		response.InternalError(c, "记忆服务未启用")
		return
	}

	memory, err := h.memoryService.GetCoreMemory(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}

	if memory == nil {
		response.Success(c, gin.H{
			"message": "暂无记忆数据",
		})
		return
	}

	response.Success(c, memory)
}

// GetMyAnalysis 获取我的分析结果（用于显示自己的状态）
// GET /api/agent/my-analysis
func (h *AgentHandler) GetMyAnalysis(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if h.memoryService == nil {
		response.Success(c, gin.H{
			"analysis": nil,
		})
		return
	}

	// 从缓存获取分析结果
	analysisMap, _ := h.memoryService.GetCachedAnalysisByUserIDs(c.Request.Context(), []string{userID})
	analysis := analysisMap[userID]

	response.Success(c, gin.H{
		"analysis": analysis,
	})
}

// GetHolmesFreeProbability 获取好友有空概率列表（福尔摩斯版，带完整推理过程）
// GET /api/friends/holmes-probability
func (h *AgentHandler) GetHolmesFreeProbability(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	// 获取福尔摩斯分析列表
	result, err := h.agentService.GetFriendsHolmesAnalysis(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}

	response.Success(c, result)
}

// GetHolmesAnalysis 获取单个好友的福尔摩斯分析详情
// GET /api/friends/:id/holmes
func (h *AgentHandler) GetHolmesAnalysis(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	friendID := c.Param("id")
	if friendID == "" {
		response.ParamError(c, "好友ID不能为空")
		return
	}

	// TODO: 验证好友关系

	// 获取福尔摩斯分析
	result, err := h.agentService.GetHolmesAnalysis(c.Request.Context(), friendID)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}

	if result == nil {
		response.Success(c, gin.H{
			"message":   "暂无分析数据",
			"friend_id": friendID,
		})
		return
	}

	response.Success(c, result)
}

// ReportStatusStream 流式上报状态（SSE 实时输出推理过程）
// POST /api/agent/status/stream
func (h *AgentHandler) ReportStatusStream(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var req model.ExtendedStatusReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 禁用 nginx 缓冲

	// 获取 ResponseWriter 用于 flush
	w := c.Writer

	// 使用独立 context，不受客户端断连影响
	sseCtx, sseCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer sseCancel()

	// 流式执行福尔摩斯推理
	result, err := h.agentService.ReportExtendedStatusStream(sseCtx, userID, &req, func(event interface{}) {
		// 将事件序列化为 JSON
		data, err := json.Marshal(event)
		if err != nil {
			return
		}

		// 写入 SSE 格式
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	})

	if err != nil {
		// 发送错误事件
		errEvent := map[string]string{
			"type":    "error",
			"content": err.Error(),
		}
		data, _ := json.Marshal(errEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
		return
	}

	// ✅ 立即缓存分析结果（转换为 AnalysisResult 格式）
	if result != nil && h.memoryService != nil {
		analysisResult := &model.AnalysisResult{
			Availability: model.AvailabilityAnalysis{
				Status:      getStatusFromProbability(result.Result.Probability),
				Probability: result.Result.Probability,
				Reason:      result.Result.Summary,
				Confidence:  result.Result.Confidence,
			},
			LifeStatus: model.LifeStatus{
				Emoji:       result.Result.Emoji,
				Label:       result.Result.Summary,
				Description: result.Result.Summary,
			},
			UpdatedAt: time.Now(),
		}

		// 异步缓存，不阻塞响应（不覆盖用户手动确认的状态）
		go func() {
			ctx := context.Background()
			existingCache, _ := h.memoryService.GetCachedAnalysisByUserIDs(ctx, []string{userID})
			if existing := existingCache[userID]; existing != nil && !existing.IsAIGuess {
				fmt.Printf("[Holmes 缓存] 跳过：用户已手动确认状态 user=%s\n", userID)
				return
			}
			err := h.memoryService.CacheAnalysisResult(ctx, userID, analysisResult)
			if err != nil {
				fmt.Printf("[Holmes 缓存] 失败 user=%s error=%v\n", userID, err)
			} else {
				fmt.Printf("[Holmes 缓存] 成功 user=%s status=%s probability=%d\n",
					userID, analysisResult.Availability.Status, analysisResult.Availability.Probability)
			}
		}()
	}

	// 发送最终结果
	doneEvent := map[string]interface{}{
		"type":   "done",
		"result": result,
	}
	data, _ := json.Marshal(doneEvent)
	fmt.Fprintf(w, "data: %s\n\n", data)
	w.Flush()
}

// ReportStatus2Stream Holmes 2.0 流式上报状态（SSE 实时输出叙事推理过程）
// POST /api/agent/status/stream2
func (h *AgentHandler) ReportStatus2Stream(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var req model.ExtendedStatusReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer

	// 使用独立 context，不受客户端断连影响
	sse2Ctx, sse2Cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer sse2Cancel()

	// 流式执行 Holmes 2.0 推理
	result, err := h.agentService.ReportExtendedStatus2Stream(sse2Ctx, userID, &req, func(event interface{}) {
		data, err := json.Marshal(event)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	})

	if err != nil {
		errEvent := map[string]string{
			"type":    "error",
			"content": err.Error(),
		}
		data, _ := json.Marshal(errEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
		return
	}

	// 缓存分析结果
	if result != nil && h.memoryService != nil {
		// 从 Holmes2Result 转换为 AnalysisResult
		analysisResult := &model.AnalysisResult{
			Availability: model.AvailabilityAnalysis{
				Status:      getStatusFromProbability(result.Result.Probability),
				Probability: result.Result.Probability,
				Reason:      result.Result.Summary,
				Confidence:  result.Result.Confidence,
			},
			LifeStatus: model.LifeStatus{
				Emoji:       result.Result.Emoji,
				Label:       result.Result.Summary,
				Description: result.Creative.Narrative,
			},
			UpdatedAt: time.Now(),
		}

		// 添加心情信息
		if result.Creative != nil && result.Creative.Mood != nil {
			mood := result.Creative.Mood
			if mood.Valence > 0.5 {
				analysisResult.Mood = "积极"
			} else if mood.Valence > 0 {
				analysisResult.Mood = "平静"
			} else if mood.Valence > -0.5 {
				analysisResult.Mood = "低落"
			} else {
				analysisResult.Mood = "消极"
			}
		}

		// 异步缓存（不覆盖用户手动确认的状态）
		go func() {
			ctx := context.Background()
			existingCache, _ := h.memoryService.GetCachedAnalysisByUserIDs(ctx, []string{userID})
			if existing := existingCache[userID]; existing != nil && !existing.IsAIGuess {
				fmt.Printf("[Holmes 2.0 缓存] 跳过：用户已手动确认状态 user=%s\n", userID)
				return
			}
			err := h.memoryService.CacheAnalysisResult(ctx, userID, analysisResult)
			if err != nil {
				fmt.Printf("[Holmes 2.0 缓存] 失败 user=%s error=%v\n", userID, err)
			} else {
				fmt.Printf("[Holmes 2.0 缓存] 成功 user=%s scene=%s\n",
					userID, result.Creative.Scene)
			}
		}()
	}

	// 发送最终结果
	doneEvent := map[string]interface{}{
		"type":   "done",
		"result": result,
	}
	data, _ := json.Marshal(doneEvent)
	fmt.Fprintf(w, "data: %s\n\n", data)
	w.Flush()
}

// getStatusFromProbability 根据概率确定状态文本
func getStatusFromProbability(probability int) string {
	if probability >= 70 {
		return "有空"
	} else if probability >= 40 {
		return "可能有空"
	} else {
		return "忙碌"
	}
}

// GenerateStatusOptionsStream 流式生成状态选项（SSE）
// POST /api/agent/status-options
func (h *AgentHandler) GenerateStatusOptionsStream(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var req model.ExtendedStatusReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer

	// 使用独立 context，不受客户端断连影响
	optCtx, optCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer optCancel()

	// 获取用户最近的状态记忆（用于推理）
	var recentMemory []*model.UserStatusMemory
	if h.memoryService != nil {
		recentMemory, _ = h.memoryService.GetRecentUserStatusMemory(optCtx, userID, 10)
	}

	// 流式执行状态选项生成
	result, err := h.agentService.GenerateStatusOptionsStream(optCtx, userID, &req, recentMemory, func(event interface{}) {
		data, err := json.Marshal(event)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	})

	if err != nil {
		errEvent := map[string]string{
			"type":    "error",
			"content": err.Error(),
		}
		data, _ := json.Marshal(errEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
		return
	}

	// 发送状态选项
	if result != nil {
		optionsEvent := map[string]interface{}{
			"type": "options",
			"data": result,
		}
		data, _ := json.Marshal(optionsEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	}

	// 发送完成信号
	fmt.Fprintf(w, "data: [DONE]\n\n")
	w.Flush()
}

// GetOnboardingStatusOptions 获取引导流程的状态选项
// POST /api/agent/onboarding-status-options
func (h *AgentHandler) GetOnboardingStatusOptions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req struct {
		ProfileType string `json:"profile_type" binding:"required"`
		City        string `json:"city,omitempty"` // 城市名称（如"上海"、"北京"）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误: profile_type 必填")
		return
	}

	// 验证 profile_type
	validTypes := map[string]bool{
		"office_worker": true,
		"student":       true,
		"freelancer":    true,
		"entrepreneur":  true,
		"investor":      true,
		"parent":        true,
		"retired":       true,
	}
	if !validTypes[req.ProfileType] {
		response.ParamError(c, "无效的 profile_type")
		return
	}

	result, err := h.agentService.GetOnboardingStatusOptions(c.Request.Context(), req.ProfileType, req.City)
	if err != nil {
		fmt.Printf("[引导选项] 获取失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "获取状态选项失败")
		return
	}

	response.Success(c, result)
}

// SelectStatus 选择状态并记录
// POST /api/agent/select-status
func (h *AgentHandler) SelectStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req model.SelectStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	if h.memoryService == nil {
		response.InternalError(c, "记忆服务未启用")
		return
	}

	// 选择状态并记录到记忆
	err := h.memoryService.SelectStatus(c.Request.Context(), userID, &req)
	if err != nil {
		fmt.Printf("[选择状态] 失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "保存失败")
		return
	}

	// V3 推断结果智能合并到已有 schedule：
	// - 截断包含当前时刻的 V4 item（endTime → now）
	// - V3 item 的 endTime = 下一个 item 的 startTime（或 23:59）
	// - 不新建记录，避免同天多条冲突
	if h.scheduleRepo != nil {
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		currentTime := now.Format("15:04")

		existing, _ := h.scheduleRepo.GetLatestByUserAndDate(c.Request.Context(), userID, today)
		if existing != nil {
			// 1. 截断包含当前时刻的 item
			for i := range existing.Items {
				it := &existing.Items[i]
				if it.StartTime <= currentTime && currentTime < it.EndTime {
					fmt.Printf("[选择状态] 截断 V4 item %s-%s → %s-%s\n",
						it.StartTime, it.EndTime, it.StartTime, currentTime)
					it.EndTime = currentTime
					break
				}
			}

			// 2. 找 now 之后最近 item 的 startTime 作为 V3 的 endTime
			//    默认 2 小时，不超过 23:59，也不超过下一个 item
			v3EndTime := addTimeStr(currentTime, 120)
			for _, it := range existing.Items {
				if it.StartTime > currentTime {
					if it.StartTime < v3EndTime {
						v3EndTime = it.StartTime
					}
					break
				}
			}

			// 3. 追加 V3 item
			v3Item := model.ScheduleItem{
				StartTime: currentTime,
				EndTime:   v3EndTime,
				Emoji:     req.Emoji,
				Status:    req.Status,
				Executed:  true,
			}
			existing.Items = append(existing.Items, v3Item)

			if updateErr := h.scheduleRepo.Update(c.Request.Context(), existing); updateErr != nil {
				fmt.Printf("[选择状态] 更新时刻表失败 user=%s error=%v\n", userID, updateErr)
			} else {
				fmt.Printf("[选择状态] 时刻表已更新 user=%s V3 item %s-%s\n", userID, currentTime, v3EndTime)
			}
		} else {
			// 无已有 schedule，创建新记录（默认 2 小时）
			schedule := &model.StatusSchedule{
				UserID:       userID,
				ScheduleDate: now,
				Items: model.ScheduleItems{{
					StartTime: currentTime,
					EndTime:   addTimeStr(currentTime, 120),
					Emoji:     req.Emoji,
					Status:    req.Status,
					Executed:  true,
				}},
				CurrentIndex: 0,
				Status:       model.ScheduleStatusActive,
			}
			if createErr := h.scheduleRepo.Create(c.Request.Context(), schedule); createErr != nil {
				fmt.Printf("[选择状态] 创建时刻表失败 user=%s error=%v\n", userID, createErr)
			} else {
				fmt.Printf("[选择状态] 时刻表已创建 user=%s schedule_id=%d\n", userID, schedule.ID)
			}
		}
	}

	fmt.Printf("[选择状态] 成功 user=%s emoji=%s status=%s\n", userID, req.Emoji, req.Status)

	response.Success(c, gin.H{
		"success": true,
		"message": "状态已更新",
	})
}

// VoiceScheduleStream 语音状态时刻表 SSE 流（V4 版本）
// POST /api/agent/voice-schedule/stream
// Content-Type: multipart/form-data
func (h *AgentHandler) VoiceScheduleStream(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	// 优先使用 V4 服务
	if h.voiceScheduleServiceV4 == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "语音服务未启用"})
		return
	}

	// 获取上传的音频文件
	file, err := c.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传音频文件"})
		return
	}

	// 读取音频数据
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}
	defer f.Close()

	audioData := make([]byte, file.Size)
	if _, err := f.Read(audioData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

	// 获取音频格式
	audioFormat := "m4a"
	if ct := file.Header.Get("Content-Type"); ct != "" {
		if ct == "audio/wav" || ct == "audio/wave" {
			audioFormat = "wav"
		} else if ct == "audio/mpeg" || ct == "audio/mp3" {
			audioFormat = "mp3"
		}
	}

	// 获取可选的 session_id（用于多轮对话保持上下文）
	sessionID := c.PostForm("session_id")

	fmt.Printf("[VoiceScheduleV4] 收到音频 user=%s size=%d format=%s session=%s\n", userID, len(audioData), audioFormat, sessionID)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer

	// defer 保证 [DONE] 必达（无论成功/失败/panic）
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[VoiceScheduleV4] panic recovered: %v\n", r)
			errEvent := model.V4Event{Type: model.V4EventTypeError, Message: "服务异常，请重试"}
			data, _ := json.Marshal(errEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		w.Flush()
	}()

	// 使用独立 context，不受客户端断连影响
	// V4 流程包含 ASR + LLM + 工具执行，即使客户端断开也应完成操作并保存状态
	v4Ctx, v4Cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer v4Cancel()

	// 使用 V4 服务处理语音输入
	_, err = h.voiceScheduleServiceV4.ProcessVoiceInput(v4Ctx, userID, sessionID, audioData, audioFormat, func(event *model.V4Event) {
		data, err := json.Marshal(event)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	})

	if err != nil {
		errEvent := model.V4Event{
			Type:    model.V4EventTypeError,
			Message: err.Error(),
		}
		data, _ := json.Marshal(errEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	}
}

// VoiceScheduleInteract 语音时刻表后续交互
// POST /api/agent/voice-schedule/interact
// 支持 JSON 和 multipart/form-data
func (h *AgentHandler) VoiceScheduleInteract(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	if h.voiceScheduleService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "语音服务未启用"})
		return
	}

	var req model.VoiceScheduleInteractionRequest
	var audioData []byte

	contentType := c.ContentType()
	if contentType == "application/json" {
		// JSON 格式
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
			return
		}
	} else {
		// multipart/form-data 格式
		req.SessionID = c.PostForm("session_id")
		req.Action = model.VoiceScheduleAction(c.PostForm("action"))

		// 尝试获取音频文件
		file, err := c.FormFile("audio")
		if err == nil {
			f, err := file.Open()
			if err == nil {
				audioData = make([]byte, file.Size)
				f.Read(audioData)
				f.Close()
			}
		}
	}

	if req.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id 必填"})
		return
	}

	fmt.Printf("[VoiceSchedule] 交互 user=%s session=%s action=%s\n", userID, req.SessionID, req.Action)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer

	// defer 保证 [DONE] 必达
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[VoiceScheduleInteract] panic recovered: %v\n", r)
			errEvent := model.VoiceScheduleEvent{Type: model.VSEventError, Message: "服务异常，请重试"}
			data, _ := json.Marshal(errEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		w.Flush()
	}()

	// 处理交互
	err := h.voiceScheduleService.ProcessInteraction(c.Request.Context(), userID, &req, audioData, func(event *model.VoiceScheduleEvent) {
		data, err := json.Marshal(event)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	})

	if err != nil {
		errEvent := model.VoiceScheduleEvent{
			Type:    model.VSEventError,
			Message: err.Error(),
		}
		data, _ := json.Marshal(errEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	}
}

// VoiceScheduleText 语音时刻表文本测试接口（V4 版本，跳过语音识别）
// POST /api/agent/voice-schedule/text
func (h *AgentHandler) VoiceScheduleText(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	// 优先使用 V4 服务
	if h.voiceScheduleServiceV4 == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "语音服务未启用"})
		return
	}

	var req struct {
		SessionID  string                          `json:"session_id"`  // 可选，用于继续会话
		Text       string                          `json:"text" binding:"required"`
		SensorData *model.ExtendedStatusReportRequest `json:"sensor_data"` // 可选，一键生成时附带传感器数据
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: text 必填"})
		return
	}

	fmt.Printf("[VoiceScheduleTextV4] user=%s session=%s text=%s hasSensor=%v\n", userID, req.SessionID, req.Text, req.SensorData != nil)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer

	// defer 保证 [DONE] 必达
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[VoiceScheduleTextV4] panic recovered: %v\n", r)
			errEvent := model.V4Event{Type: model.V4EventTypeError, Message: "服务异常，请重试"}
			data, _ := json.Marshal(errEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		w.Flush()
	}()

	// 使用独立 context，不受客户端断连影响
	textCtx, textCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer textCancel()

	// 使用 V4 服务处理文本输入
	_, err := h.voiceScheduleServiceV4.ProcessTextInput(textCtx, userID, req.SessionID, req.Text, req.SensorData, func(event *model.V4Event) {
		data, err := json.Marshal(event)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	})

	if err != nil {
		errEvent := model.V4Event{
			Type:    model.V4EventTypeError,
			Message: err.Error(),
		}
		data, _ := json.Marshal(errEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	}
}

// VoiceScheduleTextV4 V4 版本语音时刻表文本接口
// POST /api/agent/voice-schedule/v4/text
// 新架构：简化的 while 循环 + 工具调用，不使用状态机
func (h *AgentHandler) VoiceScheduleTextV4(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	if h.voiceScheduleServiceV4 == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "V4 语音服务未启用"})
		return
	}

	var req struct {
		SessionID  string                          `json:"session_id"`  // 可选，用于继续会话
		Text       string                          `json:"text" binding:"required"`
		SensorData *model.ExtendedStatusReportRequest `json:"sensor_data"` // 可选，一键生成时附带传感器数据
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: text 必填"})
		return
	}

	fmt.Printf("[VoiceScheduleTextV4] user=%s session=%s text=%s hasSensor=%v\n", userID, req.SessionID, req.Text, req.SensorData != nil)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer

	// defer 保证 [DONE] 必达
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[VoiceScheduleTextV4-v4] panic recovered: %v\n", r)
			errEvent := model.V4Event{Type: model.V4EventTypeError, Message: "服务异常，请重试"}
			data, _ := json.Marshal(errEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		w.Flush()
	}()

	// 使用独立 context，不受客户端断连影响
	v4TextCtx, v4TextCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer v4TextCancel()

	// 使用 V4 服务处理
	_, err := h.voiceScheduleServiceV4.ProcessTextInput(v4TextCtx, userID, req.SessionID, req.Text, req.SensorData, func(event *model.V4Event) {
		data, err := json.Marshal(event)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	})

	if err != nil {
		errEvent := model.V4Event{
			Type:    model.V4EventTypeError,
			Message: err.Error(),
		}
		data, _ := json.Marshal(errEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	}
}

// AgentChatStream Agent 聊天流式接口（Tool Use Loop）
// POST /api/agent/chat/stream
func (h *AgentHandler) AgentChatStream(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	if h.agentChatService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Agent Chat 服务未启用"})
		return
	}

	var req service.AgentChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: message 必填"})
		return
	}

	fmt.Printf("[AgentChatStream] user=%s session=%s message=%s\n", userID, req.SessionID, req.Message)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer

	// 流式执行 Agent Chat
	_, err := h.agentChatService.Chat(c.Request.Context(), userID, &req, func(event *agent.AgentStreamEvent) {
		data, err := event.ToJSON()
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	})

	if err != nil {
		errEvent := agent.NewErrorEvent(err.Error())
		data, _ := errEvent.ToJSON()
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
		return
	}
}

// AgentChat Agent 聊天非流式接口
// POST /api/agent/chat
func (h *AgentHandler) AgentChat(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if h.agentChatService == nil {
		response.InternalError(c, "Agent Chat 服务未启用")
		return
	}

	var req service.AgentChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误: message 必填")
		return
	}

	fmt.Printf("[AgentChat] user=%s session=%s message=%s\n", userID, req.SessionID, req.Message)

	result, err := h.agentChatService.Chat(c.Request.Context(), userID, &req, nil)
	if err != nil {
		fmt.Printf("[AgentChat] error: %v\n", err)
		response.InternalError(c, "处理失败")
		return
	}

	response.Success(c, result)
}

// GetMySchedule 获取指定日期的时刻表
// GET /api/agent/my-schedule/:date
func (h *AgentHandler) GetMySchedule(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if h.scheduleRepo == nil {
		response.InternalError(c, "时刻表服务未启用")
		return
	}

	// 解析日期参数
	dateStr := c.Param("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		response.Error(c, 400, "日期格式无效，应为 YYYY-MM-DD")
		return
	}

	// 获取时刻表
	schedule, err := h.scheduleRepo.GetActiveByUserAndDate(c.Request.Context(), userID, date)
	if err != nil {
		fmt.Printf("[GetMySchedule] 获取失败 user=%s date=%s error=%v\n", userID, dateStr, err)
		response.InternalError(c, "获取失败")
		return
	}

	if schedule == nil {
		response.Success(c, gin.H{
			"date":  dateStr,
			"items": []interface{}{},
		})
		return
	}

	response.Success(c, gin.H{
		"date":  dateStr,
		"items": schedule.Items,
	})
}

// SetMySchedule 设置指定日期的时刻表（直接替换，用于测试）
// PUT /api/agent/my-schedule/:date
func (h *AgentHandler) SetMySchedule(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if h.scheduleRepo == nil {
		response.InternalError(c, "时刻表服务未启用")
		return
	}

	// 解析日期参数
	dateStr := c.Param("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		response.Error(c, 400, "日期格式无效，应为 YYYY-MM-DD")
		return
	}

	// 解析请求体
	var req struct {
		Items []model.ScheduleItem `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求体格式无效")
		return
	}

	// 获取现有时刻表或创建新的
	schedule, err := h.scheduleRepo.GetActiveByUserAndDate(c.Request.Context(), userID, date)
	if err != nil {
		fmt.Printf("[SetMySchedule] 获取现有时刻表失败 user=%s date=%s error=%v\n", userID, dateStr, err)
		response.InternalError(c, "获取失败")
		return
	}

	if schedule == nil {
		// 创建新时刻表
		schedule = &model.StatusSchedule{
			UserID:       userID,
			ScheduleDate: date,
			Items:        req.Items,
			Status:       model.ScheduleStatusActive,
			Visibility:   model.VisibilityPrivate,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := h.scheduleRepo.Create(c.Request.Context(), schedule); err != nil {
			fmt.Printf("[SetMySchedule] 创建时刻表失败 user=%s date=%s error=%v\n", userID, dateStr, err)
			response.InternalError(c, "创建失败")
			return
		}
	} else {
		// 更新现有时刻表
		schedule.Items = req.Items
		schedule.UpdatedAt = time.Now()
		if err := h.scheduleRepo.Update(c.Request.Context(), schedule); err != nil {
			fmt.Printf("[SetMySchedule] 更新时刻表失败 user=%s date=%s error=%v\n", userID, dateStr, err)
			response.InternalError(c, "更新失败")
			return
		}
	}

	fmt.Printf("[SetMySchedule] 成功设置时刻表 user=%s date=%s items=%d\n", userID, dateStr, len(req.Items))
	response.Success(c, gin.H{
		"date":  dateStr,
		"items": schedule.Items,
	})
}

// GetMyScheduleHistory 获取我的状态时刻表历史（分页）
// GET /api/agent/my-schedule/history
func (h *AgentHandler) GetMyScheduleHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if h.scheduleRepo == nil {
		response.InternalError(c, "时刻表服务未启用")
		return
	}

	// 解析查询参数
	limitStr := c.DefaultQuery("limit", "20")
	beforeDate := c.Query("before_date") // YYYY-MM-DD 格式，可选

	limit := 20
	if l, err := parseLimit(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	// 获取时刻表历史
	schedules, err := h.scheduleRepo.GetUserScheduleHistory(c.Request.Context(), userID, beforeDate, limit+1)
	if err != nil {
		fmt.Printf("[GetMyScheduleHistory] 获取失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "获取失败")
		return
	}

	// 判断是否有更多数据
	hasMore := len(schedules) > limit
	if hasMore {
		schedules = schedules[:limit]
	}

	// 获取最早的时刻表日期
	oldestDate, _ := h.scheduleRepo.GetUserOldestScheduleDate(c.Request.Context(), userID)

	// 按日期分组并格式化响应
	daySchedules := formatSchedulesByDate(schedules)

	response.Success(c, gin.H{
		"schedules":   daySchedules,
		"has_more":    hasMore,
		"oldest_date": oldestDate,
	})
}

// parseLimit 解析 limit 参数
func parseLimit(s string) (int, error) {
	var limit int
	_, err := fmt.Sscanf(s, "%d", &limit)
	return limit, err
}

// GetUserSchedule 获取好友的时刻表（只返回当前和未来的条目）
// GET /api/v1/agent/user/:userId/schedule
func (h *AgentHandler) GetUserSchedule(c *gin.Context) {
	viewerID := middleware.GetUserID(c)
	if viewerID == "" {
		response.Unauthorized(c)
		return
	}

	if h.scheduleRepo == nil {
		response.InternalError(c, "时刻表服务未启用")
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		response.ParamError(c, "用户 ID 不能为空")
		return
	}

	// 检查好友关系
	if h.friendshipRepo != nil {
		areFriends, err := h.friendshipRepo.AreFriends(c.Request.Context(), viewerID, targetUserID)
		if err != nil {
			fmt.Printf("[GetUserSchedule] 检查好友关系失败 viewer=%s target=%s error=%v\n", viewerID, targetUserID, err)
			response.InternalError(c, "检查好友关系失败")
			return
		}
		if !areFriends {
			response.Error(c, 403, "只能查看好友的行程表")
			return
		}
	}

	// 获取最近的时刻表数据
	schedules, err := h.scheduleRepo.GetUserScheduleHistory(c.Request.Context(), targetUserID, "", 10)
	if err != nil {
		fmt.Printf("[GetUserSchedule] 获取失败 target=%s error=%v\n", targetUserID, err)
		response.InternalError(c, "获取失败")
		return
	}

	// 过滤：只保留今天及未来的时刻表，今天的条目过滤掉已结束的
	now := time.Now()
	todayStr := now.Format("2006-01-02")
	currentTime := now.Format("15:04")

	var filtered []*model.StatusSchedule
	for _, s := range schedules {
		dateStr := s.ScheduleDate.Format("2006-01-02")

		if dateStr < todayStr {
			// 过去的日期，跳过
			continue
		}

		if dateStr == todayStr {
			// 今天：过滤掉已结束的条目
			var activeItems model.ScheduleItems
			for _, item := range s.Items {
				// 跨午夜时段（end < start）不会结束于今天，保留
				if item.EndTime < item.StartTime {
					activeItems = append(activeItems, item)
				} else if item.EndTime > currentTime {
					activeItems = append(activeItems, item)
				}
			}
			if len(activeItems) > 0 {
				filteredSchedule := *s
				filteredSchedule.Items = activeItems
				filtered = append(filtered, &filteredSchedule)
			}
		} else {
			// 未来的日期，全部保留
			filtered = append(filtered, s)
		}
	}

	// 可见性过滤：Booking 条目根据查看者身份过滤
	if h.bookingService != nil && viewerID != targetUserID {
		for _, s := range filtered {
			s.Items = model.ScheduleItems(
				h.bookingService.FilterScheduleForViewer(c.Request.Context(), []model.ScheduleItem(s.Items), viewerID),
			)
		}
	}

	// 格式化响应
	daySchedules := formatSchedulesByDate(filtered)

	response.Success(c, gin.H{
		"schedules":   daySchedules,
		"has_more":    false,
		"oldest_date": "",
	})
}

// UpdateScheduleItemRequest 更新时刻表条目请求
type UpdateScheduleItemRequest struct {
	OldStartTime string `json:"old_start_time" binding:"required"`
	OldEndTime   string `json:"old_end_time" binding:"required"`
	NewStartTime string `json:"new_start_time" binding:"required"`
	NewEndTime   string `json:"new_end_time" binding:"required"`
	Emoji        string `json:"emoji" binding:"required"`
	Status       string `json:"status" binding:"required"`
	Highlight    *bool `json:"highlight,omitempty"`      // 高亮状态（有空），nil 表示不修改
	RemindBefore *int  `json:"remind_before,omitempty"` // 提前提醒分钟数，nil=不修改，0=取消，>0=设置
}

// UpdateScheduleItem 更新时刻表条目
// PUT /api/agent/my-schedule/:date/item
func (h *AgentHandler) UpdateScheduleItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if h.scheduleRepo == nil {
		response.InternalError(c, "时刻表服务未启用")
		return
	}

	date := c.Param("date")
	if date == "" {
		response.ParamError(c, "日期不能为空")
		return
	}

	var req UpdateScheduleItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("[UpdateScheduleItem] 参数绑定失败: %v\n", err)
		response.ParamError(c, "参数错误")
		return
	}

	// 解析日期
	scheduleDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		response.ParamError(c, "日期格式错误，应为 YYYY-MM-DD")
		return
	}

	// 获取当天的时刻表（不限 status，允许编辑已完成的时刻表）
	schedule, err := h.scheduleRepo.GetLatestByUserAndDate(c.Request.Context(), userID, scheduleDate)
	if err != nil || schedule == nil {
		response.NotFound(c, "未找到该日期的时刻表")
		return
	}

	// 查找并更新条目
	found := false
	foundIdx := -1
	for i, item := range schedule.Items {
		if item.StartTime == req.OldStartTime && item.EndTime == req.OldEndTime {
			highlight := item.Highlight // 默认保留原值
			if req.Highlight != nil {
				highlight = *req.Highlight
			}
			remindBefore := item.RemindBefore // 默认保留原值
			if req.RemindBefore != nil {
				remindBefore = *req.RemindBefore
			}
			// 仅当 emoji 或 status 实际变更时才清除 AI 标记
			isAIGuess := item.IsAIGuess
			if req.Emoji != item.Emoji || req.Status != item.Status {
				isAIGuess = false
			}
			schedule.Items[i] = model.ScheduleItem{
				StartTime:    req.NewStartTime,
				EndTime:      req.NewEndTime,
				Emoji:        req.Emoji,
				Status:       req.Status,
				Executed:     item.Executed,
				IsAIGuess:    isAIGuess,
				GifURL:       item.GifURL,
				GiphyQuery:   item.GiphyQuery,
				Highlight:    highlight,
				RemindBefore: remindBefore,
			}
			found = true
			foundIdx = i
			break
		}
	}

	if !found {
		response.NotFound(c, "未找到该时段")
		return
	}

	// 检测时间冲突（仅在时间实际变更时检查，避免跨午夜等既有时段的虚假冲突）
	timeChanged := req.NewStartTime != req.OldStartTime || req.NewEndTime != req.OldEndTime
	if timeChanged {
		for j, other := range schedule.Items {
			if j == foundIdx {
				continue
			}
			if timesOverlap(req.NewStartTime, req.NewEndTime, other.StartTime, other.EndTime) {
				response.ParamError(c, fmt.Sprintf("与 %s-%s %s 时间冲突", other.StartTime, other.EndTime, other.Status))
				return
			}
		}
	}

	// 按 StartTime 重新排序（编辑时间后顺序可能错乱，影响首页 overlay 查找）
	sort.Slice(schedule.Items, func(i, j int) bool {
		return schedule.Items[i].StartTime < schedule.Items[j].StartTime
	})

	// 保存更新
	if err := h.scheduleRepo.Update(c.Request.Context(), schedule); err != nil {
		fmt.Printf("[UpdateScheduleItem] 保存失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "保存失败")
		return
	}

	fmt.Printf("[UpdateScheduleItem] 成功 user=%s date=%s slot=%s-%s\n",
		userID, date, req.NewStartTime, req.NewEndTime)

	response.Success(c, gin.H{
		"success": true,
		"message": "更新成功",
	})
}

// DeleteScheduleItem 删除时刻表条目
// DELETE /api/agent/my-schedule/:date/item
func (h *AgentHandler) DeleteScheduleItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if h.scheduleRepo == nil {
		response.InternalError(c, "时刻表服务未启用")
		return
	}

	date := c.Param("date")
	if date == "" {
		response.ParamError(c, "日期不能为空")
		return
	}

	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	if startTime == "" || endTime == "" {
		response.ParamError(c, "start_time 和 end_time 必填")
		return
	}

	// 解析日期
	scheduleDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		response.ParamError(c, "日期格式错误，应为 YYYY-MM-DD")
		return
	}

	// 获取当天所有非取消的时刻表（同一天可能有多条记录）
	schedules, err := h.scheduleRepo.GetAllByUserAndDate(c.Request.Context(), userID, scheduleDate)
	if err != nil || len(schedules) == 0 {
		response.NotFound(c, "未找到该日期的时刻表")
		return
	}

	// 在所有 schedule 中查找并删除条目
	var targetSchedule *model.StatusSchedule
	for _, sch := range schedules {
		for _, item := range sch.Items {
			if item.StartTime == startTime && item.EndTime == endTime {
				targetSchedule = sch
				break
			}
		}
		if targetSchedule != nil {
			break
		}
	}

	if targetSchedule == nil {
		response.NotFound(c, "未找到该时段")
		return
	}

	newItems := make(model.ScheduleItems, 0, len(targetSchedule.Items))
	for _, item := range targetSchedule.Items {
		if item.StartTime == startTime && item.EndTime == endTime {
			continue
		}
		newItems = append(newItems, item)
	}
	targetSchedule.Items = newItems

	// 保存更新（如果没有条目了，标记为 cancelled）
	if len(newItems) == 0 {
		targetSchedule.Status = model.ScheduleStatusCancelled
	}
	if err := h.scheduleRepo.Update(c.Request.Context(), targetSchedule); err != nil {
		fmt.Printf("[DeleteScheduleItem] 保存失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "保存失败")
		return
	}

	fmt.Printf("[DeleteScheduleItem] 成功 user=%s date=%s slot=%s-%s\n",
		userID, date, startTime, endTime)

	response.Success(c, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

// timesOverlap 检测两个时段是否重叠（支持跨午夜）
func timesOverlap(s1, e1, s2, e2 string) bool {
	s1m := parseTimeMinutes(s1)
	e1m := parseTimeMinutes(e1)
	s2m := parseTimeMinutes(s2)
	e2m := parseTimeMinutes(e2)

	// 拆分为子区间（处理跨午夜）
	type interval struct{ start, end int }
	var ranges1, ranges2 []interval

	if e1m <= s1m {
		ranges1 = []interval{{s1m, 24 * 60}, {0, e1m}}
	} else {
		ranges1 = []interval{{s1m, e1m}}
	}

	if e2m <= s2m {
		ranges2 = []interval{{s2m, 24 * 60}, {0, e2m}}
	} else {
		ranges2 = []interval{{s2m, e2m}}
	}

	for _, r1 := range ranges1 {
		for _, r2 := range ranges2 {
			if r1.start < r2.end && r2.start < r1.end {
				return true
			}
		}
	}
	return false
}

// parseTimeMinutes 将 HH:MM 格式转换为分钟数
func parseTimeMinutes(t string) int {
	var h, m int
	fmt.Sscanf(t, "%d:%d", &h, &m)
	return h*60 + m
}

// DayScheduleResponse 按日期分组的时刻表响应
type DayScheduleResponse struct {
	ScheduleDate string               `json:"schedule_date"`
	Items        []model.ScheduleItem `json:"items"`
	Status       string               `json:"status"`
}

// formatSchedulesByDate 将时刻表按日期分组
func formatSchedulesByDate(schedules []*model.StatusSchedule) []DayScheduleResponse {
	// 使用 map 按日期分组，保留每天最新的时刻表
	dateMap := make(map[string]*model.StatusSchedule)
	dateOrder := make([]string, 0)

	for _, s := range schedules {
		dateStr := s.ScheduleDate.Format("2006-01-02")
		if _, exists := dateMap[dateStr]; !exists {
			dateMap[dateStr] = s
			dateOrder = append(dateOrder, dateStr)
		}
	}

	// 按日期顺序构建响应，items 按 start_time 排序
	result := make([]DayScheduleResponse, 0, len(dateOrder))
	for _, dateStr := range dateOrder {
		s := dateMap[dateStr]
		items := make([]model.ScheduleItem, len(s.Items))
		copy(items, s.Items)
		sort.Slice(items, func(i, j int) bool {
			return items[i].StartTime < items[j].StartTime
		})
		result = append(result, DayScheduleResponse{
			ScheduleDate: dateStr,
			Items:        items,
			Status:       string(s.Status),
		})
	}

	return result
}

// AgentVoiceChatStream Agent 语音聊天流式接口（语音转文字 + Tool Use Loop）
// POST /api/agent/voice/stream
func (h *AgentHandler) AgentVoiceChatStream(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	if h.agentChatService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Agent Chat 服务未启用"})
		return
	}

	// 解析表单数据
	sessionID := c.PostForm("session_id")
	audioFormat := c.PostForm("format")
	if audioFormat == "" {
		audioFormat = "m4a"
	}

	// 读取音频文件
	file, _, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少音频文件"})
		return
	}
	defer file.Close()

	audioData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取音频文件失败"})
		return
	}

	fmt.Printf("[AgentVoiceChatStream] user=%s session=%s format=%s audioSize=%d\n",
		userID, sessionID, audioFormat, len(audioData))

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer

	// 构建请求
	req := &service.AgentVoiceChatRequest{
		SessionID:   sessionID,
		AudioData:   audioData,
		AudioFormat: audioFormat,
	}

	// 流式执行语音聊天
	_, _, err = h.agentChatService.VoiceChat(c.Request.Context(), userID, req, func(event *agent.AgentStreamEvent) {
		data, err := event.ToJSON()
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	})

	if err != nil {
		errEvent := agent.NewErrorEvent(err.Error())
		data, _ := errEvent.ToJSON()
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
		return
	}
}

// ========== 模型测试接口 ==========

// modelTestService 模型测试服务实例
var modelTestService *service.ModelTestService

// SetModelTestService 设置模型测试服务
func (h *AgentHandler) SetModelTestService(svc *service.ModelTestService) {
	modelTestService = svc
}

// TestModelSingle 单个测试用例
// POST /api/agent/test/model/single
func (h *AgentHandler) TestModelSingle(c *gin.Context) {
	if modelTestService == nil {
		response.InternalError(c, "模型测试服务未启用")
		return
	}

	var req struct {
		CaseID   int    `json:"case_id" binding:"required"`
		Provider string `json:"provider" binding:"required"` // "qwen" 或 "kimi"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误: case_id 和 provider 必填")
		return
	}

	// 验证 provider
	var provider agent.LLMProvider
	switch req.Provider {
	case "qwen":
		provider = agent.ProviderQwen
	case "kimi":
		provider = agent.ProviderKimi
	default:
		response.ParamError(c, "无效的 provider，可选值: qwen, kimi")
		return
	}

	// 查找测试用例
	var testCase *service.ModelTestCase
	for _, tc := range modelTestService.GetTestCases() {
		if tc.ID == req.CaseID {
			testCase = &tc
			break
		}
	}
	if testCase == nil {
		response.ParamError(c, "测试用例不存在")
		return
	}

	fmt.Printf("[ModelTest] 运行单个测试 case=%d provider=%s input=%s\n",
		req.CaseID, req.Provider, testCase.Input)

	// 运行测试
	result, err := modelTestService.RunSingleTest(c.Request.Context(), *testCase, provider)
	if err != nil {
		response.InternalError(c, fmt.Sprintf("测试失败: %v", err))
		return
	}

	response.Success(c, result)
}

// TestModelComparison 运行完整模型对比测试
// POST /api/agent/test/model/comparison
func (h *AgentHandler) TestModelComparison(c *gin.Context) {
	if modelTestService == nil {
		response.InternalError(c, "模型测试服务未启用")
		return
	}

	fmt.Printf("[ModelTest] 开始完整模型对比测试...\n")

	// 这可能需要较长时间，设置较长的超时
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
	defer cancel()

	report, err := modelTestService.RunComparison(ctx)
	if err != nil {
		response.InternalError(c, fmt.Sprintf("测试失败: %v", err))
		return
	}

	fmt.Printf("[ModelTest] 测试完成，Qwen: %.1f, Kimi: %.1f\n",
		report.QwenSummary.OverallScore, report.KimiSummary.OverallScore)

	response.Success(c, report)
}

// TestModelReport 获取 Markdown 格式报告
// POST /api/agent/test/model/report
func (h *AgentHandler) TestModelReport(c *gin.Context) {
	if modelTestService == nil {
		response.InternalError(c, "模型测试服务未启用")
		return
	}

	var req struct {
		Report *service.ModelComparisonReport `json:"report" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误: report 必填")
		return
	}

	markdown := modelTestService.GenerateMarkdownReport(req.Report)

	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.String(http.StatusOK, markdown)
}

// GetTestCases 获取所有测试用例
// GET /api/agent/test/model/cases
func (h *AgentHandler) GetTestCases(c *gin.Context) {
	if modelTestService == nil {
		response.InternalError(c, "模型测试服务未启用")
		return
	}

	cases := modelTestService.GetTestCases()
	response.Success(c, gin.H{
		"total": len(cases),
		"cases": cases,
	})
}

// TestModelByCategory 按分类运行测试
// POST /api/agent/test/model/category
func (h *AgentHandler) TestModelByCategory(c *gin.Context) {
	if modelTestService == nil {
		response.InternalError(c, "模型测试服务未启用")
		return
	}

	var req struct {
		Category string `json:"category" binding:"required"`
		Provider string `json:"provider" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	var provider agent.LLMProvider
	switch req.Provider {
	case "qwen":
		provider = agent.ProviderQwen
	case "kimi":
		provider = agent.ProviderKimi
	default:
		response.ParamError(c, "无效的 provider")
		return
	}

	cases := modelTestService.GetTestCasesByCategory(req.Category)
	if len(cases) == 0 {
		response.ParamError(c, "该分类没有测试用例")
		return
	}

	fmt.Printf("[ModelTest] 按分类测试 category=%s provider=%s count=%d\n",
		req.Category, req.Provider, len(cases))

	var results []service.ModelTestResult
	for _, tc := range cases {
		result, err := modelTestService.RunSingleTest(c.Request.Context(), tc, provider)
		if err != nil {
			response.InternalError(c, fmt.Sprintf("测试失败: %v", err))
			return
		}
		results = append(results, *result)

		// 添加延迟避免速率限制
		time.Sleep(500 * time.Millisecond)
	}

	response.Success(c, gin.H{
		"category": req.Category,
		"provider": req.Provider,
		"results":  results,
	})
}

// ========== Persona 生成测试接口 ==========

// SetInferencePersonaService 设置 Persona 生成服务（测试用）
func (h *AgentHandler) SetInferencePersonaService(svc *service.InferencePersonaService, memRepo *repository.MemoryRepository) {
	h.inferencePersonaService = svc
	h.memoryRepo = memRepo
}

// TestPersonaGenerate 手动触发 persona 生成（测试端点）
// POST /api/v1/agent/test/persona-generate
func (h *AgentHandler) TestPersonaGenerate(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if h.inferencePersonaService == nil {
		response.InternalError(c, "Persona 服务未配置")
		return
	}

	// 1. 查看生成前的 core_memories 状态
	var beforePersona, beforeStats string
	if h.memoryRepo != nil {
		before, err := h.memoryRepo.GetCoreMemory(c.Request.Context(), userID)
		if err == nil && before != nil {
			beforePersona = before.PersonaText
			beforeStats = before.TimePatternStats
		}
	}

	// 2. 执行 persona 生成
	startTime := time.Now()
	err := h.inferencePersonaService.GeneratePersona(c.Request.Context(), userID)
	elapsed := time.Since(startTime)

	if err != nil {
		response.InternalError(c, fmt.Sprintf("Persona 生成失败: %v", err))
		return
	}

	// 3. 读取生成后的结果
	var afterPersona, afterStats string
	var generatedAt *time.Time
	if h.memoryRepo != nil {
		after, err := h.memoryRepo.GetCoreMemory(c.Request.Context(), userID)
		if err == nil && after != nil {
			afterPersona = after.PersonaText
			afterStats = after.TimePatternStats
			generatedAt = after.PersonaGeneratedAt
		}
	}

	// 4. 解析 stats JSON 为可读格式
	var statsObj interface{}
	if afterStats != "" {
		json.Unmarshal([]byte(afterStats), &statsObj)
	}

	// 5. 模拟推断注入效果：提取当前时段分布
	currentSlot := service.ExtractCurrentSlotDistribution(afterStats, time.Now())

	response.Success(c, gin.H{
		"user_id":     userID,
		"elapsed_ms":  elapsed.Milliseconds(),
		"before": gin.H{
			"persona_text":       beforePersona,
			"time_pattern_stats": beforeStats,
		},
		"after": gin.H{
			"persona_text":        afterPersona,
			"time_pattern_stats":  statsObj,
			"persona_generated_at": generatedAt,
		},
		"inference_injection": gin.H{
			"persona":              afterPersona,
			"current_slot_history": currentSlot,
			"note":                 "以上两个字段会在 V3 推断时注入到 LLM prompt 中",
		},
	})
}

// ========== 当下状态推理接口 ==========

// InferStatus AI 推断当下状态
// POST /api/v1/agent/infer-status
func (h *AgentHandler) InferStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	// 解析传感器数据（可选，如果前端没传就用最近缓存的）
	var req model.ExtendedStatusReportRequest
	c.ShouldBindJSON(&req)

	// 调用 AgentService 进行推理
	result, err := h.agentService.InferCurrentStatus(c.Request.Context(), userID, &req)
	if err != nil {
		fmt.Printf("[InferStatus] 推理失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "推理失败")
		return
	}

	fmt.Printf("[InferStatus] 推理成功 user=%s emoji=%s activity=%s is_available=%v\n",
		userID, result.Emoji, result.Activity, result.IsAvailable)

	// 推理成功后，直接更新首页缓存和时刻表
	// 这样用户不需要额外点击"确认发布"就能看到结果
	if h.agentService != nil {
		feedbackReq := &model.StatusFeedbackRequest{
			CorrectedEmoji:       result.Emoji,
			CorrectedActivity:    result.Activity,
			CorrectedPlace:       result.Place,
			CorrectedIsAvailable: &result.IsAvailable,
			GifURL:               result.GifURL,
			GiphyQuery:           result.GiphyQuery,
		}
		if err := h.agentService.SaveStatusFeedback(c.Request.Context(), userID, feedbackReq); err != nil {
			fmt.Printf("[InferStatus] 更新首页状态失败 user=%s error=%v\n", userID, err)
		} else {
			fmt.Printf("[InferStatus] 首页状态已更新 user=%s\n", userID)
		}
	}

	response.Success(c, result)
}

// StatusFeedback 状态反馈（用户修正状态，存入记忆）
// POST /api/v1/agent/status-feedback
func (h *AgentHandler) StatusFeedback(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req model.StatusFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	// 保存到记忆
	err := h.agentService.SaveStatusFeedback(c.Request.Context(), userID, &req)
	if err != nil {
		fmt.Printf("[StatusFeedback] 保存失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "保存失败")
		return
	}

	fmt.Printf("[StatusFeedback] 保存成功 user=%s corrected=%s %s\n",
		userID, req.CorrectedEmoji, req.CorrectedActivity)

	response.Success(c, gin.H{
		"success": true,
		"message": "反馈已记录",
	})
}

// UploadGif 客户端上传 GIF 文件，后端转存到 COS
// POST /api/v1/agent/upload-gif
func (h *AgentHandler) UploadGif(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	file, err := c.FormFile("gif")
	if err != nil {
		response.ParamError(c, "请上传 GIF 文件")
		return
	}

	// 限制文件大小 5MB
	if file.Size > 5*1024*1024 {
		response.ParamError(c, "GIF 文件不能超过 5MB")
		return
	}

	f, err := file.Open()
	if err != nil {
		response.InternalError(c, "读取文件失败")
		return
	}
	defer f.Close()

	// 上传到 COS
	cosURL, err := h.agentService.UploadGifToCOS(c.Request.Context(), f, file.Size)
	if err != nil {
		fmt.Printf("[UploadGif] 上传失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "上传失败")
		return
	}

	fmt.Printf("[UploadGif] 上传成功 user=%s size=%d url=%s\n", userID, file.Size, cosURL)

	response.Success(c, gin.H{
		"gif_url": cosURL,
	})
}

// GetSTSCredentials 获取 COS 临时上传凭证
// GET /api/v1/agent/sts
func (h *AgentHandler) GetSTSCredentials(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if h.stsSecretID == "" || h.stsBucket == "" {
		response.InternalError(c, "STS 未配置")
		return
	}

	prefix := "gif/status/"
	cred, err := tencent.GenerateSTSCredentials(h.stsSecretID, h.stsSecretKey, h.stsBucket, h.stsRegion, prefix)
	if err != nil {
		fmt.Printf("[STS] 生成临时凭证失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "获取上传凭证失败")
		return
	}

	response.Success(c, gin.H{"sts": cred})
}

// ========== V2 Agent-based 状态推断接口 ==========

// InferStatusV2 AI 推断当下状态（V4 全量）
// POST /api/v1/agent/infer-status-v2
func (h *AgentHandler) InferStatusV2(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req model.ExtendedStatusReportRequest
	c.ShouldBindJSON(&req)

	h.inferStatusV4Sync(c, userID, &req)
}

// inferStatusV4Sync V4 同步推断（一键生成，用户主动触发，不受推断锁限制）
func (h *AgentHandler) inferStatusV4Sync(c *gin.Context, userID string, req *model.ExtendedStatusReportRequest) {
	v4Ctx, v4Cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer v4Cancel()

	// 聚合上下文后直接调用 InferStatus（绕过 InferWithAgent 的推断锁）
	inferenceContext := h.voiceScheduleServiceV4.GatherInferenceContext(v4Ctx, userID)
	inferResp, err := h.voiceScheduleServiceV4.InferStatus(v4Ctx, userID, req, inferenceContext, nil)
	if err != nil {
		fmt.Printf("[InferStatusV4] 推断失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "推断失败")
		return
	}

	if inferResp != nil && inferResp.Result != nil {
		fmt.Printf("[InferStatusV4] 推断成功 user=%s emoji=%s activity=%s\n",
			userID, inferResp.Result.Emoji, inferResp.Result.Activity)
	}

	response.Success(c, inferResp)
}

// InferStatusV2Stream Agent-based AI 推断当下状态（SSE 流式版，支持 V3/V4 灰度切换）
// POST /api/v1/agent/infer-status-v2/stream
func (h *AgentHandler) InferStatusV2Stream(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var req model.ExtendedStatusReportRequest
	c.ShouldBindJSON(&req)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer

	// 启动心跳保活，防止 iOS URLSession 认为流已结束
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(1500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Fprintf(w, ":keepalive\n\n")
				w.Flush()
			}
		}
	}()

	// 全量走 V4 流式推断
	h.inferStatusV4Stream(c, w, userID, &req)
}

// inferStatusV4Stream V4 流式推断（SSE，将 V4Event 转换为 InferenceStreamEvent 格式）
func (h *AgentHandler) inferStatusV4Stream(_ *gin.Context, w http.ResponseWriter, userID string, req *model.ExtendedStatusReportRequest) {
	flusher, _ := w.(http.Flusher)
	flush := func() {
		if flusher != nil {
			flusher.Flush()
		}
	}

	// defer 保证 [DONE] 必达
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[InferStatusV4Stream] panic recovered: %v\n", r)
			errEvent := inferenceStreamEvent{Type: "error", Message: "服务异常，请重试"}
			data, _ := json.Marshal(errEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flush()
	}()

	// 聚合推断上下文
	inferenceContext := h.voiceScheduleServiceV4.GatherInferenceContext(context.Background(), userID)

	v4Ctx, v4Cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer v4Cancel()

	// V4 推断，将 V4Event 转为 InferenceStreamEvent
	inferResp, err := h.voiceScheduleServiceV4.InferStatus(v4Ctx, userID, req, inferenceContext, func(event *model.V4Event) {
		var streamEvent *inferenceStreamEvent
		switch event.Type {
		case model.V4EventTypePhase:
			streamEvent = &inferenceStreamEvent{
				Type:    "phase",
				Message: event.Message,
			}
		case model.V4EventTypeStatusUpdated:
			streamEvent = &inferenceStreamEvent{
				Type: "result",
				Data: map[string]interface{}{
					"result": &model.CurrentStatusInference{
						Emoji:    event.Emoji,
						Activity: event.Status,
					},
				},
				Message: event.Message,
			}
		case model.V4EventTypeToolStart:
			streamEvent = &inferenceStreamEvent{
				Type:    "tool",
				Message: "正在推断状态...",
			}
		default:
			return
		}
		if streamEvent != nil {
			data, _ := json.Marshal(streamEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flush()
		}
	})

	if err != nil {
		errEvent := inferenceStreamEvent{Type: "error", Message: err.Error()}
		data, _ := json.Marshal(errEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flush()
		return
	}

	// 推断结果不自动入库，等用户点击"确认发布"后通过 status-feedback 接口入库
	if inferResp != nil && inferResp.Result != nil {
		// 发送最终结果事件（包含 GIF 信息，供客户端展示）
		resultEvent := inferenceStreamEvent{
			Type: "result",
			Data: map[string]interface{}{
				"result": inferResp.Result,
			},
			Message: "推断完成，等待确认发布",
		}
		data, _ := json.Marshal(resultEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flush()
	}
}

// InferStatusRespond 接收用户选择（V4 已废弃此流程，保留接口兼容客户端）
// POST /api/v1/agent/infer-status-v2/respond
func (h *AgentHandler) InferStatusRespond(c *gin.Context) {
	// V4 推断始终返回 Phase=completed，不再有 awaiting_choice 流程
	// 保留此接口避免旧版客户端调用时崩溃
	fmt.Printf("[InferStatusRespond] 已废弃（V4 无 awaiting_choice 流程）\n")
	response.Success(c, gin.H{
		"message": "V4 推断不需要用户选择，此接口已废弃",
	})
}

// ========== V3/V4 推断 A/B 对比测试 ==========

// InferenceABResult 单次 A/B 对比结果
type InferenceABResult struct {
	Version    string                         `json:"version"`     // "v3" 或 "v4"
	Result     *model.InferenceResponse       `json:"result"`      // 推断结果
	DurationMs int64                          `json:"duration_ms"` // 耗时（毫秒）
	Error      string                         `json:"error,omitempty"`
}

// InferenceABComparisonResponse A/B 对比完整响应
type InferenceABComparisonResponse struct {
	UserID    string              `json:"user_id"`
	Timestamp string              `json:"timestamp"`
	V3        *InferenceABResult  `json:"v3"`
	V4        *InferenceABResult  `json:"v4"`
	// 差异摘要
	Diff      *InferenceABDiff    `json:"diff"`
}

// InferenceABDiff 推断差异分析
type InferenceABDiff struct {
	EmojiMatch    bool   `json:"emoji_match"`    // emoji 是否一致
	ActivityMatch bool   `json:"activity_match"` // activity 是否一致
	AvailMatch    bool   `json:"avail_match"`    // is_available 是否一致
	V3Faster      bool   `json:"v3_faster"`      // V3 是否更快
	TimeDiffMs    int64  `json:"time_diff_ms"`   // 耗时差（V4 - V3，正数表示 V4 慢）
	Summary       string `json:"summary"`        // 一句话摘要
}

// InferStatusABCompare V3/V4 推断 A/B 对比（已废弃，V3 已下线，仅运行 V4）
// POST /api/v1/agent/infer-status-v2/ab-compare
func (h *AgentHandler) InferStatusABCompare(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req model.ExtendedStatusReportRequest
	c.ShouldBindJSON(&req)

	if h.voiceScheduleServiceV4 == nil {
		response.InternalError(c, "V4 推断服务未启用")
		return
	}

	// V3 已下线，仅运行 V4
	start := time.Now()
	v4Ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	inference, err := h.voiceScheduleServiceV4.InferWithAgent(v4Ctx, userID, &req)
	v4Result := &InferenceABResult{
		Version:    "v4",
		DurationMs: time.Since(start).Milliseconds(),
	}
	if err != nil {
		v4Result.Error = err.Error()
	} else {
		v4Result.Result = &model.InferenceResponse{
			Phase:  model.InferencePhaseCompleted,
			Result: inference,
		}
	}

	response.Success(c, &InferenceABComparisonResponse{
		UserID:    userID,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		V3:        &InferenceABResult{Version: "v3", Error: "V3 已下线"},
		V4:        v4Result,
		Diff:      &InferenceABDiff{Summary: "V3 已下线，仅展示 V4 结果"},
	})
}

// InferStatusABBatch V3/V4 推断 A/B 批量对比（连续多轮对比，统计一致率）
// POST /api/v1/agent/infer-status-v2/ab-batch
func (h *AgentHandler) InferStatusABBatch(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req struct {
		SensorData model.ExtendedStatusReportRequest `json:"sensor_data"`
		Rounds     int                               `json:"rounds"` // 对比轮数（默认 3）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}
	if req.Rounds <= 0 {
		req.Rounds = 3
	}
	if req.Rounds > 10 {
		req.Rounds = 10
	}

	if h.voiceScheduleServiceV4 == nil {
		response.InternalError(c, "推断服务未启用")
		return
	}

	fmt.Printf("[AB Batch] 开始 V4 多轮测试 user=%s rounds=%d\n", userID, req.Rounds)

	type roundResult struct {
		Round int                `json:"round"`
		V4    *InferenceABResult `json:"v4"`
	}

	var rounds []roundResult
	var totalV4Ms int64

	for i := 0; i < req.Rounds; i++ {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		inference, err := h.voiceScheduleServiceV4.InferWithAgent(ctx, userID, &req.SensorData)
		cancel()

		r := &InferenceABResult{Version: "v4", DurationMs: time.Since(start).Milliseconds()}
		if err != nil {
			r.Error = err.Error()
		} else {
			r.Result = &model.InferenceResponse{
				Phase:  model.InferencePhaseCompleted,
				Result: inference,
			}
		}
		totalV4Ms += r.DurationMs

		rounds = append(rounds, roundResult{
			Round: i + 1,
			V4:    r,
		})
	}

	n := req.Rounds
	summary := map[string]interface{}{
		"rounds":    n,
		"avg_v4_ms": totalV4Ms / int64(n),
		"note":      "V3 已下线，仅展示 V4 结果",
	}

	fmt.Printf("[AB Batch] V4 多轮完成 user=%s rounds=%d avg=%dms\n",
		userID, n, totalV4Ms/int64(n))

	response.Success(c, map[string]interface{}{
		"user_id":   userID,
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"summary":   summary,
		"rounds":    rounds,
	})
}

// analyzeInferenceDiff 分析 V3 和 V4 推断结果的差异
func analyzeInferenceDiff(v3, v4 *InferenceABResult) *InferenceABDiff {
	diff := &InferenceABDiff{
		V3Faster:   v3.DurationMs < v4.DurationMs,
		TimeDiffMs: v4.DurationMs - v3.DurationMs,
	}

	// 任一出错时
	if v3.Error != "" || v4.Error != "" {
		diff.Summary = fmt.Sprintf("V3: %s | V4: %s",
			inferErrOrOk(v3), inferErrOrOk(v4))
		return diff
	}

	// 提取两边的 result
	var v3r, v4r *model.CurrentStatusInference
	if v3.Result != nil {
		v3r = v3.Result.Result
	}
	if v4.Result != nil {
		v4r = v4.Result.Result
	}

	if v3r == nil || v4r == nil {
		diff.Summary = "至少一方无推断结果"
		return diff
	}

	diff.EmojiMatch = v3r.Emoji == v4r.Emoji
	diff.ActivityMatch = v3r.Activity == v4r.Activity
	diff.AvailMatch = v3r.IsAvailable == v4r.IsAvailable

	// 生成摘要
	if diff.EmojiMatch && diff.ActivityMatch {
		diff.Summary = fmt.Sprintf("一致: %s%s", v3r.Emoji, v3r.Activity)
	} else {
		diff.Summary = fmt.Sprintf("V3=%s%s V4=%s%s", v3r.Emoji, v3r.Activity, v4r.Emoji, v4r.Activity)
	}

	// 追加耗时信息
	faster := "V3"
	if !diff.V3Faster {
		faster = "V4"
	}
	diff.Summary += fmt.Sprintf(" | %s快%dms", faster, abs64(diff.TimeDiffMs))

	return diff
}

func inferErrOrOk(r *InferenceABResult) string {
	if r.Error != "" {
		return "错误:" + r.Error
	}
	if r.Result != nil && r.Result.Result != nil {
		return r.Result.Result.Emoji + r.Result.Result.Activity
	}
	return "无结果"
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// ========== V3/V4 推断全面 A/B 对比测试 ==========

// inferenceABLLMKey 存储 LLM API Key（由 main.go 注入）
var inferenceABLLMKey string
var inferenceABLLMModel string

// SetInferenceABConfig 设置 A/B 对比测试的 LLM 配置
func (h *AgentHandler) SetInferenceABConfig(apiKey, modelName string) {
	inferenceABLLMKey = apiKey
	inferenceABLLMModel = modelName
}

// TestInferenceAB V3/V4 全面对比测试（返回 JSON 报告）
// POST /api/v1/agent/test/inference-ab
func (h *AgentHandler) TestInferenceAB(c *gin.Context) {
	if inferenceABLLMKey == "" {
		response.InternalError(c, "LLM 未配置，无法运行对比测试")
		return
	}

	var req struct {
		PersonaIDs  []string `json:"persona_ids"`  // 为空=全部 25 个
		Concurrency int      `json:"concurrency"`  // 默认 3
	}
	c.ShouldBindJSON(&req)

	config := service.InferenceABConfig{
		PersonaIDs:  req.PersonaIDs,
		Concurrency: req.Concurrency,
	}

	engine := service.NewInferenceABEngine(inferenceABLLMKey, inferenceABLLMModel, config)

	personaDesc := "全部"
	if len(req.PersonaIDs) > 0 {
		personaDesc = strings.Join(req.PersonaIDs, ",")
	}
	fmt.Printf("[InferenceAB] 开始全面对比 personas=%s\n", personaDesc)

	// 长超时（25 画像可能需要 15 分钟）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	report, err := engine.RunFullComparison(ctx)
	if err != nil {
		response.InternalError(c, fmt.Sprintf("对比测试失败: %v", err))
		return
	}

	fmt.Printf("[InferenceAB] 完成: V3=%.1f%% V4=%.1f%% 一致=%.1f%% 耗时=%s\n",
		report.V3Accuracy, report.V4Accuracy, report.EmojiMatchRate,
		report.Duration.Round(time.Second))

	response.Success(c, report)
}

// TestInferenceABReport V3/V4 全面对比测试（返回 Markdown 报告）
// POST /api/v1/agent/test/inference-ab/report
func (h *AgentHandler) TestInferenceABReport(c *gin.Context) {
	if inferenceABLLMKey == "" {
		response.InternalError(c, "LLM 未配置，无法运行对比测试")
		return
	}

	var req struct {
		PersonaIDs  []string `json:"persona_ids"`
		Concurrency int      `json:"concurrency"`
	}
	c.ShouldBindJSON(&req)

	config := service.InferenceABConfig{
		PersonaIDs:  req.PersonaIDs,
		Concurrency: req.Concurrency,
	}

	engine := service.NewInferenceABEngine(inferenceABLLMKey, inferenceABLLMModel, config)

	personaDesc := "全部"
	if len(req.PersonaIDs) > 0 {
		personaDesc = strings.Join(req.PersonaIDs, ",")
	}
	fmt.Printf("[InferenceAB Report] 开始全面对比 personas=%s\n", personaDesc)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	report, err := engine.RunFullComparison(ctx)
	if err != nil {
		response.InternalError(c, fmt.Sprintf("对比测试失败: %v", err))
		return
	}

	markdown := service.GenerateInferenceABReport(report)

	fmt.Printf("[InferenceAB Report] 完成: V3=%.1f%% V4=%.1f%% 耗时=%s\n",
		report.V3Accuracy, report.V4Accuracy, report.Duration.Round(time.Second))

	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.String(http.StatusOK, markdown)
}

