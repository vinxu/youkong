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
	agentChatService    *service.AgentChatService
}

func NewConversationHandler(conversationService *service.ConversationService, agentChatService *service.AgentChatService) *ConversationHandler {
	return &ConversationHandler{
		conversationService: conversationService,
		agentChatService:    agentChatService,
	}
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

type CreateConversationRequest struct {
	PartnerID string `json:"partnerId" binding:"required"`
}

func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误：需要 partnerId")
		return
	}

	userID := middleware.GetUserID(c)
	convResp, err := h.conversationService.GetOrCreateConversationWithResponse(c.Request.Context(), userID, req.PartnerID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, convResp)
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

// AgentReply 让 Agent 生成回复
// POST /api/v1/conversations/:id/agent-reply
func (h *ConversationHandler) AgentReply(c *gin.Context) {
	conversationID := c.Param("id")
	if conversationID == "" {
		response.ParamError(c, "会话ID不能为空")
		return
	}

	if h.agentChatService == nil {
		response.InternalError(c, "你的元婴罢工了")
		return
	}

	userID := middleware.GetUserID(c)
	reply, err := h.agentChatService.GenerateReply(c.Request.Context(), conversationID, userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, reply)
}
