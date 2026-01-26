package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"youkong/internal/middleware"
	"youkong/internal/pkg/poster"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

type InvitationHandler struct {
	invitationService *service.InvitationService
	posterGenerator   *poster.Generator
}

func NewInvitationHandler(invitationService *service.InvitationService, posterGenerator *poster.Generator) *InvitationHandler {
	return &InvitationHandler{
		invitationService: invitationService,
		posterGenerator:   posterGenerator,
	}
}

type CreateInvitationRequest struct {
	CircleID    string `json:"circleId"`
	MaxUses     int    `json:"maxUses"`
	ExpiresDays int    `json:"expiresDays"`
}

func (h *InvitationHandler) CreateInvitation(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	result, err := h.invitationService.Create(c.Request.Context(), userID, &service.CreateInvitationRequest{
		CircleID:    req.CircleID,
		MaxUses:     req.MaxUses,
		ExpiresDays: req.ExpiresDays,
	})
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *InvitationHandler) GetMyInvitations(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	result, err := h.invitationService.GetMyInvitations(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *InvitationHandler) GetInvitationByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.ParamError(c, "邀请码不能为空")
		return
	}

	result, err := h.invitationService.GetByCode(c.Request.Context(), code)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}
	if result == nil {
		response.NotFound(c, "邀请不存在")
		return
	}

	response.Success(c, result)
}

func (h *InvitationHandler) DisableInvitation(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	id := c.Param("id")
	if id == "" {
		response.ParamError(c, "邀请ID不能为空")
		return
	}

	if err := h.invitationService.Disable(c.Request.Context(), id, userID); err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "邀请已禁用"})
}

func (h *InvitationHandler) AcceptInvitation(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	code := c.Param("code")
	if code == "" {
		response.ParamError(c, "邀请码不能为空")
		return
	}

	joinedCircle, err := h.invitationService.Accept(c.Request.Context(), code, userID)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":      "已接受邀请",
		"joinedCircle": joinedCircle,
	})
}

func (h *InvitationHandler) GetInvitationDetail(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	id := c.Param("id")
	if id == "" {
		response.ParamError(c, "邀请ID不能为空")
		return
	}

	result, err := h.invitationService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}
	if result == nil {
		response.NotFound(c, "邀请不存在")
		return
	}

	response.Success(c, result)
}

func (h *InvitationHandler) GetPoster(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	id := c.Param("id")
	if id == "" {
		response.ParamError(c, "邀请ID不能为空")
		return
	}

	invitation, err := h.invitationService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}
	if invitation == nil {
		response.NotFound(c, "邀请不存在")
		return
	}

	// 验证是否是邀请创建者
	if invitation.Inviter == nil || invitation.Inviter.ID != userID {
		response.Forbidden(c, "无权限获取此海报")
		return
	}

	// 生成海报
	data := &poster.PosterData{
		InviteCode: invitation.Code,
		InviteURL:  invitation.InviteURL,
	}
	if invitation.Inviter != nil {
		data.InviterNickname = invitation.Inviter.Nickname
		data.InviterAvatar = invitation.Inviter.Avatar
	}
	if invitation.Circle != nil {
		data.CircleName = invitation.Circle.Name
		data.CircleEmoji = invitation.Circle.Emoji
	}

	posterBytes, err := h.posterGenerator.GeneratePoster(data)
	if err != nil {
		response.Error(c, response.CodeInternalError, "生成海报失败: "+err.Error())
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=\"invite_poster.png\"")
	c.Data(http.StatusOK, "image/png", posterBytes)
}

func (h *InvitationHandler) GetQRCode(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	id := c.Param("id")
	if id == "" {
		response.ParamError(c, "邀请ID不能为空")
		return
	}

	invitation, err := h.invitationService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}
	if invitation == nil {
		response.NotFound(c, "邀请不存在")
		return
	}

	// 生成二维码
	qrBytes, err := h.posterGenerator.GenerateQRCode(invitation.InviteURL, 300)
	if err != nil {
		response.Error(c, response.CodeInternalError, "生成二维码失败: "+err.Error())
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=\"invite_qrcode.png\"")
	c.Data(http.StatusOK, "image/png", qrBytes)
}
