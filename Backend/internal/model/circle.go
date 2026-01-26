package model

import "time"

type Circle struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Emoji     string    `db:"emoji" json:"emoji"`
	Color     string    `db:"color" json:"color"`
	OwnerID   string    `db:"owner_id" json:"ownerId"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

type CircleMember struct {
	ID       string    `db:"id" json:"id"`
	CircleID string    `db:"circle_id" json:"circleId"`
	UserID   string    `db:"user_id" json:"userId"`
	JoinedAt time.Time `db:"joined_at" json:"joinedAt"`
}

type CircleWithMembers struct {
	Circle
	Members []UserProfile `json:"members"`
}

type CircleDetail struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Emoji       string        `json:"emoji"`
	Color       string        `json:"color"`
	OwnerID     string        `json:"ownerId"`
	MemberCount int           `json:"memberCount"`
	Members     []UserProfile `json:"members,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
}
