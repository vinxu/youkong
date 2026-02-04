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
func (s *UserProfileService) GetProfile(userID string) (*model.UserProfileData, error) {
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
func (s *UserProfileService) UpsertProfile(userID string, req *model.UserProfileRequest) (*model.UserProfileData, error) {
	// 构建 UserProfile
	profile := &model.UserProfileData{
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

// 简化保存用户画像（只保存 profile_type）
func (s *UserProfileService) SaveSimpleProfile(userID string, profileType string) error {
	// 映射 profile_type 到 occupation_type
	occupationType := model.MapProfileTypeToOccupation(profileType)

	// 使用默认值创建画像
	profile := &model.UserProfileData{
		UserID:            userID,
		OccupationType:    occupationType,
		WorkSchedule:      model.WorkScheduleFlexible,           // 默认弹性
		LifestyleType:     model.LifestyleBalanced,              // 默认均衡
		ExerciseFrequency: model.ExerciseOccasional,             // 默认偶尔
		SocialPreference:  model.SocialModeratelySocial,         // 默认适度社交
		FrequentLocations: model.FrequentLocations{},            // 空数组
		Preferences:       model.Preferences{},                  // 空 map
		UpdatedAt:         time.Now(),
	}

	return s.profileRepo.Upsert(profile)
}

// 获取用户的 ProfileType（简化版）
func (s *UserProfileService) GetSimpleProfileType(userID string) (string, error) {
	profile, err := s.profileRepo.Get(userID)
	if err != nil {
		return "", err
	}
	if profile == nil {
		return "", nil
	}

	// 反向映射 occupation_type 到 profile_type
	switch profile.OccupationType {
	case model.OccupationOfficeWorker:
		return "office_worker", nil
	case model.OccupationStudent:
		return "student", nil
	case model.OccupationFreelancer:
		return "freelancer", nil
	case model.OccupationEntrepreneur:
		return "entrepreneur", nil
	case model.OccupationInvestor:
		return "investor", nil
	case model.OccupationHomemaker:
		return "parent", nil
	case model.OccupationRetired:
		return "retired", nil
	default:
		return string(profile.OccupationType), nil
	}
}
