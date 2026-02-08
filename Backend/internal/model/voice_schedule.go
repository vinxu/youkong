package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
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
	StartTime  string `json:"start_time"`             // HH:MM 格式
	EndTime    string `json:"end_time"`               // HH:MM 格式
	Emoji      string `json:"emoji"`
	Status     string `json:"status"`
	Executed   bool   `json:"executed"`
	IsAIGuess  bool   `json:"is_ai_guess,omitempty"`  // 是否为 AI 推测的状态
	GifURL     string `json:"gif_url,omitempty"`      // GIF 动图 URL
	GiphyQuery string `json:"giphy_query,omitempty"`  // Giphy 搜索词
	Highlight  bool   `json:"highlight,omitempty"`    // 高亮状态（有空）
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
	HidePastEvents    bool               `db:"hide_past_events" json:"hide_past_events"`
	DefaultVisibility ScheduleVisibility `db:"default_visibility" json:"default_visibility"`
	DefaultCircleIDs  StringArray        `db:"default_circle_ids" json:"default_circle_ids,omitempty"`
	ShowCity          bool               `db:"show_city" json:"show_city"` // 是否显示城市，默认 true
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
	HistorySummary      string                 `json:"history_summary,omitempty"`      // 对话历史摘要（超过3轮时生成）
	RetryCount          int                    `json:"retry_count,omitempty"`          // 当前重试次数（验证失败重试）
	LastOperation       OperationType          `json:"last_operation,omitempty"`       // 上次操作类型

	// ========== 时刻表累积机制（v2 新增）==========
	ScheduleSnapshots []ScheduleSnapshot `json:"schedule_snapshots,omitempty"` // 时刻表快照历史

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
	VSEventCancelled        VoiceScheduleEventType = "cancelled"         // 取消操作
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

	// ========== v3 架构改进：撤销与 UX 增强 ==========
	AwaitingAction   string       `json:"awaiting_action,omitempty"`   // 等待的操作类型：approval/input/undo/none
	AvailableActions []ActionHint `json:"available_actions,omitempty"` // 可执行的操作提示
	CanUndo            bool         `json:"can_undo,omitempty"`            // 是否可撤销
	UndoDeadline       int64        `json:"undo_deadline,omitempty"`       // 撤销截止时间（Unix 毫秒）
	UndoRemaining      int64        `json:"undo_remaining,omitempty"`      // 剩余撤销时间（秒）
	NeedsConfirmation  bool         `json:"needs_confirmation,omitempty"`  // 需要用户确认（delete/modify 操作）
	PendingOperation   string       `json:"pending_operation,omitempty"`   // 待确认的操作类型
}

// ActionHint 操作提示（用于客户端显示快捷按钮）
type ActionHint struct {
	Action      string `json:"action"`                // 操作类型：confirm/reject/modify/undo
	Label       string `json:"label"`                 // 显示文本
	Shortcut    string `json:"shortcut,omitempty"`    // 快捷指令，如 "好的"
	Description string `json:"description,omitempty"` // 操作说明
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
	VSActionUndo       VoiceScheduleAction = "undo" // v3 新增：撤销操作
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
	Operation       string              `json:"operation,omitempty"`      // 操作类型：create/modify/delete/query（用于智能合并）
	TargetIndex     *int                `json:"target_index,omitempty"`   // 修改/删除时指定目标时段索引（0-based）
}

// ========== 时刻表累积机制（v2 新增）==========
// 注意：OperationType 定义在 voice_schedule_schema.go 中，避免重复

// IntentResult 意图分发器返回结果（v2 新增：Tool Calling + CoT 混合机制）
// 用于解决 update_status vs create 等意图混淆问题
type IntentResult struct {
	// Action 意图类型：create/modify/confirm/cancel/update_status/query/chat
	Action string `json:"action"`

	// Thinking LLM 的分析过程（Chain of Thought）
	// 至少包含 3 条分析，便于调试和追溯
	Thinking []string `json:"thinking"`

	// Confidence 置信度 0.0-1.0
	Confidence float64 `json:"confidence"`

	// Entities 提取的实体（时间、状态、目标等）
	Entities map[string]interface{} `json:"entities,omitempty"`

	// TargetDate 目标日期（从 entities 中提取，YYYY-MM-DD 格式）
	TargetDate string `json:"target_date,omitempty"`

	// StatusInfo 当前状态信息（update_status 时使用）
	StatusInfo *CurrentStatusGuess `json:"status_info,omitempty"`
}

// ScheduleSnapshot 时刻表快照（用于记录每次 LLM 返回后的时刻表状态）
type ScheduleSnapshot struct {
	Round     int            `json:"round"`     // 对话轮次
	Action    string         `json:"action"`    // 操作类型：create/modify/delete/replace
	Schedule  []ScheduleItem `json:"schedule"`  // 快照时的时刻表
	Timestamp time.Time      `json:"timestamp"` // 记录时间
	Reason    string         `json:"reason"`    // 操作原因/用户原始输入
}

// SequenceContext 衔接上下文（用于检测"开完会去健身"等连续活动表达）
type SequenceContext struct {
	PreviousActivity ScheduleItem `json:"previous_activity"` // 前一个活动
	PreviousIndex    int          `json:"previous_index"`    // 前一个活动的索引
	EndTime          string       `json:"end_time"`          // 前一个活动的结束时间
	DetectedKeyword  string       `json:"detected_keyword"`  // 检测到的衔接关键词
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
// v3 架构改进：统一 Phase 状态机，废弃 V1 的 State
type ConversationPhase string

const (
	PhaseIdle               ConversationPhase = "idle"                // 闲置/聊天
	PhaseUnderstanding      ConversationPhase = "understanding"       // 理解意图
	PhaseDiscussing         ConversationPhase = "discussing"          // 讨论确认
	PhasePlanning           ConversationPhase = "planning"            // 生成计划
	PhaseAwaitingApproval   ConversationPhase = "awaiting_approval"   // 显式等待确认状态
	PhaseProcessingApproval ConversationPhase = "processing_approval" // 处理审批中
	PhaseApproval           ConversationPhase = "approval"            // 等待审批（兼容旧代码）
	PhaseExecution          ConversationPhase = "execution"           // 执行保存
	PhaseConfirmedUndoable  ConversationPhase = "confirmed_undoable"  // 已确认但可撤销（5分钟内）
	PhaseCompleted          ConversationPhase = "completed"           // 完成
)

// ValidPhaseTransitions 有效的阶段转换映射
// v3 架构改进：严格的状态转换验证
var ValidPhaseTransitions = map[ConversationPhase][]ConversationPhase{
	PhaseIdle:               {PhaseUnderstanding},
	PhaseUnderstanding:      {PhasePlanning, PhaseDiscussing, PhaseIdle},
	PhaseDiscussing:         {PhasePlanning, PhaseDiscussing, PhaseIdle},
	PhasePlanning:           {PhaseAwaitingApproval, PhasePlanning, PhaseApproval},
	PhaseAwaitingApproval:   {PhaseProcessingApproval, PhasePlanning, PhaseIdle},
	PhaseProcessingApproval: {PhaseExecution, PhasePlanning, PhaseIdle},
	PhaseApproval:           {PhaseExecution, PhasePlanning, PhaseIdle},           // 兼容
	PhaseExecution:          {PhaseConfirmedUndoable, PhaseCompleted},
	PhaseConfirmedUndoable:  {PhaseCompleted, PhasePlanning},                      // 撤销后回到 Planning
	PhaseCompleted:          {PhaseIdle, PhaseUnderstanding},
}

// IsValidTransition 检查阶段转换是否有效
func IsValidTransition(from, to ConversationPhase) bool {
	validTargets, exists := ValidPhaseTransitions[from]
	if !exists {
		return false
	}
	for _, valid := range validTargets {
		if valid == to {
			return true
		}
	}
	return false
}

// ========== v3 架构改进：执行历史与撤销机制 ==========

// ExecutionHistory 执行历史记录（用于回滚/撤销）
type ExecutionHistory struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	SessionID     string         `json:"session_id"`
	Action        string         `json:"action"`                   // save/modify/delete
	BeforeState   []ScheduleItem `json:"before_state"`             // 执行前的时刻表状态
	AfterState    []ScheduleItem `json:"after_state"`              // 执行后的时刻表状态
	TargetDate    time.Time      `json:"target_date"`              // 目标日期
	Timestamp     time.Time      `json:"timestamp"`                // 执行时间
	CanUndo       bool           `json:"can_undo"`                 // 是否可撤销
	UndoDeadline  time.Time      `json:"undo_deadline"`            // 撤销截止时间
	UndoReason    string         `json:"undo_reason,omitempty"`    // 如果不可撤销，原因是什么
}

// UndoWindow 撤销窗口时长
const UndoWindow = 5 * time.Minute

// executionHistoryCounter 用于生成唯一 ID 的原子计数器
var executionHistoryCounter uint64

// NewExecutionHistory 创建执行历史记录
// 使用时间戳+原子计数器确保高并发下 ID 唯一性
func NewExecutionHistory(userID, sessionID, action string, before, after []ScheduleItem, targetDate time.Time) *ExecutionHistory {
	now := time.Now()
	// 使用原子操作增加计数器，确保并发安全
	counter := atomic.AddUint64(&executionHistoryCounter, 1)
	return &ExecutionHistory{
		ID:           fmt.Sprintf("exec_%d_%d", now.UnixNano(), counter),
		UserID:       userID,
		SessionID:    sessionID,
		Action:       action,
		BeforeState:  before,
		AfterState:   after,
		TargetDate:   targetDate,
		Timestamp:    now,
		CanUndo:      true,
		UndoDeadline: now.Add(UndoWindow),
	}
}

// IsUndoable 检查是否还在可撤销窗口内
func (h *ExecutionHistory) IsUndoable() bool {
	if !h.CanUndo {
		return false
	}
	return time.Now().Before(h.UndoDeadline)
}

// RemainingUndoTime 返回剩余可撤销时间（秒）
func (h *ExecutionHistory) RemainingUndoTime() int64 {
	if !h.IsUndoable() {
		return 0
	}
	return int64(time.Until(h.UndoDeadline).Seconds())
}

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

// ========== 时间感知注解（Phase 7: 工具层时间感知）==========

// TimeNotice 工具返回值中的时间注解
// 附加在 plan_activities / view_schedule 的返回值中，帮助 LLM 生成更智能的回复
type TimeNotice struct {
	Type    string `json:"type"`     // "conflict" | "past" | "ongoing" | "soon"
	Detail  string `json:"detail"`   // 人可读描述
	ItemIdx int    `json:"item_idx"` // 关联的 item 索引，-1 表示通用
}

// ========== V4 架构：简化的 Session 和消息结构 ==========

// V4Message V4 版本的消息结构（兼容 OpenAI 格式）
type V4Message struct {
	Role       string        `json:"role"`                  // system, user, assistant, tool
	Content    string        `json:"content,omitempty"`     // 消息内容
	ToolCalls  []V4ToolCall  `json:"tool_calls,omitempty"`  // 工具调用（assistant 消息）
	ToolCallID string        `json:"tool_call_id,omitempty"` // 工具调用 ID（tool 消息）
	Name       string        `json:"name,omitempty"`        // 工具名称（tool 消息）
}

// V4ToolCall V4 版本的工具调用
type V4ToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"` // function
	Function V4ToolFunction `json:"function"`
}

// V4ToolFunction 工具函数信息
type V4ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串
}

// V4Session V4 版本的简化会话结构
// 设计理念：像 Claude CLI 一样，不使用复杂的状态机，只保存消息历史
type V4Session struct {
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`

	// 消息历史（核心：包含完整的对话上下文）
	Messages []V4Message `json:"messages"`

	// 历史对话摘要（上下文压缩后保留的语义信息）
	Summary string `json:"summary,omitempty"`

	// 待确认的时刻表（调用 update_schedule 后保存，等待用户确认）
	PendingSchedule []ScheduleItem `json:"pending_schedule,omitempty"`
	PendingDate     time.Time      `json:"pending_date,omitempty"`

	// 目标日期（从用户输入中解析）
	TargetDate time.Time `json:"target_date,omitempty"`

	// 当前时刻表（从数据库加载）
	CurrentSchedule  []ScheduleItem `json:"current_schedule,omitempty"`
	TomorrowSchedule []ScheduleItem `json:"tomorrow_schedule,omitempty"` // 明日时刻表

	// 可见性设置
	Visibility ScheduleVisibility `json:"visibility,omitempty"`
	CircleIDs  []string           `json:"circle_ids,omitempty"`

	// ========== 好友交互相关字段 ==========
	// 待发送的消息（调用 send_message 后保存，等待用户确认）
	PendingMessage *V4PendingMessage `json:"pending_message,omitempty"`
	// 待发送的日程邀请（调用 create_schedule_invite 后保存，等待用户确认）
	PendingInvite *V4PendingInvite `json:"pending_invite,omitempty"`
	// 待确认的删除操作（delete_schedule / remove_friend 后保存，等待用户确认）
	PendingDeletion *V4PendingDeletion `json:"pending_deletion,omitempty"`
	// 最近查询到的好友（用于上下文关联，如"给他发消息"）
	LastQueriedFriend *V4FriendInfo `json:"last_queried_friend,omitempty"`

	// ========== 扩展功能相关字段 ==========
	// 通讯录上下文
	ContactHashes   []string               `json:"contact_hashes,omitempty"`   // 上传的手机号哈希
	MatchedContacts []ContactMatchResult   `json:"matched_contacts,omitempty"` // 匹配结果缓存

	// 设备数据缓存（减少重复查询）
	CachedDeviceData *V4DeviceDataCache `json:"cached_device_data,omitempty"`

	// 行为建议上下文
	LastSuggestionContext string `json:"last_suggestion_context,omitempty"`

	// 传感器数据（一键生成时传入，不持久化到 Redis，仅本次请求使用）
	SensorData *ExtendedStatusReportRequest `json:"-"`

	// 时间预处理注解（Layer 1 解析结果，不持久化，仅本次请求使用）
	TemporalAnnotation string `json:"-"`

	// 目标日期的安排（从用户输入解析目标日期后按需加载，不持久化）
	TargetDateSchedule []ScheduleItem `json:"-"`
	TargetDateLabel    string         `json:"-"` // 如 "2026-02-10(周二)" 用于系统提示词显示
}

// V4DeviceDataCache 设备数据缓存
type V4DeviceDataCache struct {
	Location  *V4LocationData  `json:"location,omitempty"`
	Calendar  *V4CalendarData  `json:"calendar,omitempty"`
	Screen    *V4ScreenData    `json:"screen,omitempty"`
	Device    *V4DeviceState   `json:"device,omitempty"`
	Status    *V4StatusData    `json:"status,omitempty"`
	CachedAt  time.Time        `json:"cached_at"`
}

// V4LocationData 位置数据
type V4LocationData struct {
	PlaceType      string `json:"place_type"`       // home/work/cafe/transit/...
	PlaceName      string `json:"place_name"`       // 地点名称
	City           string `json:"city"`             // 城市
	AtPlaceSince   int    `json:"at_place_since"`   // 在此位置多久（分钟）
}

// V4CalendarData 日历数据
type V4CalendarData struct {
	HasCurrentEvent   bool   `json:"has_current_event"`
	CurrentEvent      string `json:"current_event,omitempty"`       // 当前日程标题
	EventEndMinutes   int    `json:"event_end_minutes,omitempty"`   // 当前事件还有多久结束
	NextEventIn       int    `json:"next_event_in,omitempty"`       // 下个日程多久后
	NextEvent         string `json:"next_event,omitempty"`          // 下个日程标题
	TodayRemaining    int    `json:"today_remaining,omitempty"`     // 今天剩余日程数
}

// V4ScreenData 屏幕使用数据
type V4ScreenData struct {
	IsActive        bool   `json:"is_active"`
	ActivityType    string `json:"activity_type"`     // entertainment/productivity/communication/idle
	SessionDuration int    `json:"session_duration"`  // 本次使用时长（分钟）
}

// V4DeviceState 设备状态
type V4DeviceState struct {
	BatteryLevel int    `json:"battery_level"`  // 0-100
	IsCharging   bool   `json:"is_charging"`
	NetworkType  string `json:"network_type"`   // wifi/cellular
	FocusModeOn  bool   `json:"focus_mode_on"`
	Headphones   bool   `json:"headphones"`
}

// V4StatusData 状态分析数据
type V4StatusData struct {
	Availability string `json:"availability"`   // 有空/忙碌/可能有空
	Probability  int    `json:"probability"`    // 0-100
	Emoji        string `json:"emoji"`
	LifeStatus   string `json:"life_status"`    // 生活状态描述
}

// V4PendingMessage 待发送的消息
type V4PendingMessage struct {
	FriendID   string `json:"friend_id"`
	FriendName string `json:"friend_name"`
	Message    string `json:"message"`
}

// V4PendingInvite 待发送的日程邀请
type V4PendingInvite struct {
	FriendID   string `json:"friend_id"`
	FriendName string `json:"friend_name"`
	Date       string `json:"date"`        // YYYY-MM-DD
	StartTime  string `json:"start_time"`  // HH:MM
	EndTime    string `json:"end_time"`    // HH:MM
	Activity   string `json:"activity"`
	Location   string `json:"location,omitempty"`
	Message    string `json:"message,omitempty"`
}

// V4PendingDeletion 待确认的删除操作
type V4PendingDeletion struct {
	Type           string         `json:"type"`                        // "schedule" | "friend"
	Date           string         `json:"date,omitempty"`              // 日期（schedule 类型）
	Target         string         `json:"target,omitempty"`            // 删除目标关键词或 "all"
	DeletedItems   []ScheduleItem `json:"deleted_items,omitempty"`     // 将被删除的条目
	RemainingItems []ScheduleItem `json:"remaining_items,omitempty"`   // 删除后剩余的条目
	FriendID       string         `json:"friend_id,omitempty"`         // 好友 ID（friend 类型）
	FriendName     string         `json:"friend_name,omitempty"`       // 好友名字（friend 类型）
}

// V4FriendInfo 好友信息（用于工具返回和上下文）
type V4FriendInfo struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Avatar             string `json:"avatar,omitempty"`
	Probability        int    `json:"probability"`                   // 有空概率 0-100, -1无数据
	Status             string `json:"status,omitempty"`              // 当前状态描述（life_status_label）
	Emoji              string `json:"emoji,omitempty"`               // 状态emoji（life_status_emoji）
	City               string `json:"city,omitempty"`                // 所在城市
	Confidence         string `json:"confidence"`                    // high/medium/low
	AvailabilityStatus string `json:"availability_status,omitempty"` // 有空状态：有空/忙碌/可能有空
}

// NewV4Session 创建新的 V4 会话
func NewV4Session(userID, sessionID string) *V4Session {
	return &V4Session{
		UserID:    userID,
		SessionID: sessionID,
		CreatedAt: time.Now(),
		Messages:  []V4Message{},
	}
}

// AddMessage 添加消息到历史
func (s *V4Session) AddMessage(role, content string) {
	s.Messages = append(s.Messages, V4Message{
		Role:    role,
		Content: content,
	})
}

// AddAssistantMessageWithToolCalls 添加带工具调用的 assistant 消息
func (s *V4Session) AddAssistantMessageWithToolCalls(content string, toolCalls []V4ToolCall) {
	s.Messages = append(s.Messages, V4Message{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	})
}

// AddToolResult 添加工具执行结果
func (s *V4Session) AddToolResult(toolCallID, toolName, result string) {
	s.Messages = append(s.Messages, V4Message{
		Role:       "tool",
		Content:    result,
		ToolCallID: toolCallID,
		Name:       toolName,
	})
}

// GetRecentMessages 获取最近的消息（用于构建 Prompt，控制上下文长度）
func (s *V4Session) GetRecentMessages(maxTurns int) []V4Message {
	if len(s.Messages) <= maxTurns*2 {
		return s.Messages
	}
	// 保留最近的 N 轮对话
	return s.Messages[len(s.Messages)-maxTurns*2:]
}

// HasPendingSchedule 是否有待确认的时刻表
func (s *V4Session) HasPendingSchedule() bool {
	return len(s.PendingSchedule) > 0
}

// ClearPendingSchedule 清除待确认的时刻表
func (s *V4Session) ClearPendingSchedule() {
	s.PendingSchedule = nil
	s.PendingDate = time.Time{}
}

// HasPendingMessage 是否有待发送的消息
func (s *V4Session) HasPendingMessage() bool {
	return s.PendingMessage != nil
}

// HasPendingInvite 是否有待发送的日程邀请
func (s *V4Session) HasPendingInvite() bool {
	return s.PendingInvite != nil
}

// HasPendingDeletion 是否有待确认的删除操作
func (s *V4Session) HasPendingDeletion() bool {
	return s.PendingDeletion != nil
}

// ClearPendingDeletion 清除待确认的删除操作
func (s *V4Session) ClearPendingDeletion() {
	s.PendingDeletion = nil
}

// HasAnyPending 是否有任何待确认的内容
func (s *V4Session) HasAnyPending() bool {
	return s.HasPendingSchedule() || s.HasPendingMessage() || s.HasPendingInvite() || s.HasPendingDeletion()
}

// ClearPendingMessage 清除待发送的消息
func (s *V4Session) ClearPendingMessage() {
	s.PendingMessage = nil
}

// ClearPendingInvite 清除待发送的日程邀请
func (s *V4Session) ClearPendingInvite() {
	s.PendingInvite = nil
}

// ClearAllPending 清除所有待确认的内容
func (s *V4Session) ClearAllPending() {
	s.ClearPendingSchedule()
	s.ClearPendingMessage()
	s.ClearPendingInvite()
	s.ClearPendingDeletion()
}

// V4EventType V4 版本的 SSE 事件类型
type V4EventType string

const (
	V4EventTypeSessionStart       V4EventType = "session_start"
	V4EventTypeTranscript         V4EventType = "transcript"
	V4EventTypeThinking           V4EventType = "thinking"
	V4EventTypeSchedulePreview    V4EventType = "schedule_preview"
	V4EventTypeScheduleSaved      V4EventType = "schedule_saved"
	V4EventTypeStatusUpdated      V4EventType = "status_updated"
	V4EventTypePreferenceUpdated  V4EventType = "preference_updated"
	V4EventTypeChat               V4EventType = "chat"
	V4EventTypeChatStream         V4EventType = "chat_stream"     // 流式文本片段
	V4EventTypeChatStreamEnd      V4EventType = "chat_stream_end" // 流式输出结束
	V4EventTypeError              V4EventType = "error"

	// 阶段暴露事件（让用户知道当前在做什么）
	V4EventTypePhase     V4EventType = "phase"      // 处理阶段（理解请求、调用工具、生成回复）
	V4EventTypeToolStart V4EventType = "tool_start" // 工具开始执行
	V4EventTypeToolEnd   V4EventType = "tool_end"   // 工具执行完成

	// 删除相关事件
	V4EventTypeDeletePreview   V4EventType = "delete_preview"   // 删除预览（待确认）
	V4EventTypeScheduleDeleted V4EventType = "schedule_deleted" // 日程已删除
	V4EventTypeFriendRemoved   V4EventType = "friend_removed"   // 好友已删除

	// 好友相关事件
	V4EventTypeFriendsResult   V4EventType = "friends_result"   // 好友查询结果
	V4EventTypeMessagePreview  V4EventType = "message_preview"  // 消息预览（待确认）
	V4EventTypeMessageSent     V4EventType = "message_sent"     // 消息已发送
	V4EventTypeInvitePreview   V4EventType = "invite_preview"   // 日程邀请预览（待确认）
	V4EventTypeInviteSent      V4EventType = "invite_sent"      // 日程邀请已发送
)

// V4Event V4 版本的 SSE 事件
type V4Event struct {
	Type      V4EventType    `json:"type"`
	SessionID string         `json:"session_id,omitempty"`
	Message   string         `json:"message,omitempty"`
	Items     []ScheduleItem `json:"items,omitempty"`
	Date      string         `json:"date,omitempty"`
	Emoji     string         `json:"emoji,omitempty"`
	Status    string         `json:"status,omitempty"`

	// 查询模式标识（查询已有时刻表时为 true，前端不显示确认按钮）
	IsQuery bool `json:"is_query,omitempty"`

	// 阶段暴露字段
	Phase    string `json:"phase,omitempty"`     // 阶段名称：understanding, tool_calling, generating
	ToolName string `json:"tool_name,omitempty"` // 工具名称（tool_start/tool_end 时使用）
	Loop     int    `json:"loop,omitempty"`      // 当前循环次数

	// 好友相关字段
	Friends         []V4FriendInfo    `json:"friends,omitempty"`          // 好友列表
	Total           int               `json:"total,omitempty"`            // 总数
	FilterApplied   string            `json:"filter_applied,omitempty"`   // 应用的筛选条件
	PendingMessage  *V4PendingMessage `json:"pending_message,omitempty"`  // 待发送消息预览
	PendingInvite   *V4PendingInvite   `json:"pending_invite,omitempty"`   // 待发送邀请预览
	PendingDeletion *V4PendingDeletion `json:"pending_deletion,omitempty"` // 待确认删除预览
	DeletedCount    int                `json:"deleted_count,omitempty"`    // 已删除条目数
	MessageID       string             `json:"message_id,omitempty"`       // 已发送消息ID
	SentTo          string            `json:"sent_to,omitempty"`          // 发送给谁
	AwaitingConfirm bool              `json:"awaiting_confirm,omitempty"` // 是否等待确认
}
