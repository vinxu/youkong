package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

type WechatRepository struct {
	db *sqlx.DB
}

func NewWechatRepository(db *sqlx.DB) *WechatRepository {
	return &WechatRepository{db: db}
}

func (r *WechatRepository) Create(ctx context.Context, wechatUser *model.WechatUser) error {
	query := `INSERT INTO wechat_users (id, user_id, openid, unionid, nickname, avatar, created_at)
              VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		wechatUser.ID, wechatUser.UserID, wechatUser.OpenID, wechatUser.UnionID,
		wechatUser.Nickname, wechatUser.Avatar, wechatUser.CreatedAt)
	return err
}

func (r *WechatRepository) GetByOpenID(ctx context.Context, openID string) (*model.WechatUser, error) {
	var wechatUser model.WechatUser
	query := `SELECT * FROM wechat_users WHERE openid = ?`
	err := r.db.GetContext(ctx, &wechatUser, query, openID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &wechatUser, nil
}

func (r *WechatRepository) GetByUnionID(ctx context.Context, unionID string) (*model.WechatUser, error) {
	var wechatUser model.WechatUser
	query := `SELECT * FROM wechat_users WHERE unionid = ?`
	err := r.db.GetContext(ctx, &wechatUser, query, unionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &wechatUser, nil
}

func (r *WechatRepository) GetByUserID(ctx context.Context, userID string) (*model.WechatUser, error) {
	var wechatUser model.WechatUser
	query := `SELECT * FROM wechat_users WHERE user_id = ?`
	err := r.db.GetContext(ctx, &wechatUser, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &wechatUser, nil
}

func (r *WechatRepository) Update(ctx context.Context, wechatUser *model.WechatUser) error {
	query := `UPDATE wechat_users SET nickname = ?, avatar = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, wechatUser.Nickname, wechatUser.Avatar, wechatUser.ID)
	return err
}

func (r *WechatRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM wechat_users WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
