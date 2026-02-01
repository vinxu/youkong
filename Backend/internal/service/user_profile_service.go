package service

import (
	"database/sql"
	"errors"
	"time"
	"youkong/internal/model"
	"youkong/internal/repository"
)

type UserProfileService struct {
	profileRepo *repository.UserProfileRepository
}

func NewUserProfileService(profileRepo *repository.UserProfileRepository) *UserProfileService {
	return &UserProfileService{
		profileRepo: profileRepo,
	}
}

// 获取用户画像
func (s *UserProfileService) GetProfile(userID string) (*model.UserProfile, error) {
	profile, err := s.profileRepo.Get(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 用户画像不存在
		}
		return nil, err
	}
	return profile, nil
}

// 创建或更新用户画像
func (s *UserProfileService) UpsertProfile(userID string, req *model.UserProfileRequest) (*model.UserProfile, error) {
	// 构建 UserProfile
	profile := &model.UserProfile{
		UserID:            userID,
		OccupationType:    req.OccupationType,
		WorkSchedule:      req.WorkSchedule,
		TypicalWorkHours:  req.TypicalWorkHours,
		LifestyleType:     req.LifestyleType,
		ExerciseFrequency: req.ExerciseFrequency,
		SocialPreference:  req.SocialPreference,
		FrequentLocations: req.FrequentLocations,
		Preferences:       req.Preferences,
		UpdatedAt:         time.Now(),
	}

	// 保存到数据库
	err := s.profileRepo.Upsert(profile)
	if err != nil {
		return nil, err
	}

	// 返回保存后的画像
	return s.GetProfile(userID)
}

// 检查用户画像是否已完成
func (s *UserProfileService) IsProfileComplete(userID string) (bool, error) {
	return s.profileRepo.Exists(userID)
}

// 删除用户画像
func (s *UserProfileService) DeleteProfile(userID string) error {
	return s.profileRepo.Delete(userID)
}
