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
	agentService *service.AgentService
}

// NewAgentHandler 创建 Agent 处理器
func NewAgentHandler(agentService *service.AgentService) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
	}
}

// ReportStatus 上报状态
// POST /api/agent/status
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

	response.Success(c, gin.H{
		"success":        true,
		"next_report_in": 60, // 下次上报间隔（秒）
	})
}

// GetFreeProbability 获取好友有空概率列表
// GET /api/friends/free-probability
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
