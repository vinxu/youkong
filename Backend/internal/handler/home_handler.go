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

// GeneratePoster 生成分享海报
// POST /api/v1/home/poster
// TODO: 实现海报生成逻辑
func (h *HomeHandler) GeneratePoster(c *gin.Context) {
	// userID := middleware.GetUserID(c)

	// 暂时返回未实现
	response.Error(c, response.CodeInternalError, "海报生成功能暂未实现，将在 Phase 4 完成")
}
