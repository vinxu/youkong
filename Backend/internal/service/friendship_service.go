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
	friendshipRepo     *repository.FriendshipRepository
	userRepo           *repository.UserRepository
	invitationRepo     *repository.InvitationRepository
	circleRepo         *repository.CircleRepository
	friendRequestRepo  *repository.FriendRequestRepository
}

func NewFriendshipService(
	friendshipRepo *repository.FriendshipRepository,
	userRepo *repository.UserRepository,
	invitationRepo *repository.InvitationRepository,
	circleRepo *repository.CircleRepository,
	friendRequestRepo *repository.FriendRequestRepository,
) *FriendshipService {
	return &FriendshipService{
		friendshipRepo:    friendshipRepo,
		userRepo:          userRepo,
		invitationRepo:    invitationRepo,
		circleRepo:        circleRepo,
		friendRequestRepo: friendRequestRepo,
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

// AddFriendByPhone 通过手机号添加好友
func (s *FriendshipService) AddFriendByPhone(ctx context.Context, userID, phone string) (*model.AddFriendByPhoneResponse, error) {
	// 1. 根据手机号查找用户
	friend, err := s.userRepo.GetByPhone(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if friend == nil {
		return nil, fmt.Errorf("该手机号未注册")
	}

	// 2. 不能添加自己
	if friend.ID == userID {
		return nil, fmt.Errorf("不能添加自己为好友")
	}

	// 3. 检查是否已是好友
	areFriends, err := s.friendshipRepo.AreFriends(ctx, userID, friend.ID)
	if err != nil {
		return nil, fmt.Errorf("检查好友状态失败: %w", err)
	}
	if areFriends {
		return &model.AddFriendByPhoneResponse{
			User:  friend.ToProfile(),
			Added: false, // 已是好友
		}, nil
	}

	// 4. 创建双向好友关系
	now := time.Now()
	friendship1 := &model.Friendship{
		ID:        uuid.New().String(),
		UserID:    userID,
		FriendID:  friend.ID,
		Source:    model.FriendshipSourceSearch,
		CreatedAt: now,
	}
	friendship2 := &model.Friendship{
		ID:        uuid.New().String(),
		UserID:    friend.ID,
		FriendID:  userID,
		Source:    model.FriendshipSourceSearch,
		CreatedAt: now,
	}

	if err := s.friendshipRepo.Create(ctx, friendship1); err != nil {
		return nil, fmt.Errorf("添加好友失败: %w", err)
	}
	if err := s.friendshipRepo.Create(ctx, friendship2); err != nil {
		_ = s.friendshipRepo.Delete(ctx, userID, friend.ID)
		return nil, fmt.Errorf("添加好友失败: %w", err)
	}

	return &model.AddFriendByPhoneResponse{
		User:  friend.ToProfile(),
		Added: true,
	}, nil
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

// ==================== 好友请求相关方法 ====================

// SendFriendRequest 发送好友请求
func (s *FriendshipService) SendFriendRequest(ctx context.Context, fromUserID, phone, message string) (*model.SendFriendRequestResponse, error) {
	// 1. 根据手机号查找用户
	toUser, err := s.userRepo.GetByPhone(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if toUser == nil {
		return nil, fmt.Errorf("该手机号未注册")
	}

	// 2. 不能添加自己
	if toUser.ID == fromUserID {
		return nil, fmt.Errorf("不能添加自己为好友")
	}

	// 3. 检查是否已是好友
	areFriends, err := s.friendshipRepo.AreFriends(ctx, fromUserID, toUser.ID)
	if err != nil {
		return nil, fmt.Errorf("检查好友状态失败: %w", err)
	}
	if areFriends {
		return &model.SendFriendRequestResponse{
			User:   toUser.ToProfile(),
			Status: "ALREADY_FRIENDS",
		}, nil
	}

	// 4. 检查是否已有待处理的请求
	existingRequest, err := s.friendRequestRepo.GetPendingRequest(ctx, fromUserID, toUser.ID)
	if err != nil {
		return nil, fmt.Errorf("检查请求状态失败: %w", err)
	}
	if existingRequest != nil {
		return &model.SendFriendRequestResponse{
			RequestID: existingRequest.ID,
			User:      toUser.ToProfile(),
			Status:    "ALREADY_REQUESTED",
		}, nil
	}

	// 5. 检查对方是否已经向我发送了请求（如果是，直接同意）
	reverseRequest, err := s.friendRequestRepo.GetPendingRequest(ctx, toUser.ID, fromUserID)
	if err != nil {
		return nil, fmt.Errorf("检查请求状态失败: %w", err)
	}
	if reverseRequest != nil {
		// 对方已经向我发送请求，直接同意并成为好友
		if err := s.acceptFriendRequest(ctx, reverseRequest); err != nil {
			return nil, err
		}
		return &model.SendFriendRequestResponse{
			RequestID: reverseRequest.ID,
			User:      toUser.ToProfile(),
			Status:    "ACCEPTED",
		}, nil
	}

	// 6. 创建新的好友请求
	request := &model.FriendRequest{
		ID:         uuid.New().String(),
		FromUserID: fromUserID,
		ToUserID:   toUser.ID,
		Message:    message,
		Status:     model.FriendRequestStatusPending,
	}
	if err := s.friendRequestRepo.Create(ctx, request); err != nil {
		return nil, fmt.Errorf("创建好友请求失败: %w", err)
	}

	return &model.SendFriendRequestResponse{
		RequestID: request.ID,
		User:      toUser.ToProfile(),
		Status:    "PENDING",
	}, nil
}

// GetReceivedRequests 获取收到的好友请求
func (s *FriendshipService) GetReceivedRequests(ctx context.Context, userID string) ([]*model.FriendRequestInfo, error) {
	requests, err := s.friendRequestRepo.GetReceivedRequests(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取好友请求失败: %w", err)
	}

	// 获取发送者信息
	userIDs := make([]string, 0, len(requests))
	for _, req := range requests {
		userIDs = append(userIDs, req.FromUserID)
	}

	users, err := s.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	userMap := make(map[string]*model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]*model.FriendRequestInfo, 0, len(requests))
	for _, req := range requests {
		if user, ok := userMap[req.FromUserID]; ok {
			result = append(result, &model.FriendRequestInfo{
				ID:        req.ID,
				User:      user.ToProfile(),
				Message:   req.Message,
				Status:    string(req.Status),
				CreatedAt: req.CreatedAt,
			})
		}
	}

	return result, nil
}

// GetSentRequests 获取发出的好友请求
func (s *FriendshipService) GetSentRequests(ctx context.Context, userID string) ([]*model.FriendRequestInfo, error) {
	requests, err := s.friendRequestRepo.GetSentRequests(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取好友请求失败: %w", err)
	}

	// 获取接收者信息
	userIDs := make([]string, 0, len(requests))
	for _, req := range requests {
		userIDs = append(userIDs, req.ToUserID)
	}

	users, err := s.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	userMap := make(map[string]*model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]*model.FriendRequestInfo, 0, len(requests))
	for _, req := range requests {
		if user, ok := userMap[req.ToUserID]; ok {
			result = append(result, &model.FriendRequestInfo{
				ID:        req.ID,
				User:      user.ToProfile(),
				Message:   req.Message,
				Status:    string(req.Status),
				CreatedAt: req.CreatedAt,
			})
		}
	}

	return result, nil
}

// HandleFriendRequest 处理好友请求（同意/拒绝）
func (s *FriendshipService) HandleFriendRequest(ctx context.Context, userID, requestID string, accept bool) error {
	// 1. 获取请求
	request, err := s.friendRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("获取请求失败: %w", err)
	}
	if request == nil {
		return fmt.Errorf("请求不存在")
	}

	// 2. 验证是否是接收者
	if request.ToUserID != userID {
		return fmt.Errorf("无权操作此请求")
	}

	// 3. 验证状态
	if request.Status != model.FriendRequestStatusPending {
		return fmt.Errorf("请求已处理")
	}

	if accept {
		return s.acceptFriendRequest(ctx, request)
	} else {
		return s.friendRequestRepo.UpdateStatus(ctx, requestID, model.FriendRequestStatusRejected)
	}
}

// acceptFriendRequest 接受好友请求并创建好友关系
func (s *FriendshipService) acceptFriendRequest(ctx context.Context, request *model.FriendRequest) error {
	// 1. 更新请求状态
	if err := s.friendRequestRepo.UpdateStatus(ctx, request.ID, model.FriendRequestStatusAccepted); err != nil {
		return fmt.Errorf("更新请求状态失败: %w", err)
	}

	// 2. 创建双向好友关系
	now := time.Now()
	friendship1 := &model.Friendship{
		ID:        uuid.New().String(),
		UserID:    request.FromUserID,
		FriendID:  request.ToUserID,
		Source:    model.FriendshipSourceRequest,
		CreatedAt: now,
	}
	friendship2 := &model.Friendship{
		ID:        uuid.New().String(),
		UserID:    request.ToUserID,
		FriendID:  request.FromUserID,
		Source:    model.FriendshipSourceRequest,
		CreatedAt: now,
	}

	if err := s.friendshipRepo.Create(ctx, friendship1); err != nil {
		return fmt.Errorf("创建好友关系失败: %w", err)
	}
	if err := s.friendshipRepo.Create(ctx, friendship2); err != nil {
		_ = s.friendshipRepo.Delete(ctx, request.FromUserID, request.ToUserID)
		return fmt.Errorf("创建好友关系失败: %w", err)
	}

	return nil
}

// GetPendingRequestCount 获取待处理请求数量
func (s *FriendshipService) GetPendingRequestCount(ctx context.Context, userID string) (int, error) {
	return s.friendRequestRepo.GetPendingCount(ctx, userID)
}
