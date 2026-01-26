package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"youkong/internal/model"
	"youkong/internal/repository"
)

type CircleService struct {
	circleRepo *repository.CircleRepository
	userRepo   *repository.UserRepository
}

func NewCircleService(circleRepo *repository.CircleRepository, userRepo *repository.UserRepository) *CircleService {
	return &CircleService{
		circleRepo: circleRepo,
		userRepo:   userRepo,
	}
}

type CreateCircleRequest struct {
	Name  string `json:"name" binding:"required,max=20"`
	Emoji string `json:"emoji" binding:"required"`
	Color string `json:"color" binding:"required"`
}

func (s *CircleService) Create(ctx context.Context, ownerID string, req *CreateCircleRequest) (*model.Circle, error) {
	circle := &model.Circle{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Emoji:     req.Emoji,
		Color:     req.Color,
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.circleRepo.Create(ctx, circle); err != nil {
		return nil, fmt.Errorf("创建圈子失败: %w", err)
	}

	// 将创建者加入圈子
	member := &model.CircleMember{
		ID:       uuid.New().String(),
		CircleID: circle.ID,
		UserID:   ownerID,
		JoinedAt: time.Now(),
	}
	if err := s.circleRepo.AddMember(ctx, member); err != nil {
		return nil, fmt.Errorf("添加创建者到圈子失败: %w", err)
	}

	return circle, nil
}

func (s *CircleService) GetByID(ctx context.Context, id string) (*model.CircleDetail, error) {
	circle, err := s.circleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取圈子失败: %w", err)
	}
	if circle == nil {
		return nil, nil
	}

	memberIDs, err := s.circleRepo.GetMemberUserIDs(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取圈子成员失败: %w", err)
	}

	users, err := s.userRepo.GetByIDs(ctx, memberIDs)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	members := make([]model.UserProfile, 0, len(users))
	for _, u := range users {
		members = append(members, *u.ToProfile())
	}

	return &model.CircleDetail{
		ID:          circle.ID,
		Name:        circle.Name,
		Emoji:       circle.Emoji,
		Color:       circle.Color,
		OwnerID:     circle.OwnerID,
		MemberCount: len(members),
		Members:     members,
		CreatedAt:   circle.CreatedAt,
	}, nil
}

func (s *CircleService) GetMyCircles(ctx context.Context, userID string) ([]*model.CircleDetail, error) {
	// 获取用户拥有的圈子
	ownedCircles, err := s.circleRepo.GetByOwnerID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取拥有的圈子失败: %w", err)
	}

	// 获取用户加入的圈子
	joinedCircles, err := s.circleRepo.GetCirclesByMemberID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取加入的圈子失败: %w", err)
	}

	// 合并并去重
	circleMap := make(map[string]*model.Circle)
	for _, c := range ownedCircles {
		circleMap[c.ID] = c
	}
	for _, c := range joinedCircles {
		circleMap[c.ID] = c
	}

	result := make([]*model.CircleDetail, 0, len(circleMap))
	for _, circle := range circleMap {
		memberCount, _ := s.circleRepo.GetMemberCount(ctx, circle.ID)
		result = append(result, &model.CircleDetail{
			ID:          circle.ID,
			Name:        circle.Name,
			Emoji:       circle.Emoji,
			Color:       circle.Color,
			OwnerID:     circle.OwnerID,
			MemberCount: memberCount,
			CreatedAt:   circle.CreatedAt,
		})
	}

	return result, nil
}

func (s *CircleService) Update(ctx context.Context, id, ownerID string, req *CreateCircleRequest) (*model.Circle, error) {
	circle, err := s.circleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取圈子失败: %w", err)
	}
	if circle == nil {
		return nil, fmt.Errorf("圈子不存在")
	}
	if circle.OwnerID != ownerID {
		return nil, fmt.Errorf("无权限修改此圈子")
	}

	circle.Name = req.Name
	circle.Emoji = req.Emoji
	circle.Color = req.Color

	if err := s.circleRepo.Update(ctx, circle); err != nil {
		return nil, fmt.Errorf("更新圈子失败: %w", err)
	}

	return circle, nil
}

func (s *CircleService) Delete(ctx context.Context, id, ownerID string) error {
	circle, err := s.circleRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("获取圈子失败: %w", err)
	}
	if circle == nil {
		return fmt.Errorf("圈子不存在")
	}
	if circle.OwnerID != ownerID {
		return fmt.Errorf("无权限删除此圈子")
	}

	if err := s.circleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除圈子失败: %w", err)
	}

	return nil
}

func (s *CircleService) AddMember(ctx context.Context, circleID, ownerID, memberUserID string) error {
	circle, err := s.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return fmt.Errorf("获取圈子失败: %w", err)
	}
	if circle == nil {
		return fmt.Errorf("圈子不存在")
	}
	if circle.OwnerID != ownerID {
		return fmt.Errorf("无权限添加成员")
	}

	// 检查用户是否存在
	user, err := s.userRepo.GetByID(ctx, memberUserID)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}
	if user == nil {
		return fmt.Errorf("用户不存在")
	}

	// 检查是否已经是成员
	isMember, err := s.circleRepo.IsMember(ctx, circleID, memberUserID)
	if err != nil {
		return fmt.Errorf("检查成员状态失败: %w", err)
	}
	if isMember {
		return fmt.Errorf("用户已经是圈子成员")
	}

	member := &model.CircleMember{
		ID:       uuid.New().String(),
		CircleID: circleID,
		UserID:   memberUserID,
		JoinedAt: time.Now(),
	}

	if err := s.circleRepo.AddMember(ctx, member); err != nil {
		return fmt.Errorf("添加成员失败: %w", err)
	}

	return nil
}

func (s *CircleService) RemoveMember(ctx context.Context, circleID, ownerID, memberUserID string) error {
	circle, err := s.circleRepo.GetByID(ctx, circleID)
	if err != nil {
		return fmt.Errorf("获取圈子失败: %w", err)
	}
	if circle == nil {
		return fmt.Errorf("圈子不存在")
	}
	if circle.OwnerID != ownerID {
		return fmt.Errorf("无权限移除成员")
	}

	// 不能移除自己（圈主）
	if memberUserID == ownerID {
		return fmt.Errorf("不能移除圈主")
	}

	if err := s.circleRepo.RemoveMember(ctx, circleID, memberUserID); err != nil {
		return fmt.Errorf("移除成员失败: %w", err)
	}

	return nil
}

func (s *CircleService) GetUserCircleIDs(ctx context.Context, userID string) ([]string, error) {
	return s.circleRepo.GetUserCircleIDs(ctx, userID)
}
