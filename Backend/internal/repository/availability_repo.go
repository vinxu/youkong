package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

type AvailabilityRepository struct {
	db *sqlx.DB
}

func NewAvailabilityRepository(db *sqlx.DB) *AvailabilityRepository {
	return &AvailabilityRepository{db: db}
}

func (r *AvailabilityRepository) Create(ctx context.Context, availability *model.Availability) error {
	query := `INSERT INTO availabilities
              (id, user_id, start_time, end_time, location_type, location_name, latitude, longitude, status, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		availability.ID, availability.UserID, availability.StartTime, availability.EndTime,
		availability.LocationType, availability.LocationName, availability.Latitude, availability.Longitude,
		availability.Status, availability.CreatedAt, availability.UpdatedAt)
	return err
}

func (r *AvailabilityRepository) GetByID(ctx context.Context, id string) (*model.Availability, error) {
	var availability model.Availability
	query := `SELECT * FROM availabilities WHERE id = ?`
	err := r.db.GetContext(ctx, &availability, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &availability, nil
}

func (r *AvailabilityRepository) GetByUserID(ctx context.Context, userID string, status model.AvailabilityStatus) ([]*model.Availability, error) {
	var availabilities []*model.Availability
	query := `SELECT * FROM availabilities WHERE user_id = ? AND status = ? ORDER BY start_time ASC`
	err := r.db.SelectContext(ctx, &availabilities, query, userID, status)
	return availabilities, err
}

func (r *AvailabilityRepository) GetActiveByUserID(ctx context.Context, userID string) ([]*model.Availability, error) {
	return r.GetByUserID(ctx, userID, model.AvailabilityStatusActive)
}

func (r *AvailabilityRepository) UpdateStatus(ctx context.Context, id string, status model.AvailabilityStatus) error {
	query := `UPDATE availabilities SET status = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	return err
}

func (r *AvailabilityRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM availabilities WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *AvailabilityRepository) AddCircles(ctx context.Context, availabilityID string, circleIDs []string) error {
	if len(circleIDs) == 0 {
		return nil
	}
	query := `INSERT INTO availability_circles (availability_id, circle_id) VALUES (?, ?)`
	for _, circleID := range circleIDs {
		if _, err := r.db.ExecContext(ctx, query, availabilityID, circleID); err != nil {
			return err
		}
	}
	return nil
}

func (r *AvailabilityRepository) GetCircleIDs(ctx context.Context, availabilityID string) ([]string, error) {
	var circleIDs []string
	query := `SELECT circle_id FROM availability_circles WHERE availability_id = ?`
	err := r.db.SelectContext(ctx, &circleIDs, query, availabilityID)
	return circleIDs, err
}

func (r *AvailabilityRepository) GetVisibleAvailabilities(ctx context.Context, userCircleIDs []string) ([]*model.Availability, error) {
	if len(userCircleIDs) == 0 {
		return []*model.Availability{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT DISTINCT a.* FROM availabilities a
		INNER JOIN availability_circles ac ON a.id = ac.availability_id
		WHERE ac.circle_id IN (?)
		AND a.status = ?
		AND a.end_time > ?
		ORDER BY a.start_time ASC
	`, userCircleIDs, model.AvailabilityStatusActive, time.Now())
	if err != nil {
		return nil, err
	}

	query = r.db.Rebind(query)
	var availabilities []*model.Availability
	err = r.db.SelectContext(ctx, &availabilities, query, args...)
	return availabilities, err
}

func (r *AvailabilityRepository) ExpireOldAvailabilities(ctx context.Context) (int64, error) {
	query := `UPDATE availabilities SET status = ?, updated_at = ? WHERE status = ? AND end_time < ?`
	result, err := r.db.ExecContext(ctx, query, model.AvailabilityStatusExpired, time.Now(), model.AvailabilityStatusActive, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
