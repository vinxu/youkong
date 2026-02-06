package model

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"time"
)

// NullRawMessage 支持 NULL 的 json.RawMessage
type NullRawMessage json.RawMessage

// Scan 实现 sql.Scanner 接口
func (n *NullRawMessage) Scan(value interface{}) error {
	if value == nil {
		*n = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		// 如果是字符串 "null"，视为 nil
		if string(v) == "null" {
			*n = nil
			return nil
		}
		*n = NullRawMessage(v)
	case string:
		// 如果是字符串 "null"，视为 nil
		if v == "null" {
			*n = nil
			return nil
		}
		*n = NullRawMessage(v)
	}
	return nil
}

// Value 实现 driver.Valuer 接口
func (n NullRawMessage) Value() (driver.Value, error) {
	if n == nil {
		return nil, nil
	}
	return []byte(n), nil
}

// MarshalJSON 实现 json.Marshaler
func (n NullRawMessage) MarshalJSON() ([]byte, error) {
	if n == nil || len(n) == 0 {
		return []byte("null"), nil
	}
	// 验证是否为有效 JSON
	if !json.Valid(n) {
		// 如果不是有效 JSON，返回 null
		return []byte("null"), nil
	}
	return json.RawMessage(n).MarshalJSON()
}

type MessageType string

const (
	MessageTypeText            MessageType = "TEXT"
	MessageTypeAvailabilityCard MessageType = "AVAILABILITY_CARD"
	MessageTypeConfirmRequest  MessageType = "CONFIRM_REQUEST"
	MessageTypeConfirmResponse MessageType = "CONFIRM_RESPONSE"
	MessageTypeScheduleInvite  MessageType = "SCHEDULE_INVITE" // 日程邀请卡片
)

type Conversation struct {
	ID            string       `db:"id" json:"id"`
	User1ID       string       `db:"user1_id" json:"user1Id"`
	User2ID       string       `db:"user2_id" json:"user2Id"`
	LastMessageAt sql.NullTime `db:"last_message_at" json:"lastMessageAt,omitempty"`
	CreatedAt     time.Time    `db:"created_at" json:"createdAt"`
}

type Message struct {
	ID             string          `db:"id" json:"id"`
	ConversationID string          `db:"conversation_id" json:"conversationId"`
	SenderID       string          `db:"sender_id" json:"senderId"`
	Type           MessageType     `db:"type" json:"type"`
	Content        sql.NullString  `db:"content" json:"content,omitempty"`
	Metadata       NullRawMessage  `db:"metadata" json:"metadata,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"createdAt"`
	ReadAt         sql.NullTime    `db:"read_at" json:"readAt,omitempty"`
}

type ConversationResponse struct {
	ID          string       `json:"id"`
	Partner     UserProfile  `json:"partner"`
	LastMessage *MessageResponse `json:"lastMessage,omitempty"`
	UnreadCount int          `json:"unreadCount"`
	CreatedAt   time.Time    `json:"createdAt"`
}

type MessageResponse struct {
	ID        string          `json:"id"`
	Sender    UserProfile     `json:"sender"`
	Type      MessageType     `json:"type"`
	Content   string          `json:"content,omitempty"`
	Metadata  NullRawMessage  `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
	IsRead    bool            `json:"isRead"`
}

func (m *Message) ToResponse(sender *User) *MessageResponse {
	resp := &MessageResponse{
		ID:        m.ID,
		Sender:    *sender.ToProfile(),
		Type:      m.Type,
		Metadata:  m.Metadata,
		CreatedAt: m.CreatedAt,
		IsRead:    m.ReadAt.Valid,
	}
	if m.Content.Valid {
		resp.Content = m.Content.String
	}
	return resp
}
