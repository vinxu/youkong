package handler

import (
	"github.com/gin-gonic/gin"
	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

type FriendshipHandler struct {
	friendshipService *service.FriendshipService
}

func NewFriendshipHandler(friendshipService *service.FriendshipService) *FriendshipHandler {
	return &FriendshipHandler{friendshipService: friendshipService}
}

func (h *FriendshipHandler) GetFriends(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	result, err := h.friendshipService.GetFriends(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *FriendshipHandler) RemoveFriend(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	friendID := c.Param("userId")
	if friendID == "" {
		response.ParamError(c, "好友ID不能为空")
		return
	}

	if err := h.friendshipService.RemoveFriend(c.Request.Context(), userID, friendID); err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "已删除好友"})
}

func (h *FriendshipHandler) GetInvitedByMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	result, err := h.friendshipService.GetInvitedByMe(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *FriendshipHandler) GetInvitedMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	result, err := h.friendshipService.GetInvitedMe(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// AddFriendByPhone 通过手机号加好友
// POST /api/v1/friends/add-by-phone
func (h *FriendshipHandler) AddFriendByPhone(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req model.AddFriendByPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "手机号格式错误，需要11位数字")
		return
	}

	result, err := h.friendshipService.AddFriendByPhone(c.Request.Context(), userID, req.Phone)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, result)
}
