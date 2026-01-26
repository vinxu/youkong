package handler

import (
	"github.com/gin-gonic/gin"
	"youkong/internal/middleware"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

type CircleHandler struct {
	circleService *service.CircleService
}

func NewCircleHandler(circleService *service.CircleService) *CircleHandler {
	return &CircleHandler{circleService: circleService}
}

func (h *CircleHandler) GetCircles(c *gin.Context) {
	userID := middleware.GetUserID(c)
	circles, err := h.circleService.GetMyCircles(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, circles)
}

func (h *CircleHandler) CreateCircle(c *gin.Context) {
	var req service.CreateCircleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	userID := middleware.GetUserID(c)
	circle, err := h.circleService.Create(c.Request.Context(), userID, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, circle)
}

func (h *CircleHandler) GetCircle(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.ParamError(c, "圈子ID不能为空")
		return
	}

	circle, err := h.circleService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if circle == nil {
		response.NotFound(c, "圈子不存在")
		return
	}

	response.Success(c, circle)
}

func (h *CircleHandler) UpdateCircle(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.ParamError(c, "圈子ID不能为空")
		return
	}

	var req service.CreateCircleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	userID := middleware.GetUserID(c)
	circle, err := h.circleService.Update(c.Request.Context(), id, userID, &req)
	if err != nil {
		response.Error(c, response.CodeForbidden, err.Error())
		return
	}

	response.Success(c, circle)
}

func (h *CircleHandler) DeleteCircle(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.ParamError(c, "圈子ID不能为空")
		return
	}

	userID := middleware.GetUserID(c)
	if err := h.circleService.Delete(c.Request.Context(), id, userID); err != nil {
		response.Error(c, response.CodeForbidden, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

type AddMemberRequest struct {
	UserID string `json:"userId" binding:"required"`
}

func (h *CircleHandler) AddMember(c *gin.Context) {
	circleID := c.Param("id")
	if circleID == "" {
		response.ParamError(c, "圈子ID不能为空")
		return
	}

	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	ownerID := middleware.GetUserID(c)
	if err := h.circleService.AddMember(c.Request.Context(), circleID, ownerID, req.UserID); err != nil {
		response.Error(c, response.CodeForbidden, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "添加成功"})
}

func (h *CircleHandler) RemoveMember(c *gin.Context) {
	circleID := c.Param("id")
	memberUserID := c.Param("userId")
	if circleID == "" || memberUserID == "" {
		response.ParamError(c, "参数不完整")
		return
	}

	ownerID := middleware.GetUserID(c)
	if err := h.circleService.RemoveMember(c.Request.Context(), circleID, ownerID, memberUserID); err != nil {
		response.Error(c, response.CodeForbidden, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "移除成功"})
}
