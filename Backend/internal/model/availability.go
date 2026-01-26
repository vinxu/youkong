package model

import (
	"database/sql"
	"time"
)

type LocationType string

const (
	LocationTypePreset   LocationType = "PRESET"
	LocationTypeFlexible LocationType = "FLEXIBLE"
	LocationTypeCustom   LocationType = "CUSTOM"
)

type AvailabilityStatus string

const (
	AvailabilityStatusActive    AvailabilityStatus = "ACTIVE"
	AvailabilityStatusExpired   AvailabilityStatus = "EXPIRED"
	AvailabilityStatusCancelled AvailabilityStatus = "CANCELLED"
	AvailabilityStatusFulfilled AvailabilityStatus = "FULFILLED"
)

type Availability struct {
	ID           string             `db:"id" json:"id"`
	UserID       string             `db:"user_id" json:"userId"`
	StartTime    time.Time          `db:"start_time" json:"startTime"`
	EndTime      time.Time          `db:"end_time" json:"endTime"`
	LocationType LocationType       `db:"location_type" json:"locationType"`
	LocationName sql.NullString     `db:"location_name" json:"locationName,omitempty"`
	Latitude     sql.NullFloat64    `db:"latitude" json:"latitude,omitempty"`
	Longitude    sql.NullFloat64    `db:"longitude" json:"longitude,omitempty"`
	Status       AvailabilityStatus `db:"status" json:"status"`
	CreatedAt    time.Time          `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time          `db:"updated_at" json:"updatedAt"`
}

type AvailabilityCircle struct {
	AvailabilityID string `db:"availability_id"`
	CircleID       string `db:"circle_id"`
}

type Location struct {
	Type      LocationType `json:"type"`
	Name      string       `json:"name,omitempty"`
	Latitude  float64      `json:"latitude,omitempty"`
	Longitude float64      `json:"longitude,omitempty"`
}

type AvailabilityResponse struct {
	ID        string             `json:"id"`
	User      UserProfile        `json:"user"`
	StartTime time.Time          `json:"startTime"`
	EndTime   time.Time          `json:"endTime"`
	Location  Location           `json:"location"`
	Status    AvailabilityStatus `json:"status"`
	CircleIDs []string           `json:"circleIds,omitempty"`
	CreatedAt time.Time          `json:"createdAt"`
}

func (a *Availability) ToResponse(user *User, circleIDs []string) *AvailabilityResponse {
	loc := Location{
		Type: a.LocationType,
	}
	if a.LocationName.Valid {
		loc.Name = a.LocationName.String
	}
	if a.Latitude.Valid {
		loc.Latitude = a.Latitude.Float64
	}
	if a.Longitude.Valid {
		loc.Longitude = a.Longitude.Float64
	}

	return &AvailabilityResponse{
		ID:        a.ID,
		User:      *user.ToProfile(),
		StartTime: a.StartTime,
		EndTime:   a.EndTime,
		Location:  loc,
		Status:    a.Status,
		CircleIDs: circleIDs,
		CreatedAt: a.CreatedAt,
	}
}
