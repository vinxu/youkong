package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

// ContextRepository 对话上下文数据访问层
type ContextRepository struct {
	db *sqlx.DB
}

// NewContextRepository 创建对话上下文数据访问层
func NewContextRepository(db *sqlx.DB) *ContextRepository {
	return &ContextRepository{db: db}
}

// GetByConversationID 根据会话ID获取上下文
func (r *ContextRepository) GetByConversationID(ctx context.Context, conversationID string) (*model.ConversationContext, error) {
	var convCtx model.ConversationContext
	query := `SELECT * FROM conversation_contexts WHERE conversation_id = ?`
	err := r.db.GetContext(ctx, &convCtx, query, conversationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &convCtx, nil
}

// Create 创建上下文
func (r *ContextRepository) Create(ctx context.Context, convCtx *model.ConversationContext) error {
	query := `INSERT INTO conversation_contexts
              (conversation_id, messages, token_count, summary, last_sync_msg_id, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		convCtx.ConversationID,
		convCtx.Messages,
		convCtx.TokenCount,
		convCtx.Summary,
		convCtx.LastSyncMsgID,
		now,
		now,
	)
	return err
}

// Update 更新上下文
func (r *ContextRepository) Update(ctx context.Context, convCtx *model.ConversationContext) error {
	query := `UPDATE conversation_contexts SET
              messages = ?,
              token_count = ?,
              summary = ?,
              last_sync_msg_id = ?,
              updated_at = ?
              WHERE conversation_id = ?`
	_, err := r.db.ExecContext(ctx, query,
		convCtx.Messages,
		convCtx.TokenCount,
		convCtx.Summary,
		convCtx.LastSyncMsgID,
		time.Now(),
		convCtx.ConversationID,
	)
	return err
}

// Delete 删除上下文
func (r *ContextRepository) Delete(ctx context.Context, conversationID string) error {
	query := `DELETE FROM conversation_contexts WHERE conversation_id = ?`
	_, err := r.db.ExecContext(ctx, query, conversationID)
	return err
}

// GetOrCreate 获取或创建上下文
func (r *ContextRepository) GetOrCreate(ctx context.Context, conversationID string, initialMessages string) (*model.ConversationContext, error) {
	convCtx, err := r.GetByConversationID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if convCtx != nil {
		return convCtx, nil
	}

	// 创建新的上下文
	now := time.Now()
	newCtx := &model.ConversationContext{
		ConversationID: conversationID,
		Messages:       initialMessages,
		TokenCount:     len(initialMessages) / 4,
		Summary:        "",
		LastSyncMsgID:  "",
		UpdatedAt:      now,
		CreatedAt:      now,
	}
	if err := r.Create(ctx, newCtx); err != nil {
		return nil, err
	}
	return newCtx, nil
}

// UpdateLastSyncMsgID 更新最后同步的消息ID
func (r *ContextRepository) UpdateLastSyncMsgID(ctx context.Context, conversationID, msgID string) error {
	query := `UPDATE conversation_contexts SET last_sync_msg_id = ?, updated_at = NOW() WHERE conversation_id = ?`
	_, err := r.db.ExecContext(ctx, query, msgID, conversationID)
	return err
}

// ResetContext 重置上下文（保留总结）
func (r *ContextRepository) ResetContext(ctx context.Context, conversationID, messages, summary string) error {
	query := `UPDATE conversation_contexts SET
              messages = ?,
              token_count = ?,
              summary = ?,
              updated_at = NOW()
              WHERE conversation_id = ?`
	_, err := r.db.ExecContext(ctx, query, messages, len(messages)/4, summary, conversationID)
	return err
}
