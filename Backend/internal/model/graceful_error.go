package model

// ErrorSeverity 错误严重程度
type ErrorSeverity int

const (
	// SeverityInfo 信息性，不影响流程（如未知 action，可以继续对话）
	SeverityInfo ErrorSeverity = iota

	// SeverityRecoverable 可恢复，可以降级处理（如 LLM 超时可重试）
	SeverityRecoverable

	// SeverityFatal 致命，需要用户介入（如数据库不可用）
	SeverityFatal
)

// String 返回严重程度的字符串表示
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityRecoverable:
		return "recoverable"
	case SeverityFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// RecoveryStrategy 恢复策略
type RecoveryStrategy string

const (
	// StrategyRetry 重试策略
	StrategyRetry RecoveryStrategy = "retry"

	// StrategyFallback 降级策略（使用默认值）
	StrategyFallback RecoveryStrategy = "fallback"

	// StrategyExplain 解释策略（让 LLM 向用户解释）
	StrategyExplain RecoveryStrategy = "explain"

	// StrategyIgnore 忽略策略（继续执行）
	StrategyIgnore RecoveryStrategy = "ignore"
)

// OperationOutcome 操作结果（替代简单的 error，提供更丰富的上下文）
// 用于统一系统中各种操作的结果表示，便于优雅降级处理
type OperationOutcome struct {
	// Success 是否成功
	Success bool `json:"success"`

	// Severity 严重程度
	Severity ErrorSeverity `json:"severity"`

	// Code 错误代码（用于归类和统计）
	// 示例：UNKNOWN_ACTION, LLM_TIMEOUT, DB_ERROR, SESSION_SAVE_FAILED
	Code string `json:"code"`

	// Message 内部消息（日志用，开发者可读）
	Message string `json:"message"`

	// UserMessage 用户消息（可选，如果已有预定义消息）
	UserMessage string `json:"user_message,omitempty"`

	// Context 上下文数据（用于 LLM 生成解释）
	Context map[string]interface{} `json:"context,omitempty"`

	// CanRetry 是否可重试
	CanRetry bool `json:"can_retry"`

	// RetryCount 已重试次数
	RetryCount int `json:"retry_count,omitempty"`

	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries,omitempty"`

	// Suggestions 建议操作（硬编码的建议，LLM 可能会覆盖）
	Suggestions []string `json:"suggestions,omitempty"`

	// OriginalError 原始错误（如果有）
	OriginalError error `json:"-"`
}

// NewSuccessOutcome 创建成功结果
func NewSuccessOutcome() *OperationOutcome {
	return &OperationOutcome{
		Success:  true,
		Severity: SeverityInfo,
	}
}

// NewInfoOutcome 创建信息性结果（不影响流程）
func NewInfoOutcome(code, message string) *OperationOutcome {
	return &OperationOutcome{
		Success:  false,
		Severity: SeverityInfo,
		Code:     code,
		Message:  message,
	}
}

// NewRecoverableOutcome 创建可恢复结果
func NewRecoverableOutcome(code, message string) *OperationOutcome {
	return &OperationOutcome{
		Success:    false,
		Severity:   SeverityRecoverable,
		Code:       code,
		Message:    message,
		CanRetry:   true,
		MaxRetries: 3,
	}
}

// NewFatalOutcome 创建致命错误结果
func NewFatalOutcome(code, message string, err error) *OperationOutcome {
	return &OperationOutcome{
		Success:       false,
		Severity:      SeverityFatal,
		Code:          code,
		Message:       message,
		CanRetry:      false,
		OriginalError: err,
	}
}

// WithContext 添加上下文
func (o *OperationOutcome) WithContext(key string, value interface{}) *OperationOutcome {
	if o.Context == nil {
		o.Context = make(map[string]interface{})
	}
	o.Context[key] = value
	return o
}

// WithUserMessage 添加用户消息
func (o *OperationOutcome) WithUserMessage(msg string) *OperationOutcome {
	o.UserMessage = msg
	return o
}

// WithSuggestions 添加建议
func (o *OperationOutcome) WithSuggestions(suggestions ...string) *OperationOutcome {
	o.Suggestions = suggestions
	return o
}

// ShouldNotifyUser 是否应该通知用户
func (o *OperationOutcome) ShouldNotifyUser() bool {
	// 成功时不需要特别通知（除非有特殊消息）
	if o.Success {
		return o.UserMessage != ""
	}
	// 失败时都应该通知
	return true
}

// IsFatal 是否是致命错误
func (o *OperationOutcome) IsFatal() bool {
	return o.Severity == SeverityFatal
}

// IsRecoverable 是否可恢复
func (o *OperationOutcome) IsRecoverable() bool {
	return o.Severity == SeverityRecoverable && o.CanRetry && o.RetryCount < o.MaxRetries
}

// IncrementRetry 增加重试计数
func (o *OperationOutcome) IncrementRetry() {
	o.RetryCount++
}

// Error 实现 error 接口
func (o *OperationOutcome) Error() string {
	if o.OriginalError != nil {
		return o.Message + ": " + o.OriginalError.Error()
	}
	return o.Message
}

// ErrorPattern 错误模式定义
type ErrorPattern struct {
	// Code 错误代码
	Code string

	// Severity 严重程度
	Severity ErrorSeverity

	// CanAutoRecover 是否可以自动恢复
	CanAutoRecover bool

	// Strategy 恢复策略
	Strategy RecoveryStrategy

	// DefaultUserMessage 默认用户消息（如果 LLM 不可用）
	DefaultUserMessage string

	// DefaultSuggestions 默认建议
	DefaultSuggestions []string
}

// LLMExplanation LLM 生成的解释结果
type LLMExplanation struct {
	// UserMessage 给用户的消息
	UserMessage string `json:"user_message"`

	// SuggestedActions 建议的操作
	SuggestedActions []string `json:"suggested_actions"`

	// CanContinue 是否可以继续对话
	CanContinue bool `json:"can_continue"`
}
