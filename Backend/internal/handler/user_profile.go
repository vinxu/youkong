package handler

import (
	"net/http"
	"youkong/internal/model"
	"youkong/internal/pkg/response"
	"youkong/internal/service"

	"github.com/gin-gonic/gin"
)

type UserProfileHandler struct {
	profileService *service.UserProfileService
}

func NewUserProfileHandler(profileService *service.UserProfileService) *UserProfileHandler {
	return &UserProfileHandler{
		profileService: profileService,
	}
}

// 获取当前用户画像
// GET /api/v1/profile
func (h *UserProfileHandler) GetMyProfile(c *gin.Context) {
	userID := c.GetString("userID")

	profile, err := h.profileService.GetProfile(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取用户画像失败")
		return
	}

	if profile == nil {
		// 用户画像不存在，返回空对象
		response.Success(c, gin.H{
			"exists": false,
			"profile": nil,
		})
		return
	}

	response.Success(c, gin.H{
		"exists": true,
		"profile": profile,
	})
}

// 创建或更新用户画像
// PUT /api/v1/profile
func (h *UserProfileHandler) UpsertProfile(c *gin.Context) {
	userID := c.GetString("userID")

	var req model.UserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	profile, err := h.profileService.UpsertProfile(userID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "保存用户画像失败")
		return
	}

	response.Success(c, profile)
}

// 检查用户画像是否已完成
// GET /api/v1/profile/status
func (h *UserProfileHandler) GetProfileStatus(c *gin.Context) {
	userID := c.GetString("userID")

	isComplete, err := h.profileService.IsProfileComplete(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "检查用户画像失败")
		return
	}

	response.Success(c, gin.H{
		"is_complete": isComplete,
	})
}

// 删除用户画像
// DELETE /api/v1/profile
func (h *UserProfileHandler) DeleteProfile(c *gin.Context) {
	userID := c.GetString("userID")

	err := h.profileService.DeleteProfile(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "删除用户画像失败")
		return
	}

	response.Success(c, gin.H{
		"message": "用户画像已删除",
	})
}

// 简化保存用户画像（引导流程用）
// POST /api/v1/profile/simple
func (h *UserProfileHandler) SaveSimpleProfile(c *gin.Context) {
	userID := c.GetString("userID")

	var req model.SimpleProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	err := h.profileService.SaveSimpleProfile(userID, req.ProfileType, req.Gender, req.MBTI)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "保存用户画像失败")
		return
	}

	response.Success(c, gin.H{
		"message":      "画像已保存",
		"profile_type": req.ProfileType,
		"gender":       req.Gender,
		"mbti":         req.MBTI,
	})
}
