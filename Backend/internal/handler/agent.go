package handler

import (
	"net/http"

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

	// 1. 保存实时状态到 Redis（兼容旧逻辑）
	legacyReq := &model.StatusReportRequest{
		Screen:   req.Screen,
		Location: req.Location,
	}
	if err := h.agentService.ReportStatus(c.Request.Context(), userID, legacyReq); err != nil {
		response.InternalError(c, "上报失败")
		return
	}

	// 2. 分析状态并更新记忆
	var analysisResult *model.AnalysisResult
	if h.memoryService != nil {
		result, err := h.memoryService.AnalyzeAndUpdateMemory(c.Request.Context(), userID, &req)
		if err == nil {
			analysisResult = result
		}
	}

	// 3. 返回增强响应
	resp := model.StatusReportResponse{
		Success:      true,
		NextReportIn: 60,
		Analysis:     analysisResult,
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

	// 获取基础有空概率
	result, err := h.agentService.GetFriendsFreeProbability(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}

	// 增强：添加生活状态
	if h.memoryService != nil && result != nil && len(result.Friends) > 0 {
		friendIDs := make([]string, len(result.Friends))
		for i, f := range result.Friends {
			friendIDs[i] = f.FriendID
		}

		// 批量获取缓存的分析结果
		analysisMap, err := h.memoryService.GetCachedAnalysisByUserIDs(c.Request.Context(), friendIDs)
		if err == nil && len(analysisMap) > 0 {
			// 构建增强响应
			enhanced := make([]model.EnhancedFriendRecommendation, len(result.Friends))
			for i, f := range result.Friends {
				enhanced[i] = model.EnhancedFriendRecommendation{
					FriendID:    f.FriendID,
					Name:        f.Name,
					Avatar:      f.Avatar,
					Probability: f.Probability,
					Confidence:  f.Confidence,
					Reason:      f.Reason,
					Color:       f.Color,
					UpdatedAt:   f.UpdatedAt,
				}
				if analysis, ok := analysisMap[f.FriendID]; ok {
					enhanced[i].LifeStatus = &analysis.LifeStatus
				}
			}

			response.Success(c, model.EnhancedFreeProbabilityResponse{
				Friends:     enhanced,
				GeneratedAt: result.GeneratedAt,
			})
			return
		}
	}

	response.Success(c, result)
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
