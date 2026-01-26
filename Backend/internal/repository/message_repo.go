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
