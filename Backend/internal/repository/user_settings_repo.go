package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

type UserSettingsRepository struct {
	db *sqlx.DB
}

func NewUserSettingsRepository(db *sqlx.DB) *UserSettingsRepository {
	return &UserSettingsRepository{db: db}
}

// Get 获取用户设置
func (r *UserSettingsRepository) Get(ctx context.Context, userID string) (*model.UserSettings, error) {
	var settings model.UserSettings
	query := `SELECT * FROM user_settings WHERE user_id = ?`
	err := r.db.GetContext(ctx, &settings, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// 返回默认设置
			return &model.UserSettings{
				UserID:             userID,
				AutoPredictEnabled: false,
			}, nil
		}
		return nil, err
	}
	return &settings, nil
}

// Upsert 更新或插入用户设置
func (r *UserSettingsRepository) Upsert(ctx context.Context, settings *model.UserSettings) error {
	query := `INSERT INTO user_settings (user_id, auto_predict_enabled, created_at, updated_at)
              VALUES (?, ?, NOW(), NOW())
              ON DUPLICATE KEY UPDATE
              auto_predict_enabled = VALUES(auto_predict_enabled),
              updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query, settings.UserID, settings.AutoPredictEnabled)
	return err
}

// GetAllWithAutoPredictEnabled 获取所有开启自动推测的用户 ID
func (r *UserSettingsRepository) GetAllWithAutoPredictEnabled(ctx context.Context) ([]string, error) {
	var userIDs []string
	query := `SELECT user_id FROM user_settings WHERE auto_predict_enabled = 1`
	err := r.db.SelectContext(ctx, &userIDs, query)
	if err != nil {
		return nil, err
	}
	return userIDs, nil
}
