package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"youkong/internal/model"
)

// BookingRepository 预约数据访问层
type BookingRepository struct {
	db *sqlx.DB
}

// NewBookingRepository 创建预约 Repository
func NewBookingRepository(db *sqlx.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

// Create 创建预约
func (r *BookingRepository) Create(ctx context.Context, booking *model.Booking) error {
	query := `
		INSERT INTO bookings (id, creator_id, title, emoji, schedule_date, start_time, end_time, location, note, status, message_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		booking.ID,
		booking.CreatorID,
		booking.Title,
		booking.Emoji,
		booking.ScheduleDate,
		booking.StartTime,
		booking.EndTime,
		booking.Location,
		booking.Note,
		booking.Status,
		booking.MessageID,
	)
	return err
}

// GetByID 根据 ID 获取预约
func (r *BookingRepository) GetByID(ctx context.Context, id string) (*model.Booking, error) {
	var booking model.Booking
	query := `SELECT * FROM bookings WHERE id = ?`
	err := r.db.GetContext(ctx, &booking, query, id)
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

// UpdateStatus 更新预约状态
func (r *BookingRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE bookings SET status = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// UpdateMessageID 更新预约关联的消息 ID
func (r *BookingRepository) UpdateMessageID(ctx context.Context, id string, messageID string) error {
	query := `UPDATE bookings SET message_id = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, messageID, id)
	return err
}

// GetByUserID 获取用户相关的预约（作为创建者或参与者）
func (r *BookingRepository) GetByUserID(ctx context.Context, userID string, limit int) ([]*model.Booking, error) {
	var bookings []*model.Booking
	query := `
		SELECT DISTINCT b.* FROM bookings b
		LEFT JOIN booking_participants bp ON b.id = bp.booking_id
		WHERE (b.creator_id = ? OR bp.user_id = ?)
		  AND b.status != 'cancelled'
		ORDER BY b.schedule_date DESC, b.start_time DESC
		LIMIT ?
	`
	err := r.db.SelectContext(ctx, &bookings, query, userID, userID, limit)
	if err != nil {
		return nil, err
	}
	return bookings, nil
}

// GetUpcomingByUserID 获取用户即将到来的预约（confirmed 状态，未来日期）
func (r *BookingRepository) GetUpcomingByUserID(ctx context.Context, userID string) ([]*model.Booking, error) {
	var bookings []*model.Booking
	query := `
		SELECT DISTINCT b.* FROM bookings b
		LEFT JOIN booking_participants bp ON b.id = bp.booking_id
		WHERE (b.creator_id = ? OR (bp.user_id = ? AND bp.status = 'accepted'))
		  AND b.status = 'confirmed'
		  AND b.schedule_date >= CURDATE()
		ORDER BY b.schedule_date ASC, b.start_time ASC
	`
	err := r.db.SelectContext(ctx, &bookings, query, userID, userID)
	if err != nil {
		return nil, err
	}
	return bookings, nil
}

// AddParticipant 添加参与者
func (r *BookingRepository) AddParticipant(ctx context.Context, bookingID, userID string) error {
	query := `
		INSERT INTO booking_participants (booking_id, user_id, status, created_at)
		VALUES (?, ?, 'pending', NOW())
		ON DUPLICATE KEY UPDATE status = 'pending'
	`
	_, err := r.db.ExecContext(ctx, query, bookingID, userID)
	return err
}

// UpdateParticipantStatus 更新参与者状态
func (r *BookingRepository) UpdateParticipantStatus(ctx context.Context, bookingID, userID, status string) error {
	query := `
		UPDATE booking_participants
		SET status = ?, responded_at = NOW()
		WHERE booking_id = ? AND user_id = ?
	`
	_, err := r.db.ExecContext(ctx, query, status, bookingID, userID)
	return err
}

// GetParticipants 获取预约的所有参与者
func (r *BookingRepository) GetParticipants(ctx context.Context, bookingID string) ([]*model.BookingParticipant, error) {
	var participants []*model.BookingParticipant
	query := `SELECT * FROM booking_participants WHERE booking_id = ?`
	err := r.db.SelectContext(ctx, &participants, query, bookingID)
	if err != nil {
		return nil, err
	}
	return participants, nil
}

// GetParticipant 获取单个参与者
func (r *BookingRepository) GetParticipant(ctx context.Context, bookingID, userID string) (*model.BookingParticipant, error) {
	var p model.BookingParticipant
	query := `SELECT * FROM booking_participants WHERE booking_id = ? AND user_id = ?`
	err := r.db.GetContext(ctx, &p, query, bookingID, userID)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// IsParticipant 检查用户是否是预约参与方（创建者或参与者）
func (r *BookingRepository) IsParticipant(ctx context.Context, bookingID, userID string) (bool, error) {
	// 先检查是否是创建者
	var count int
	query := `SELECT COUNT(*) FROM bookings WHERE id = ? AND creator_id = ?`
	err := r.db.GetContext(ctx, &count, query, bookingID, userID)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	// 再检查是否是参与者
	query = `SELECT COUNT(*) FROM booking_participants WHERE booking_id = ? AND user_id = ?`
	err = r.db.GetContext(ctx, &count, query, bookingID, userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// MarkNotified 标记参与者已发送提醒
func (r *BookingRepository) MarkNotified(ctx context.Context, bookingID, userID string) error {
	query := `UPDATE booking_participants SET notified = TRUE WHERE booking_id = ? AND user_id = ?`
	_, err := r.db.ExecContext(ctx, query, bookingID, userID)
	return err
}

// GetUnnotifiedUpcoming 获取未通知的即将开始的预约（用于提醒）
func (r *BookingRepository) GetUnnotifiedUpcoming(ctx context.Context, beforeTime time.Time) ([]*model.Booking, error) {
	var bookings []*model.Booking
	query := `
		SELECT b.* FROM bookings b
		WHERE b.status = 'confirmed'
		  AND b.schedule_date = CURDATE()
		  AND b.start_time <= ?
		  AND EXISTS (
			SELECT 1 FROM booking_participants bp
			WHERE bp.booking_id = b.id AND bp.notified = FALSE AND bp.status = 'accepted'
		  )
	`
	err := r.db.SelectContext(ctx, &bookings, query, beforeTime.Format("15:04"))
	if err != nil {
		return nil, err
	}
	return bookings, nil
}

// BookingWithCreator 预约 + 创建者名字（用于被邀请方查看）
type BookingWithCreator struct {
	model.Booking
	CreatorName string `db:"creator_name"`
}

// GetPendingForUser 获取用户的待处理邀请（被邀请方视角）
func (r *BookingRepository) GetPendingForUser(ctx context.Context, userID string) ([]*BookingWithCreator, error) {
	var bookings []*BookingWithCreator
	query := `
		SELECT b.*, u.nickname as creator_name
		FROM bookings b
		JOIN booking_participants bp ON b.id = bp.booking_id
		JOIN users u ON b.creator_id = u.id
		WHERE bp.user_id = ?
		  AND bp.status = 'pending'
		  AND b.status IN ('pending', 'confirmed')
		  AND b.schedule_date >= CURDATE()
		ORDER BY b.schedule_date ASC, b.start_time ASC
	`
	err := r.db.SelectContext(ctx, &bookings, query, userID)
	if err != nil {
		return nil, err
	}
	return bookings, nil
}

// GetBookingsByDateRange 获取日期范围内用户参与的预约
func (r *BookingRepository) GetBookingsByDateRange(ctx context.Context, userID string, startDate, endDate time.Time) ([]*model.Booking, error) {
	var bookings []*model.Booking
	query := `
		SELECT DISTINCT b.* FROM bookings b
		LEFT JOIN booking_participants bp ON b.id = bp.booking_id
		WHERE (b.creator_id = ? OR (bp.user_id = ? AND bp.status = 'accepted'))
		  AND b.status IN ('pending', 'confirmed')
		  AND b.schedule_date BETWEEN ? AND ?
		ORDER BY b.start_time ASC
	`
	err := r.db.SelectContext(ctx, &bookings, query, userID, userID,
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	return bookings, nil
}
