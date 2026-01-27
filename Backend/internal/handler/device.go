package handler

import (
	"github.com/gin-gonic/gin"
	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

// DeviceHandler 设备相关处理器
type DeviceHandler struct {
	notificationService *service.NotificationService
}

// NewDeviceHandler 创建设备处理器
func NewDeviceHandler(notificationService *service.NotificationService) *DeviceHandler {
	return &DeviceHandler{
		notificationService: notificationService,
	}
}

// RegisterToken 注册设备 Token
// POST /api/v1/devices/token
func (h *DeviceHandler) RegisterToken(c *gin.Context) {
	var req model.RegisterDeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if err := h.notificationService.RegisterDeviceToken(c.Request.Context(), userID, &req); err != nil {
		response.InternalError(c, "注册设备 Token 失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// UnregisterToken 注销设备 Token
// DELETE /api/v1/devices/token
func (h *DeviceHandler) UnregisterToken(c *gin.Context) {
	var req model.UnregisterDeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误: "+err.Error())
		return
	}

	if err := h.notificationService.UnregisterDeviceToken(c.Request.Context(), req.Token); err != nil {
		response.InternalError(c, "注销设备 Token 失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}
