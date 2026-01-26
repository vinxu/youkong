package handler

import (
	"github.com/gin-gonic/gin"
	"youkong/internal/middleware"
	"youkong/internal/pkg/response"
	"youkong/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
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
