package model

import "time"

// CardRoom 卡片房间（每人每天一个）
type CardRoom struct {
	ID         string    `json:"id" db:"id"`
	OwnerID    string    `json:"owner_id" db:"owner_id"`
	Emoji      string    `json:"emoji" db:"emoji"`
	StatusText string    `json:"status_text" db:"status_text"`
	RoomDate   string    `json:"room_date" db:"room_date"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// CardRoomMember 房间成员
type CardRoomMember struct {
	RoomID   string    `json:"room_id" db:"room_id"`
	UserID   string    `json:"user_id" db:"user_id"`
	JoinedAt time.Time `json:"joined_at" db:"joined_at"`
}

// CardRoomMemberBrief 成员简要信息（用于 API 返回）
type CardRoomMemberBrief struct {
	UserID        string `json:"user_id"`
	Nickname      string `json:"nickname"`
	Avatar        string `json:"avatar,omitempty"`
	Gender        string `json:"gender,omitempty"`
	RiveCharacter string `json:"rive_character,omitempty"`
}

// JoinCardRequest 加入卡片房间请求
type JoinCardRequest struct {
	OwnerID    string `json:"owner_id" binding:"required"`
	Emoji      string `json:"emoji"`       // 当前卡片显示的 emoji
	StatusText string `json:"status_text"` // 当前卡片显示的状态文本
}

// JoinCardResponse 加入卡片房间响应
type JoinCardResponse struct {
	RoomID  string                `json:"room_id"`
	Members []CardRoomMemberBrief `json:"members"`
}

// CardRoomDetail 房间详情
type CardRoomDetail struct {
	ID         string                `json:"id"`
	OwnerID    string                `json:"owner_id"`
	Emoji      string                `json:"emoji"`
	StatusText string                `json:"status_text"`
	RoomDate   string                `json:"room_date"`
	Members    []CardRoomMemberBrief `json:"members"`
	CreatedAt  time.Time             `json:"created_at"`
}

// CardRoomMessage 房间消息
type CardRoomMessage struct {
	ID        string    `json:"id" db:"id"`
	RoomID    string    `json:"room_id" db:"room_id"`
	SenderID  string    `json:"sender_id" db:"sender_id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// SendRoomMessageRequest 发送房间消息请求
type SendRoomMessageRequest struct {
	Content string `json:"content" binding:"required"`
}
