package model

import "time"

// Booking 社交预约
type Booking struct {
	ID           string    `db:"id" json:"id"`
	CreatorID    string    `db:"creator_id" json:"creator_id"`
	Title        string    `db:"title" json:"title"`
	Emoji        string    `db:"emoji" json:"emoji"`
	ScheduleDate time.Time `db:"schedule_date" json:"schedule_date"`
	StartTime    string    `db:"start_time" json:"start_time"`
	EndTime      string    `db:"end_time" json:"end_time"`
	Location     string    `db:"location" json:"location,omitempty"`
	Note         string    `db:"note" json:"note,omitempty"`
	Status       string    `db:"status" json:"status"` // pending/confirmed/cancelled
	MessageID    string    `db:"message_id" json:"message_id,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// BookingParticipant 预约参与者
type BookingParticipant struct {
	ID          int64      `db:"id" json:"id"`
	BookingID   string     `db:"booking_id" json:"booking_id"`
	UserID      string     `db:"user_id" json:"user_id"`
	Status      string     `db:"status" json:"status"` // pending/accepted/declined
	RespondedAt *time.Time `db:"responded_at" json:"responded_at,omitempty"`
	Notified    bool       `db:"notified" json:"notified"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}

// BookingDetail 预约详情（含参与者信息）
type BookingDetail struct {
	Booking
	CreatorName    string               `json:"creator_name"`
	Participants   []BookingParticipantInfo `json:"participants"`
}

// BookingParticipantInfo 参与者信息（含用户基础信息）
type BookingParticipantInfo struct {
	UserID      string     `json:"user_id"`
	Nickname    string     `json:"nickname"`
	Avatar      string     `json:"avatar,omitempty"`
	Status      string     `json:"status"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
}

// BookingRespondRequest 预约响应请求
type BookingRespondRequest struct {
	Action string `json:"action" binding:"required,oneof=accept decline"`
}
