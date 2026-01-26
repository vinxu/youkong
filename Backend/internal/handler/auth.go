package handler

import (
	"github.com/gin-gonic/gin"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

type AuthHandler struct {
	authService   *service.AuthService
	wechatService *service.WechatService
}

func NewAuthHandler(authService *service.AuthService, wechatService *service.WechatService) *AuthHandler {
	return &AuthHandler{
		authService:   authService,
		wechatService: wechatService,
	}
}

type SendSMSRequest struct {
	Phone string `json:"phone" binding:"required,len=11"`
}

func (h *AuthHandler) SendSMS(c *gin.Context) {
	var req SendSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "手机号格式错误")
		return
	}

	if err := h.authService.SendSMSCode(c.Request.Context(), req.Phone); err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "验证码已发送"})
}

type VerifySMSRequest struct {
	Phone string `json:"phone" binding:"required,len=11"`
	Code  string `json:"code" binding:"required,len=6"`
}

func (h *AuthHandler) VerifySMS(c *gin.Context) {
	var req VerifySMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	result, err := h.authService.VerifyAndLogin(c.Request.Context(), req.Phone, req.Code)
	if err != nil {
		response.Error(c, response.CodeUnauthorized, err.Error())
		return
	}

	response.Success(c, result)
}

type RefreshTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "token不能为空")
		return
	}

	newToken, err := h.authService.RefreshToken(c.Request.Context(), req.Token)
	if err != nil {
		response.TokenExpired(c)
		return
	}

	response.Success(c, gin.H{"token": newToken})
}

type WechatLoginRequest struct {
	Code       string `json:"code" binding:"required"`
	InviteCode string `json:"inviteCode"`
}

func (h *AuthHandler) WechatLogin(c *gin.Context) {
	if h.wechatService == nil {
		response.Error(c, response.CodeInternalError, "微信登录功能未启用")
		return
	}

	var req WechatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "微信授权码不能为空")
		return
	}

	result, err := h.wechatService.Login(c.Request.Context(), req.Code, req.InviteCode)
	if err != nil {
		response.Error(c, response.CodeUnauthorized, err.Error())
		return
	}

	response.Success(c, result)
}
