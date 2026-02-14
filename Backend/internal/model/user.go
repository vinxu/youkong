package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID           string     `db:"id" json:"id"`
	Phone        string     `db:"phone" json:"phone,omitempty"`
	PhoneHash    string     `db:"phone_hash" json:"-"`
	Nickname     string     `db:"nickname" json:"nickname"`
	Avatar       string     `db:"avatar" json:"avatar,omitempty"`
	City         *string    `db:"city" json:"city,omitempty"`
	WechatBound  bool       `db:"wechat_bound" json:"wechatBound"`
	CreatedAt    time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updatedAt"`
	LastActiveAt *time.Time `db:"last_active_at" json:"lastActiveAt,omitempty"`
}

// UserPermissions 用户权限状态
type UserPermissions struct {
	ScreenTime bool `json:"screen_time"` // 屏幕使用时间权限
	Location   bool `json:"location"`    // 地理位置权限
	Contacts   bool `json:"contacts"`    // 通讯录权限
}

// UserProfile 用户简化信息
type UserProfile struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar,omitempty"`
}

// ToProfile 转换为简化信息
func (u *User) ToProfile() *UserProfile {
	return &UserProfile{
		ID:       u.ID,
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
	}
}

// GetPhone 获取手机号
func (u *User) GetPhone() string {
	return u.Phone
}

// GetAvatar 获取头像URL
func (u *User) GetAvatar() string {
	return u.Avatar
}
