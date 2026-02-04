package agent

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SessionConfig 会话配置
type SessionConfig struct {
	TokenBudget      int           // Token 预算，默认 8000
	MaxIterations    int           // 最大迭代次数，默认 10
	SummaryThreshold int           // 触发总结的阈值，默认 6000
	TTL              time.Duration // 会话过期时间，默认 30 分钟
}

// DefaultSessionConfig 默认会话配置
var DefaultSessionConfig = SessionConfig{
	TokenBudget:      8000,
	MaxIterations:    10,
	SummaryThreshold: 6000,
	TTL:              30 * time.Minute,
}

// AgentSession Agent 会话状态
type AgentSession struct {
	SessionID string         `json:"session_id"`
	UserID    string         `json:"user_id"`
	Messages  []AgentMessage `json:"messages"`

	// 上下文管理
	TokenCount  int    `json:"token_count"`
	TokenBudget int    `json:"token_budget"`
	Summary     string `json:"summary,omitempty"` // 历史对话摘要

	// 循环控制
	CurrentIteration int `json:"current_iteration"`
	MaxIterations    int `json:"max_iterations"`

	// 元数据
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewSession 创建新会话
func NewSession(userID string, config *SessionConfig) *AgentSession {
	if config == nil {
		config = &DefaultSessionConfig
	}

	now := time.Now()
	return &AgentSession{
		SessionID:        uuid.New().String(),
		UserID:           userID,
		Messages:         []AgentMessage{},
		TokenCount:       0,
		TokenBudget:      config.TokenBudget,
		CurrentIteration: 0,
		MaxIterations:    config.MaxIterations,
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        now.Add(config.TTL),
	}
}

// AddMessage 添加消息
func (s *AgentSession) AddMessage(msg AgentMessage) {
	s.Messages = append(s.Messages, msg)
	s.TokenCount += msg.EstimateTokens()
	s.UpdatedAt = time.Now()
}

// AddSystemMessage 添加系统消息（如果不存在则添加到开头）
func (s *AgentSession) AddSystemMessage(content string) {
	// 检查是否已有系统消息
	for i, msg := range s.Messages {
		if msg.Role == "system" {
			// 替换现有系统消息
			s.TokenCount -= msg.EstimateTokens()
			s.Messages[i] = NewSystemMessage(content)
			s.TokenCount += s.Messages[i].EstimateTokens()
			s.UpdatedAt = time.Now()
			return
		}
	}

	// 在开头插入新的系统消息
	newMsg := NewSystemMessage(content)
	s.Messages = append([]AgentMessage{newMsg}, s.Messages...)
	s.TokenCount += newMsg.EstimateTokens()
	s.UpdatedAt = time.Now()
}

// GetMessages 获取所有消息
func (s *AgentSession) GetMessages() []AgentMessage {
	return s.Messages
}

// GetLastMessage 获取最后一条消息
func (s *AgentSession) GetLastMessage() *AgentMessage {
	if len(s.Messages) == 0 {
		return nil
	}
	return &s.Messages[len(s.Messages)-1]
}

// NeedsSummarization 检查是否需要总结压缩
func (s *AgentSession) NeedsSummarization(threshold int) bool {
	if threshold <= 0 {
		threshold = DefaultSessionConfig.SummaryThreshold
	}
	return s.TokenCount > threshold
}

// CanContinue 检查是否可以继续迭代
func (s *AgentSession) CanContinue() bool {
	return s.CurrentIteration < s.MaxIterations && !s.IsExpired()
}

// IsExpired 检查会话是否过期
func (s *AgentSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// ResetIteration 重置迭代计数
func (s *AgentSession) ResetIteration() {
	s.CurrentIteration = 0
}

// IncrementIteration 增加迭代计数
func (s *AgentSession) IncrementIteration() {
	s.CurrentIteration++
}

// SetSummary 设置历史摘要
func (s *AgentSession) SetSummary(summary string) {
	s.Summary = summary
}

// RebuildWithSummary 使用摘要重建消息列表
func (s *AgentSession) RebuildWithSummary(summary string, keepLastN int) {
	if len(s.Messages) <= keepLastN+1 { // +1 for system message
		return
	}

	var systemMsg *AgentMessage
	startIdx := 0

	// 保留系统消息
	if len(s.Messages) > 0 && s.Messages[0].Role == "system" {
		systemMsg = &s.Messages[0]
		startIdx = 1
	}

	// 获取最近的消息
	recentStart := len(s.Messages) - keepLastN
	if recentStart < startIdx {
		recentStart = startIdx
	}
	recentMsgs := s.Messages[recentStart:]

	// 重建消息列表
	newMessages := []AgentMessage{}

	if systemMsg != nil {
		newMessages = append(newMessages, *systemMsg)
	}

	// 添加摘要消息
	summaryMsg := NewSystemMessage("[历史对话摘要]\n" + summary)
	newMessages = append(newMessages, summaryMsg)

	// 添加最近消息
	newMessages = append(newMessages, recentMsgs...)

	// 更新会话
	s.Messages = newMessages
	s.Summary = summary
	s.RecalculateTokens()
	s.UpdatedAt = time.Now()
}

// RecalculateTokens 重新计算 Token 数量
func (s *AgentSession) RecalculateTokens() {
	s.TokenCount = 0
	for _, msg := range s.Messages {
		s.TokenCount += msg.EstimateTokens()
	}
}

// Extend 延长会话过期时间
func (s *AgentSession) Extend(duration time.Duration) {
	s.ExpiresAt = time.Now().Add(duration)
	s.UpdatedAt = time.Now()
}

// ToJSON 序列化为 JSON
func (s *AgentSession) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// FromJSON 从 JSON 反序列化
func (s *AgentSession) FromJSON(data []byte) error {
	return json.Unmarshal(data, s)
}

// Clone 克隆会话
func (s *AgentSession) Clone() *AgentSession {
	clone := &AgentSession{
		SessionID:        s.SessionID,
		UserID:           s.UserID,
		Messages:         make([]AgentMessage, len(s.Messages)),
		TokenCount:       s.TokenCount,
		TokenBudget:      s.TokenBudget,
		Summary:          s.Summary,
		CurrentIteration: s.CurrentIteration,
		MaxIterations:    s.MaxIterations,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
		ExpiresAt:        s.ExpiresAt,
	}
	copy(clone.Messages, s.Messages)
	return clone
}
