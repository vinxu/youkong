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
