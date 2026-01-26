package handler

import (
	"github.com/gin-gonic/gin"
	"youkong/internal/middleware"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

type AvailabilityHandler struct {
	availabilityService *service.AvailabilityService
}

func NewAvailabilityHandler(availabilityService *service.AvailabilityService) *AvailabilityHandler {
	return &AvailabilityHandler{availabilityService: availabilityService}
}

func (h *AvailabilityHandler) GetFriendsAvailabilities(c *gin.Context) {
	userID := middleware.GetUserID(c)
	availabilities, err := h.availabilityService.GetFriendsAvailabilities(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, availabilities)
}

func (h *AvailabilityHandler) GetMyAvailabilities(c *gin.Context) {
	userID := middleware.GetUserID(c)
	availabilities, err := h.availabilityService.GetMyAvailabilities(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, availabilities)
}

func (h *AvailabilityHandler) CreateAvailability(c *gin.Context) {
	var req service.CreateAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	availability, err := h.availabilityService.Create(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, response.CodeParamError, err.Error())
		return
	}

	response.Success(c, availability)
}

func (h *AvailabilityHandler) CancelAvailability(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.ParamError(c, "有空状态ID不能为空")
		return
	}

	userID := middleware.GetUserID(c)
	if err := h.availabilityService.Cancel(c.Request.Context(), id, userID); err != nil {
		response.Error(c, response.CodeForbidden, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "已取消"})
}
