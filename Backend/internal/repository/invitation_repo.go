package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

type InvitationRepository struct {
	db *sqlx.DB
}

func NewInvitationRepository(db *sqlx.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

func (r *InvitationRepository) Create(ctx context.Context, invitation *model.Invitation) error {
	query := `INSERT INTO invitations (id, code, inviter_id, circle_id, max_uses, use_count, expires_at, status, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		invitation.ID, invitation.Code, invitation.InviterID, invitation.CircleID,
		invitation.MaxUses, invitation.UseCount, invitation.ExpiresAt, invitation.Status,
		invitation.CreatedAt, invitation.UpdatedAt)
	return err
}

func (r *InvitationRepository) GetByID(ctx context.Context, id string) (*model.Invitation, error) {
	var invitation model.Invitation
	query := `SELECT * FROM invitations WHERE id = ?`
	err := r.db.GetContext(ctx, &invitation, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &invitation, nil
}

func (r *InvitationRepository) GetByCode(ctx context.Context, code string) (*model.Invitation, error) {
	var invitation model.Invitation
	query := `SELECT * FROM invitations WHERE code = ?`
	err := r.db.GetContext(ctx, &invitation, query, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &invitation, nil
}

func (r *InvitationRepository) GetByInviterID(ctx context.Context, inviterID string) ([]*model.Invitation, error) {
	var invitations []*model.Invitation
	query := `SELECT * FROM invitations WHERE inviter_id = ? ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &invitations, query, inviterID)
	return invitations, err
}

func (r *InvitationRepository) Update(ctx context.Context, invitation *model.Invitation) error {
	query := `UPDATE invitations SET max_uses = ?, use_count = ?, expires_at = ?, status = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query,
		invitation.MaxUses, invitation.UseCount, invitation.ExpiresAt, invitation.Status,
		time.Now(), invitation.ID)
	return err
}

func (r *InvitationRepository) IncrementUseCount(ctx context.Context, id string) error {
	query := `UPDATE invitations SET use_count = use_count + 1, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *InvitationRepository) UpdateStatus(ctx context.Context, id string, status model.InvitationStatus) error {
	query := `UPDATE invitations SET status = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *InvitationRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM invitations WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *InvitationRepository) CountByInviterToday(ctx context.Context, inviterID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM invitations WHERE inviter_id = ? AND DATE(created_at) = CURDATE()`
	err := r.db.GetContext(ctx, &count, query, inviterID)
	return count, err
}

func (r *InvitationRepository) CreateRecord(ctx context.Context, record *model.InvitationRecord) error {
	query := `INSERT INTO invitation_records (id, invitation_id, inviter_id, invitee_id, circle_id, created_at)
              VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		record.ID, record.InvitationID, record.InviterID, record.InviteeID,
		record.CircleID, record.CreatedAt)
	return err
}

func (r *InvitationRepository) GetRecordByInvitationAndInvitee(ctx context.Context, invitationID, inviteeID string) (*model.InvitationRecord, error) {
	var record model.InvitationRecord
	query := `SELECT * FROM invitation_records WHERE invitation_id = ? AND invitee_id = ?`
	err := r.db.GetContext(ctx, &record, query, invitationID, inviteeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func (r *InvitationRepository) GetRecordsByInviterID(ctx context.Context, inviterID string) ([]*model.InvitationRecord, error) {
	var records []*model.InvitationRecord
	query := `SELECT * FROM invitation_records WHERE inviter_id = ? ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &records, query, inviterID)
	return records, err
}

func (r *InvitationRepository) GetRecordsByInviteeID(ctx context.Context, inviteeID string) ([]*model.InvitationRecord, error) {
	var records []*model.InvitationRecord
	query := `SELECT * FROM invitation_records WHERE invitee_id = ? ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &records, query, inviteeID)
	return records, err
}
