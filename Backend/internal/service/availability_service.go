package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"youkong/internal/model"
	"youkong/internal/repository"
)

type AvailabilityService struct {
	availabilityRepo *repository.AvailabilityRepository
	circleRepo       *repository.CircleRepository
	userRepo         *repository.UserRepository
}

func NewAvailabilityService(
	availabilityRepo *repository.AvailabilityRepository,
	circleRepo *repository.CircleRepository,
	userRepo *repository.UserRepository,
) *AvailabilityService {
	return &AvailabilityService{
		availabilityRepo: availabilityRepo,
		circleRepo:       circleRepo,
		userRepo:         userRepo,
	}
}

type CreateAvailabilityRequest struct {
	StartTime    time.Time             `json:"startTime" binding:"required"`
	EndTime      time.Time             `json:"endTime" binding:"required"`
	LocationType model.LocationType    `json:"locationType" binding:"required"`
	LocationName string                `json:"locationName"`
	Latitude     float64               `json:"latitude"`
	Longitude    float64               `json:"longitude"`
	CircleIDs    []string              `json:"circleIds" binding:"required,min=1"`
}

func (s *AvailabilityService) Create(ctx context.Context, userID string, req *CreateAvailabilityRequest) (*model.AvailabilityResponse, error) {
	// 验证时间
	if req.StartTime.Before(time.Now()) {
		return nil, fmt.Errorf("开始时间不能早于当前时间")
	}
	if req.EndTime.Before(req.StartTime) {
		return nil, fmt.Errorf("结束时间不能早于开始时间")
	}

	// 验证圈子归属
	for _, circleID := range req.CircleIDs {
		isMember, err := s.circleRepo.IsMember(ctx, circleID, userID)
		if err != nil {
			return nil, fmt.Errorf("验证圈子失败: %w", err)
		}
		if !isMember {
			return nil, fmt.Errorf("您不是圈子 %s 的成员", circleID)
		}
	}

	availability := &model.Availability{
		ID:           uuid.New().String(),
		UserID:       userID,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		LocationType: req.LocationType,
		Status:       model.AvailabilityStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if req.LocationName != "" {
		availability.LocationName = sql.NullString{String: req.LocationName, Valid: true}
	}
	if req.Latitude != 0 {
		availability.Latitude = sql.NullFloat64{Float64: req.Latitude, Valid: true}
	}
	if req.Longitude != 0 {
		availability.Longitude = sql.NullFloat64{Float64: req.Longitude, Valid: true}
	}

	if err := s.availabilityRepo.Create(ctx, availability); err != nil {
		return nil, fmt.Errorf("创建有空状态失败: %w", err)
	}

	if err := s.availabilityRepo.AddCircles(ctx, availability.ID, req.CircleIDs); err != nil {
		return nil, fmt.Errorf("关联圈子失败: %w", err)
	}

	user, _ := s.userRepo.GetByID(ctx, userID)
	return availability.ToResponse(user, req.CircleIDs), nil
}

func (s *AvailabilityService) GetMyAvailabilities(ctx context.Context, userID string) ([]*model.AvailabilityResponse, error) {
	availabilities, err := s.availabilityRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取有空状态失败: %w", err)
	}

	user, _ := s.userRepo.GetByID(ctx, userID)
	result := make([]*model.AvailabilityResponse, 0, len(availabilities))
	for _, a := range availabilities {
		circleIDs, _ := s.availabilityRepo.GetCircleIDs(ctx, a.ID)
		result = append(result, a.ToResponse(user, circleIDs))
	}

	return result, nil
}

func (s *AvailabilityService) GetFriendsAvailabilities(ctx context.Context, userID string) ([]*model.AvailabilityResponse, error) {
	// 获取用户所在的所有圈子
	circleIDs, err := s.circleRepo.GetUserCircleIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户圈子失败: %w", err)
	}

	if len(circleIDs) == 0 {
		return []*model.AvailabilityResponse{}, nil
	}

	// 获取这些圈子可见的有空状态
	availabilities, err := s.availabilityRepo.GetVisibleAvailabilities(ctx, circleIDs)
	if err != nil {
		return nil, fmt.Errorf("获取朋友有空状态失败: %w", err)
	}

	// 收集所有用户ID
	userIDs := make([]string, 0, len(availabilities))
	for _, a := range availabilities {
		if a.UserID != userID { // 排除自己
			userIDs = append(userIDs, a.UserID)
		}
	}

	// 批量获取用户信息
	users, err := s.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	userMap := make(map[string]*model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]*model.AvailabilityResponse, 0)
	for _, a := range availabilities {
		if a.UserID == userID { // 排除自己
			continue
		}
		user := userMap[a.UserID]
		if user == nil {
			continue
		}
		circleIDs, _ := s.availabilityRepo.GetCircleIDs(ctx, a.ID)
		result = append(result, a.ToResponse(user, circleIDs))
	}

	return result, nil
}

func (s *AvailabilityService) Cancel(ctx context.Context, id, userID string) error {
	availability, err := s.availabilityRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("获取有空状态失败: %w", err)
	}
	if availability == nil {
		return fmt.Errorf("有空状态不存在")
	}
	if availability.UserID != userID {
		return fmt.Errorf("无权限取消此有空状态")
	}
	if availability.Status != model.AvailabilityStatusActive {
		return fmt.Errorf("有空状态已结束")
	}

	if err := s.availabilityRepo.UpdateStatus(ctx, id, model.AvailabilityStatusCancelled); err != nil {
		return fmt.Errorf("取消有空状态失败: %w", err)
	}

	return nil
}

func (s *AvailabilityService) ExpireOldAvailabilities(ctx context.Context) (int64, error) {
	return s.availabilityRepo.ExpireOldAvailabilities(ctx)
}
