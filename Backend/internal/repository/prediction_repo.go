package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"youkong/internal/model"
)

// PredictionRepository 推测任务数据访问层
type PredictionRepository struct {
	db *sqlx.DB
}

// NewPredictionRepository 创建推测任务 Repository
func NewPredictionRepository(db *sqlx.DB) *PredictionRepository {
	return &PredictionRepository{db: db}
}

// Create 创建推测任务
func (r *PredictionRepository) Create(ctx context.Context, task *model.PredictionTask) error {
	query := `
		INSERT INTO prediction_tasks (
			id, user_id, status, start_time, end_time, target_date,
			predicted_items, existing_items, merged_preview,
			reasoning_content, reasoning_steps, user_context,
			error_message, progress, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, NOW(), NOW()
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		task.ID,
		task.UserID,
		task.Status,
		task.StartTime,
		task.EndTime,
		task.TargetDate,
		task.PredictedItems,
		task.ExistingItems,
		task.MergedPreview,
		task.ReasoningContent,
		task.ReasoningSteps,
		task.UserContext,
		task.ErrorMessage,
		task.Progress,
	)
	return err
}

// GetByID 根据 ID 获取推测任务
func (r *PredictionRepository) GetByID(ctx context.Context, id string) (*model.PredictionTask, error) {
	var task model.PredictionTask
	query := `SELECT * FROM prediction_tasks WHERE id = ?`
	err := r.db.GetContext(ctx, &task, query, id)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetLatestByUser 获取用户最近的推测任务
func (r *PredictionRepository) GetLatestByUser(ctx context.Context, userID string) (*model.PredictionTask, error) {
	var task model.PredictionTask
	query := `
		SELECT * FROM prediction_tasks
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &task, query, userID)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetPendingByUser 获取用户待处理（待确认）的推测任务
func (r *PredictionRepository) GetPendingByUser(ctx context.Context, userID string) (*model.PredictionTask, error) {
	var task model.PredictionTask
	query := `
		SELECT * FROM prediction_tasks
		WHERE user_id = ? AND status = 'completed' AND confirmed_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &task, query, userID)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetProcessingByUser 获取用户正在处理的推测任务
func (r *PredictionRepository) GetProcessingByUser(ctx context.Context, userID string) (*model.PredictionTask, error) {
	var task model.PredictionTask
	query := `
		SELECT * FROM prediction_tasks
		WHERE user_id = ? AND status IN ('pending', 'processing')
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &task, query, userID)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByUserAndDate 获取用户指定日期的推测任务
func (r *PredictionRepository) GetByUserAndDate(ctx context.Context, userID string, targetDate time.Time) (*model.PredictionTask, error) {
	var task model.PredictionTask
	query := `
		SELECT * FROM prediction_tasks
		WHERE user_id = ? AND target_date = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &task, query, userID, targetDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateStatus 更新任务状态
func (r *PredictionRepository) UpdateStatus(ctx context.Context, id string, status model.PredictionStatus) error {
	query := `UPDATE prediction_tasks SET status = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// UpdateProgress 更新任务进度
func (r *PredictionRepository) UpdateProgress(ctx context.Context, id string, progress int) error {
	query := `UPDATE prediction_tasks SET progress = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, progress, id)
	return err
}

// UpdateResult 更新推测结果
func (r *PredictionRepository) UpdateResult(ctx context.Context, task *model.PredictionTask) error {
	query := `
		UPDATE prediction_tasks SET
			status = ?,
			predicted_items = ?,
			existing_items = ?,
			merged_preview = ?,
			reasoning_content = ?,
			reasoning_steps = ?,
			user_context = ?,
			progress = ?,
			completed_at = ?,
			updated_at = NOW()
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		task.Status,
		task.PredictedItems,
		task.ExistingItems,
		task.MergedPreview,
		task.ReasoningContent,
		task.ReasoningSteps,
		task.UserContext,
		task.Progress,
		task.CompletedAt,
		task.ID,
	)
	return err
}

// UpdateError 更新错误信息
func (r *PredictionRepository) UpdateError(ctx context.Context, id string, errorMsg string) error {
	query := `
		UPDATE prediction_tasks SET
			status = 'failed',
			error_message = ?,
			updated_at = NOW()
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, errorMsg, id)
	return err
}

// MarkConfirmed 标记任务已确认
func (r *PredictionRepository) MarkConfirmed(ctx context.Context, id string) error {
	now := time.Now()
	query := `UPDATE prediction_tasks SET confirmed_at = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, now, id)
	return err
}

// MarkCancelled 标记任务已取消
func (r *PredictionRepository) MarkCancelled(ctx context.Context, id string) error {
	query := `UPDATE prediction_tasks SET status = 'cancelled', updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ListByUser 获取用户的推测任务列表
func (r *PredictionRepository) ListByUser(ctx context.Context, userID string, limit int) ([]*model.PredictionTask, error) {
	var tasks []*model.PredictionTask
	query := `
		SELECT * FROM prediction_tasks
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	err := r.db.SelectContext(ctx, &tasks, query, userID, limit)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// DeleteOldTasks 删除用户旧的推测任务（保留最近 N 条）
func (r *PredictionRepository) DeleteOldTasks(ctx context.Context, userID string, keepCount int) error {
	query := `
		DELETE FROM prediction_tasks
		WHERE user_id = ? AND id NOT IN (
			SELECT id FROM (
				SELECT id FROM prediction_tasks
				WHERE user_id = ?
				ORDER BY created_at DESC
				LIMIT ?
			) AS recent
		)
	`
	_, err := r.db.ExecContext(ctx, query, userID, userID, keepCount)
	return err
}
