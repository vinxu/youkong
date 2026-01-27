package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

type MessageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) CreateConversation(ctx context.Context, conv *model.Conversation) error {
	query := `INSERT INTO conversations (id, user1_id, user2_id, last_message_at, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, conv.ID, conv.User1ID, conv.User2ID, conv.LastMessageAt, conv.CreatedAt)
	return err
}

func (r *MessageRepository) GetConversationByID(ctx context.Context, id string) (*model.Conversation, error) {
	var conv model.Conversation
	query := `SELECT * FROM conversations WHERE id = ?`
	err := r.db.GetContext(ctx, &conv, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &conv, nil
}

func (r *MessageRepository) GetConversationByUsers(ctx context.Context, user1ID, user2ID string) (*model.Conversation, error) {
	var conv model.Conversation
	query := `SELECT * FROM conversations WHERE (user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)`
	err := r.db.GetContext(ctx, &conv, query, user1ID, user2ID, user2ID, user1ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &conv, nil
}

func (r *MessageRepository) GetConversationsByUserID(ctx context.Context, userID string) ([]*model.Conversation, error) {
	var convs []*model.Conversation
	query := `SELECT * FROM conversations WHERE user1_id = ? OR user2_id = ? ORDER BY last_message_at DESC`
	err := r.db.SelectContext(ctx, &convs, query, userID, userID)
	return convs, err
}

func (r *MessageRepository) UpdateConversationLastMessage(ctx context.Context, id string) error {
	query := `UPDATE conversations SET last_message_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *MessageRepository) CreateMessage(ctx context.Context, msg *model.Message) error {
	query := `INSERT INTO messages (id, conversation_id, sender_id, type, content, metadata, created_at)
              VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		msg.ID, msg.ConversationID, msg.SenderID, msg.Type, msg.Content, msg.Metadata, msg.CreatedAt)
	return err
}

func (r *MessageRepository) GetMessageByID(ctx context.Context, id string) (*model.Message, error) {
	var msg model.Message
	query := `SELECT * FROM messages WHERE id = ?`
	err := r.db.GetContext(ctx, &msg, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &msg, nil
}

func (r *MessageRepository) GetMessagesByConversationID(ctx context.Context, conversationID string, limit, offset int) ([]*model.Message, error) {
	var messages []*model.Message
	query := `SELECT * FROM messages WHERE conversation_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.db.SelectContext(ctx, &messages, query, conversationID, limit, offset)
	return messages, err
}

func (r *MessageRepository) MarkAsRead(ctx context.Context, messageID string) error {
	query := `UPDATE messages SET read_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), messageID)
	return err
}

func (r *MessageRepository) MarkAllAsRead(ctx context.Context, conversationID, userID string) error {
	query := `UPDATE messages SET read_at = ? WHERE conversation_id = ? AND sender_id != ? AND read_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, time.Now(), conversationID, userID)
	return err
}

func (r *MessageRepository) GetUnreadCount(ctx context.Context, conversationID, userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM messages WHERE conversation_id = ? AND sender_id != ? AND read_at IS NULL`
	err := r.db.GetContext(ctx, &count, query, conversationID, userID)
	return count, err
}

func (r *MessageRepository) GetLastMessage(ctx context.Context, conversationID string) (*model.Message, error) {
	var msg model.Message
	query := `SELECT * FROM messages WHERE conversation_id = ? ORDER BY created_at DESC LIMIT 1`
	err := r.db.GetContext(ctx, &msg, query, conversationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &msg, nil
}

// GetMessagesAfterID 获取指定消息ID之后的消息
func (r *MessageRepository) GetMessagesAfterID(ctx context.Context, conversationID, afterMsgID string, limit int) ([]*model.Message, error) {
	var messages []*model.Message

	if afterMsgID == "" {
		// 没有指定消息ID，返回空
		return messages, nil
	}

	// 先获取指定消息的创建时间
	var afterMsg model.Message
	query := `SELECT * FROM messages WHERE id = ?`
	err := r.db.GetContext(ctx, &afterMsg, query, afterMsgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return messages, nil
		}
		return nil, err
	}

	// 获取该时间之后的消息
	query = `SELECT * FROM messages WHERE conversation_id = ? AND created_at > ? ORDER BY created_at ASC LIMIT ?`
	err = r.db.SelectContext(ctx, &messages, query, conversationID, afterMsg.CreatedAt, limit)
	return messages, err
}

// GetByConversationID 获取会话的消息（按时间正序）
func (r *MessageRepository) GetByConversationID(ctx context.Context, conversationID string, limit int) ([]*model.Message, error) {
	var messages []*model.Message
	query := `SELECT * FROM messages WHERE conversation_id = ? ORDER BY created_at ASC LIMIT ?`
	err := r.db.SelectContext(ctx, &messages, query, conversationID, limit)
	return messages, err
}

// GetTotalUnreadCount 获取用户所有会话的总未读消息数
func (r *MessageRepository) GetTotalUnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM messages m
		JOIN conversations c ON m.conversation_id = c.id
		WHERE (c.user1_id = ? OR c.user2_id = ?)
		  AND m.sender_id != ?
		  AND m.read_at IS NULL
	`
	err := r.db.GetContext(ctx, &count, query, userID, userID, userID)
	return count, err
}
