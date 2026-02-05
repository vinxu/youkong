package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// UserMemoryDocument 用户记忆文档
// 用于存储跨会话的用户记忆，实现个性化上下文
type UserMemoryDocument struct {
	UserID    string    `db:"user_id" json:"user_id"`
	Version   int       `db:"version" json:"version"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	// 用户偏好（类似 settings.json）
	Preferences UserMemoryPreferences `db:"preferences" json:"preferences"`

	// 日程模式（自动学习）
	SchedulePatterns SchedulePatternList `db:"schedule_patterns" json:"schedule_patterns"`

	// 交互风格（自动学习）
	InteractionStyle UserInteractionStyle `db:"interaction_style" json:"interaction_style"`

	// 关键事实（用户明确告知）
	KeyFacts KeyFactList `db:"key_facts" json:"key_facts"`

	// 会话摘要（最近 N 次会话的压缩）
	SessionSummaries SessionSummaryList `db:"session_summaries" json:"session_summaries"`
}

// UserMemoryPreferences 用户偏好设置
type UserMemoryPreferences struct {
	HidePastEvents    bool   `json:"hide_past_events"`     // 不显示过去的日程
	DefaultViewDays   int    `json:"default_view_days"`    // 默认查看天数
	PreferredTimezone string `json:"preferred_timezone"`   // 偏好时区
	Language          string `json:"language"`             // 语言偏好
}

// Value 实现 driver.Valuer 接口
func (p UserMemoryPreferences) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan 实现 sql.Scanner 接口
func (p *UserMemoryPreferences) Scan(value interface{}) error {
	if value == nil {
		*p = UserMemoryPreferences{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("invalid type for UserMemoryPreferences")
	}

	return json.Unmarshal(data, p)
}

// SchedulePattern 日程模式（自动学习）
type SchedulePattern struct {
	Pattern     string   `json:"pattern"`      // 如 "工作日 9-18 点通常在公司"
	Confidence  float64  `json:"confidence"`   // 置信度 0-1
	LearnedFrom []string `json:"learned_from"` // 学习来源（会话ID列表）
	CreatedAt   string   `json:"created_at"`   // 首次学习时间
}

// SchedulePatternList JSON 数组类型
type SchedulePatternList []SchedulePattern

// Value 实现 driver.Valuer 接口
func (s SchedulePatternList) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *SchedulePatternList) Scan(value interface{}) error {
	if value == nil {
		*s = []SchedulePattern{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("invalid type for SchedulePatternList")
	}

	return json.Unmarshal(data, s)
}

// UserInteractionStyle 用户交互风格
type UserInteractionStyle struct {
	PrefersBrief      bool `json:"prefers_brief"`       // 喜欢简短回复
	ConfirmBeforeSave bool `json:"confirm_before_save"` // 保存前确认
	ShowReasoning     bool `json:"show_reasoning"`      // 显示推理过程
}

// Value 实现 driver.Valuer 接口
func (s UserInteractionStyle) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *UserInteractionStyle) Scan(value interface{}) error {
	if value == nil {
		*s = UserInteractionStyle{
			ConfirmBeforeSave: true, // 默认保存前确认
		}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("invalid type for UserInteractionStyle")
	}

	return json.Unmarshal(data, s)
}

// KeyFact 关键事实（用户明确告知需要记住的信息）
type KeyFact struct {
	Fact      string `json:"fact"`       // 如 "周三下午固定健身"
	Source    string `json:"source"`     // 用户原话
	LearnedAt string `json:"learned_at"` // 学习时间
}

// KeyFactList JSON 数组类型
type KeyFactList []KeyFact

// Value 实现 driver.Valuer 接口
func (k KeyFactList) Value() (driver.Value, error) {
	if k == nil {
		return "[]", nil
	}
	return json.Marshal(k)
}

// Scan 实现 sql.Scanner 接口
func (k *KeyFactList) Scan(value interface{}) error {
	if value == nil {
		*k = []KeyFact{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("invalid type for KeyFactList")
	}

	return json.Unmarshal(data, k)
}

// SessionSummary 会话摘要
type SessionSummary struct {
	SessionID string   `json:"session_id"` // 会话 ID
	Date      string   `json:"date"`       // 会话日期
	Summary   string   `json:"summary"`    // 简短摘要
	KeyPoints []string `json:"key_points"` // 关键决策点
}

// SessionSummaryList JSON 数组类型
type SessionSummaryList []SessionSummary

// Value 实现 driver.Valuer 接口
func (s SessionSummaryList) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *SessionSummaryList) Scan(value interface{}) error {
	if value == nil {
		*s = []SessionSummary{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("invalid type for SessionSummaryList")
	}

	return json.Unmarshal(data, s)
}

// ========== 上下文压缩相关模型 ==========

// SessionInsights 会话洞察（从会话中学习到的信息）
type SessionInsights struct {
	// 新发现的日程模式
	Patterns []SchedulePattern `json:"patterns,omitempty"`

	// 偏好设置更新
	PreferenceUpdates map[string]interface{} `json:"preference_updates,omitempty"`

	// 新发现的关键事实
	KeyFacts []KeyFact `json:"key_facts,omitempty"`

	// 会话摘要
	Summary    string   `json:"summary,omitempty"`
	KeyPoints  []string `json:"key_points,omitempty"`
	Decisions  []string `json:"decisions,omitempty"`  // 用户做的关键决策
	Remembers  []string `json:"remembers,omitempty"`  // 需要记住的信息
}

// CompressionTrigger 压缩触发条件
type CompressionTrigger struct {
	TokenCount   int  `json:"token_count"`   // 当前 token 数
	MessageCount int  `json:"message_count"` // 消息数量
	SessionEnd   bool `json:"session_end"`   // 会话结束
}

// 压缩触发阈值
const (
	TokenThreshold   = 6000 // Token 数量阈值
	MessageThreshold = 12   // 消息数量阈值
	MaxSessionSummaries = 5 // 最多保留的会话摘要数量
	MaxSchedulePatterns = 10 // 最多保留的日程模式数量
	MaxKeyFacts = 20 // 最多保留的关键事实数量
)
