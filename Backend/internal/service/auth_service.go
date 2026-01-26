package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"youkong/internal/model"
	"youkong/internal/pkg/jwt"
	"youkong/internal/repository"
)

type AuthService struct {
	userRepo   *repository.UserRepository
	smsService *SMSService
	jwtManager *jwt.Manager
}

func NewAuthService(userRepo *repository.UserRepository, smsService *SMSService, jwtManager *jwt.Manager) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		smsService: smsService,
		jwtManager: jwtManager,
	}
}

type LoginResult struct {
	Token    string       `json:"token"`
	User     *model.User  `json:"user"`
	IsNewUser bool        `json:"isNewUser"`
}

func (s *AuthService) SendSMSCode(ctx context.Context, phone string) error {
	return s.smsService.SendVerifyCode(ctx, phone)
}

func (s *AuthService) VerifyAndLogin(ctx context.Context, phone, code string) (*LoginResult, error) {
	// 验证验证码
	valid, err := s.smsService.VerifyCode(ctx, phone, code)
	if err != nil {
		return nil, fmt.Errorf("验证失败: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("验证码错误或已过期")
	}

	// 查找或创建用户
	user, err := s.userRepo.GetByPhone(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	isNewUser := user == nil
	if isNewUser {
		// 创建新用户
		user = &model.User{
			ID:        uuid.New().String(),
			Phone:     phone,
			PhoneHash: s.hashPhone(phone),
			Nickname:  s.generateDefaultNickname(phone),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("创建用户失败: %w", err)
		}
	}

	// 更新最后活跃时间
	_ = s.userRepo.UpdateLastActive(ctx, user.ID)

	// 生成Token
	token, err := s.jwtManager.GenerateToken(user.ID, user.Phone)
	if err != nil {
		return nil, fmt.Errorf("生成Token失败: %w", err)
	}

	return &LoginResult{
		Token:     token,
		User:      user,
		IsNewUser: isNewUser,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, oldToken string) (string, error) {
	return s.jwtManager.RefreshToken(oldToken)
}

func (s *AuthService) hashPhone(phone string) string {
	hash := sha256.Sum256([]byte(phone))
	return hex.EncodeToString(hash[:])
}

func (s *AuthService) generateDefaultNickname(phone string) string {
	// 使用手机号后4位生成默认昵称
	if len(phone) >= 4 {
		return "用户" + phone[len(phone)-4:]
	}
	return "用户" + phone
}
