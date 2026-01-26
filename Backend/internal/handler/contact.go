package handler

import (
	"github.com/gin-gonic/gin"

	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

// ContactHandler 通讯录处理器
type ContactHandler struct {
	contactService *service.ContactService
}

// NewContactHandler 创建通讯录处理器
func NewContactHandler(contactService *service.ContactService) *ContactHandler {
	return &ContactHandler{
		contactService: contactService,
	}
}

// MatchContacts 匹配通讯录
// POST /api/v1/contacts/match
func (h *ContactHandler) MatchContacts(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req model.ContactMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误: phone_hashes 必填且长度1-1000")
		return
	}

	result, err := h.contactService.MatchContacts(c.Request.Context(), userID, req.PhoneHashes)
	if err != nil {
		response.InternalError(c, "匹配失败")
		return
	}

	response.Success(c, result)
}

// BatchAddFriends 批量添加好友
// POST /api/v1/contacts/add-friends
func (h *ContactHandler) BatchAddFriends(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req model.BatchAddFriendsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误: user_ids 必填且长度1-100")
		return
	}

	result, err := h.contactService.BatchAddFriends(c.Request.Context(), userID, req.UserIDs)
	if err != nil {
		response.InternalError(c, "添加失败")
		return
	}

	response.Success(c, result)
}
