package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"youkong/internal/middleware"
	"youkong/internal/model"
	"youkong/internal/pkg/poster"
	"youkong/internal/pkg/response"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
	"youkong/internal/service"
)

type UserHandler struct {
	userService          *service.UserService
	posterGenerator      *poster.Generator
	inviteBaseURL        string
	messageRepo          *repository.MessageRepository
	userSettingsRepo     *repository.UserSettingsRepository
	scheduleRepo         *repository.ScheduleRepository
	friendshipRepo       *repository.FriendshipRepository
	redisClient          *tencent.RedisClient
}

func NewUserHandler(userService *service.UserService, posterGenerator *poster.Generator, inviteBaseURL string, messageRepo *repository.MessageRepository, userSettingsRepo *repository.UserSettingsRepository, scheduleRepo *repository.ScheduleRepository, friendshipRepo *repository.FriendshipRepository) *UserHandler {
	return &UserHandler{
		userService:          userService,
		posterGenerator:      posterGenerator,
		inviteBaseURL:        inviteBaseURL,
		messageRepo:          messageRepo,
		userSettingsRepo:     userSettingsRepo,
		scheduleRepo:         scheduleRepo,
		friendshipRepo:       friendshipRepo,
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

	// 获取时刻表偏好（包含 show_city）
	showCity := true // 默认显示城市
	if h.scheduleRepo != nil {
		pref, err := h.scheduleRepo.GetUserPreference(c.Request.Context(), userID)
		if err == nil && pref != nil {
			showCity = pref.ShowCity
		}
	}

	// 查询 AI 就绪详情
	aiReadyDetails := h.buildAIReadyDetails(c, userID)

	response.Success(c, gin.H{
		"auto_predict_enabled": settings.AutoPredictEnabled,
		"show_city":            showCity,
		"ai_ready_details":     aiReadyDetails,
	})
}

// buildAIReadyDetails 构建 AI 就绪详情（查询好友数和行程数）
func (h *UserHandler) buildAIReadyDetails(c *gin.Context, userID string) gin.H {
	hasInvitedFriend := false
	hasVoiceSchedule := false

	// 查询好友数
	if h.friendshipRepo != nil {
		count, err := h.friendshipRepo.CountFriends(c.Request.Context(), userID)
		if err == nil && count > 0 {
			hasInvitedFriend = true
		}
	}

	// 查询行程数
	if h.scheduleRepo != nil {
		schedules, err := h.scheduleRepo.GetUserScheduleHistory(c.Request.Context(), userID, "", 1)
		if err == nil && len(schedules) > 0 {
			hasVoiceSchedule = true
		}
	}

	return gin.H{
		"has_invited_friend": hasInvitedFriend,
		"has_voice_schedule": hasVoiceSchedule,
	}
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
		ShowCity           *bool `json:"show_city"`
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

	// 处理 show_city 设置（在 user_schedule_preferences 表中）
	showCity := true // 默认值
	if h.scheduleRepo != nil {
		// 获取当前偏好
		pref, _ := h.scheduleRepo.GetUserPreference(c.Request.Context(), userID)
		if pref == nil {
			pref = &model.UserSchedulePreference{
				UserID:            userID,
				HidePastEvents:    false,
				DefaultVisibility: "all_friends",
				ShowCity:          true,
			}
		}

		// 更新 show_city
		if req.ShowCity != nil {
			pref.ShowCity = *req.ShowCity
		}
		showCity = pref.ShowCity

		// 保存偏好
		if err := h.scheduleRepo.UpsertUserPreference(c.Request.Context(), pref); err != nil {
			response.InternalError(c, "保存城市显示设置失败: "+err.Error())
			return
		}
	}

	aiReadyDetails := h.buildAIReadyDetails(c, userID)

	response.Success(c, gin.H{
		"auto_predict_enabled": settings.AutoPredictEnabled,
		"show_city":            showCity,
		"ai_ready_details":     aiReadyDetails,
	})
}

func (h *UserHandler) SetRedisClient(rc *tencent.RedisClient) {
	h.redisClient = rc
}

// CheckAppVersion 检查 App 版本更新
// GET /api/v1/app/version?platform=ios&current_version=1.0.0
func (h *UserHandler) CheckAppVersion(c *gin.Context) {
	currentVersion := c.Query("current_version")
	if currentVersion == "" {
		response.ParamError(c, "current_version 参数不能为空")
		return
	}

	if h.redisClient == nil {
		response.InternalError(c, "Redis 未初始化")
		return
	}

	// 从 Redis Hash 读取版本配置
	config, err := h.redisClient.HGetAll(c.Request.Context(), "app:version:config")
	if err != nil || len(config) == 0 {
		// Redis 中无配置，返回无需更新
		response.Success(c, gin.H{
			"latest_version": currentVersion,
			"min_version":    currentVersion,
			"update_url":     "",
			"changelog":      "",
			"force_update":   false,
		})
		return
	}

	latestVersion := config["latest_version"]
	minVersion := config["min_version"]
	updateURL := config["update_url"]
	changelog := config["changelog"]

	if latestVersion == "" {
		latestVersion = currentVersion
	}
	if minVersion == "" {
		minVersion = "0.0.0"
	}

	// 判断是否强制更新: current_version < min_version
	forceUpdate := compareVersions(currentVersion, minVersion) < 0

	response.Success(c, gin.H{
		"latest_version": latestVersion,
		"min_version":    minVersion,
		"update_url":     updateURL,
		"changelog":      changelog,
		"force_update":   forceUpdate,
	})
}

// compareVersions 比较语义版本号
// 返回: -1 (a < b), 0 (a == b), 1 (a > b)
func compareVersions(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var numA, numB int
		if i < len(partsA) {
			numA, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			numB, _ = strconv.Atoi(partsB[i])
		}
		if numA < numB {
			return -1
		}
		if numA > numB {
			return 1
		}
	}
	return 0
}
