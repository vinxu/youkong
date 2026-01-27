package model

import (
	"time"
)

// Platform 设备平台
type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
)

// DeviceToken 设备推送令牌
type DeviceToken struct {
	ID         int64     `db:"id" json:"id"`
	UserID     string    `db:"user_id" json:"userId"`
	Token      string    `db:"token" json:"token"`
	Platform   Platform  `db:"platform" json:"platform"`
	DeviceName string    `db:"device_name" json:"deviceName,omitempty"`
	AppVersion string    `db:"app_version" json:"appVersion,omitempty"`
	IsActive   bool      `db:"is_active" json:"isActive"`
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt  time.Time `db:"updated_at" json:"updatedAt"`
}

// RegisterDeviceTokenRequest 注册设备 Token 请求
type RegisterDeviceTokenRequest struct {
	Token      string `json:"token" binding:"required"`
	Platform   string `json:"platform" binding:"required,oneof=ios android"`
	DeviceName string `json:"deviceName"`
	AppVersion string `json:"appVersion"`
}

// UnregisterDeviceTokenRequest 注销设备 Token 请求
type UnregisterDeviceTokenRequest struct {
	Token string `json:"token" binding:"required"`
}
