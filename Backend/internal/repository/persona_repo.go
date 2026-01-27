package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

// PersonaRepository 人设数据访问层
type PersonaRepository struct {
	db *sqlx.DB
}

// NewPersonaRepository 创建人设数据访问层
func NewPersonaRepository(db *sqlx.DB) *PersonaRepository {
	return &PersonaRepository{db: db}
}

// GetByUserID 获取用户的人设
func (r *PersonaRepository) GetByUserID(ctx context.Context, userID string) (*model.UserPersona, error) {
	var persona model.UserPersona
	query := `SELECT * FROM user_personas WHERE user_id = ?`
	err := r.db.GetContext(ctx, &persona, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &persona, nil
}

// Create 创建人设
func (r *PersonaRepository) Create(ctx context.Context, persona *model.UserPersona) error {
	query := `INSERT INTO user_personas
              (user_id, personality, speaking_style, interests, social_habits, common_phrases, emoji_usage, message_length, confidence_score, sample_count, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		persona.UserID,
		persona.Personality,
		persona.SpeakingStyle,
		persona.Interests,
		persona.SocialHabits,
		persona.CommonPhrases,
		persona.EmojiUsage,
		persona.MessageLength,
		persona.ConfidenceScore,
		persona.SampleCount,
		now,
		now,
	)
	return err
}

// Update 更新人设
func (r *PersonaRepository) Update(ctx context.Context, persona *model.UserPersona) error {
	query := `UPDATE user_personas SET
              personality = ?,
              speaking_style = ?,
              interests = ?,
              social_habits = ?,
              common_phrases = ?,
              emoji_usage = ?,
              message_length = ?,
              confidence_score = ?,
              sample_count = ?,
              updated_at = ?
              WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query,
		persona.Personality,
		persona.SpeakingStyle,
		persona.Interests,
		persona.SocialHabits,
		persona.CommonPhrases,
		persona.EmojiUsage,
		persona.MessageLength,
		persona.ConfidenceScore,
		persona.SampleCount,
		time.Now(),
		persona.UserID,
	)
	return err
}

// Upsert 插入或更新人设
func (r *PersonaRepository) Upsert(ctx context.Context, persona *model.UserPersona) error {
	query := `INSERT INTO user_personas
              (user_id, personality, speaking_style, interests, social_habits, common_phrases, emoji_usage, message_length, confidence_score, sample_count, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
              ON DUPLICATE KEY UPDATE
              personality = VALUES(personality),
              speaking_style = VALUES(speaking_style),
              interests = VALUES(interests),
              social_habits = VALUES(social_habits),
              common_phrases = VALUES(common_phrases),
              emoji_usage = VALUES(emoji_usage),
              message_length = VALUES(message_length),
              confidence_score = VALUES(confidence_score),
              sample_count = VALUES(sample_count),
              updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		persona.UserID,
		persona.Personality,
		persona.SpeakingStyle,
		persona.Interests,
		persona.SocialHabits,
		persona.CommonPhrases,
		persona.EmojiUsage,
		persona.MessageLength,
		persona.ConfidenceScore,
		persona.SampleCount,
	)
	return err
}

// IncrementSampleCount 增加样本数量并更新置信度
func (r *PersonaRepository) IncrementSampleCount(ctx context.Context, userID string) error {
	query := `UPDATE user_personas SET
              sample_count = sample_count + 1,
              confidence_score = LEAST(100, (sample_count + 1) * 2),
              updated_at = NOW()
              WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// GetOrCreate 获取或创建默认人设
func (r *PersonaRepository) GetOrCreate(ctx context.Context, userID string) (*model.UserPersona, error) {
	persona, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if persona != nil {
		return persona, nil
	}

	// 创建默认人设
	defaultPersona := model.DefaultPersona(userID)
	if err := r.Create(ctx, defaultPersona); err != nil {
		return nil, err
	}
	return defaultPersona, nil
}
