package model

import (
	"encoding/json"
	"time"
)

// LLMMessage LLM 消息格式
type LLMMessage struct {
	Role    string `json:"role"`    // system/user/assistant
	Content string `json:"content"`
}

// ConversationContext 对话上下文 - 每对好友一个 LLM 会话窗口
type ConversationContext struct {
	ID             int64     `json:"id" db:"id"`
	ConversationID string    `json:"conversation_id" db:"conversation_id"`
	Messages       string    `json:"messages" db:"messages"`     // JSON 存储 []LLMMessage
	TokenCount     int       `json:"token_count" db:"token_count"` // 估算 token 数
	Summary        string    `json:"summary" db:"summary"`       // 上下文总结（用于重建）
	LastSyncMsgID  string    `json:"last_sync_msg_id" db:"last_sync_msg_id"` // 最后同步的消息ID
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// GetMessages 解析消息列表
func (c *ConversationContext) GetMessages() ([]LLMMessage, error) {
	if c.Messages == "" || c.Messages == "null" {
		return []LLMMessage{}, nil
	}
	var messages []LLMMessage
	err := json.Unmarshal([]byte(c.Messages), &messages)
	return messages, err
}

// SetMessages 设置消息列表
func (c *ConversationContext) SetMessages(messages []LLMMessage) error {
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	c.Messages = string(data)
	// 估算 token 数（粗略：每4个字符约1个token）
	c.TokenCount = len(c.Messages) / 4
	return nil
}

// AddMessage 追加一条消息
func (c *ConversationContext) AddMessage(role, content string) error {
	messages, err := c.GetMessages()
	if err != nil {
		return err
	}
	messages = append(messages, LLMMessage{Role: role, Content: content})
	return c.SetMessages(messages)
}

// NeedsSummary 是否需要总结（token 超过阈值）
func (c *ConversationContext) NeedsSummary(threshold int) bool {
	return c.TokenCount > threshold
}

// Reset 重置上下文（保留 system prompt）
func (c *ConversationContext) Reset(systemPrompt, summary string) error {
	messages := []LLMMessage{
		{Role: "system", Content: systemPrompt},
	}
	c.Summary = summary
	return c.SetMessages(messages)
}

// AgentReplyResponse Agent 回复响应
type AgentReplyResponse struct {
	ID        string      `json:"id"`
	Sender    UserProfile `json:"sender"`
	Type      MessageType `json:"type"`
	Content   string      `json:"content"`
	CreatedAt time.Time   `json:"createdAt"`
}
