package handler

import (
	"github.com/gin-gonic/gin"

	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

// BookingHandler 预约处理器
type BookingHandler struct {
	bookingService *service.BookingService
}

// NewBookingHandler 创建预约处理器
func NewBookingHandler(bookingService *service.BookingService) *BookingHandler {
	return &BookingHandler{bookingService: bookingService}
}

// RespondToBooking POST /bookings/:id/respond
func (h *BookingHandler) RespondToBooking(c *gin.Context) {
	userID := middleware.GetUserID(c)
	bookingID := c.Param("id")

	var req model.BookingRespondRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请提供 action: accept 或 decline")
		return
	}

	booking, err := h.bookingService.RespondToBooking(c.Request.Context(), bookingID, userID, req.Action)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	msg := "已接受邀请，约会已加入你的行程"
	if req.Action == "decline" {
		msg = "已拒绝邀请"
	}

	response.SuccessWithMessage(c, msg, gin.H{
		"booking":          booking,
		"schedule_updated": req.Action == "accept",
	})
}

// GetBookings GET /bookings
func (h *BookingHandler) GetBookings(c *gin.Context) {
	userID := middleware.GetUserID(c)

	bookings, err := h.bookingService.GetUserBookings(c.Request.Context(), userID, 20)
	if err != nil {
		response.Error(c, response.CodeInternalError, "获取预约列表失败")
		return
	}

	response.Success(c, gin.H{
		"bookings": bookings,
		"total":    len(bookings),
	})
}

// GetBooking GET /bookings/:id
func (h *BookingHandler) GetBooking(c *gin.Context) {
	userID := middleware.GetUserID(c)
	bookingID := c.Param("id")

	detail, err := h.bookingService.GetBooking(c.Request.Context(), bookingID, userID)
	if err != nil {
		response.Error(c, response.CodeNotFound, err.Error())
		return
	}

	response.Success(c, detail)
}

// CancelBooking DELETE /bookings/:id
func (h *BookingHandler) CancelBooking(c *gin.Context) {
	userID := middleware.GetUserID(c)
	bookingID := c.Param("id")

	if err := h.bookingService.CancelBooking(c.Request.Context(), bookingID, userID); err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "预约已取消", nil)
}
