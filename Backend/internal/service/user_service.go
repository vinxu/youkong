package service

import (
	"context"
	"fmt"

	"youkong/internal/model"
	"youkong/internal/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) GetByID(ctx context.Context, id string) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}
	return user, nil
}

func (s *UserService) Update(ctx context.Context, id string, nickname, avatar string) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}

	if nickname != "" {
		user.Nickname = nickname
	}
	if avatar != "" {
		user.Avatar = avatar
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	return user, nil
}

func (s *UserService) Search(ctx context.Context, keyword string, limit int) ([]*model.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	users, err := s.userRepo.SearchByNickname(ctx, keyword, limit)
	if err != nil {
		return nil, fmt.Errorf("搜索用户失败: %w", err)
	}

	return users, nil
}

func (s *UserService) GetByIDs(ctx context.Context, ids []string) ([]*model.User, error) {
	return s.userRepo.GetByIDs(ctx, ids)
}
