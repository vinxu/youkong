package service

import (
	"strings"

	"youkong/internal/model"
)

// ErrorClassifier 错误分类器
// 根据错误代码和上下文决定如何处理错误
type ErrorClassifier struct {
	patterns map[string]*model.ErrorPattern
}

// NewErrorClassifier 创建错误分类器
func NewErrorClassifier() *ErrorClassifier {
	c := &ErrorClassifier{
		patterns: make(map[string]*model.ErrorPattern),
	}
	c.registerDefaultPatterns()
	return c
}

// registerDefaultPatterns 注册默认错误模式
func (c *ErrorClassifier) registerDefaultPatterns() {
	// ========== 信息性错误（不影响流程） ==========

	// 未知 action：让 LLM 解释
	c.patterns["UNKNOWN_ACTION"] = &model.ErrorPattern{
		Code:               "UNKNOWN_ACTION",
		Severity:           model.SeverityInfo,
		CanAutoRecover:     false,
		Strategy:           model.StrategyExplain,
		DefaultUserMessage: "抱歉，我暂时不支持这个操作。您可以尝试创建时刻表、查询时刻表或修改状态。",
		DefaultSuggestions: []string{"创建时刻表", "查询时刻表", "修改状态"},
	}

	// 空 action：让 LLM 解释
	c.patterns["EMPTY_ACTION"] = &model.ErrorPattern{
		Code:               "EMPTY_ACTION",
		Severity:           model.SeverityInfo,
		CanAutoRecover:     false,
		Strategy:           model.StrategyExplain,
		DefaultUserMessage: "我没有理解您的意思，请换个说法试试。",
		DefaultSuggestions: []string{"告诉我今天的安排", "创建时刻表"},
	}

	// ========== 可恢复错误 ==========

	// LLM 超时：重试
	c.patterns["LLM_TIMEOUT"] = &model.ErrorPattern{
		Code:               "LLM_TIMEOUT",
		Severity:           model.SeverityRecoverable,
		CanAutoRecover:     true,
		Strategy:           model.StrategyRetry,
		DefaultUserMessage: "思考了太久，让我再试一次...",
		DefaultSuggestions: []string{},
	}

	// LLM 解析失败：降级到友好提示
	c.patterns["LLM_PARSE_ERROR"] = &model.ErrorPattern{
		Code:               "LLM_PARSE_ERROR",
		Severity:           model.SeverityRecoverable,
		CanAutoRecover:     true,
		Strategy:           model.StrategyFallback,
		DefaultUserMessage: "我没有完全理解，能再说一遍吗？",
		DefaultSuggestions: []string{"今天下午开会", "查看今天的安排"},
	}

	// LLM 调用失败：重试
	c.patterns["LLM_CALL_FAILED"] = &model.ErrorPattern{
		Code:               "LLM_CALL_FAILED",
		Severity:           model.SeverityRecoverable,
		CanAutoRecover:     true,
		Strategy:           model.StrategyRetry,
		DefaultUserMessage: "网络有点慢，稍等一下...",
		DefaultSuggestions: []string{},
	}

	// 会话保存失败：通知用户但继续
	c.patterns["SESSION_SAVE_FAILED"] = &model.ErrorPattern{
		Code:               "SESSION_SAVE_FAILED",
		Severity:           model.SeverityRecoverable,
		CanAutoRecover:     false,
		Strategy:           model.StrategyExplain,
		DefaultUserMessage: "保存进度时遇到问题，操作可能需要重来。",
		DefaultSuggestions: []string{"重新开始"},
	}

	// ========== 致命错误 ==========

	// 数据库错误
	c.patterns["DB_ERROR"] = &model.ErrorPattern{
		Code:               "DB_ERROR",
		Severity:           model.SeverityFatal,
		CanAutoRecover:     false,
		Strategy:           model.StrategyExplain,
		DefaultUserMessage: "系统暂时出了点问题，请稍后再试。",
		DefaultSuggestions: []string{"稍后重试"},
	}

	// Redis 错误
	c.patterns["REDIS_ERROR"] = &model.ErrorPattern{
		Code:               "REDIS_ERROR",
		Severity:           model.SeverityRecoverable,
		CanAutoRecover:     false,
		Strategy:           model.StrategyExplain,
		DefaultUserMessage: "缓存服务暂时不可用，操作可能会慢一点。",
		DefaultSuggestions: []string{},
	}

	// ASR 识别失败
	c.patterns["ASR_FAILED"] = &model.ErrorPattern{
		Code:               "ASR_FAILED",
		Severity:           model.SeverityRecoverable,
		CanAutoRecover:     false,
		Strategy:           model.StrategyExplain,
		DefaultUserMessage: "没听清楚，能再说一遍吗？",
		DefaultSuggestions: []string{"再说一遍", "改用文字输入"},
	}

	// 权限不足
	c.patterns["PERMISSION_DENIED"] = &model.ErrorPattern{
		Code:               "PERMISSION_DENIED",
		Severity:           model.SeverityFatal,
		CanAutoRecover:     false,
		Strategy:           model.StrategyExplain,
		DefaultUserMessage: "没有权限执行这个操作。",
		DefaultSuggestions: []string{},
	}

	// 功能不支持
	c.patterns["FEATURE_NOT_SUPPORTED"] = &model.ErrorPattern{
		Code:               "FEATURE_NOT_SUPPORTED",
		Severity:           model.SeverityInfo,
		CanAutoRecover:     false,
		Strategy:           model.StrategyExplain,
		DefaultUserMessage: "这个功能还在开发中。",
		DefaultSuggestions: []string{"创建时刻表", "查询时刻表"},
	}

	// 会话过期
	c.patterns["SESSION_EXPIRED"] = &model.ErrorPattern{
		Code:               "SESSION_EXPIRED",
		Severity:           model.SeverityInfo,
		CanAutoRecover:     false,
		Strategy:           model.StrategyExplain,
		DefaultUserMessage: "对话已超时，请重新开始。",
		DefaultSuggestions: []string{"重新开始"},
	}

	// 无法撤销
	c.patterns["UNDO_NOT_AVAILABLE"] = &model.ErrorPattern{
		Code:               "UNDO_NOT_AVAILABLE",
		Severity:           model.SeverityInfo,
		CanAutoRecover:     false,
		Strategy:           model.StrategyExplain,
		DefaultUserMessage: "没有可以撤销的操作。",
		DefaultSuggestions: []string{},
	}

	// 撤销超时
	c.patterns["UNDO_EXPIRED"] = &model.ErrorPattern{
		Code:               "UNDO_EXPIRED",
		Severity:           model.SeverityInfo,
		CanAutoRecover:     false,
		Strategy:           model.StrategyExplain,
		DefaultUserMessage: "撤销时间已过（5分钟内可撤销）。",
		DefaultSuggestions: []string{"重新创建时刻表"},
	}
}

// Classify 对操作结果进行分类
func (c *ErrorClassifier) Classify(outcome *model.OperationOutcome) *ClassificationResult {
	if outcome.Success {
		return &ClassificationResult{
			Pattern:        nil,
			CanAutoRecover: false,
			Strategy:       model.StrategyIgnore,
		}
	}

	// 精确匹配
	if pattern, ok := c.patterns[outcome.Code]; ok {
		return &ClassificationResult{
			Pattern:        pattern,
			CanAutoRecover: pattern.CanAutoRecover,
			Strategy:       pattern.Strategy,
		}
	}

	// 模糊匹配（基于错误代码前缀）
	for code, pattern := range c.patterns {
		if strings.HasPrefix(outcome.Code, code) {
			return &ClassificationResult{
				Pattern:        pattern,
				CanAutoRecover: pattern.CanAutoRecover,
				Strategy:       pattern.Strategy,
			}
		}
	}

	// 默认策略：让 LLM 解释
	return &ClassificationResult{
		Pattern:        nil,
		CanAutoRecover: false,
		Strategy:       model.StrategyExplain,
	}
}

// GetPattern 获取指定代码的错误模式
func (c *ErrorClassifier) GetPattern(code string) *model.ErrorPattern {
	return c.patterns[code]
}

// RegisterPattern 注册新的错误模式
func (c *ErrorClassifier) RegisterPattern(pattern *model.ErrorPattern) {
	c.patterns[pattern.Code] = pattern
}

// ClassificationResult 分类结果
type ClassificationResult struct {
	// Pattern 匹配的错误模式（可能为 nil）
	Pattern *model.ErrorPattern

	// CanAutoRecover 是否可以自动恢复
	CanAutoRecover bool

	// Strategy 推荐的恢复策略
	Strategy model.RecoveryStrategy
}

// GetDefaultMessage 获取默认消息
func (r *ClassificationResult) GetDefaultMessage() string {
	if r.Pattern != nil {
		return r.Pattern.DefaultUserMessage
	}
	return "遇到了一些问题，请稍后再试。"
}

// GetDefaultSuggestions 获取默认建议
func (r *ClassificationResult) GetDefaultSuggestions() []string {
	if r.Pattern != nil {
		return r.Pattern.DefaultSuggestions
	}
	return []string{"重试"}
}
