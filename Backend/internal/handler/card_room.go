package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

// CardRoomHandler 卡片房间处理器
type CardRoomHandler struct {
	cardRoomService *service.CardRoomService
}

// NewCardRoomHandler 创建卡片房间处理器
func NewCardRoomHandler(cardRoomService *service.CardRoomService) *CardRoomHandler {
	return &CardRoomHandler{cardRoomService: cardRoomService}
}

// JoinCard POST /api/v1/card/join
func (h *CardRoomHandler) JoinCard(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req model.JoinCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请提供 owner_id")
		return
	}

	resp, err := h.cardRoomService.JoinCard(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, resp)
}

// GetRoom GET /api/v1/card/room/:room_id
func (h *CardRoomHandler) GetRoom(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		response.ParamError(c, "缺少 room_id")
		return
	}

	resp, err := h.cardRoomService.GetRoom(c.Request.Context(), userID, roomID)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, resp)
}

// GetRoomMessages GET /api/v1/card/room/:room_id/messages
func (h *CardRoomHandler) GetRoomMessages(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		response.ParamError(c, "缺少 room_id")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	resp, err := h.cardRoomService.GetRoomMessages(c.Request.Context(), userID, roomID, limit, offset)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, resp)
}

// SendRoomMessage POST /api/v1/card/room/:room_id/messages
func (h *CardRoomHandler) SendRoomMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		response.ParamError(c, "缺少 room_id")
		return
	}

	var req model.SendRoomMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请提供消息内容")
		return
	}

	resp, err := h.cardRoomService.SendRoomMessage(c.Request.Context(), userID, roomID, &req)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, resp)
}
