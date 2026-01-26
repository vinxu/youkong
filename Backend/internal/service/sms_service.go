package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"youkong/internal/pkg/tencent"
)

// 测试账号配置 - 前端开发测试用
var testAccounts = map[string]string{
	"13800000001": "111111", // 测试账号1
	"13800000002": "222222", // 测试账号2
	"13800000003": "333333", // 测试账号3
	"13800138000": "123456", // 测试账号4
}

type SMSService struct {
	smsClient   *tencent.SMSClient
	redisClient *tencent.RedisClient
}

func NewSMSService(smsClient *tencent.SMSClient, redisClient *tencent.RedisClient) *SMSService {
	return &SMSService{
		smsClient:   smsClient,
		redisClient: redisClient,
	}
}

// isTestAccount 检查是否为测试账号
func (s *SMSService) isTestAccount(phone string) (string, bool) {
	code, exists := testAccounts[phone]
	return code, exists
}

func (s *SMSService) SendVerifyCode(ctx context.Context, phone string) error {
	// 检查是否为测试账号
	if code, isTest := s.isTestAccount(phone); isTest {
		// 测试账号直接保存固定验证码，不发送短信
		if err := s.redisClient.SetSMSCode(ctx, phone, code); err != nil {
			return fmt.Errorf("保存验证码失败: %w", err)
		}
		return nil
	}

	// 检查发送频率限制
	canSend, err := s.redisClient.CheckSMSRateLimit(ctx, phone)
	if err != nil {
		return fmt.Errorf("检查发送频率失败: %w", err)
	}
	if !canSend {
		return fmt.Errorf("发送太频繁，请稍后再试")
	}

	// 生成6位验证码
	code := s.generateCode()

	// 发送短信
	if err := s.smsClient.SendVerifyCode(phone, code); err != nil {
		return err
	}

	// 保存验证码到Redis
	if err := s.redisClient.SetSMSCode(ctx, phone, code); err != nil {
		return fmt.Errorf("保存验证码失败: %w", err)
	}

	// 设置发送频率限制
	if err := s.redisClient.SetSMSRateLimit(ctx, phone); err != nil {
		return fmt.Errorf("设置发送频率限制失败: %w", err)
	}

	return nil
}

func (s *SMSService) VerifyCode(ctx context.Context, phone, code string) (bool, error) {
	// 测试账号直接验证固定验证码
	if testCode, isTest := s.isTestAccount(phone); isTest {
		return code == testCode, nil
	}

	savedCode, err := s.redisClient.GetSMSCode(ctx, phone)
	if err != nil {
		return false, nil // 验证码不存在或已过期
	}

	if savedCode != code {
		return false, nil
	}

	// 验证成功后删除验证码
	_ = s.redisClient.DeleteSMSCode(ctx, phone)

	return true, nil
}

func (s *SMSService) generateCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}
