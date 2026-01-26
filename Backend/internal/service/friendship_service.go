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

type FriendshipService struct {
	friendshipRepo *repository.FriendshipRepository
	userRepo       *repository.UserRepository
	invitationRepo *repository.InvitationRepository
	circleRepo     *repository.CircleRepository
}

func NewFriendshipService(
	friendshipRepo *repository.FriendshipRepository,
	userRepo *repository.UserRepository,
	invitationRepo *repository.InvitationRepository,
	circleRepo *repository.CircleRepository,
) *FriendshipService {
	return &FriendshipService{
		friendshipRepo: friendshipRepo,
		userRepo:       userRepo,
		invitationRepo: invitationRepo,
		circleRepo:     circleRepo,
	}
}

func (s *FriendshipService) GetFriends(ctx context.Context, userID string) ([]*model.FriendInfo, error) {
	friendships, err := s.friendshipRepo.GetFriendsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取好友列表失败: %w", err)
	}

	friendIDs := make([]string, 0, len(friendships))
	for _, f := range friendships {
		friendIDs = append(friendIDs, f.FriendID)
	}

	users, err := s.userRepo.GetByIDs(ctx, friendIDs)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	userMap := make(map[string]*model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]*model.FriendInfo, 0, len(friendships))
	for _, f := range friendships {
		if user, ok := userMap[f.FriendID]; ok {
			result = append(result, &model.FriendInfo{
				User:      user.ToProfile(),
				Source:    string(f.Source),
				CreatedAt: f.CreatedAt,
			})
		}
	}

	return result, nil
}

func (s *FriendshipService) GetFriendIDs(ctx context.Context, userID string) ([]string, error) {
	return s.friendshipRepo.GetFriendIDs(ctx, userID)
}

func (s *FriendshipService) AreFriends(ctx context.Context, userID, friendID string) (bool, error) {
	return s.friendshipRepo.AreFriends(ctx, userID, friendID)
}

func (s *FriendshipService) AddFriend(ctx context.Context, userID, friendID string, source model.FriendshipSource) error {
	if userID == friendID {
		return fmt.Errorf("不能添加自己为好友")
	}

	// 检查是否已经是好友
	areFriends, err := s.friendshipRepo.AreFriends(ctx, userID, friendID)
	if err != nil {
		return fmt.Errorf("检查好友状态失败: %w", err)
	}
	if areFriends {
		return fmt.Errorf("已经是好友")
	}

	// 检查好友是否存在
	friend, err := s.userRepo.GetByID(ctx, friendID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if friend == nil {
		return fmt.Errorf("用户不存在")
	}

	now := time.Now()

	// 创建双向好友关系
	friendship1 := &model.Friendship{
		ID:        uuid.New().String(),
		UserID:    userID,
		FriendID:  friendID,
		Source:    source,
		CreatedAt: now,
	}
	friendship2 := &model.Friendship{
		ID:        uuid.New().String(),
		UserID:    friendID,
		FriendID:  userID,
		Source:    source,
		CreatedAt: now,
	}

	if err := s.friendshipRepo.Create(ctx, friendship1); err != nil {
		return fmt.Errorf("创建好友关系失败: %w", err)
	}
	if err := s.friendshipRepo.Create(ctx, friendship2); err != nil {
		// 回滚第一条
		_ = s.friendshipRepo.Delete(ctx, userID, friendID)
		return fmt.Errorf("创建好友关系失败: %w", err)
	}

	return nil
}

func (s *FriendshipService) RemoveFriend(ctx context.Context, userID, friendID string) error {
	areFriends, err := s.friendshipRepo.AreFriends(ctx, userID, friendID)
	if err != nil {
		return fmt.Errorf("检查好友状态失败: %w", err)
	}
	if !areFriends {
		return fmt.Errorf("不是好友关系")
	}

	return s.friendshipRepo.DeleteBidirectional(ctx, userID, friendID)
}

func (s *FriendshipService) GetInvitedByMe(ctx context.Context, userID string) ([]*model.FriendWithInvitation, error) {
	records, err := s.invitationRepo.GetRecordsByInviterID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取邀请记录失败: %w", err)
	}

	inviteeIDs := make([]string, 0, len(records))
	for _, r := range records {
		inviteeIDs = append(inviteeIDs, r.InviteeID)
	}

	users, err := s.userRepo.GetByIDs(ctx, inviteeIDs)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	userMap := make(map[string]*model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]*model.FriendWithInvitation, 0, len(records))
	for _, r := range records {
		if user, ok := userMap[r.InviteeID]; ok {
			item := &model.FriendWithInvitation{
				User:      user.ToProfile(),
				Source:    string(model.FriendshipSourceInvitation),
				CreatedAt: r.CreatedAt,
			}

			// 获取圈子名称
			if r.CircleID.Valid {
				circle, _ := s.circleRepo.GetByID(ctx, r.CircleID.String)
				if circle != nil {
					item.CircleName = circle.Name
				}
			}

			result = append(result, item)
		}
	}

	return result, nil
}

func (s *FriendshipService) GetInvitedMe(ctx context.Context, userID string) ([]*model.FriendWithInvitation, error) {
	records, err := s.invitationRepo.GetRecordsByInviteeID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取邀请记录失败: %w", err)
	}

	inviterIDs := make([]string, 0, len(records))
	for _, r := range records {
		inviterIDs = append(inviterIDs, r.InviterID)
	}

	users, err := s.userRepo.GetByIDs(ctx, inviterIDs)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	userMap := make(map[string]*model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]*model.FriendWithInvitation, 0, len(records))
	for _, r := range records {
		if user, ok := userMap[r.InviterID]; ok {
			item := &model.FriendWithInvitation{
				User:      user.ToProfile(),
				Source:    string(model.FriendshipSourceInvitation),
				CreatedAt: r.CreatedAt,
			}

			if r.CircleID.Valid {
				circle, _ := s.circleRepo.GetByID(ctx, r.CircleID.String)
				if circle != nil {
					item.CircleName = circle.Name
				}
			}

			result = append(result, item)
		}
	}

	return result, nil
}

func (s *FriendshipService) AddFriendWithInvitation(ctx context.Context, userID, friendID, invitationID string) error {
	if userID == friendID {
		return fmt.Errorf("不能添加自己为好友")
	}

	areFriends, _ := s.friendshipRepo.AreFriends(ctx, userID, friendID)
	if areFriends {
		return nil // 已经是好友，不报错
	}

	now := time.Now()
	invitationIDNull := sql.NullString{String: invitationID, Valid: invitationID != ""}

	friendship1 := &model.Friendship{
		ID:           uuid.New().String(),
		UserID:       userID,
		FriendID:     friendID,
		Source:       model.FriendshipSourceInvitation,
		InvitationID: invitationIDNull,
		CreatedAt:    now,
	}
	friendship2 := &model.Friendship{
		ID:           uuid.New().String(),
		UserID:       friendID,
		FriendID:     userID,
		Source:       model.FriendshipSourceInvitation,
		InvitationID: invitationIDNull,
		CreatedAt:    now,
	}

	_ = s.friendshipRepo.Create(ctx, friendship1)
	_ = s.friendshipRepo.Create(ctx, friendship2)

	return nil
}
