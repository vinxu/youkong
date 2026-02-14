package repository

import (
	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

type UserProfileRepository struct {
	db *sqlx.DB
}

func NewUserProfileRepository(db *sqlx.DB) *UserProfileRepository {
	return &UserProfileRepository{db: db}
}

// 获取用户画像
func (r *UserProfileRepository) Get(userID string) (*model.UserProfileData, error) {
	var profile model.UserProfileData
	query := `
		SELECT user_id, gender, mbti, occupation_type, work_schedule, typical_work_hours,
		       lifestyle_type, exercise_frequency, social_preference,
		       frequent_locations, preferences, created_at, updated_at
		FROM user_profiles
		WHERE user_id = ?
	`
	err := r.db.Get(&profile, query, userID)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// 创建或更新用户画像
func (r *UserProfileRepository) Upsert(profile *model.UserProfileData) error {
	query := `
		INSERT INTO user_profiles (
			user_id, gender, mbti, occupation_type, work_schedule, typical_work_hours,
			lifestyle_type, exercise_frequency, social_preference,
			frequent_locations, preferences
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			gender = VALUES(gender),
			mbti = VALUES(mbti),
			occupation_type = VALUES(occupation_type),
			work_schedule = VALUES(work_schedule),
			typical_work_hours = VALUES(typical_work_hours),
			lifestyle_type = VALUES(lifestyle_type),
			exercise_frequency = VALUES(exercise_frequency),
			social_preference = VALUES(social_preference),
			frequent_locations = VALUES(frequent_locations),
			preferences = VALUES(preferences),
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(query,
		profile.UserID,
		profile.Gender,
		profile.MBTI,
		profile.OccupationType,
		profile.WorkSchedule,
		profile.TypicalWorkHours,
		profile.LifestyleType,
		profile.ExerciseFrequency,
		profile.SocialPreference,
		profile.FrequentLocations,
		profile.Preferences,
	)
	return err
}

// 删除用户画像
func (r *UserProfileRepository) Delete(userID string) error {
	query := `DELETE FROM user_profiles WHERE user_id = ?`
	_, err := r.db.Exec(query, userID)
	return err
}

// 检查用户画像是否存在
func (r *UserProfileRepository) Exists(userID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM user_profiles WHERE user_id = ?`
	err := r.db.Get(&count, query, userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
