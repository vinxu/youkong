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

// GetLatestPlusOneMap 批量获取某用户对每个好友最近一次 +1 的时间
func (r *InteractionRepository) GetLatestPlusOneMap(ctx context.Context, senderID string, receiverIDs []string) (map[string]time.Time, error) {
	result := make(map[string]time.Time)
	if len(receiverIDs) == 0 {
		return result, nil
	}

	query, args, err := sqlx.In(`
		SELECT receiver_id, MAX(created_at) as latest
		FROM interactions
		WHERE sender_id = ? AND receiver_id IN (?) AND action_label = '+1'
		GROUP BY receiver_id
	`, senderID, receiverIDs)
	if err != nil {
		return result, err
	}

	query = r.db.Rebind(query)

	type row struct {
		ReceiverID string    `db:"receiver_id"`
		Latest     time.Time `db:"latest"`
	}
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return result, err
	}

	for _, r := range rows {
		result[r.ReceiverID] = r.Latest
	}
	return result, nil
}

// GetTodayPlusOnes 获取某用户某天收到的所有 +1 互动记录
func (r *InteractionRepository) GetTodayPlusOnes(ctx context.Context, receiverID string, date string) ([]model.Interaction, error) {
	var interactions []model.Interaction
	query := `
		SELECT * FROM interactions
		WHERE receiver_id = ? AND action_label = '+1' AND DATE(created_at) = ?
		ORDER BY created_at ASC
	`
	err := r.db.SelectContext(ctx, &interactions, query, receiverID, date)
	return interactions, err
}

// GetPlusOnesSince 批量获取每人在指定时间之后收到的所有 +1 互动记录
func (r *InteractionRepository) GetPlusOnesSince(ctx context.Context, receiverIDs []string, since time.Time) (map[string][]model.Interaction, error) {
	result := make(map[string][]model.Interaction)
	if len(receiverIDs) == 0 {
		return result, nil
	}

	query, args, err := sqlx.In(`
		SELECT * FROM interactions
		WHERE receiver_id IN (?) AND action_label = '+1' AND created_at >= ?
		ORDER BY created_at ASC
	`, receiverIDs, since)
	if err != nil {
		return result, err
	}

	query = r.db.Rebind(query)

	var interactions []model.Interaction
	if err := r.db.SelectContext(ctx, &interactions, query, args...); err != nil {
		return result, err
	}

	for _, inter := range interactions {
		result[inter.ReceiverID] = append(result[inter.ReceiverID], inter)
	}
	return result, nil
}

// GetPlusOnesSinceTime 批量获取每人在指定时间之后收到的 +1 次数
func (r *InteractionRepository) GetPlusOnesSinceTime(ctx context.Context, receiverIDs []string, sinceTime time.Time) (map[string]int, error) {
	counts := make(map[string]int)
	if len(receiverIDs) == 0 {
		return counts, nil
	}

	query, args, err := sqlx.In(`
		SELECT receiver_id, COUNT(*) as cnt
		FROM interactions
		WHERE receiver_id IN (?) AND action_label = '+1' AND created_at >= ?
		GROUP BY receiver_id
	`, receiverIDs, sinceTime)
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
