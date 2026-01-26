package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"youkong/internal/middleware"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

type ConversationHandler struct {
	conversationService *service.ConversationService
}

func NewConversationHandler(conversationService *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{conversationService: conversationService}
}

func (h *ConversationHandler) GetConversations(c *gin.Context) {
	userID := middleware.GetUserID(c)
	conversations, err := h.conversationService.GetConversations(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, conversations)
}

func (h *ConversationHandler) GetMessages(c *gin.Context) {
	conversationID := c.Param("id")
	if conversationID == "" {
		response.ParamError(c, "会话ID不能为空")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	userID := middleware.GetUserID(c)
	messages, err := h.conversationService.GetMessages(c.Request.Context(), conversationID, userID, limit, offset)
	if err != nil {
		response.Error(c, response.CodeForbidden, err.Error())
		return
	}

	response.Success(c, messages)
}

func (h *ConversationHandler) SendMessage(c *gin.Context) {
	conversationID := c.Param("id")
	if conversationID == "" {
		response.ParamError(c, "会话ID不能为空")
		return
	}

	var req service.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	userID := middleware.GetUserID(c)
	message, err := h.conversationService.SendMessage(c.Request.Context(), conversationID, userID, &req)
	if err != nil {
		response.Error(c, response.CodeForbidden, err.Error())
		return
	}

	response.Success(c, message)
}
