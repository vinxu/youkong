package handler

import (
	"github.com/gin-gonic/gin"

	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

// PredictionHandler AI 状态推测处理器
type PredictionHandler struct {
	predictionService *service.PredictionService
}

// NewPredictionHandler 创建推测处理器
func NewPredictionHandler(predictionService *service.PredictionService) *PredictionHandler {
	return &PredictionHandler{
		predictionService: predictionService,
	}
}

// StartPrediction 开始推测任务
// POST /api/v1/agent/prediction/start
func (h *PredictionHandler) StartPrediction(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req model.StartPredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体
		req = model.StartPredictionRequest{}
	}

	task, err := h.predictionService.StartPrediction(c.Request.Context(), userID, req.TargetDate)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.Success(c, task.ToResponse())
}

// GetPrediction 获取推测任务状态
// GET /api/v1/agent/prediction/:id
func (h *PredictionHandler) GetPrediction(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		response.ParamError(c, "任务 ID 不能为空")
		return
	}

	task, err := h.predictionService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		response.NotFound(c, "任务不存在")
		return
	}

	// 验证用户权限
	if task.UserID != userID {
		response.Forbidden(c, "无权访问此任务")
		return
	}

	response.Success(c, task.ToResponse())
}

// GetLatestPrediction 获取最近的推测任务
// GET /api/v1/agent/prediction/latest
func (h *PredictionHandler) GetLatestPrediction(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	task, err := h.predictionService.GetLatestTask(c.Request.Context(), userID)
	if err != nil {
		response.Success(c, nil) // 没有任务返回 null
		return
	}

	response.Success(c, task.ToResponse())
}

// GetPendingPrediction 获取待确认的推测任务
// GET /api/v1/agent/prediction/pending
func (h *PredictionHandler) GetPendingPrediction(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	task, err := h.predictionService.GetPendingTask(c.Request.Context(), userID)
	if err != nil {
		response.Success(c, nil) // 没有待确认任务返回 null
		return
	}

	response.Success(c, task.ToResponse())
}

// ConfirmPrediction 确认推测结果
// POST /api/v1/agent/prediction/:id/confirm
func (h *PredictionHandler) ConfirmPrediction(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		response.ParamError(c, "任务 ID 不能为空")
		return
	}

	var req model.ConfirmPredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体（使用默认合并结果）
		req = model.ConfirmPredictionRequest{}
	}

	err := h.predictionService.ConfirmPrediction(c.Request.Context(), taskID, userID, req.ModifiedItems)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.Success(c, gin.H{
		"success": true,
		"message": "推测结果已确认并保存",
	})
}

// RejectPrediction 放弃推测结果
// POST /api/v1/agent/prediction/:id/reject
func (h *PredictionHandler) RejectPrediction(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		response.ParamError(c, "任务 ID 不能为空")
		return
	}

	err := h.predictionService.RejectPrediction(c.Request.Context(), taskID, userID)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.Success(c, gin.H{
		"success": true,
		"message": "已放弃推测结果",
	})
}
