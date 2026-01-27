package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"youkong/internal/pkg/jwt"
	"youkong/internal/pkg/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源（生产环境应限制）
	},
}

// WSHandler WebSocket 处理器
type WSHandler struct {
	wsManager  *ws.Manager
	jwtManager *jwt.Manager
}

// NewWSHandler 创建 WebSocket 处理器
func NewWSHandler(wsManager *ws.Manager, jwtManager *jwt.Manager) *WSHandler {
	return &WSHandler{
		wsManager:  wsManager,
		jwtManager: jwtManager,
	}
}

// HandleWS 处理 WebSocket 连接
// GET /ws?token=<jwt_token>
func (h *WSHandler) HandleWS(c *gin.Context) {
	// 从 query 参数获取 token
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	// 验证 token
	claims, err := h.jwtManager.ParseToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// 升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// 处理连接（阻塞直到断开）
	h.wsManager.HandleConnection(claims.UserID, conn)
}

// GetWSManager 获取 WebSocket 管理器（供其他服务使用）
func (h *WSHandler) GetWSManager() *ws.Manager {
	return h.wsManager
}
