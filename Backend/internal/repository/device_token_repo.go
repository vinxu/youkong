package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

type DeviceTokenRepository struct {
	db *sqlx.DB
}

func NewDeviceTokenRepository(db *sqlx.DB) *DeviceTokenRepository {
	return &DeviceTokenRepository{db: db}
}

// Upsert 创建或更新设备 Token
func (r *DeviceTokenRepository) Upsert(ctx context.Context, dt *model.DeviceToken) error {
	query := `
		INSERT INTO device_tokens (user_id, token, platform, device_name, app_version, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, TRUE, ?, ?)
		ON DUPLICATE KEY UPDATE
			user_id = VALUES(user_id),
			platform = VALUES(platform),
			device_name = VALUES(device_name),
			app_version = VALUES(app_version),
			is_active = TRUE,
			updated_at = VALUES(updated_at)
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		dt.UserID, dt.Token, dt.Platform, dt.DeviceName, dt.AppVersion, now, now)
	return err
}

// GetByToken 根据 Token 获取设备信息
func (r *DeviceTokenRepository) GetByToken(ctx context.Context, token string) (*model.DeviceToken, error) {
	var dt model.DeviceToken
	query := `SELECT * FROM device_tokens WHERE token = ?`
	err := r.db.GetContext(ctx, &dt, query, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &dt, nil
}

// GetActiveByUserID 获取用户所有活跃设备 Token
func (r *DeviceTokenRepository) GetActiveByUserID(ctx context.Context, userID string) ([]*model.DeviceToken, error) {
	var tokens []*model.DeviceToken
	query := `SELECT * FROM device_tokens WHERE user_id = ? AND is_active = TRUE`
	err := r.db.SelectContext(ctx, &tokens, query, userID)
	return tokens, err
}

// GetActiveByUserIDs 批量获取用户的活跃设备 Token
func (r *DeviceTokenRepository) GetActiveByUserIDs(ctx context.Context, userIDs []string) ([]*model.DeviceToken, error) {
	if len(userIDs) == 0 {
		return []*model.DeviceToken{}, nil
	}
	query, args, err := sqlx.In(`SELECT * FROM device_tokens WHERE user_id IN (?) AND is_active = TRUE`, userIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var tokens []*model.DeviceToken
	err = r.db.SelectContext(ctx, &tokens, query, args...)
	return tokens, err
}

// Deactivate 停用设备 Token
func (r *DeviceTokenRepository) Deactivate(ctx context.Context, token string) error {
	query := `UPDATE device_tokens SET is_active = FALSE, updated_at = ? WHERE token = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), token)
	return err
}

// DeactivateByUserID 停用用户所有设备 Token
func (r *DeviceTokenRepository) DeactivateByUserID(ctx context.Context, userID string) error {
	query := `UPDATE device_tokens SET is_active = FALSE, updated_at = ? WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	return err
}

// DeleteInactiveOlderThan 删除指定时间之前的非活跃 Token
func (r *DeviceTokenRepository) DeleteInactiveOlderThan(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM device_tokens WHERE is_active = FALSE AND updated_at < ?`
	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
