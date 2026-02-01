package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// 职业类型
type OccupationType string

const (
	OccupationStudent       OccupationType = "student"
	OccupationOfficeWorker  OccupationType = "office_worker"
	OccupationFreelancer    OccupationType = "freelancer"
	OccupationShiftWorker   OccupationType = "shift_worker"
	OccupationEntrepreneur  OccupationType = "entrepreneur"
	OccupationHomemaker     OccupationType = "homemaker"
	OccupationRetired       OccupationType = "retired"
	OccupationOther         OccupationType = "other"
)

// 工作时间规律
type WorkSchedule string

const (
	WorkScheduleRegular9to5 WorkSchedule = "regular_9to5"
	WorkScheduleFlexible    WorkSchedule = "flexible"
	WorkScheduleShift       WorkSchedule = "shift"
	WorkScheduleIrregular   WorkSchedule = "irregular"
	WorkScheduleNone        WorkSchedule = "none"
)

// 生活方式类型
type LifestyleType string

const (
	LifestyleEarlyBird LifestyleType = "early_bird"
	LifestyleNightOwl  LifestyleType = "night_owl"
	LifestyleBalanced  LifestyleType = "balanced"
)

// 运动频率
type ExerciseFrequency string

const (
	ExerciseDaily      ExerciseFrequency = "daily"
	ExerciseRegular    ExerciseFrequency = "regular"
	ExerciseOccasional ExerciseFrequency = "occasional"
	ExerciseRarely     ExerciseFrequency = "rarely"
)

// 社交倾向
type SocialPreference string

const (
	SocialVerySocial       SocialPreference = "very_social"
	SocialModeratelySocial SocialPreference = "moderately_social"
	SocialIntrovert        SocialPreference = "introvert"
)

// 工作时间段
type WorkHours struct {
	Start string `json:"start"` // 如 "09:00"
	End   string `json:"end"`   // 如 "18:00"
}

// Scan 实现 sql.Scanner 接口
func (w *WorkHours) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, w)
}

// Value 实现 driver.Valuer 接口
func (w WorkHours) Value() (driver.Value, error) {
	return json.Marshal(w)
}

// 常驻地点
type FrequentLocation struct {
	Type string  `json:"type"` // "home", "work", "gym", etc.
	Name string  `json:"name"` // 如 "望京"
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

// FrequentLocations 类型（用于 JSON 数组）
type FrequentLocations []FrequentLocation

// Scan 实现 sql.Scanner 接口
func (f *FrequentLocations) Scan(value interface{}) error {
	if value == nil {
		*f = []FrequentLocation{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, f)
}

// Value 实现 driver.Valuer 接口
func (f FrequentLocations) Value() (driver.Value, error) {
	if len(f) == 0 {
		return json.Marshal([]FrequentLocation{})
	}
	return json.Marshal(f)
}

// 用户画像
type UserProfile struct {
	UserID             string              `db:"user_id" json:"user_id"`
	OccupationType     OccupationType      `db:"occupation_type" json:"occupation_type"`
	WorkSchedule       WorkSchedule        `db:"work_schedule" json:"work_schedule"`
	TypicalWorkHours   *WorkHours          `db:"typical_work_hours" json:"typical_work_hours,omitempty"`
	LifestyleType      LifestyleType       `db:"lifestyle_type" json:"lifestyle_type"`
	ExerciseFrequency  ExerciseFrequency   `db:"exercise_frequency" json:"exercise_frequency"`
	SocialPreference   SocialPreference    `db:"social_preference" json:"social_preference"`
	FrequentLocations  FrequentLocations   `db:"frequent_locations" json:"frequent_locations"`
	Preferences        map[string]string   `db:"preferences" json:"preferences,omitempty"`
	CreatedAt          time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time           `db:"updated_at" json:"updated_at"`
}

// 创建/更新用户画像请求
type UserProfileRequest struct {
	OccupationType    OccupationType     `json:"occupation_type" binding:"required,oneof=student office_worker freelancer shift_worker entrepreneur homemaker retired other"`
	WorkSchedule      WorkSchedule       `json:"work_schedule" binding:"required,oneof=regular_9to5 flexible shift irregular none"`
	TypicalWorkHours  *WorkHours         `json:"typical_work_hours,omitempty"`
	LifestyleType     LifestyleType      `json:"lifestyle_type" binding:"required,oneof=early_bird night_owl balanced"`
	ExerciseFrequency ExerciseFrequency  `json:"exercise_frequency" binding:"required,oneof=daily regular occasional rarely"`
	SocialPreference  SocialPreference   `json:"social_preference" binding:"required,oneof=very_social moderately_social introvert"`
	FrequentLocations []FrequentLocation `json:"frequent_locations,omitempty"`
	Preferences       map[string]string  `json:"preferences,omitempty"`
}
