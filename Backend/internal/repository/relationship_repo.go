package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

// RelationshipRepository 关系数据访问层
type RelationshipRepository struct {
	db *sqlx.DB
}

// NewRelationshipRepository 创建关系数据访问层
func NewRelationshipRepository(db *sqlx.DB) *RelationshipRepository {
	return &RelationshipRepository{db: db}
}

// Get 获取指定的关系画像
func (r *RelationshipRepository) Get(ctx context.Context, userID, friendID string) (*model.Relationship, error) {
	var rel model.Relationship
	query := `SELECT * FROM relationships WHERE user_id = ? AND friend_id = ?`
	err := r.db.GetContext(ctx, &rel, query, userID, friendID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rel, nil
}

// Create 创建关系画像
func (r *RelationshipRepository) Create(ctx context.Context, rel *model.Relationship) error {
	query := `INSERT INTO relationships
              (user_id, friend_id, closeness, interact_style, common_topics, tone_with_this, joke_level, shared_memory, message_count, confidence_score, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		rel.UserID,
		rel.FriendID,
		rel.Closeness,
		rel.InteractStyle,
		rel.CommonTopics,
		rel.ToneWithThis,
		rel.JokeLevel,
		rel.SharedMemory,
		rel.MessageCount,
		rel.ConfidenceScore,
		now,
		now,
	)
	return err
}

// Update 更新关系画像
func (r *RelationshipRepository) Update(ctx context.Context, rel *model.Relationship) error {
	query := `UPDATE relationships SET
              closeness = ?,
              interact_style = ?,
              common_topics = ?,
              tone_with_this = ?,
              joke_level = ?,
              shared_memory = ?,
              message_count = ?,
              confidence_score = ?,
              updated_at = ?
              WHERE user_id = ? AND friend_id = ?`
	_, err := r.db.ExecContext(ctx, query,
		rel.Closeness,
		rel.InteractStyle,
		rel.CommonTopics,
		rel.ToneWithThis,
		rel.JokeLevel,
		rel.SharedMemory,
		rel.MessageCount,
		rel.ConfidenceScore,
		time.Now(),
		rel.UserID,
		rel.FriendID,
	)
	return err
}

// Upsert 插入或更新关系画像
func (r *RelationshipRepository) Upsert(ctx context.Context, rel *model.Relationship) error {
	query := `INSERT INTO relationships
              (user_id, friend_id, closeness, interact_style, common_topics, tone_with_this, joke_level, shared_memory, message_count, confidence_score, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
              ON DUPLICATE KEY UPDATE
              closeness = VALUES(closeness),
              interact_style = VALUES(interact_style),
              common_topics = VALUES(common_topics),
              tone_with_this = VALUES(tone_with_this),
              joke_level = VALUES(joke_level),
              shared_memory = VALUES(shared_memory),
              message_count = VALUES(message_count),
              confidence_score = VALUES(confidence_score),
              updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		rel.UserID,
		rel.FriendID,
		rel.Closeness,
		rel.InteractStyle,
		rel.CommonTopics,
		rel.ToneWithThis,
		rel.JokeLevel,
		rel.SharedMemory,
		rel.MessageCount,
		rel.ConfidenceScore,
	)
	return err
}

// IncrementMessageCount 增加消息计数
func (r *RelationshipRepository) IncrementMessageCount(ctx context.Context, userID, friendID string) error {
	query := `UPDATE relationships SET
              message_count = message_count + 1,
              confidence_score = LEAST(100, (message_count + 1)),
              updated_at = NOW()
              WHERE user_id = ? AND friend_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID, friendID)
	return err
}

// GetOrCreate 获取或创建默认关系画像
func (r *RelationshipRepository) GetOrCreate(ctx context.Context, userID, friendID string) (*model.Relationship, error) {
	rel, err := r.Get(ctx, userID, friendID)
	if err != nil {
		return nil, err
	}
	if rel != nil {
		return rel, nil
	}

	// 创建默认关系画像
	defaultRel := model.DefaultRelationship(userID, friendID)
	if err := r.Create(ctx, defaultRel); err != nil {
		return nil, err
	}
	return defaultRel, nil
}

// GetByUserID 获取用户的所有关系画像
func (r *RelationshipRepository) GetByUserID(ctx context.Context, userID string) ([]*model.Relationship, error) {
	var rels []*model.Relationship
	query := `SELECT * FROM relationships WHERE user_id = ? ORDER BY message_count DESC`
	err := r.db.SelectContext(ctx, &rels, query, userID)
	if err != nil {
		return nil, err
	}
	return rels, nil
}
