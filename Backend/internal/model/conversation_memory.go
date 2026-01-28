package model

import (
	"database/sql"
	"encoding/json"
	"time"
)

// MemoryType 记忆类型
type MemoryType string

const (
	MemoryTypeCommitment MemoryType = "commitment" // 约定/承诺
	MemoryTypeEmotion    MemoryType = "emotion"    // 情感表达
	MemoryTypeTopic      MemoryType = "topic"      // 重要话题
	MemoryTypePattern    MemoryType = "pattern"    // 行为模式
)

// MemoryItem 单条记忆
type MemoryItem struct {
	Type        MemoryType `json:"type"`         // 记忆类型
	Content     string     `json:"content"`      // 信息内容
	Timestamp   time.Time  `json:"timestamp"`    // 原始时间戳
	TimeContext string     `json:"time_context"` // 时间上下文描述
	Sender      string     `json:"sender"`       // 谁说的
	Importance  int        `json:"importance"`   // 重要程度 1-5
}

// ConversationMemory 会话记忆
type ConversationMemory struct {
	ID             int64          `db:"id"`
	ConversationID string         `db:"conversation_id"`
	Memories       sql.NullString `db:"memories"`       // JSON 字符串
	Summary        sql.NullString `db:"summary"`        // 对话总结
	LastSummaryAt  sql.NullTime   `db:"last_summary_at"`
	TotalMessages  int            `db:"total_messages"`
	CreatedAt      time.Time      `db:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at"`
}

// GetMemories 解析记忆列表
func (m *ConversationMemory) GetMemories() ([]MemoryItem, error) {
	if !m.Memories.Valid || m.Memories.String == "" {
		return []MemoryItem{}, nil
	}

	var items []MemoryItem
	if err := json.Unmarshal([]byte(m.Memories.String), &items); err != nil {
		return nil, err
	}
	return items, nil
}

// SetMemories 设置记忆列表
func (m *ConversationMemory) SetMemories(items []MemoryItem) error {
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	m.Memories = sql.NullString{String: string(data), Valid: true}
	return nil
}

// AddMemory 添加一条记忆
func (m *ConversationMemory) AddMemory(item MemoryItem) error {
	items, err := m.GetMemories()
	if err != nil {
		return err
	}

	// 限制记忆数量，保留最重要的
	items = append(items, item)
	if len(items) > 20 {
		items = pruneMemories(items, 15)
	}

	return m.SetMemories(items)
}

// pruneMemories 修剪记忆，保留最重要的 N 条
func pruneMemories(items []MemoryItem, keepCount int) []MemoryItem {
	if len(items) <= keepCount {
		return items
	}

	// 按重要性排序（简单实现：保留高重要性的 + 最近的）
	// commitment 类型始终保留
	var commitments []MemoryItem
	var others []MemoryItem

	for _, item := range items {
		if item.Type == MemoryTypeCommitment && item.Importance >= 4 {
			commitments = append(commitments, item)
		} else {
			others = append(others, item)
		}
	}

	// 按时间倒序保留 others
	if len(others) > keepCount-len(commitments) {
		others = others[len(others)-(keepCount-len(commitments)):]
	}

	result := append(commitments, others...)
	return result
}

// CuratedContext Curator 输出的精选上下文
type CuratedContext struct {
	Context       string       `json:"context"`        // 精选的对话上下文
	KeyPoints     []string     `json:"key_points"`     // 关键点列表
	MemoryUpdates []MemoryItem `json:"memory_updates"` // 需要更新的记忆
	State         string       `json:"state"`          // 对话状态: waiting_for_reply, in_discussion, ending
}
