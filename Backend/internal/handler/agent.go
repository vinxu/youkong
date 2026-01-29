package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

// AgentHandler Agent 处理器
type AgentHandler struct {
	agentService  *service.AgentService
	memoryService *service.MemoryService
}

// NewAgentHandler 创建 Agent 处理器
func NewAgentHandler(agentService *service.AgentService, memoryService *service.MemoryService) *AgentHandler {
	return &AgentHandler{
		agentService:  agentService,
		memoryService: memoryService,
	}
}

// ReportStatus 上报状态（增强版，支持完整数据和实时分析）
// POST /api/agent/status
func (h *AgentHandler) ReportStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	// 支持扩展的状态上报请求
	var req model.ExtendedStatusReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	// 1. 使用福尔摩斯推理框架进行分析
	holmesResult, err := h.agentService.ReportExtendedStatus(c.Request.Context(), userID, &req)
	if err != nil {
		// 福尔摩斯分析失败，降级到旧逻辑
		legacyReq := &model.StatusReportRequest{
			Screen:   req.Screen,
			Location: req.Location,
		}
		if err := h.agentService.ReportStatus(c.Request.Context(), userID, legacyReq); err != nil {
			response.InternalError(c, "上报失败")
			return
		}
	}

	// 2. 同时更新核心记忆（保持向后兼容）
	var analysisResult *model.AnalysisResult
	if h.memoryService != nil {
		result, err := h.memoryService.AnalyzeAndUpdateMemory(c.Request.Context(), userID, &req)
		if err == nil {
			analysisResult = result
		}
	}

	// 3. 构建响应
	resp := struct {
		Success      bool                  `json:"success"`
		NextReportIn int                   `json:"next_report_in"`
		Analysis     *model.AnalysisResult `json:"analysis,omitempty"`
		Holmes       *model.HolmesResult   `json:"holmes,omitempty"` // 福尔摩斯分析结果
	}{
		Success:      true,
		NextReportIn: 60,
		Analysis:     analysisResult,
		Holmes:       holmesResult,
	}

	response.Success(c, resp)
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

	// 发送最终结果
	doneEvent := map[string]interface{}{
		"type":   "done",
		"result": result,
	}
	data, _ := json.Marshal(doneEvent)
	fmt.Fprintf(w, "data: %s\n\n", data)
	w.Flush()
}
