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

// AddFriendByPhone 通过手机号加好友（直接添加，无需确认）
// POST /api/v1/friends/add-by-phone
// Deprecated: 建议使用 SendFriendRequest 发送好友请求
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

// SendFriendRequest 发送好友请求
// POST /api/v1/friends/request
func (h *FriendshipHandler) SendFriendRequest(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req model.SendFriendRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "手机号格式错误，需要11位数字")
		return
	}

	result, err := h.friendshipService.SendFriendRequest(c.Request.Context(), userID, req.Phone, req.Message)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// GetReceivedRequests 获取收到的好友请求
// GET /api/v1/friends/requests/received
func (h *FriendshipHandler) GetReceivedRequests(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	result, err := h.friendshipService.GetReceivedRequests(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// GetSentRequests 获取发出的好友请求
// GET /api/v1/friends/requests/sent
func (h *FriendshipHandler) GetSentRequests(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	result, err := h.friendshipService.GetSentRequests(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, result)
}

// HandleFriendRequest 处理好友请求（同意/拒绝）
// POST /api/v1/friends/requests/:id/handle
func (h *FriendshipHandler) HandleFriendRequest(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	requestID := c.Param("id")
	if requestID == "" {
		response.ParamError(c, "请求ID不能为空")
		return
	}

	var req model.HandleFriendRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	if err := h.friendshipService.HandleFriendRequest(c.Request.Context(), userID, requestID, req.Accept); err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	msg := "已拒绝"
	if req.Accept {
		msg = "已同意，你们已成为好友"
	}
	response.Success(c, gin.H{"message": msg})
}

// GetPendingRequestCount 获取待处理请求数量
// GET /api/v1/friends/requests/count
func (h *FriendshipHandler) GetPendingRequestCount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	count, err := h.friendshipService.GetPendingRequestCount(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, gin.H{"count": count})
}
