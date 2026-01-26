package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"youkong/internal/model"
	"youkong/internal/pkg/jwt"
	"youkong/internal/pkg/wechat"
	"youkong/internal/repository"
)

type WechatService struct {
	wechatRepo     *repository.WechatRepository
	userRepo       *repository.UserRepository
	invitationRepo *repository.InvitationRepository
	friendshipRepo *repository.FriendshipRepository
	circleRepo     *repository.CircleRepository
	wechatClient   *wechat.Client
	jwtManager     *jwt.Manager
}

func NewWechatService(
	wechatRepo *repository.WechatRepository,
	userRepo *repository.UserRepository,
	invitationRepo *repository.InvitationRepository,
	friendshipRepo *repository.FriendshipRepository,
	circleRepo *repository.CircleRepository,
	wechatClient *wechat.Client,
	jwtManager *jwt.Manager,
) *WechatService {
	return &WechatService{
		wechatRepo:     wechatRepo,
		userRepo:       userRepo,
		invitationRepo: invitationRepo,
		friendshipRepo: friendshipRepo,
		circleRepo:     circleRepo,
		wechatClient:   wechatClient,
		jwtManager:     jwtManager,
	}
}

type WechatLoginResult struct {
	Token        string             `json:"token"`
	User         *model.User        `json:"user"`
	IsNewUser    bool               `json:"isNewUser"`
	JoinedCircle *model.CircleInfo  `json:"joinedCircle,omitempty"`
}

func (s *WechatService) Login(ctx context.Context, code string, inviteCode string) (*WechatLoginResult, error) {
	if s.wechatClient == nil {
		return nil, fmt.Errorf("微信客户端未配置")
	}

	// 获取微信用户信息
	wechatUserInfo, err := s.wechatClient.GetUserInfo(code)
	if err != nil {
		return nil, fmt.Errorf("获取微信用户信息失败: %w", err)
	}

	// 查找是否已绑定
	existingWechat, err := s.wechatRepo.GetByOpenID(ctx, wechatUserInfo.OpenID)
	if err != nil {
		return nil, fmt.Errorf("查询微信绑定失败: %w", err)
	}

	var user *model.User
	isNewUser := false
	var joinedCircle *model.CircleInfo

	if existingWechat != nil {
		// 已绑定，获取用户
		user, err = s.userRepo.GetByID(ctx, existingWechat.UserID)
		if err != nil {
			return nil, fmt.Errorf("获取用户失败: %w", err)
		}
		if user == nil {
			return nil, fmt.Errorf("用户不存在")
		}

		// 更新微信信息
		existingWechat.Nickname = wechatUserInfo.Nickname
		existingWechat.Avatar = wechatUserInfo.Avatar
		_ = s.wechatRepo.Update(ctx, existingWechat)
	} else {
		// 新用户注册
		isNewUser = true
		now := time.Now()

		user = &model.User{
			ID:          uuid.New().String(),
			Nickname:    wechatUserInfo.Nickname,
			Avatar:      wechatUserInfo.Avatar,
			WechatBound: true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("创建用户失败: %w", err)
		}

		// 创建微信绑定
		wechatUser := &model.WechatUser{
			ID:        uuid.New().String(),
			UserID:    user.ID,
			OpenID:    wechatUserInfo.OpenID,
			UnionID:   wechatUserInfo.UnionID,
			Nickname:  wechatUserInfo.Nickname,
			Avatar:    wechatUserInfo.Avatar,
			CreatedAt: now,
		}
		if err := s.wechatRepo.Create(ctx, wechatUser); err != nil {
			return nil, fmt.Errorf("创建微信绑定失败: %w", err)
		}

		// 处理邀请码
		if inviteCode != "" {
			joinedCircle, err = s.processInvitation(ctx, inviteCode, user.ID)
			if err != nil {
				// 邀请处理失败不影响登录
				fmt.Printf("处理邀请失败: %v\n", err)
			}
		}
	}

	// 更新最后活跃时间
	_ = s.userRepo.UpdateLastActive(ctx, user.ID)

	// 生成Token
	token, err := s.jwtManager.GenerateToken(user.ID, user.Phone)
	if err != nil {
		return nil, fmt.Errorf("生成Token失败: %w", err)
	}

	return &WechatLoginResult{
		Token:        token,
		User:         user,
		IsNewUser:    isNewUser,
		JoinedCircle: joinedCircle,
	}, nil
}

func (s *WechatService) processInvitation(ctx context.Context, inviteCode string, inviteeID string) (*model.CircleInfo, error) {
	// 获取邀请
	invitation, err := s.invitationRepo.GetByCode(ctx, inviteCode)
	if err != nil {
		return nil, fmt.Errorf("查询邀请失败: %w", err)
	}
	if invitation == nil {
		return nil, fmt.Errorf("邀请不存在")
	}

	if !invitation.IsValid() {
		return nil, fmt.Errorf("邀请已失效")
	}

	// 不能邀请自己
	if invitation.InviterID == inviteeID {
		return nil, fmt.Errorf("不能接受自己的邀请")
	}

	// 检查是否已接受过
	existingRecord, err := s.invitationRepo.GetRecordByInvitationAndInvitee(ctx, invitation.ID, inviteeID)
	if err != nil {
		return nil, fmt.Errorf("查询邀请记录失败: %w", err)
	}
	if existingRecord != nil {
		return nil, fmt.Errorf("已接受过此邀请")
	}

	now := time.Now()

	// 创建邀请记录
	record := &model.InvitationRecord{
		ID:           uuid.New().String(),
		InvitationID: invitation.ID,
		InviterID:    invitation.InviterID,
		InviteeID:    inviteeID,
		CircleID:     invitation.CircleID,
		CreatedAt:    now,
	}
	if err := s.invitationRepo.CreateRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("创建邀请记录失败: %w", err)
	}

	// 增加使用次数
	_ = s.invitationRepo.IncrementUseCount(ctx, invitation.ID)

	// 创建双向好友关系
	friendship1 := &model.Friendship{
		ID:           uuid.New().String(),
		UserID:       invitation.InviterID,
		FriendID:     inviteeID,
		Source:       model.FriendshipSourceInvitation,
		InvitationID: invitation.CircleID,
		CreatedAt:    now,
	}
	friendship2 := &model.Friendship{
		ID:           uuid.New().String(),
		UserID:       inviteeID,
		FriendID:     invitation.InviterID,
		Source:       model.FriendshipSourceInvitation,
		InvitationID: invitation.CircleID,
		CreatedAt:    now,
	}
	_ = s.friendshipRepo.Create(ctx, friendship1)
	_ = s.friendshipRepo.Create(ctx, friendship2)

	// 加入圈子
	var joinedCircle *model.CircleInfo
	if invitation.CircleID.Valid {
		circleID := invitation.CircleID.String
		circle, err := s.circleRepo.GetByID(ctx, circleID)
		if err == nil && circle != nil {
			// 检查是否已经是成员
			isMember, _ := s.circleRepo.IsMember(ctx, circleID, inviteeID)
			if !isMember {
				member := &model.CircleMember{
					ID:       uuid.New().String(),
					CircleID: circleID,
					UserID:   inviteeID,
					JoinedAt: now,
				}
				_ = s.circleRepo.AddMember(ctx, member)
			}

			memberCount, _ := s.circleRepo.GetMemberCount(ctx, circleID)
			joinedCircle = &model.CircleInfo{
				ID:          circle.ID,
				Name:        circle.Name,
				Emoji:       circle.Emoji,
				Color:       circle.Color,
				MemberCount: memberCount,
			}
		}
	}

	return joinedCircle, nil
}

func (s *WechatService) BindWechat(ctx context.Context, userID string, code string) error {
	if s.wechatClient == nil {
		return fmt.Errorf("微信客户端未配置")
	}

	// 检查是否已绑定
	existing, err := s.wechatRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("查询绑定状态失败: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("已绑定微信")
	}

	// 获取微信用户信息
	wechatUserInfo, err := s.wechatClient.GetUserInfo(code)
	if err != nil {
		return fmt.Errorf("获取微信用户信息失败: %w", err)
	}

	// 检查该微信是否已被其他账号绑定
	existingByOpenID, err := s.wechatRepo.GetByOpenID(ctx, wechatUserInfo.OpenID)
	if err != nil {
		return fmt.Errorf("查询微信绑定失败: %w", err)
	}
	if existingByOpenID != nil {
		return fmt.Errorf("该微信已绑定其他账号")
	}

	// 创建绑定
	wechatUser := &model.WechatUser{
		ID:        uuid.New().String(),
		UserID:    userID,
		OpenID:    wechatUserInfo.OpenID,
		UnionID:   wechatUserInfo.UnionID,
		Nickname:  wechatUserInfo.Nickname,
		Avatar:    wechatUserInfo.Avatar,
		CreatedAt: time.Now(),
	}
	if err := s.wechatRepo.Create(ctx, wechatUser); err != nil {
		return fmt.Errorf("创建微信绑定失败: %w", err)
	}

	// 更新用户微信绑定状态
	user, _ := s.userRepo.GetByID(ctx, userID)
	if user != nil {
		user.WechatBound = true
		_ = s.userRepo.Update(ctx, user)
	}

	return nil
}
