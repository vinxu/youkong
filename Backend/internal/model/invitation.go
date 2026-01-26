package model

import (
	"database/sql"
	"time"
)

type InvitationStatus string

const (
	InvitationStatusActive   InvitationStatus = "ACTIVE"
	InvitationStatusDisabled InvitationStatus = "DISABLED"
	InvitationStatusExpired  InvitationStatus = "EXPIRED"
)

type Invitation struct {
	ID        string           `db:"id" json:"id"`
	Code      string           `db:"code" json:"code"`
	InviterID string           `db:"inviter_id" json:"inviterId"`
	CircleID  sql.NullString   `db:"circle_id" json:"-"`
	MaxUses   int              `db:"max_uses" json:"maxUses"`
	UseCount  int              `db:"use_count" json:"useCount"`
	ExpiresAt sql.NullTime     `db:"expires_at" json:"-"`
	Status    InvitationStatus `db:"status" json:"status"`
	CreatedAt time.Time        `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time        `db:"updated_at" json:"updatedAt"`
}

func (i *Invitation) GetCircleID() string {
	if i.CircleID.Valid {
		return i.CircleID.String
	}
	return ""
}

func (i *Invitation) GetExpiresAt() *time.Time {
	if i.ExpiresAt.Valid {
		return &i.ExpiresAt.Time
	}
	return nil
}

func (i *Invitation) IsValid() bool {
	if i.Status != InvitationStatusActive {
		return false
	}
	if i.ExpiresAt.Valid && time.Now().After(i.ExpiresAt.Time) {
		return false
	}
	if i.UseCount >= i.MaxUses {
		return false
	}
	return true
}

type InvitationRecord struct {
	ID           string         `db:"id" json:"id"`
	InvitationID string         `db:"invitation_id" json:"invitationId"`
	InviterID    string         `db:"inviter_id" json:"inviterId"`
	InviteeID    string         `db:"invitee_id" json:"inviteeId"`
	CircleID     sql.NullString `db:"circle_id" json:"-"`
	CreatedAt    time.Time      `db:"created_at" json:"createdAt"`
}

func (r *InvitationRecord) GetCircleID() string {
	if r.CircleID.Valid {
		return r.CircleID.String
	}
	return ""
}

type InvitationDetail struct {
	ID         string       `json:"id"`
	Code       string       `json:"code"`
	InviteURL  string       `json:"inviteUrl"`
	QRCodeURL  string       `json:"qrcodeUrl,omitempty"`
	Inviter    *UserProfile `json:"inviter,omitempty"`
	Circle     *CircleInfo  `json:"circle,omitempty"`
	MaxUses    int          `json:"maxUses"`
	UseCount   int          `json:"useCount"`
	ExpiresAt  *time.Time   `json:"expiresAt,omitempty"`
	Status     string       `json:"status"`
	IsValid    bool         `json:"isValid"`
	CreatedAt  time.Time    `json:"createdAt"`
}

type CircleInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Emoji       string `json:"emoji"`
	Color       string `json:"color,omitempty"`
	MemberCount int    `json:"memberCount,omitempty"`
}

type InvitationPublicInfo struct {
	Inviter *UserProfile `json:"inviter"`
	Circle  *CircleInfo  `json:"circle,omitempty"`
	IsValid bool         `json:"isValid"`
}
