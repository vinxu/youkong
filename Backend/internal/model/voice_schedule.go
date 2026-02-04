package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// ScheduleStatus 时刻表状态
type ScheduleStatus string

const (
	ScheduleStatusActive    ScheduleStatus = "active"
	ScheduleStatusCompleted ScheduleStatus = "completed"
	ScheduleStatusCancelled ScheduleStatus = "cancelled"
)

// ScheduleItem 时刻表条目
type ScheduleItem struct {
	StartTime string `json:"start_time"` // HH:MM 格式
	EndTime   string `json:"end_time"`   // HH:MM 格式
	Emoji     string `json:"emoji"`
	Status    string `json:"status"`
	Executed  bool   `json:"executed"`
}

// ScheduleItems JSON 数组类型（用于数据库存储）
type ScheduleItems []ScheduleItem

// Value 实现 driver.Valuer 接口
func (s ScheduleItems) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *ScheduleItems) Scan(value interface{}) error {
	if value == nil {
		*s = []ScheduleItem{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("invalid type for ScheduleItems")
	}

	return json.Unmarshal(data, s)
}

// StatusSchedule 状态时刻表（数据库模型）
type StatusSchedule struct {
	ID           int64              `db:"id" json:"id"`
	UserID       string             `db:"user_id" json:"user_id"`
	ScheduleDate time.Time          `db:"schedule_date" json:"schedule_date"`
	Items        ScheduleItems      `db:"items" json:"items"`
	CurrentIndex int                `db:"current_index" json:"current_index"`
	Status       ScheduleStatus     `db:"status" json:"status"`
	Visibility   ScheduleVisibility `db:"visibility" json:"visibility"`
	CircleIDs    StringArray        `db:"circle_ids" json:"circle_ids,omitempty"`
	CreatedAt    time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `db:"updated_at" json:"updated_at"`
}

// StringArray JSON 数组类型（用于数据库存储）
type StringArray []string

// Value 实现 driver.Valuer 接口
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("invalid type for StringArray")
	}

	return json.Unmarshal(data, s)
}

// UserSchedulePreference 用户时刻表偏好设置
type UserSchedulePreference struct {
	UserID            string             `db:"user_id" json:"user_id"`
	DefaultVisibility ScheduleVisibility `db:"default_visibility" json:"default_visibility"`
	DefaultCircleIDs  StringArray        `db:"default_circle_ids" json:"default_circle_ids,omitempty"`
	CreatedAt         time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `db:"updated_at" json:"updated_at"`
}

// VoiceScheduleSession 语音时刻表会话（Redis 存储）
type VoiceScheduleSession struct {
	UserID            string            `json:"user_id"`
	SessionID         string            `json:"session_id"`
	CurrentSchedule   []ScheduleItem    `json:"current_schedule"`
	PendingQuestions  []ClarifyQuestion `json:"pending_questions,omitempty"`
	PartialSchedule   []ScheduleItem    `json:"partial_schedule,omitempty"`
	TranscriptHistory []string          `json:"transcript_history"`
	State             string            `json:"state"` // initial, clarifying, schedule_ready, confirmed (旧字段，保持兼容)
	CurrentStatusGuess *CurrentStatusGuess `json:"current_status_guess,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`

	// Plan Mode 上下文
	UserContext         *CompressedUserContext `json:"user_context,omitempty"`         // 压缩后的用户上下文
	ConversationHistory []ConversationTurn     `json:"conversation_history,omitempty"` // 对话历史
	LastReasoning       []string               `json:"last_reasoning,omitempty"`       // 最近一次的推理依据

	// 可见性设置
	Visibility ScheduleVisibility `json:"visibility,omitempty"` // 可见性
	CircleIDs  []string           `json:"circle_ids,omitempty"` // 指定圈子 ID

	// 目标日期（支持"明天"、"后天"等）
	TargetDate time.Time `json:"target_date,omitempty"` // 时刻表对应的日期，默认为今天

	// ========== 多阶段对话状态机字段（新增）==========
	Phase          ConversationPhase   `json:"phase,omitempty"`           // 当前阶段
	IntentSummary  *IntentSummary      `json:"intent_summary,omitempty"`  // 意图摘要
	DraftPlan      *DraftPlan          `json:"draft_plan,omitempty"`      // 计划草案
	Clarifications []ClarificationItem `json:"clarifications,omitempty"` // 待澄清项列表
	PhaseHistory   []PhaseTransition   `json:"phase_history,omitempty"`   // 阶段转换历史
}

// ClarifyQuestion 澄清问题
type ClarifyQuestion struct {
	ID         string   `json:"id"`
	Question   string   `json:"question"`
	Options    []string `json:"options"`
	AllowVoice bool     `json:"allow_voice"`
}

// CurrentStatusGuess 当前状态猜测
type CurrentStatusGuess struct {
	Emoji  string `json:"emoji"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// VoiceScheduleEventType SSE 事件类型
type VoiceScheduleEventType string

const (
	VSEventSessionStart     VoiceScheduleEventType = "session_start"
	VSEventRecognizing      VoiceScheduleEventType = "recognizing"
	VSEventTranscript       VoiceScheduleEventType = "transcript"
	VSEventProgress         VoiceScheduleEventType = "progress"         // 过程反馈
	VSEventThinking         VoiceScheduleEventType = "thinking"
	VSEventClarify          VoiceScheduleEventType = "clarify"
	VSEventSchedule         VoiceScheduleEventType = "schedule"
	VSEventCurrentStatus    VoiceScheduleEventType = "current_status"
	VSEventChat             VoiceScheduleEventType = "chat"             // 聊天回复（非时刻表操作）
	VSEventVisibilityPrompt VoiceScheduleEventType = "visibility_prompt" // 可见性选择
	VSEventCircleList       VoiceScheduleEventType = "circle_list"       // 圈子列表
	VSEventConfirmed        VoiceScheduleEventType = "confirmed"
	VSEventError            VoiceScheduleEventType = "error"

	// ========== 多阶段对话状态机事件（新增）==========
	VSEventPhaseChange    VoiceScheduleEventType = "phase_change"    // 阶段转换
	VSEventIntentSummary  VoiceScheduleEventType = "intent_summary"  // 意图理解结果
	VSEventDiscussion     VoiceScheduleEventType = "discussion"      // 讨论消息
	VSEventDraftPlan      VoiceScheduleEventType = "draft_plan"      // 计划草案（待审批）
	VSEventApprovalPrompt VoiceScheduleEventType = "approval_prompt" // 审批提示
)

// VoiceScheduleEvent SSE 事件
type VoiceScheduleEvent struct {
	Type       VoiceScheduleEventType `json:"type"`
	SessionID  string                 `json:"session_id,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Text       string                 `json:"text,omitempty"`
	Questions  []ClarifyQuestion      `json:"questions,omitempty"`
	Items      []ScheduleItem         `json:"items,omitempty"`
	Emoji      string                 `json:"emoji,omitempty"`
	StatusText string                 `json:"status_text,omitempty"`
	Reason     string                 `json:"reason,omitempty"`
	Message    string                 `json:"message,omitempty"`
	Partial    bool                   `json:"partial,omitempty"`

	// Progress 事件专用字段
	Action string `json:"action,omitempty"` // 进度动作类型
	Detail string `json:"detail,omitempty"` // 详细信息

	// 推理相关字段
	Reasoning []string `json:"reasoning,omitempty"` // 推理依据

	// 可见性相关字段
	Visibility string              `json:"visibility,omitempty"` // 默认可见性
	Circles    []CircleInfoCompact `json:"circles,omitempty"`    // 圈子列表

	// 查询模式标识（查询已有时刻表时为 true，前端不显示确认按钮）
	IsQuery bool `json:"is_query,omitempty"`

	// ========== 多阶段对话状态机字段（新增）==========
	Phase          ConversationPhase   `json:"phase,omitempty"`           // 当前阶段
	PreviousPhase  ConversationPhase   `json:"previous_phase,omitempty"`  // 上一阶段
	IntentSummary  *IntentSummary      `json:"intent_summary,omitempty"`  // 意图摘要
	DraftPlan      *DraftPlan          `json:"draft_plan,omitempty"`      // 计划草案
	Clarifications []ClarificationItem `json:"clarifications,omitempty"` // 待澄清项
	CanApprove     bool                `json:"can_approve,omitempty"`     // 是否可以审批
}

// CircleInfoCompact 圈子信息（用于语音时刻表可见性选择）
// 注意：与 invitation.go 中的 CircleInfo 字段相似，但 JSON 标签不同
type CircleInfoCompact struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Emoji       string `json:"emoji"`
	MemberCount int    `json:"member_count"`
}

// VoiceScheduleAction 交互动作
type VoiceScheduleAction string

const (
	VSActionAnswer     VoiceScheduleAction = "answer"
	VSActionVoice      VoiceScheduleAction = "voice"
	VSActionSupplement VoiceScheduleAction = "supplement"
	VSActionConfirm    VoiceScheduleAction = "confirm"
	VSActionCancel     VoiceScheduleAction = "cancel"
)

// VoiceScheduleInteractionRequest 后续交互请求
type VoiceScheduleInteractionRequest struct {
	SessionID string                        `json:"session_id" binding:"required"`
	Action    VoiceScheduleAction           `json:"action" binding:"required"`
	Data      *VoiceScheduleInteractionData `json:"data,omitempty"`
}

// VoiceScheduleInteractionData 交互数据
type VoiceScheduleInteractionData struct {
	Answers    map[string]string  `json:"answers,omitempty"`    // 问题 ID -> 回答
	AudioData  string             `json:"audio_data,omitempty"` // Base64 编码的音频数据
	Text       string             `json:"text,omitempty"`       // 文本输入
	Visibility ScheduleVisibility `json:"visibility,omitempty"` // 可见性设置
	CircleIDs  []string           `json:"circle_ids,omitempty"` // 指定圈子 ID
}

// LLMVoiceAnalysisResult LLM 分析结果
type LLMVoiceAnalysisResult struct {
	Action          string              `json:"action"` // create, modify, cancel, guess, clarify, query, replace, chat
	Schedule        []ScheduleItem      `json:"schedule,omitempty"`
	CancelledItems  []string            `json:"cancelled_items,omitempty"`
	Questions       []ClarifyQuestion   `json:"questions,omitempty"`
	PartialSchedule []ScheduleItem      `json:"partial_schedule,omitempty"`
	CurrentStatus   *CurrentStatusGuess `json:"current_status,omitempty"`
	Message         string              `json:"message,omitempty"`        // chat 动作的回复内容
	Reason          string              `json:"reason,omitempty"`
	Reasoning       []string            `json:"reasoning,omitempty"`      // 推理依据
	Thinking        string              `json:"thinking,omitempty"`       // 思考过程
	NeedThinking    bool                `json:"need_thinking,omitempty"`  // 是否需要深度思考
	TargetDate      string              `json:"target_date,omitempty"`    // 目标日期：YYYY-MM-DD 格式（兼容 today/tomorrow 等旧格式）
	DateReasoning   string              `json:"date_reasoning,omitempty"` // 日期推理过程（调试用）
}

// ========== Plan Mode 上下文数据结构 ==========

// UserContext 用户完整上下文（供 LLM 推理使用）
type UserContext struct {
	// 基础信息
	Profile     *UserProfileData `json:"profile"`      // 用户画像
	CurrentTime string           `json:"current_time"` // 当前时间
	Weekday     string           `json:"weekday"`      // 星期几

	// 已有行程
	TodaySchedules []ScheduleItem `json:"today_schedules"` // 今日已确定的行程

	// 历史记忆
	RecentStatuses []StatusMemoryItem `json:"recent_statuses"` // 最近状态记录
	CoreMemory     *CoreMemory        `json:"core_memory"`     // 核心记忆

	// 设备数据（实时）
	DeviceData *DeviceContextData `json:"device_data"` // 设备采集数据
}

// StatusMemoryItem 状态记忆条目
type StatusMemoryItem struct {
	Emoji     string `json:"emoji"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// DeviceContextData 设备上下文数据
type DeviceContextData struct {
	// 位置
	CurrentPlace    string `json:"current_place"`      // 当前位置类型 (home/work/leisure/transit)
	PlaceName       string `json:"place_name"`         // 地点名称（反向地理编码）
	AtPlaceSince    int    `json:"at_place_since"`     // 在此位置待了多久(分钟)

	// 运动
	IsMoving      bool   `json:"is_moving"`       // 是否在移动
	MovementType  string `json:"movement_type"`   // 移动类型 (walking/driving/stationary)
	StepsLastHour int    `json:"steps_last_hour"` // 最近一小时步数

	// 日历
	CurrentEvent  string `json:"current_event"`   // 当前日历事件
	NextEvent     string `json:"next_event"`      // 下一个事件
	NextEventIn   int    `json:"next_event_in"`   // 下一个事件多久后开始(分钟)

	// 屏幕
	IsScreenActive bool   `json:"is_screen_active"` // 屏幕是否亮着
	ActivityType   string `json:"activity_type"`    // 活动类型
}

// CompressedUserContext 压缩后的用户上下文（控制在 ~1000 字符内）
type CompressedUserContext struct {
	// 用户画像摘要（~100 字符）
	ProfileSummary string `json:"profile_summary"` // "上班族，北京，通常9-18点工作"

	// 今日行程摘要（~200 字符）
	TodayScheduleSummary string `json:"today_schedule_summary"` // "已有2个行程：12:00午餐、14:00开会"

	// 历史规律洞察（复用 CoreMemory 设计，~200 字符）
	BehaviorInsights string `json:"behavior_insights"` // "工作日午餐通常12:00-13:00"
	TimePatterns     string `json:"time_patterns"`     // "周末常睡懒觉到10点"
	LocationPrefs    string `json:"location_prefs"`    // "午餐常去公司楼下"

	// 当前设备状态摘要（~100 字符）
	DeviceStateSummary string `json:"device_state_summary"` // "在公司，静止30分钟，日历显示14:00有会"
}

// ConversationTurn 对话轮次
type ConversationTurn struct {
	Role    string `json:"role"`    // "user" | "assistant"
	Content string `json:"content"` // 用户语音转文本 或 AI 的 thinking
	Time    string `json:"time"`
}

// ========== Progress 过程反馈事件 ==========

// ProgressAction 进度动作类型
type ProgressAction string

const (
	ProgressRecognizing      ProgressAction = "recognizing"       // 正在识别语音
	ProgressLoadingContext   ProgressAction = "loading_context"   // 正在加载上下文
	ProgressCheckingCalendar ProgressAction = "checking_calendar" // 查看日历
	ProgressCheckingHistory  ProgressAction = "checking_history"  // 查看历史习惯
	ProgressCheckingLocation ProgressAction = "checking_location" // 查看位置
	ProgressCheckingSocial   ProgressAction = "checking_social"   // 查看社交上下文
	ProgressAnalyzing        ProgressAction = "thinking"          // 分析意图
	ProgressGenerating       ProgressAction = "generating"        // 生成时刻表
)

// ========== 可见性控制 ==========

// ScheduleVisibility 时刻表可见性
type ScheduleVisibility string

const (
	VisibilityAllFriends ScheduleVisibility = "all_friends" // 所有好友可见
	VisibilityCircles    ScheduleVisibility = "circles"     // 指定圈子可见
	VisibilityPrivate    ScheduleVisibility = "private"     // 仅自己可见
)

// ========== 多阶段对话状态机 ==========

// ConversationPhase 对话阶段
type ConversationPhase string

const (
	PhaseUnderstanding ConversationPhase = "understanding" // 理解意图
	PhaseDiscussing    ConversationPhase = "discussing"    // 讨论确认
	PhasePlanning      ConversationPhase = "planning"      // 生成计划
	PhaseApproval      ConversationPhase = "approval"      // 等待审批
	PhaseExecution     ConversationPhase = "execution"     // 执行保存
	PhaseCompleted     ConversationPhase = "completed"     // 完成
	PhaseIdle          ConversationPhase = "idle"          // 聊天/非时刻表
)

// IntentSummary 意图摘要（Understanding 阶段输出）
type IntentSummary struct {
	Action           string   `json:"action"`            // create/modify/query/cancel/chat
	TargetDate       string   `json:"target_date"`       // YYYY-MM-DD
	Activities       []string `json:"activities"`        // 提取的活动列表
	TimeReferences   []string `json:"time_references"`   // 时间引用 ["下午", "晚上8点"]
	HasAmbiguity     bool     `json:"has_ambiguity"`     // 是否有模糊信息
	AmbiguityReasons []string `json:"ambiguity_reasons"` // 模糊原因列表
	Confidence       float64  `json:"confidence"`        // 置信度 0.0-1.0
	Reasoning        []string `json:"reasoning"`         // 推理过程
}

// DraftPlan 计划草案（Planning 阶段输出）
type DraftPlan struct {
	Schedule  []ScheduleItem `json:"schedule"`   // 完整时刻表
	Summary   string         `json:"summary"`    // 一句话总结
	Changes   []PlanChange   `json:"changes"`    // 变更列表（对比已有时刻表）
	Reasoning []string       `json:"reasoning"`  // 推理过程
	Version   int            `json:"version"`    // 版本号（用于多次修改）
	CreatedAt time.Time      `json:"created_at"` // 创建时间
}

// PlanChange 计划变更条目
type PlanChange struct {
	Type        string `json:"type"`        // add/modify/delete
	TimeRange   string `json:"time_range"`  // "14:00-16:00"
	Description string `json:"description"` // "新增开会"
}

// ClarificationItem 待澄清项（Discussing 阶段使用）
type ClarificationItem struct {
	ID       string `json:"id"`               // 唯一标识
	Question string `json:"question"`         // 问题内容
	Reason   string `json:"reason"`           // 为什么需要澄清
	Answered bool   `json:"answered"`         // 是否已回答
	Answer   string `json:"answer,omitempty"` // 用户的回答
}

// PhaseTransition 阶段转换记录
type PhaseTransition struct {
	From      ConversationPhase `json:"from"`      // 起始阶段
	To        ConversationPhase `json:"to"`        // 目标阶段
	Reason    string            `json:"reason"`    // 转换原因
	Timestamp time.Time         `json:"timestamp"` // 转换时间
}
