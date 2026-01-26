package model

import "time"

type WechatUser struct {
	ID        string    `db:"id" json:"id"`
	UserID    string    `db:"user_id" json:"userId"`
	OpenID    string    `db:"openid" json:"openid"`
	UnionID   string    `db:"unionid" json:"unionid,omitempty"`
	Nickname  string    `db:"nickname" json:"nickname,omitempty"`
	Avatar    string    `db:"avatar" json:"avatar,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}
