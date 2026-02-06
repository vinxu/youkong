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
	Iterations    int       `json:"iterations" db:"iterations"`         // Agent 循环次数
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}
