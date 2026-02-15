package handler

import (
	"github.com/gin-gonic/gin"

	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

// InteractionHandler 互动处理器
type InteractionHandler struct {
	interactionService *service.InteractionService
}

// NewInteractionHandler 创建互动处理器
func NewInteractionHandler(interactionService *service.InteractionService) *InteractionHandler {
	return &InteractionHandler{interactionService: interactionService}
}

// SendInteraction POST /interact
func (h *InteractionHandler) SendInteraction(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req model.SendInteractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请提供 receiver_id, action_emoji, action_label, action_push_text")
		return
	}

	if userID == req.ReceiverID {
		response.ParamError(c, "不能和自己互动")
		return
	}

	if err := h.interactionService.SendInteraction(c.Request.Context(), userID, &req); err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "互动成功", nil)
}
