package model

import (
	"database/sql"
	"encoding/json"
	"time"
)

type MessageType string

const (
	MessageTypeText            MessageType = "TEXT"
	MessageTypeAvailabilityCard MessageType = "AVAILABILITY_CARD"
	MessageTypeConfirmRequest  MessageType = "CONFIRM_REQUEST"
	MessageTypeConfirmResponse MessageType = "CONFIRM_RESPONSE"
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
	Metadata       json.RawMessage `db:"metadata" json:"metadata,omitempty"`
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
	Metadata  json.RawMessage `json:"metadata,omitempty"`
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
