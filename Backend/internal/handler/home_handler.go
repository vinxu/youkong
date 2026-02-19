package handler

import (
	"github.com/gin-gonic/gin"
	"youkong/internal/middleware"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

// HomeHandler 首页处理器
type HomeHandler struct {
	homeService *service.HomeService
}

// NewHomeHandler 创建首页处理器
func NewHomeHandler(homeService *service.HomeService) *HomeHandler {
	return &HomeHandler{
		homeService: homeService,
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

// CacheGifURLs 批量缓存客户端解析的 GIF cos_url
// POST /api/v1/home/gif-cache
func (h *HomeHandler) CacheGifURLs(c *gin.Context) {
	var req struct {
		Items []service.GifCacheItem `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
		response.ParamError(c, "items 不能为空")
		return
	}
	cached := h.homeService.CacheGifURLs(c.Request.Context(), req.Items)
	response.Success(c, gin.H{"cached": cached})
}
