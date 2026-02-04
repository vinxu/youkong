package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"youkong/internal/model"
)

// ScheduleRepository 时刻表数据访问层
type ScheduleRepository struct {
	db *sqlx.DB
}

// NewScheduleRepository 创建时刻表 Repository
func NewScheduleRepository(db *sqlx.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

// Create 创建时刻表
func (r *ScheduleRepository) Create(ctx context.Context, schedule *model.StatusSchedule) error {
	// 设置默认可见性
	if schedule.Visibility == "" {
		schedule.Visibility = model.VisibilityAllFriends
	}

	query := `
		INSERT INTO status_schedules (user_id, schedule_date, items, current_index, status, visibility, circle_ids, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
	`
	result, err := r.db.ExecContext(ctx, query,
		schedule.UserID,
		schedule.ScheduleDate,
		schedule.Items,
		schedule.CurrentIndex,
		schedule.Status,
		schedule.Visibility,
		schedule.CircleIDs,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	schedule.ID = id

	// 如果是圈子可见，保存圈子关联
	if schedule.Visibility == model.VisibilityCircles && len(schedule.CircleIDs) > 0 {
		r.saveScheduleCircles(ctx, schedule.ID, schedule.CircleIDs)
	}

	return nil
}

// saveScheduleCircles 保存时刻表-圈子关联
func (r *ScheduleRepository) saveScheduleCircles(ctx context.Context, scheduleID int64, circleIDs []string) error {
	if len(circleIDs) == 0 {
		return nil
	}

	query := `INSERT INTO schedule_circles (schedule_id, circle_id) VALUES (?, ?)`
	for _, circleID := range circleIDs {
		_, err := r.db.ExecContext(ctx, query, scheduleID, circleID)
		if err != nil {
			// 忽略重复插入错误
			continue
		}
	}
	return nil
}

// GetByID 根据 ID 获取时刻表
func (r *ScheduleRepository) GetByID(ctx context.Context, id int64) (*model.StatusSchedule, error) {
	var schedule model.StatusSchedule
	query := `SELECT * FROM status_schedules WHERE id = ?`
	err := r.db.GetContext(ctx, &schedule, query, id)
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

// GetActiveByUserAndDate 获取用户某天的活跃时刻表
func (r *ScheduleRepository) GetActiveByUserAndDate(ctx context.Context, userID string, date time.Time) (*model.StatusSchedule, error) {
	var schedule model.StatusSchedule
	query := `
		SELECT * FROM status_schedules
		WHERE user_id = ? AND schedule_date = ? AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &schedule, query, userID, date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

// GetAllActiveSchedules 获取所有活跃的时刻表（用于定时任务）
func (r *ScheduleRepository) GetAllActiveSchedules(ctx context.Context, date time.Time) ([]*model.StatusSchedule, error) {
	var schedules []*model.StatusSchedule
	query := `
		SELECT * FROM status_schedules
		WHERE status = 'active' AND schedule_date = ?
	`
	err := r.db.SelectContext(ctx, &schedules, query, date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	return schedules, nil
}

// Update 更新时刻表
func (r *ScheduleRepository) Update(ctx context.Context, schedule *model.StatusSchedule) error {
	query := `
		UPDATE status_schedules
		SET items = ?, current_index = ?, status = ?, updated_at = NOW()
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		schedule.Items,
		schedule.CurrentIndex,
		schedule.Status,
		schedule.ID,
	)
	return err
}

// UpdateStatus 更新时刻表状态
func (r *ScheduleRepository) UpdateStatus(ctx context.Context, id int64, status model.ScheduleStatus) error {
	query := `UPDATE status_schedules SET status = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// MarkItemExecuted 标记某个条目为已执行
func (r *ScheduleRepository) MarkItemExecuted(ctx context.Context, id int64, index int) error {
	// 先获取当前时刻表
	schedule, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 标记条目为已执行
	if index < len(schedule.Items) {
		schedule.Items[index].Executed = true
		schedule.CurrentIndex = index + 1
	}

	// 检查是否全部完成
	allExecuted := true
	for _, item := range schedule.Items {
		if !item.Executed {
			allExecuted = false
			break
		}
	}
	if allExecuted {
		schedule.Status = model.ScheduleStatusCompleted
	}

	return r.Update(ctx, schedule)
}

// CancelUserActiveSchedules 取消用户所有活跃的时刻表
func (r *ScheduleRepository) CancelUserActiveSchedules(ctx context.Context, userID string) error {
	query := `
		UPDATE status_schedules
		SET status = 'cancelled', updated_at = NOW()
		WHERE user_id = ? AND status = 'active'
	`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// CancelUserSchedulesByDate 取消用户指定日期的活跃时刻表
func (r *ScheduleRepository) CancelUserSchedulesByDate(ctx context.Context, userID string, date time.Time) error {
	query := `
		UPDATE status_schedules
		SET status = 'cancelled', updated_at = NOW()
		WHERE user_id = ? AND status = 'active' AND DATE(schedule_date) = DATE(?)
	`
	_, err := r.db.ExecContext(ctx, query, userID, date)
	return err
}

// GetRecentByUser 获取用户最近的时刻表
func (r *ScheduleRepository) GetRecentByUser(ctx context.Context, userID string, limit int) ([]*model.StatusSchedule, error) {
	var schedules []*model.StatusSchedule
	query := `
		SELECT * FROM status_schedules
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	err := r.db.SelectContext(ctx, &schedules, query, userID, limit)
	if err != nil {
		return nil, err
	}
	return schedules, nil
}

// GetUserScheduleHistory 获取用户时刻表历史（分页）
// beforeDate: 获取此日期之前的数据（用于分页），为空则从最新开始
// limit: 每页数量
func (r *ScheduleRepository) GetUserScheduleHistory(ctx context.Context, userID string, beforeDate string, limit int) ([]*model.StatusSchedule, error) {
	var schedules []*model.StatusSchedule

	var query string
	var args []interface{}

	if beforeDate == "" {
		// 从最新开始获取
		query = `
			SELECT * FROM status_schedules
			WHERE user_id = ?
			ORDER BY schedule_date DESC, created_at DESC
			LIMIT ?
		`
		args = []interface{}{userID, limit}
	} else {
		// 获取指定日期之前的数据
		query = `
			SELECT * FROM status_schedules
			WHERE user_id = ? AND schedule_date < ?
			ORDER BY schedule_date DESC, created_at DESC
			LIMIT ?
		`
		args = []interface{}{userID, beforeDate, limit}
	}

	err := r.db.SelectContext(ctx, &schedules, query, args...)
	if err != nil {
		return nil, err
	}
	return schedules, nil
}

// GetUserOldestScheduleDate 获取用户最早的时刻表日期
func (r *ScheduleRepository) GetUserOldestScheduleDate(ctx context.Context, userID string) (string, error) {
	var oldestDate string
	query := `
		SELECT DATE_FORMAT(MIN(schedule_date), '%Y-%m-%d') as oldest_date
		FROM status_schedules
		WHERE user_id = ?
	`
	err := r.db.GetContext(ctx, &oldestDate, query, userID)
	if err != nil {
		return "", err
	}
	return oldestDate, nil
}

// CountUserSchedules 统计用户时刻表数量
func (r *ScheduleRepository) CountUserSchedules(ctx context.Context, userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM status_schedules WHERE user_id = ?`
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		return 0, err
	}
	return count, nil
}
