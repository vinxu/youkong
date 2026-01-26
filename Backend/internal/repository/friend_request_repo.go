package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

type FriendRequestRepository struct {
	db *sqlx.DB
}

func NewFriendRequestRepository(db *sqlx.DB) *FriendRequestRepository {
	return &FriendRequestRepository{db: db}
}

// Create 创建好友请求
func (r *FriendRequestRepository) Create(ctx context.Context, req *model.FriendRequest) error {
	query := `INSERT INTO friend_requests (id, from_user_id, to_user_id, message, status)
			  VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, req.ID, req.FromUserID, req.ToUserID, req.Message, req.Status)
	return err
}

// GetByID 根据ID获取好友请求
func (r *FriendRequestRepository) GetByID(ctx context.Context, id string) (*model.FriendRequest, error) {
	var req model.FriendRequest
	query := `SELECT * FROM friend_requests WHERE id = ?`
	err := r.db.GetContext(ctx, &req, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &req, err
}

// GetPendingRequest 获取待处理的请求（检查是否已有请求）
func (r *FriendRequestRepository) GetPendingRequest(ctx context.Context, fromUserID, toUserID string) (*model.FriendRequest, error) {
	var req model.FriendRequest
	query := `SELECT * FROM friend_requests
			  WHERE from_user_id = ? AND to_user_id = ? AND status = 'PENDING'`
	err := r.db.GetContext(ctx, &req, query, fromUserID, toUserID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &req, err
}

// GetReceivedRequests 获取收到的好友请求（待处理）
func (r *FriendRequestRepository) GetReceivedRequests(ctx context.Context, userID string) ([]*model.FriendRequest, error) {
	var requests []*model.FriendRequest
	query := `SELECT * FROM friend_requests
			  WHERE to_user_id = ? AND status = 'PENDING'
			  ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &requests, query, userID)
	return requests, err
}

// GetSentRequests 获取发出的好友请求
func (r *FriendRequestRepository) GetSentRequests(ctx context.Context, userID string) ([]*model.FriendRequest, error) {
	var requests []*model.FriendRequest
	query := `SELECT * FROM friend_requests
			  WHERE from_user_id = ? AND status = 'PENDING'
			  ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &requests, query, userID)
	return requests, err
}

// UpdateStatus 更新请求状态
func (r *FriendRequestRepository) UpdateStatus(ctx context.Context, id string, status model.FriendRequestStatus) error {
	query := `UPDATE friend_requests SET status = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// GetPendingCount 获取待处理请求数量
func (r *FriendRequestRepository) GetPendingCount(ctx context.Context, userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM friend_requests WHERE to_user_id = ? AND status = 'PENDING'`
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}
