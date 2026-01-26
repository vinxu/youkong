package model

import (
	"database/sql"
	"time"
)

type FriendshipSource string

const (
	FriendshipSourceInvitation FriendshipSource = "INVITATION"
	FriendshipSourceSearch     FriendshipSource = "SEARCH"
	FriendshipSourceManual     FriendshipSource = "MANUAL"
	FriendshipSourceContacts   FriendshipSource = "CONTACTS"
	FriendshipSourceRequest    FriendshipSource = "REQUEST"
)

type Friendship struct {
	ID           string           `db:"id" json:"id"`
	UserID       string           `db:"user_id" json:"userId"`
	FriendID     string           `db:"friend_id" json:"friendId"`
	Source       FriendshipSource `db:"source" json:"source"`
	InvitationID sql.NullString   `db:"invitation_id" json:"-"`
	CreatedAt    time.Time        `db:"created_at" json:"createdAt"`
}

func (f *Friendship) GetInvitationID() string {
	if f.InvitationID.Valid {
		return f.InvitationID.String
	}
	return ""
}

type FriendInfo struct {
	User      *UserProfile `json:"user"`
	Source    string       `json:"source"`
	CreatedAt time.Time    `json:"createdAt"`
}

type FriendWithInvitation struct {
	User       *UserProfile `json:"user"`
	Source     string       `json:"source"`
	CircleName string       `json:"circleName,omitempty"`
	CreatedAt  time.Time    `json:"createdAt"`
}

// AddFriendByPhoneRequest 通过手机号加好友请求
type AddFriendByPhoneRequest struct {
	Phone string `json:"phone" binding:"required,len=11"`
}

// AddFriendByPhoneResponse 通过手机号加好友响应
type AddFriendByPhoneResponse struct {
	User  *UserProfile `json:"user"`
	Added bool         `json:"added"` // true=新添加, false=已是好友
}

// ==================== 好友请求相关 ====================

type FriendRequestStatus string

const (
	FriendRequestStatusPending   FriendRequestStatus = "PENDING"
	FriendRequestStatusAccepted  FriendRequestStatus = "ACCEPTED"
	FriendRequestStatusRejected  FriendRequestStatus = "REJECTED"
	FriendRequestStatusCancelled FriendRequestStatus = "CANCELLED"
)

// FriendRequest 好友请求
type FriendRequest struct {
	ID         string              `db:"id" json:"id"`
	FromUserID string              `db:"from_user_id" json:"fromUserId"`
	ToUserID   string              `db:"to_user_id" json:"toUserId"`
	Message    string              `db:"message" json:"message,omitempty"`
	Status     FriendRequestStatus `db:"status" json:"status"`
	CreatedAt  time.Time           `db:"created_at" json:"createdAt"`
	UpdatedAt  time.Time           `db:"updated_at" json:"updatedAt"`
}

// SendFriendRequestRequest 发送好友请求
type SendFriendRequestRequest struct {
	Phone   string `json:"phone" binding:"required,len=11"`
	Message string `json:"message"` // 附言，可选
}

// SendFriendRequestResponse 发送好友请求响应
type SendFriendRequestResponse struct {
	RequestID string       `json:"requestId"`
	User      *UserProfile `json:"user"`    // 目标用户信息
	Status    string       `json:"status"`  // PENDING / ALREADY_FRIENDS / ALREADY_REQUESTED
}

// FriendRequestInfo 好友请求详情（带用户信息）
type FriendRequestInfo struct {
	ID        string       `json:"id"`
	User      *UserProfile `json:"user"`              // 对方用户信息
	Message   string       `json:"message,omitempty"` // 附言
	Status    string       `json:"status"`
	CreatedAt time.Time    `json:"createdAt"`
}

// HandleFriendRequestRequest 处理好友请求
type HandleFriendRequestRequest struct {
	Accept bool `json:"accept"` // true=同意, false=拒绝
}
