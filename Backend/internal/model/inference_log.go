package model

import "time"

// InferenceLog 推断日志
type InferenceLog struct {
	ID            int64     `json:"id" db:"id"`
	UserID        string    `json:"user_id" db:"user_id"`
	Timestamp     time.Time `json:"timestamp" db:"timestamp"`
	SensorData    string    `json:"sensor_data" db:"sensor_data"`       // JSON: 输入的传感器数据
	ToolsUsed     []string  `json:"tools_used" db:"-"`                  // 使用了哪些工具
	ToolsUsedJSON string    `json:"-" db:"tools_used"`                  // JSON 存储
	AskedUser     bool      `json:"asked_user" db:"asked_user"`         // 是否问了用户
	InitialResult string    `json:"initial_result" db:"initial_result"` // JSON: 初始推断结果
	FinalResult   string    `json:"final_result" db:"final_result"`     // JSON: 最终结果（修正后）
	UserCorrected bool      `json:"user_corrected" db:"user_corrected"` // 用户是否修正
	Confidence    string    `json:"confidence" db:"confidence"`         // 置信度
	Iterations      int       `json:"iterations" db:"iterations"`             // Agent 循环次数
	LatencyMs       int       `json:"latency_ms" db:"latency_ms"`             // 推断耗时(毫秒)
	TriggerSource   string    `json:"trigger_source" db:"trigger_source"`     // oneclick|auto_scheduler|sync
	ContextSections string    `json:"context_sections" db:"context_sections"` // JSON: 上下文段填充情况
	ThinkingTokens  int       `json:"thinking_tokens" db:"thinking_tokens"`   // thinking token 估算
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// ContextSectionsMeta 上下文段填充元数据（序列化为 JSON 存入 context_sections）
type ContextSectionsMeta struct {
	HasDeviceSignals    bool `json:"has_device_signals"`
	CorrectionsCount    int  `json:"corrections_count"`
	TimeslotCorrections int  `json:"timeslot_corrections"`
	UserScheduleCount   int  `json:"user_schedule_count"`
	AIScheduleCount     int  `json:"ai_schedule_count"`
	HasProfile          bool `json:"has_profile"`
	HasCoreMemory       bool `json:"has_core_memory"`
	RecentStatusesCount int  `json:"recent_statuses_count"`
	HasPrevInference    bool `json:"has_prev_inference"`
	HasConversation     bool `json:"has_conversation"`
}

// InferenceCorrection 推断纠正记录（持久化到 MySQL）
type InferenceCorrection struct {
	ID                int64     `json:"id" db:"id"`
	UserID            string    `json:"user_id" db:"user_id"`
	OriginalEmoji     string    `json:"original_emoji" db:"original_emoji"`
	OriginalActivity  string    `json:"original_activity" db:"original_activity"`
	CorrectedEmoji    string    `json:"corrected_emoji" db:"corrected_emoji"`
	CorrectedActivity string    `json:"corrected_activity" db:"corrected_activity"`
	CorrectedPlace    string    `json:"corrected_place" db:"corrected_place"`
	DayOfWeek         int       `json:"day_of_week" db:"day_of_week"`
	HourOfDay         int       `json:"hour_of_day" db:"hour_of_day"`
	DeviceContext     string    `json:"device_context" db:"device_context"`
	LocationType      string    `json:"location_type" db:"location_type"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// TimeslotCorrectionSummary 时段纠正聚合摘要
type TimeslotCorrectionSummary struct {
	OriginalEmoji     string `json:"original_emoji" db:"original_emoji"`
	OriginalActivity  string `json:"original_activity" db:"original_activity"`
	CorrectedEmoji    string `json:"corrected_emoji" db:"corrected_emoji"`
	CorrectedActivity string `json:"corrected_activity" db:"corrected_activity"`
	CorrectedPlace    string `json:"corrected_place" db:"corrected_place"`
	Count             int    `json:"count" db:"count"`
}
