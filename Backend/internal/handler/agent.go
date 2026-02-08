package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/agent"
	"youkong/internal/pkg/response"
	"youkong/internal/pkg/tencent"
	"youkong/internal/service"
)

// FriendshipChecker 好友关系检查接口
type FriendshipChecker interface {
	AreFriends(ctx context.Context, userID, friendID string) (bool, error)
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
	inferenceAgent         *service.StatusInferenceAgent // V2 推断 Agent
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

// SetInferenceAgent 设置 V2 推断 Agent
func (h *AgentHandler) SetInferenceAgent(agent *service.StatusInferenceAgent) {
	h.inferenceAgent = agent
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

	// 使用 Agent 推断
	if h.inferenceAgent == nil {
		response.InternalError(c, "推断服务未初始化")
		return
	}

	fmt.Printf("[上报] 开始 V3 推断 user=%s\n", userID)
	inference, err := h.inferenceAgent.InferWithAgent(c.Request.Context(), userID, &req)
	if err != nil {
		fmt.Printf("[上报] 推断失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "推断失败")
		return
	}

	fmt.Printf("[上报] 推断完成 user=%s emoji=%s activity=%s confidence=%s\n",
		userID, inference.Emoji, inference.Activity, inference.Confidence)

	// 转换为 AnalysisResult 格式（兼容现有前端）
	analysisResult := inferenceToAnalysisResult(inference)

	// 缓存分析结果
	if h.memoryService != nil {
		_ = h.memoryService.CacheAnalysisResult(c.Request.Context(), userID, analysisResult)
	}

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

	// 使用 V4 服务处理语音输入
	_, err = h.voiceScheduleServiceV4.ProcessVoiceInput(c.Request.Context(), userID, sessionID, audioData, audioFormat, func(event *model.V4Event) {
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

	// 使用 V4 服务处理文本输入
	_, err := h.voiceScheduleServiceV4.ProcessTextInput(c.Request.Context(), userID, req.SessionID, req.Text, req.SensorData, func(event *model.V4Event) {
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
		return
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	w.Flush()
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

	// 使用 V4 服务处理
	_, err := h.voiceScheduleServiceV4.ProcessTextInput(c.Request.Context(), userID, req.SessionID, req.Text, req.SensorData, func(event *model.V4Event) {
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
	Highlight    *bool  `json:"highlight,omitempty"` // 高亮状态（有空），nil 表示不修改
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
	for i, item := range schedule.Items {
		if item.StartTime == req.OldStartTime && item.EndTime == req.OldEndTime {
			highlight := item.Highlight // 默认保留原值
			if req.Highlight != nil {
				highlight = *req.Highlight
			}
			schedule.Items[i] = model.ScheduleItem{
				StartTime: req.NewStartTime,
				EndTime:   req.NewEndTime,
				Emoji:     req.Emoji,
				Status:    req.Status,
				Executed:  item.Executed,
				IsAIGuess: false, // 用户编辑后不再是 AI 推测
				GifURL:    item.GifURL,
				GiphyQuery: item.GiphyQuery,
				Highlight: highlight,
			}
			found = true
			break
		}
	}

	if !found {
		response.NotFound(c, "未找到该时段")
		return
	}

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

	// 获取当天的时刻表（不限 status，允许编辑已完成的时刻表）
	schedule, err := h.scheduleRepo.GetLatestByUserAndDate(c.Request.Context(), userID, scheduleDate)
	if err != nil || schedule == nil {
		response.NotFound(c, "未找到该日期的时刻表")
		return
	}

	// 查找并删除条目
	found := false
	newItems := make(model.ScheduleItems, 0, len(schedule.Items))
	for _, item := range schedule.Items {
		if item.StartTime == startTime && item.EndTime == endTime {
			found = true
			continue
		}
		newItems = append(newItems, item)
	}

	if !found {
		response.NotFound(c, "未找到该时段")
		return
	}

	schedule.Items = newItems

	// 保存更新
	if err := h.scheduleRepo.Update(c.Request.Context(), schedule); err != nil {
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

// InferStatusV2 Agent-based AI 推断当下状态（V3 架构：支持选项交互）
// POST /api/v1/agent/infer-status-v2
func (h *AgentHandler) InferStatusV2(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if h.inferenceAgent == nil {
		response.InternalError(c, "推断服务未启用")
		return
	}

	var req model.ExtendedStatusReportRequest
	c.ShouldBindJSON(&req)

	// 使用 V3 接口返回完整 InferenceResponse
	inferResp, err := h.inferenceAgent.InferWithAgentV3(c.Request.Context(), userID, &req)
	if err != nil {
		fmt.Printf("[InferStatusV3] 推断失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, "推断失败")
		return
	}

	// 如果直接完成（非选项交互），更新首页状态
	if inferResp.Phase == model.InferencePhaseCompleted && inferResp.Result != nil {
		fmt.Printf("[InferStatusV3] 推断成功 user=%s emoji=%s activity=%s confidence=%s\n",
			userID, inferResp.Result.Emoji, inferResp.Result.Activity, inferResp.Result.Confidence)

		if h.agentService != nil {
			feedbackReq := &model.StatusFeedbackRequest{
				CorrectedEmoji:       inferResp.Result.Emoji,
				CorrectedActivity:    inferResp.Result.Activity,
				CorrectedPlace:       inferResp.Result.Place,
				CorrectedIsAvailable: &inferResp.Result.IsAvailable,
				GifURL:               inferResp.Result.GifURL,
				GiphyQuery:           inferResp.Result.GiphyQuery,
			}
			if err := h.agentService.SaveStatusFeedback(c.Request.Context(), userID, feedbackReq); err != nil {
				fmt.Printf("[InferStatusV3] 更新首页状态失败 user=%s error=%v\n", userID, err)
			}
		}
	} else {
		fmt.Printf("[InferStatusV3] 等待用户选择 user=%s session=%s options=%d\n",
			userID, inferResp.SessionID, len(inferResp.Options))
	}

	response.Success(c, inferResp)
}

// InferStatusV2Stream Agent-based AI 推断当下状态（SSE 流式版）
// POST /api/v1/agent/infer-status-v2/stream
func (h *AgentHandler) InferStatusV2Stream(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	if h.inferenceAgent == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "推断服务未启用"})
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
				// SSE 注释行，客户端会忽略，但保持连接活跃
				fmt.Fprintf(w, ":keepalive\n\n")
				w.Flush()
			}
		}
	}()

	// 流式推断
	result, err := h.inferenceAgent.InferWithAgentStream(c.Request.Context(), userID, &req, func(event *service.InferenceStreamEvent) {
		data, err := json.Marshal(event)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
	})

	if err != nil {
		errEvent := service.InferenceStreamEvent{
			Type:    "error",
			Message: err.Error(),
		}
		data, _ := json.Marshal(errEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.Flush()
		return
	}

	// 推断成功后更新首页状态（保留 GIF 信息）
	if result != nil && h.agentService != nil {
		go func() {
			ctx := context.Background()
			feedbackReq := &model.StatusFeedbackRequest{
				CorrectedEmoji:       result.Emoji,
				CorrectedActivity:    result.Activity,
				CorrectedPlace:       result.Place,
				CorrectedIsAvailable: &result.IsAvailable,
				GifURL:               result.GifURL,
				GiphyQuery:           result.GiphyQuery,
			}
			h.agentService.SaveStatusFeedback(ctx, userID, feedbackReq)
		}()
	}

	// 发送完成信号
	fmt.Fprintf(w, "data: [DONE]\n\n")
	w.Flush()
}

// InferStatusRespond 接收用户选择（V3: 从 session 完成推断）
// POST /api/v1/agent/infer-status-v2/respond
func (h *AgentHandler) InferStatusRespond(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if h.inferenceAgent == nil {
		response.InternalError(c, "推断服务未启用")
		return
	}

	var req struct {
		SessionID     string `json:"session_id" binding:"required"`
		SelectedIndex int    `json:"selected_index"` // 用户选择的选项索引
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误: session_id 必填")
		return
	}

	fmt.Printf("[InferStatusRespond] user=%s session=%s selected=%d\n", userID, req.SessionID, req.SelectedIndex)

	result, err := h.inferenceAgent.HandleUserResponse(c.Request.Context(), userID, req.SessionID, req.SelectedIndex)
	if err != nil {
		fmt.Printf("[InferStatusRespond] 处理失败 user=%s error=%v\n", userID, err)
		response.InternalError(c, err.Error())
		return
	}

	// 更新首页状态
	if h.agentService != nil && result != nil {
		feedbackReq := &model.StatusFeedbackRequest{
			CorrectedEmoji:       result.Emoji,
			CorrectedActivity:    result.Activity,
			CorrectedPlace:       result.Place,
			CorrectedIsAvailable: &result.IsAvailable,
			GifURL:               result.GifURL,
			GiphyQuery:           result.GiphyQuery,
		}
		if err := h.agentService.SaveStatusFeedback(c.Request.Context(), userID, feedbackReq); err != nil {
			fmt.Printf("[InferStatusRespond] 更新首页状态失败 user=%s error=%v\n", userID, err)
		}
	}

	response.Success(c, &model.InferenceResponse{
		Phase:  model.InferencePhaseCompleted,
		Result: result,
	})
}

