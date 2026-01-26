package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"youkong/internal/model"
	"youkong/internal/repository"
)

const (
	DefaultMaxUses      = 100
	DefaultExpireDays   = 7
	MaxExpireDays       = 30
	RateLimitPerDay     = 10
	InviteCodeLength    = 8
	InviteCodeChars     = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 排除易混淆字符 I,O,0,1
)

type InvitationService struct {
	invitationRepo *repository.InvitationRepository
	circleRepo     *repository.CircleRepository
	userRepo       *repository.UserRepository
	friendshipRepo *repository.FriendshipRepository
	baseURL        string
}

func NewInvitationService(
	invitationRepo *repository.InvitationRepository,
	circleRepo *repository.CircleRepository,
	userRepo *repository.UserRepository,
	friendshipRepo *repository.FriendshipRepository,
	baseURL string,
) *InvitationService {
	return &InvitationService{
		invitationRepo: invitationRepo,
		circleRepo:     circleRepo,
		userRepo:       userRepo,
		friendshipRepo: friendshipRepo,
		baseURL:        baseURL,
	}
}

type CreateInvitationRequest struct {
	CircleID    string `json:"circleId"`
	MaxUses     int    `json:"maxUses"`
	ExpiresDays int    `json:"expiresDays"`
}

func (s *InvitationService) Create(ctx context.Context, inviterID string, req *CreateInvitationRequest) (*model.InvitationDetail, error) {
	// 检查频率限制
	count, err := s.invitationRepo.CountByInviterToday(ctx, inviterID)
	if err != nil {
		return nil, fmt.Errorf("查询邀请次数失败: %w", err)
	}
	if count >= RateLimitPerDay {
		return nil, fmt.Errorf("今日邀请次数已达上限(%d次)", RateLimitPerDay)
	}

	// 验证圈子（如果指定）
	var circle *model.Circle
	if req.CircleID != "" {
		circle, err = s.circleRepo.GetByID(ctx, req.CircleID)
		if err != nil {
			return nil, fmt.Errorf("查询圈子失败: %w", err)
		}
		if circle == nil {
			return nil, fmt.Errorf("圈子不存在")
		}
		if circle.OwnerID != inviterID {
			// 检查是否是圈子成员
			isMember, _ := s.circleRepo.IsMember(ctx, req.CircleID, inviterID)
			if !isMember {
				return nil, fmt.Errorf("无权限创建此圈子的邀请")
			}
		}
	}

	// 设置默认值
	maxUses := req.MaxUses
	if maxUses <= 0 {
		maxUses = DefaultMaxUses
	}

	expiresDays := req.ExpiresDays
	if expiresDays <= 0 {
		expiresDays = DefaultExpireDays
	}
	if expiresDays > MaxExpireDays {
		expiresDays = MaxExpireDays
	}

	// 生成邀请码
	code, err := s.generateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("生成邀请码失败: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(expiresDays) * 24 * time.Hour)

	invitation := &model.Invitation{
		ID:        uuid.New().String(),
		Code:      code,
		InviterID: inviterID,
		MaxUses:   maxUses,
		UseCount:  0,
		ExpiresAt: sql.NullTime{Time: expiresAt, Valid: true},
		Status:    model.InvitationStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if req.CircleID != "" {
		invitation.CircleID = sql.NullString{String: req.CircleID, Valid: true}
	}

	if err := s.invitationRepo.Create(ctx, invitation); err != nil {
		return nil, fmt.Errorf("创建邀请失败: %w", err)
	}

	// 获取邀请者信息
	inviter, _ := s.userRepo.GetByID(ctx, inviterID)
	var inviterProfile *model.UserProfile
	if inviter != nil {
		inviterProfile = inviter.ToProfile()
	}

	// 构建圈子信息
	var circleInfo *model.CircleInfo
	if circle != nil {
		memberCount, _ := s.circleRepo.GetMemberCount(ctx, circle.ID)
		circleInfo = &model.CircleInfo{
			ID:          circle.ID,
			Name:        circle.Name,
			Emoji:       circle.Emoji,
			Color:       circle.Color,
			MemberCount: memberCount,
		}
	}

	return &model.InvitationDetail{
		ID:        invitation.ID,
		Code:      invitation.Code,
		InviteURL: s.getInviteURL(invitation.Code),
		Inviter:   inviterProfile,
		Circle:    circleInfo,
		MaxUses:   invitation.MaxUses,
		UseCount:  invitation.UseCount,
		ExpiresAt: &expiresAt,
		Status:    string(invitation.Status),
		IsValid:   invitation.IsValid(),
		CreatedAt: invitation.CreatedAt,
	}, nil
}

func (s *InvitationService) GetByID(ctx context.Context, id string) (*model.InvitationDetail, error) {
	invitation, err := s.invitationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("查询邀请失败: %w", err)
	}
	if invitation == nil {
		return nil, nil
	}

	return s.toInvitationDetail(ctx, invitation)
}

func (s *InvitationService) GetByCode(ctx context.Context, code string) (*model.InvitationPublicInfo, error) {
	invitation, err := s.invitationRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("查询邀请失败: %w", err)
	}
	if invitation == nil {
		return nil, nil
	}

	// 获取邀请者信息
	inviter, _ := s.userRepo.GetByID(ctx, invitation.InviterID)
	var inviterProfile *model.UserProfile
	if inviter != nil {
		inviterProfile = inviter.ToProfile()
	}

	// 获取圈子信息
	var circleInfo *model.CircleInfo
	if invitation.CircleID.Valid {
		circle, _ := s.circleRepo.GetByID(ctx, invitation.CircleID.String)
		if circle != nil {
			memberCount, _ := s.circleRepo.GetMemberCount(ctx, circle.ID)
			circleInfo = &model.CircleInfo{
				ID:          circle.ID,
				Name:        circle.Name,
				Emoji:       circle.Emoji,
				MemberCount: memberCount,
			}
		}
	}

	return &model.InvitationPublicInfo{
		Inviter: inviterProfile,
		Circle:  circleInfo,
		IsValid: invitation.IsValid(),
	}, nil
}

func (s *InvitationService) GetMyInvitations(ctx context.Context, inviterID string) ([]*model.InvitationDetail, error) {
	invitations, err := s.invitationRepo.GetByInviterID(ctx, inviterID)
	if err != nil {
		return nil, fmt.Errorf("查询邀请列表失败: %w", err)
	}

	result := make([]*model.InvitationDetail, 0, len(invitations))
	for _, inv := range invitations {
		detail, err := s.toInvitationDetail(ctx, inv)
		if err == nil {
			result = append(result, detail)
		}
	}

	return result, nil
}

func (s *InvitationService) Disable(ctx context.Context, id, ownerID string) error {
	invitation, err := s.invitationRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("查询邀请失败: %w", err)
	}
	if invitation == nil {
		return fmt.Errorf("邀请不存在")
	}
	if invitation.InviterID != ownerID {
		return fmt.Errorf("无权限禁用此邀请")
	}

	return s.invitationRepo.UpdateStatus(ctx, id, model.InvitationStatusDisabled)
}

func (s *InvitationService) Accept(ctx context.Context, code string, inviteeID string) (*model.CircleInfo, error) {
	invitation, err := s.invitationRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("查询邀请失败: %w", err)
	}
	if invitation == nil {
		return nil, fmt.Errorf("邀请不存在")
	}

	if !invitation.IsValid() {
		return nil, fmt.Errorf("邀请已失效")
	}

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

	// 检查是否已经是好友
	areFriends, _ := s.friendshipRepo.AreFriends(ctx, invitation.InviterID, inviteeID)
	if !areFriends {
		// 创建双向好友关系
		invitationIDNull := sql.NullString{String: invitation.ID, Valid: true}
		friendship1 := &model.Friendship{
			ID:           uuid.New().String(),
			UserID:       invitation.InviterID,
			FriendID:     inviteeID,
			Source:       model.FriendshipSourceInvitation,
			InvitationID: invitationIDNull,
			CreatedAt:    now,
		}
		friendship2 := &model.Friendship{
			ID:           uuid.New().String(),
			UserID:       inviteeID,
			FriendID:     invitation.InviterID,
			Source:       model.FriendshipSourceInvitation,
			InvitationID: invitationIDNull,
			CreatedAt:    now,
		}
		_ = s.friendshipRepo.Create(ctx, friendship1)
		_ = s.friendshipRepo.Create(ctx, friendship2)
	}

	// 加入圈子
	var joinedCircle *model.CircleInfo
	if invitation.CircleID.Valid {
		circleID := invitation.CircleID.String
		circle, err := s.circleRepo.GetByID(ctx, circleID)
		if err == nil && circle != nil {
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

func (s *InvitationService) generateInviteCode() (string, error) {
	code := make([]byte, InviteCodeLength)
	charLen := big.NewInt(int64(len(InviteCodeChars)))

	for i := 0; i < InviteCodeLength; i++ {
		n, err := rand.Int(rand.Reader, charLen)
		if err != nil {
			return "", err
		}
		code[i] = InviteCodeChars[n.Int64()]
	}

	return string(code), nil
}

func (s *InvitationService) getInviteURL(code string) string {
	return s.baseURL + code
}

func (s *InvitationService) toInvitationDetail(ctx context.Context, invitation *model.Invitation) (*model.InvitationDetail, error) {
	// 获取邀请者信息
	inviter, _ := s.userRepo.GetByID(ctx, invitation.InviterID)
	var inviterProfile *model.UserProfile
	if inviter != nil {
		inviterProfile = inviter.ToProfile()
	}

	// 获取圈子信息
	var circleInfo *model.CircleInfo
	if invitation.CircleID.Valid {
		circle, _ := s.circleRepo.GetByID(ctx, invitation.CircleID.String)
		if circle != nil {
			memberCount, _ := s.circleRepo.GetMemberCount(ctx, circle.ID)
			circleInfo = &model.CircleInfo{
				ID:          circle.ID,
				Name:        circle.Name,
				Emoji:       circle.Emoji,
				Color:       circle.Color,
				MemberCount: memberCount,
			}
		}
	}

	return &model.InvitationDetail{
		ID:        invitation.ID,
		Code:      invitation.Code,
		InviteURL: s.getInviteURL(invitation.Code),
		Inviter:   inviterProfile,
		Circle:    circleInfo,
		MaxUses:   invitation.MaxUses,
		UseCount:  invitation.UseCount,
		ExpiresAt: invitation.GetExpiresAt(),
		Status:    string(invitation.Status),
		IsValid:   invitation.IsValid(),
		CreatedAt: invitation.CreatedAt,
	}, nil
}
