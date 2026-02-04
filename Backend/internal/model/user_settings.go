package model

import "time"

// UserSettings 用户设置
type UserSettings struct {
	UserID             string    `db:"user_id" json:"user_id"`
	AutoPredictEnabled bool      `db:"auto_predict_enabled" json:"auto_predict_enabled"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`
}

// UserSettingsRequest 用户设置请求
type UserSettingsRequest struct {
	AutoPredictEnabled *bool `json:"auto_predict_enabled"`
}

// UserSettingsResponse 用户设置响应
type UserSettingsResponse struct {
	AutoPredictEnabled bool `json:"auto_predict_enabled"`
}
