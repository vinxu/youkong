package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

type FriendshipRepository struct {
	db *sqlx.DB
}

func NewFriendshipRepository(db *sqlx.DB) *FriendshipRepository {
	return &FriendshipRepository{db: db}
}

func (r *FriendshipRepository) Create(ctx context.Context, friendship *model.Friendship) error {
	query := `INSERT INTO friendships (id, user_id, friend_id, source, invitation_id, created_at)
              VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		friendship.ID, friendship.UserID, friendship.FriendID, friendship.Source,
		friendship.InvitationID, friendship.CreatedAt)
	return err
}

func (r *FriendshipRepository) GetByUserAndFriend(ctx context.Context, userID, friendID string) (*model.Friendship, error) {
	var friendship model.Friendship
	query := `SELECT * FROM friendships WHERE user_id = ? AND friend_id = ?`
	err := r.db.GetContext(ctx, &friendship, query, userID, friendID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &friendship, nil
}

func (r *FriendshipRepository) AreFriends(ctx context.Context, userID, friendID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM friendships WHERE user_id = ? AND friend_id = ?`
	err := r.db.GetContext(ctx, &count, query, userID, friendID)
	return count > 0, err
}

func (r *FriendshipRepository) GetFriendsByUserID(ctx context.Context, userID string) ([]*model.Friendship, error) {
	var friendships []*model.Friendship
	query := `SELECT * FROM friendships WHERE user_id = ? ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &friendships, query, userID)
	return friendships, err
}

func (r *FriendshipRepository) GetFriendIDs(ctx context.Context, userID string) ([]string, error) {
	var friendIDs []string
	query := `SELECT friend_id FROM friendships WHERE user_id = ?`
	err := r.db.SelectContext(ctx, &friendIDs, query, userID)
	return friendIDs, err
}

func (r *FriendshipRepository) Delete(ctx context.Context, userID, friendID string) error {
	query := `DELETE FROM friendships WHERE user_id = ? AND friend_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID, friendID)
	return err
}

func (r *FriendshipRepository) DeleteBidirectional(ctx context.Context, userID, friendID string) error {
	query := `DELETE FROM friendships WHERE (user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)`
	_, err := r.db.ExecContext(ctx, query, userID, friendID, friendID, userID)
	return err
}

func (r *FriendshipRepository) CountFriends(ctx context.Context, userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM friendships WHERE user_id = ?`
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}

func (r *FriendshipRepository) GetFriendshipsBySource(ctx context.Context, userID string, source model.FriendshipSource) ([]*model.Friendship, error) {
	var friendships []*model.Friendship
	query := `SELECT * FROM friendships WHERE user_id = ? AND source = ? ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &friendships, query, userID, source)
	return friendships, err
}

func (r *FriendshipRepository) GetInvitedByMe(ctx context.Context, userID string) ([]*model.Friendship, error) {
	var friendships []*model.Friendship
	query := `SELECT f.* FROM friendships f
              INNER JOIN invitation_records ir ON f.invitation_id = ir.invitation_id
              WHERE f.user_id = ? AND ir.inviter_id = ?
              ORDER BY f.created_at DESC`
	err := r.db.SelectContext(ctx, &friendships, query, userID, userID)
	return friendships, err
}

func (r *FriendshipRepository) GetInvitedMe(ctx context.Context, userID string) ([]*model.Friendship, error) {
	var friendships []*model.Friendship
	query := `SELECT f.* FROM friendships f
              INNER JOIN invitation_records ir ON f.invitation_id = ir.invitation_id
              WHERE f.user_id = ? AND ir.invitee_id = ?
              ORDER BY f.created_at DESC`
	err := r.db.SelectContext(ctx, &friendships, query, userID, userID)
	return friendships, err
}
