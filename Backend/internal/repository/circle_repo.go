package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

type CircleRepository struct {
	db *sqlx.DB
}

func NewCircleRepository(db *sqlx.DB) *CircleRepository {
	return &CircleRepository{db: db}
}

func (r *CircleRepository) Create(ctx context.Context, circle *model.Circle) error {
	query := `INSERT INTO circles (id, name, emoji, color, owner_id, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		circle.ID, circle.Name, circle.Emoji, circle.Color, circle.OwnerID,
		circle.CreatedAt, circle.UpdatedAt)
	return err
}

func (r *CircleRepository) GetByID(ctx context.Context, id string) (*model.Circle, error) {
	var circle model.Circle
	query := `SELECT * FROM circles WHERE id = ?`
	err := r.db.GetContext(ctx, &circle, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &circle, nil
}

func (r *CircleRepository) GetByOwnerID(ctx context.Context, ownerID string) ([]*model.Circle, error) {
	var circles []*model.Circle
	query := `SELECT * FROM circles WHERE owner_id = ? ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &circles, query, ownerID)
	return circles, err
}

func (r *CircleRepository) Update(ctx context.Context, circle *model.Circle) error {
	query := `UPDATE circles SET name = ?, emoji = ?, color = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, circle.Name, circle.Emoji, circle.Color, circle.ID)
	return err
}

func (r *CircleRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM circles WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *CircleRepository) AddMember(ctx context.Context, member *model.CircleMember) error {
	query := `INSERT INTO circle_members (id, circle_id, user_id, joined_at) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, member.ID, member.CircleID, member.UserID, member.JoinedAt)
	return err
}

func (r *CircleRepository) RemoveMember(ctx context.Context, circleID, userID string) error {
	query := `DELETE FROM circle_members WHERE circle_id = ? AND user_id = ?`
	_, err := r.db.ExecContext(ctx, query, circleID, userID)
	return err
}

func (r *CircleRepository) GetMembers(ctx context.Context, circleID string) ([]*model.CircleMember, error) {
	var members []*model.CircleMember
	query := `SELECT * FROM circle_members WHERE circle_id = ?`
	err := r.db.SelectContext(ctx, &members, query, circleID)
	return members, err
}

func (r *CircleRepository) GetMemberUserIDs(ctx context.Context, circleID string) ([]string, error) {
	var userIDs []string
	query := `SELECT user_id FROM circle_members WHERE circle_id = ?`
	err := r.db.SelectContext(ctx, &userIDs, query, circleID)
	return userIDs, err
}

func (r *CircleRepository) IsMember(ctx context.Context, circleID, userID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM circle_members WHERE circle_id = ? AND user_id = ?`
	err := r.db.GetContext(ctx, &count, query, circleID, userID)
	return count > 0, err
}

func (r *CircleRepository) GetCirclesByMemberID(ctx context.Context, userID string) ([]*model.Circle, error) {
	var circles []*model.Circle
	query := `SELECT c.* FROM circles c
              INNER JOIN circle_members cm ON c.id = cm.circle_id
              WHERE cm.user_id = ?
              ORDER BY c.created_at DESC`
	err := r.db.SelectContext(ctx, &circles, query, userID)
	return circles, err
}

func (r *CircleRepository) GetUserCircleIDs(ctx context.Context, userID string) ([]string, error) {
	var circleIDs []string
	query := `SELECT circle_id FROM circle_members WHERE user_id = ?`
	err := r.db.SelectContext(ctx, &circleIDs, query, userID)
	return circleIDs, err
}

func (r *CircleRepository) GetMemberCount(ctx context.Context, circleID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM circle_members WHERE circle_id = ?`
	err := r.db.GetContext(ctx, &count, query, circleID)
	return count, err
}
