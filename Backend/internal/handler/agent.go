package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/agent"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

// AgentHandler Agent 处理器
type AgentHandler struct {
	agentService         *service.AgentService
	memoryService        *service.MemoryService
	voiceScheduleService *service.VoiceScheduleService
	agentChatService     *service.AgentChatService
	scheduleRepo         ScheduleRepositoryInterface
}

// ScheduleRepositoryInterface 时刻表 Repository 接口
type ScheduleRepositoryInterface interface {
	GetUserScheduleHistory(ctx context.Context, userID string, beforeDate string, limit int) ([]*model.StatusSchedule, error)
	GetUserOldestScheduleDate(ctx context.Context, userID string) (string, error)
	CountUserSchedules(ctx context.Context, userID string) (int, error)
	GetActiveByUserAndDate(ctx context.Context, userID string, date time.Time) (*model.StatusSchedule, error)
	Create(ctx context.Context, schedule *model.StatusSchedule) error
	Update(ctx context.Context, schedule *model.StatusSchedule) error
}

// NewAgentHandler 创建 Agent 处理器
func NewAgentHandler(agentService *service.AgentService, memoryService *service.MemoryService, voiceScheduleService *service.VoiceScheduleService, agentChatService *service.AgentChatService, scheduleRepo ScheduleRepositoryInterface) *AgentHandler {
	return &AgentHandler{
		agentService:         agentService,
		memoryService:        memoryService,
		voiceScheduleService: voiceScheduleService,
		agentChatService:     agentChatService,
		scheduleRepo:         scheduleRepo,
	}
}

// ReportStatus 上报状态（简化版，仅用于手动触发分析）
// POST /api/agent/status
func (h *AgentHandler) ReportStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req model.ExtendedStatusReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	// ✅ 同步分析并返回结果
	if h.memoryService == nil {
		response.InternalError(c, "分析服务未初始化")
		return
	}

	// 【优先级检查】如果有活跃时刻表，使用时刻表状态而不是 LLM 分析
	if scheduleResult := h.checkActiveScheduleStatus(c.Request.Context(), userID); scheduleResult != nil {
		fmt.Printf("[上报] 使用时刻表状态 user=%s emoji=%s status=%s\n",
			userID, scheduleResult.LifeStatus.Emoji, scheduleResult.LifeStatus.Label)

		// 缓存时刻表状态
		if h.memoryService != nil {
			_ = h.memoryService.CacheAnalysisResult(c.Request.Context(), userID, scheduleResult)
		}

		response.Success(c, gin.H{
			"success":  true,
			"message":  "使用时刻表状态",
			"analysis": scheduleResult,
		})
		return
	}

	fmt.Printf("[上报] 开始分析 user=%s\n", userID)
	result, err := h.memoryService.AnalyzeAndUpdateMemory(c.Request.Context(), userID, &req)
	if err != nil {
		fmt.Printf("[上报] 分析失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "分析失败")
		return
	}

	fmt.Printf("[上报] 分析完成 user=%s status=%s\n", userID, result.Availability.Status)

	// 同步 AI 推测状态到时刻表（异步执行，不阻塞响应）
	go h.syncAIGuessToSchedule(context.Background(), userID, result)

	// 返回分析结果
	response.Success(c, gin.H{
		"success":  true,
		"message":  "分析完成",
		"analysis": result,
	})
}

// checkActiveScheduleStatus 检查用户是否有活跃时刻表，如果有返回当前时段的状态
func (h *AgentHandler) checkActiveScheduleStatus(ctx context.Context, userID string) *model.AnalysisResult {
	if h.scheduleRepo == nil {
		return nil
	}

	now := time.Now()
	currentTime := now.Format("15:04")
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)

	// ============ 第一步：检查昨天的跨午夜时段 ============
	// 凌晨时段（00:00-12:00），优先检查昨天的时刻表中是否有跨午夜到今天的时段
	if currentTime < "12:00" {
		yesterdaySchedule, err := h.scheduleRepo.GetActiveByUserAndDate(ctx, userID, yesterday)
		if err == nil && yesterdaySchedule != nil && len(yesterdaySchedule.Items) > 0 {
			for _, item := range yesterdaySchedule.Items {
				// 只检查跨午夜的时段（end < start，如 22:00-09:00）
				if item.EndTime < item.StartTime {
					// 当前时间在跨午夜时段的"第二天部分"（00:00 到 end）
					if currentTime < item.EndTime {
						fmt.Printf("[Schedule] 匹配到昨天(%s)的跨午夜时段: %s-%s %s\n",
							yesterday.Format("01-02"), item.StartTime, item.EndTime, item.Status)
						return &model.AnalysisResult{
							Availability: model.AvailabilityAnalysis{
								Status:      getScheduleStatusText(item.Status),
								Probability: guessScheduleProbability(item.Status),
								Reason:      item.Status,
								Confidence:  "high",
							},
							LifeStatus: model.LifeStatus{
								Emoji:       item.Emoji,
								Label:       item.Status,
								Description: item.Status,
							},
							UpdatedAt: now,
						}
					}
				}
			}
		}
	}

	// ============ 第二步：检查今天的时刻表 ============
	schedule, err := h.scheduleRepo.GetActiveByUserAndDate(ctx, userID, today)
	if err != nil || schedule == nil || len(schedule.Items) == 0 {
		return nil
	}

	// 检查当前时间是否在今天某个时段内
	for _, item := range schedule.Items {
		// 对于跨午夜时段（end < start），只有当前时间 >= start 时才匹配
		// 跨午夜时段的"第二天部分"已在第一步处理
		if item.EndTime < item.StartTime {
			// 跨午夜：只匹配"今天部分"（start 到 23:59）
			if currentTime >= item.StartTime {
				fmt.Printf("[Schedule] 匹配到今天(%s)的跨午夜时段(今天部分): %s-%s %s\n",
					today.Format("01-02"), item.StartTime, item.EndTime, item.Status)
				return &model.AnalysisResult{
					Availability: model.AvailabilityAnalysis{
						Status:      getScheduleStatusText(item.Status),
						Probability: guessScheduleProbability(item.Status),
						Reason:      item.Status,
						Confidence:  "high",
					},
					LifeStatus: model.LifeStatus{
						Emoji:       item.Emoji,
						Label:       item.Status,
						Description: item.Status,
					},
					UpdatedAt: now,
				}
			}
		} else {
			// 普通时段（同一天内）
			if currentTime >= item.StartTime && currentTime < item.EndTime {
				fmt.Printf("[Schedule] 匹配到今天(%s)的普通时段: %s-%s %s\n",
					today.Format("01-02"), item.StartTime, item.EndTime, item.Status)
				return &model.AnalysisResult{
					Availability: model.AvailabilityAnalysis{
						Status:      getScheduleStatusText(item.Status),
						Probability: guessScheduleProbability(item.Status),
						Reason:      item.Status,
						Confidence:  "high",
					},
					LifeStatus: model.LifeStatus{
						Emoji:       item.Emoji,
						Label:       item.Status,
						Description: item.Status,
					},
					UpdatedAt: now,
				}
			}
		}
	}

	return nil
}

// isTimeInRange 检查时间是否在范围内（复用调度器逻辑）
func isTimeInRange(current, start, end string) bool {
	// 处理跨午夜的情况
	if end < start {
		return current >= start || current < end
	}
	return current >= start && current < end
}

// getScheduleStatusText 根据状态描述判断有空/忙碌
func getScheduleStatusText(status string) string {
	busyKeywords := []string{"开会", "工作", "上班", "上课", "出差", "加班", "忙"}
	for _, keyword := range busyKeywords {
		if containsStr(status, keyword) {
			return "忙碌"
		}
	}

	freeKeywords := []string{"休息", "休闲", "逛街", "刷", "玩", "躺", "看", "吃", "喝", "聚"}
	for _, keyword := range freeKeywords {
		if containsStr(status, keyword) {
			return "有空"
		}
	}

	return "可能有空"
}

// guessScheduleProbability 猜测有空概率
func guessScheduleProbability(status string) int {
	statusText := getScheduleStatusText(status)
	switch statusText {
	case "有空":
		return 80
	case "忙碌":
		return 20
	default:
		return 50
	}
}

// containsStr 检查字符串是否包含子串
func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
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

	// 流式执行福尔摩斯推理
	result, err := h.agentService.ReportExtendedStatusStream(c.Request.Context(), userID, &req, func(event interface{}) {
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

		// 异步缓存，不阻塞响应
		go func() {
			ctx := context.Background()
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

	// 流式执行 Holmes 2.0 推理
	result, err := h.agentService.ReportExtendedStatus2Stream(c.Request.Context(), userID, &req, func(event interface{}) {
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

		// 异步缓存
		go func() {
			ctx := context.Background()
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

	// 获取用户最近的状态记忆（用于推理）
	var recentMemory []*model.UserStatusMemory
	if h.memoryService != nil {
		recentMemory, _ = h.memoryService.GetRecentUserStatusMemory(c.Request.Context(), userID, 10)
	}

	// 流式执行状态选项生成
	result, err := h.agentService.GenerateStatusOptionsStream(c.Request.Context(), userID, &req, recentMemory, func(event interface{}) {
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

	// 选择状态并记录
	err := h.memoryService.SelectStatus(c.Request.Context(), userID, &req)
	if err != nil {
		fmt.Printf("[选择状态] 失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "保存失败")
		return
	}

	fmt.Printf("[选择状态] 成功 user=%s emoji=%s status=%s\n", userID, req.Emoji, req.Status)

	response.Success(c, gin.H{
		"success": true,
		"message": "状态已更新",
	})
}

// VoiceScheduleStream 语音状态时刻表 SSE 流
// POST /api/agent/voice-schedule/stream
// Content-Type: multipart/form-data
func (h *AgentHandler) VoiceScheduleStream(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	if h.voiceScheduleService == nil {
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

	fmt.Printf("[VoiceSchedule] 收到音频 user=%s size=%d format=%s session=%s\n", userID, len(audioData), audioFormat, sessionID)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer

	// 流式处理语音输入（传入 sessionID 以恢复会话）
	_, err = h.voiceScheduleService.ProcessVoiceInput(c.Request.Context(), userID, sessionID, audioData, audioFormat, func(event *model.VoiceScheduleEvent) {
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
		return
	}

	// 发送完成信号
	fmt.Fprintf(w, "data: [DONE]\n\n")
	w.Flush()
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
		return
	}

	// 发送完成信号（VoiceScheduleInteract）
	fmt.Fprintf(w, "data: [DONE]\n\n")
	w.Flush()
}

// VoiceScheduleText 语音时刻表文本测试接口（用于测试，跳过语音识别）
// POST /api/agent/voice-schedule/text
func (h *AgentHandler) VoiceScheduleText(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	if h.voiceScheduleService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "语音服务未启用"})
		return
	}

	var req struct {
		SessionID string `json:"session_id"` // 可选，用于继续会话
		Text      string `json:"text" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: text 必填"})
		return
	}

	fmt.Printf("[VoiceScheduleText] user=%s session=%s text=%s\n", userID, req.SessionID, req.Text)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer

	// 使用文本处理（跳过语音识别）
	_, err := h.voiceScheduleService.ProcessTextInput(c.Request.Context(), userID, req.SessionID, req.Text, func(event *model.VoiceScheduleEvent) {
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
		return
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	w.Flush()
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

	// 按日期顺序构建响应
	result := make([]DayScheduleResponse, 0, len(dateOrder))
	for _, dateStr := range dateOrder {
		s := dateMap[dateStr]
		result = append(result, DayScheduleResponse{
			ScheduleDate: dateStr,
			Items:        s.Items,
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
