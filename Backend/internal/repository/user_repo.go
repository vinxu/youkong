package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (id, phone, phone_hash, nickname, avatar, wechat_bound, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Phone, user.PhoneHash, user.Nickname, user.Avatar, user.WechatBound,
		user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE id = ?`
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE phone = ?`
	err := r.db.GetContext(ctx, &user, query, phone)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByPhoneHash(ctx context.Context, phoneHash string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE phone_hash = ?`
	err := r.db.GetContext(ctx, &user, query, phoneHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	query := `UPDATE users SET nickname = ?, avatar = ?, wechat_bound = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, user.Nickname, user.Avatar, user.WechatBound, time.Now(), user.ID)
	return err
}

func (r *UserRepository) UpdateLastActive(ctx context.Context, id string) error {
	query := `UPDATE users SET last_active_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *UserRepository) GetByIDs(ctx context.Context, ids []string) ([]*model.User, error) {
	if len(ids) == 0 {
		return []*model.User{}, nil
	}
	query, args, err := sqlx.In(`SELECT * FROM users WHERE id IN (?)`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var users []*model.User
	err = r.db.SelectContext(ctx, &users, query, args...)
	return users, err
}

func (r *UserRepository) SearchByNickname(ctx context.Context, keyword string, limit int) ([]*model.User, error) {
	var users []*model.User
	query := `SELECT * FROM users WHERE nickname LIKE ? LIMIT ?`
	err := r.db.SelectContext(ctx, &users, query, "%"+keyword+"%", limit)
	return users, err
}

// GetByPhoneHashes 根据手机号Hash批量查询用户
func (r *UserRepository) GetByPhoneHashes(ctx context.Context, phoneHashes []string) ([]*model.User, error) {
	if len(phoneHashes) == 0 {
		return []*model.User{}, nil
	}
	query, args, err := sqlx.In(`SELECT * FROM users WHERE phone_hash IN (?)`, phoneHashes)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var users []*model.User
	err = r.db.SelectContext(ctx, &users, query, args...)
	return users, err
}
