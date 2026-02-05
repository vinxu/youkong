package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// PredictionStatus 推测任务状态
type PredictionStatus string

const (
	PredictionPending    PredictionStatus = "pending"
	PredictionProcessing PredictionStatus = "processing"
	PredictionCompleted  PredictionStatus = "completed"
	PredictionFailed     PredictionStatus = "failed"
	PredictionCancelled  PredictionStatus = "cancelled"
)

// PredictionTask AI 状态推测任务
type PredictionTask struct {
	ID               string             `db:"id" json:"id"`
	UserID           string             `db:"user_id" json:"user_id"`
	Status           PredictionStatus   `db:"status" json:"status"`
	StartTime        time.Time          `db:"start_time" json:"start_time"`
	EndTime          time.Time          `db:"end_time" json:"end_time"`
	TargetDate       time.Time          `db:"target_date" json:"target_date"`
	PredictedItems   ScheduleItems      `db:"predicted_items" json:"predicted_items,omitempty"`
	ExistingItems    ScheduleItems      `db:"existing_items" json:"existing_items,omitempty"`
	MergedPreview    ScheduleItems      `db:"merged_preview" json:"merged_preview,omitempty"`
	ReasoningContent string             `db:"reasoning_content" json:"reasoning_content,omitempty"`
	ReasoningSteps   ReasoningStepsJSON `db:"reasoning_steps" json:"reasoning_steps,omitempty"`
	UserContext      UserContextJSON    `db:"user_context" json:"user_context,omitempty"`
	ErrorMessage     string             `db:"error_message" json:"error_message,omitempty"`
	Progress         int                `db:"progress" json:"progress"`
	CreatedAt        time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time          `db:"updated_at" json:"updated_at"`
	CompletedAt      *time.Time         `db:"completed_at" json:"completed_at,omitempty"`
	ConfirmedAt      *time.Time         `db:"confirmed_at" json:"confirmed_at,omitempty"`
}

// ReasoningStepsJSON 推理步骤 JSON 数组类型
type ReasoningStepsJSON []string

// Value 实现 driver.Valuer 接口
func (r ReasoningStepsJSON) Value() (driver.Value, error) {
	if r == nil {
		return "[]", nil
	}
	return json.Marshal(r)
}

// Scan 实现 sql.Scanner 接口
func (r *ReasoningStepsJSON) Scan(value interface{}) error {
	if value == nil {
		*r = []string{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("invalid type for ReasoningStepsJSON")
	}

	return json.Unmarshal(data, r)
}

// UserContextJSON 用户上下文 JSON 类型
type UserContextJSON struct {
	ProfileSummary       string `json:"profile_summary,omitempty"`
	TodayScheduleSummary string `json:"today_schedule_summary,omitempty"`
	BehaviorInsights     string `json:"behavior_insights,omitempty"`
	TimePatterns         string `json:"time_patterns,omitempty"`
	LocationPrefs        string `json:"location_prefs,omitempty"`
	DeviceStateSummary   string `json:"device_state_summary,omitempty"`
}

// Value 实现 driver.Valuer 接口
func (u UserContextJSON) Value() (driver.Value, error) {
	return json.Marshal(u)
}

// Scan 实现 sql.Scanner 接口
func (u *UserContextJSON) Scan(value interface{}) error {
	if value == nil {
		*u = UserContextJSON{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("invalid type for UserContextJSON")
	}

	return json.Unmarshal(data, u)
}

// PredictionTaskResponse API 响应格式
type PredictionTaskResponse struct {
	ID               string           `json:"id"`
	Status           PredictionStatus `json:"status"`
	StartTime        time.Time        `json:"start_time"`
	EndTime          time.Time        `json:"end_time"`
	TargetDate       string           `json:"target_date"` // YYYY-MM-DD 格式
	PredictedItems   []ScheduleItem   `json:"predicted_items,omitempty"`
	ExistingItems    []ScheduleItem   `json:"existing_items,omitempty"`
	MergedPreview    []ScheduleItem   `json:"merged_preview,omitempty"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	ReasoningSteps   []string         `json:"reasoning_steps,omitempty"`
	ErrorMessage     string           `json:"error_message,omitempty"`
	Progress         int              `json:"progress"`
	CreatedAt        time.Time        `json:"created_at"`
	CompletedAt      *time.Time       `json:"completed_at,omitempty"`
	ConfirmedAt      *time.Time       `json:"confirmed_at,omitempty"`
}

// ToResponse 转换为 API 响应格式
func (t *PredictionTask) ToResponse() *PredictionTaskResponse {
	return &PredictionTaskResponse{
		ID:               t.ID,
		Status:           t.Status,
		StartTime:        t.StartTime,
		EndTime:          t.EndTime,
		TargetDate:       t.TargetDate.Format("2006-01-02"),
		PredictedItems:   t.PredictedItems,
		ExistingItems:    t.ExistingItems,
		MergedPreview:    t.MergedPreview,
		ReasoningContent: t.ReasoningContent,
		ReasoningSteps:   t.ReasoningSteps,
		ErrorMessage:     t.ErrorMessage,
		Progress:         t.Progress,
		CreatedAt:        t.CreatedAt,
		CompletedAt:      t.CompletedAt,
		ConfirmedAt:      t.ConfirmedAt,
	}
}

// StartPredictionRequest 开始推测请求
type StartPredictionRequest struct {
	TargetDate string `json:"target_date,omitempty"` // 可选，默认今天，格式 YYYY-MM-DD
}

// ConfirmPredictionRequest 确认推测请求
type ConfirmPredictionRequest struct {
	ModifiedItems []ScheduleItem `json:"modified_items,omitempty"` // 可选，用户修改后的时刻表
}
