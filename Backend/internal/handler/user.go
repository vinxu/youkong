package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"youkong/internal/middleware"
	"youkong/internal/pkg/poster"
	"youkong/internal/pkg/response"
	"youkong/internal/repository"
	"youkong/internal/service"
)

type UserHandler struct {
	userService          *service.UserService
	posterGenerator      *poster.Generator
	inviteBaseURL        string
	messageRepo          *repository.MessageRepository
	userSettingsRepo     *repository.UserSettingsRepository
}

func NewUserHandler(userService *service.UserService, posterGenerator *poster.Generator, inviteBaseURL string, messageRepo *repository.MessageRepository, userSettingsRepo *repository.UserSettingsRepository) *UserHandler {
	return &UserHandler{
		userService:          userService,
		posterGenerator:      posterGenerator,
		inviteBaseURL:        inviteBaseURL,
		messageRepo:          messageRepo,
		userSettingsRepo:     userSettingsRepo,
	}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if user == nil {
		response.NotFound(c, "用户不存在")
		return
	}

	response.Success(c, user)
}

type UpdateUserRequest struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	userID := middleware.GetUserID(c)
	user, err := h.userService.Update(c.Request.Context(), userID, req.Nickname, req.Avatar)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, user)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.ParamError(c, "用户ID不能为空")
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if user == nil {
		response.NotFound(c, "用户不存在")
		return
	}

	response.Success(c, user.ToProfile())
}

func (h *UserHandler) SearchUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		response.ParamError(c, "搜索关键词不能为空")
		return
	}

	users, err := h.userService.Search(c.Request.Context(), keyword, 20)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	profiles := make([]interface{}, 0, len(users))
	for _, u := range users {
		profiles = append(profiles, u.ToProfile())
	}

	response.Success(c, profiles)
}

// GetMyPoster 获取我的邀请海报
// GET /api/v1/users/me/poster
func (h *UserHandler) GetMyPoster(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if user == nil {
		response.NotFound(c, "用户不存在")
		return
	}

	// 生成固定的邀请码（用户ID前8位，去掉横杠）
	inviteCode := strings.ReplaceAll(userID, "-", "")[:8]
	inviteURL := h.inviteBaseURL + inviteCode

	// 生成海报
	data := &poster.PosterData{
		InviterNickname: user.Nickname,
		InviterAvatar:   user.Avatar,
		InviteCode:      inviteCode,
		InviteURL:       inviteURL,
	}

	posterBytes, err := h.posterGenerator.GeneratePoster(data)
	if err != nil {
		response.Error(c, response.CodeInternalError, "生成海报失败: "+err.Error())
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=\"my_invite_poster.png\"")
	c.Data(http.StatusOK, "image/png", posterBytes)
}

// GetMyInviteInfo 获取我的邀请信息
// GET /api/v1/users/me/invite
func (h *UserHandler) GetMyInviteInfo(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if user == nil {
		response.NotFound(c, "用户不存在")
		return
	}

	// 生成固定的邀请码
	inviteCode := strings.ReplaceAll(userID, "-", "")[:8]
	inviteURL := h.inviteBaseURL + inviteCode

	response.Success(c, gin.H{
		"code":      inviteCode,
		"inviteUrl": inviteURL,
	})
}

// GetBadgeCount 获取未读消息数（用于 App Badge）
// GET /api/v1/users/me/badge
func (h *UserHandler) GetBadgeCount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	count, err := h.messageRepo.GetTotalUnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "获取未读数失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"count": count,
	})
}

// GetSettings 获取用户设置
// GET /api/v1/users/settings
func (h *UserHandler) GetSettings(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	settings, err := h.userSettingsRepo.Get(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "获取设置失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"auto_predict_enabled": settings.AutoPredictEnabled,
	})
}

// UpdateSettings 更新用户设置
// PUT /api/v1/users/settings
func (h *UserHandler) UpdateSettings(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req struct {
		AutoPredictEnabled *bool `json:"auto_predict_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误")
		return
	}

	// 获取当前设置
	settings, err := h.userSettingsRepo.Get(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "获取设置失败: "+err.Error())
		return
	}

	// 更新字段
	if req.AutoPredictEnabled != nil {
		settings.AutoPredictEnabled = *req.AutoPredictEnabled
	}
	settings.UserID = userID

	// 保存
	if err := h.userSettingsRepo.Upsert(c.Request.Context(), settings); err != nil {
		response.InternalError(c, "保存设置失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"auto_predict_enabled": settings.AutoPredictEnabled,
	})
}
