package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MessageType WebSocket 消息类型
type MessageType string

const (
	MessageTypeNewMessage MessageType = "NEW_MESSAGE"
	MessageTypePing       MessageType = "PING"
	MessageTypePong       MessageType = "PONG"
)

// WSMessage WebSocket 消息格式
type WSMessage struct {
	Type MessageType `json:"type"`
	Data any         `json:"data,omitempty"`
}

// NewMessageData 新消息数据
type NewMessageData struct {
	ConversationID string `json:"conversation_id"`
	Message        any    `json:"message"`
}

// Manager WebSocket 连接管理器
type Manager struct {
	connections map[string]*websocket.Conn // userID -> conn
	mu          sync.RWMutex
}

// NewManager 创建 WebSocket 管理器
func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]*websocket.Conn),
	}
}

// Register 注册连接
func (m *Manager) Register(userID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 关闭旧连接
	if old, ok := m.connections[userID]; ok {
		old.Close()
	}

	m.connections[userID] = conn
}

// Unregister 注销连接
func (m *Manager) Unregister(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, ok := m.connections[userID]; ok {
		conn.Close()
		delete(m.connections, userID)
	}
}

// SendToUser 发送消息给指定用户
func (m *Manager) SendToUser(userID string, msg WSMessage) error {
	m.mu.RLock()
	conn, ok := m.connections[userID]
	m.mu.RUnlock()

	if !ok {
		return nil // 用户不在线，静默忽略
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// 设置写超时
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, data)
}

// SendNewMessage 发送新消息通知
func (m *Manager) SendNewMessage(userID, conversationID string, message any) error {
	return m.SendToUser(userID, WSMessage{
		Type: MessageTypeNewMessage,
		Data: NewMessageData{
			ConversationID: conversationID,
			Message:        message,
		},
	})
}

// Broadcast 广播消息给多个用户
func (m *Manager) Broadcast(userIDs []string, msg WSMessage) {
	for _, userID := range userIDs {
		m.SendToUser(userID, msg)
	}
}

// IsOnline 检查用户是否在线
func (m *Manager) IsOnline(userID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.connections[userID]
	return ok
}

// OnlineCount 在线用户数
func (m *Manager) OnlineCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// HandleConnection 处理 WebSocket 连接（心跳维护）
func (m *Manager) HandleConnection(userID string, conn *websocket.Conn) {
	m.Register(userID, conn)
	defer m.Unregister(userID)

	// 设置读取限制和超时
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 读取消息循环（主要处理 PING）
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// 处理心跳
		if msg.Type == MessageTypePing {
			m.SendToUser(userID, WSMessage{Type: MessageTypePong})
		}
	}
}
