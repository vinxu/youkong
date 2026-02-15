package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"youkong/internal/model"
)

// InteractionRepository 互动数据访问层
type InteractionRepository struct {
	db *sqlx.DB
}

// NewInteractionRepository 创建互动 Repository
func NewInteractionRepository(db *sqlx.DB) *InteractionRepository {
	return &InteractionRepository{db: db}
}

// Create 创建互动记录
func (r *InteractionRepository) Create(ctx context.Context, interaction *model.Interaction) error {
	query := `
		INSERT INTO interactions (id, sender_id, receiver_id, action_emoji, action_label, action_push_text, created_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		interaction.ID,
		interaction.SenderID,
		interaction.ReceiverID,
		interaction.ActionEmoji,
		interaction.ActionLabel,
		interaction.ActionPushText,
	)
	return err
}

// GetLastBetween 获取两人之间最近一次互动
func (r *InteractionRepository) GetLastBetween(ctx context.Context, senderID, receiverID string) (*model.Interaction, error) {
	var interaction model.Interaction
	query := `
		SELECT * FROM interactions
		WHERE sender_id = ? AND receiver_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &interaction, query, senderID, receiverID)
	if err != nil {
		return nil, err
	}
	return &interaction, nil
}

// GetTodayCounts 批量获取今日每人收到的互动次数
func (r *InteractionRepository) GetTodayCounts(ctx context.Context, receiverIDs []string) (map[string]int, error) {
	counts := make(map[string]int)
	if len(receiverIDs) == 0 {
		return counts, nil
	}

	today := time.Now().Format("2006-01-02")

	query, args, err := sqlx.In(`
		SELECT receiver_id, COUNT(*) as cnt
		FROM interactions
		WHERE receiver_id IN (?) AND created_at >= ?
		GROUP BY receiver_id
	`, receiverIDs, today)
	if err != nil {
		return counts, err
	}

	query = r.db.Rebind(query)

	type countRow struct {
		ReceiverID string `db:"receiver_id"`
		Count      int    `db:"cnt"`
	}
	var rows []countRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return counts, err
	}

	for _, row := range rows {
		counts[row.ReceiverID] = row.Count
	}
	return counts, nil
}
