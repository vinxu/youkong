package handler

import (
	"github.com/gin-gonic/gin"
	"youkong/internal/middleware"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

// HomeHandler 首页处理器
type HomeHandler struct {
	homeService   *service.HomeService
	posterService *service.PosterService
}

// NewHomeHandler 创建首页处理器
func NewHomeHandler(homeService *service.HomeService, posterService *service.PosterService) *HomeHandler {
	return &HomeHandler{
		homeService:   homeService,
		posterService: posterService,
	}
}

// GetGrid 获取宫格数据
// GET /api/v1/home/grid
func (h *HomeHandler) GetGrid(c *gin.Context) {
	userID := middleware.GetUserID(c)

	gridData, err := h.homeService.GetGridData(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gridData)
}

// GeneratePosterRequest 生成海报请求
type GeneratePosterRequest struct {
	UserIDs    []string `json:"user_ids"`
	InviteCode string   `json:"invite_code,omitempty"`
}

// GeneratePoster 生成分享海报
// POST /api/v1/home/poster
func (h *HomeHandler) GeneratePoster(c *gin.Context) {
	var req GeneratePosterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "参数错误: "+err.Error())
		return
	}

	// 限制用户数量（最多16个）
	if len(req.UserIDs) > 16 {
		req.UserIDs = req.UserIDs[:16]
	}

	// 生成海报
	posterPath, err := h.posterService.GeneratePoster(req.UserIDs, req.InviteCode)
	if err != nil {
		response.InternalError(c, "生成海报失败: "+err.Error())
		return
	}

	// 返回海报路径（实际应该上传到 COS 并返回 URL）
	// TODO: 上传到腾讯云 COS
	response.Success(c, gin.H{
		"poster_url": posterPath,
		"message":    "海报已生成（本地路径，生产环境需上传到 COS）",
	})
}
