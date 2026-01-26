package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"youkong/internal/model"
	"youkong/internal/repository"
)

// ContactService 通讯录服务
type ContactService struct {
	userRepo       *repository.UserRepository
	friendshipRepo *repository.FriendshipRepository
}

// NewContactService 创建通讯录服务
func NewContactService(userRepo *repository.UserRepository, friendshipRepo *repository.FriendshipRepository) *ContactService {
	return &ContactService{
		userRepo:       userRepo,
		friendshipRepo: friendshipRepo,
	}
}

// MatchContacts 匹配通讯录中已注册的用户
func (s *ContactService) MatchContacts(ctx context.Context, currentUserID string, phoneHashes []string) (*model.ContactMatchResponse, error) {
	// 1. 根据手机号Hash批量查询用户
	users, err := s.userRepo.GetByPhoneHashes(ctx, phoneHashes)
	if err != nil {
		return nil, err
	}

	// 2. 获取当前用户的好友列表
	friendships, err := s.friendshipRepo.GetFriendsByUserID(ctx, currentUserID)
	if err != nil {
		return nil, err
	}

	// 构建好友ID集合
	friendIDSet := make(map[string]bool)
	for _, f := range friendships {
		friendIDSet[f.FriendID] = true
	}

	// 3. 构建匹配结果
	matches := make([]model.ContactMatchResult, 0, len(users))
	for _, user := range users {
		// 排除自己
		if user.ID == currentUserID {
			continue
		}

		matches = append(matches, model.ContactMatchResult{
			PhoneHash: user.PhoneHash,
			User:      user.ToProfile(),
			IsFriend:  friendIDSet[user.ID],
		})
	}

	return &model.ContactMatchResponse{
		Matches:    matches,
		TotalFound: len(matches),
	}, nil
}

// BatchAddFriends 批量添加好友
func (s *ContactService) BatchAddFriends(ctx context.Context, currentUserID string, userIDs []string) (*model.BatchAddFriendsResponse, error) {
	added := 0
	skipped := 0
	addedIDs := make([]string, 0)

	for _, userID := range userIDs {
		// 排除自己
		if userID == currentUserID {
			continue
		}

		// 检查是否已是好友
		isFriend, err := s.friendshipRepo.AreFriends(ctx, currentUserID, userID)
		if err != nil {
			continue
		}

		if isFriend {
			skipped++
			continue
		}

		now := time.Now()

		// 添加双向好友关系
		err = s.friendshipRepo.Create(ctx, &model.Friendship{
			ID:        uuid.New().String(),
			UserID:    currentUserID,
			FriendID:  userID,
			Source:    model.FriendshipSourceContacts,
			CreatedAt: now,
		})
		if err != nil {
			continue
		}

		err = s.friendshipRepo.Create(ctx, &model.Friendship{
			ID:        uuid.New().String(),
			UserID:    userID,
			FriendID:  currentUserID,
			Source:    model.FriendshipSourceContacts,
			CreatedAt: now,
		})
		if err != nil {
			continue
		}

		added++
		addedIDs = append(addedIDs, userID)
	}

	return &model.BatchAddFriendsResponse{
		Added:   added,
		Skipped: skipped,
		UserIDs: addedIDs,
	}, nil
}
