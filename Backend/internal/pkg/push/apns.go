package push

import (
	"context"
	"fmt"
	"log"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"
	"youkong/internal/config"
)

// APNsClient Apple 推送服务客户端
type APNsClient struct {
	client   *apns2.Client
	bundleID string
	enabled  bool
}

// NewAPNsClient 创建 APNs 客户端
func NewAPNsClient(cfg config.APNsConfig) (*APNsClient, error) {
	// 检查配置是否完整
	if cfg.KeyPath == "" || cfg.KeyID == "" || cfg.TeamID == "" || cfg.BundleID == "" {
		log.Println("[APNs] 配置不完整，APNs 推送已禁用")
		return &APNsClient{enabled: false}, nil
	}

	// 加载认证密钥
	authKey, err := token.AuthKeyFromFile(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("加载 APNs AuthKey 失败: %w", err)
	}

	// 创建 token
	tok := &token.Token{
		AuthKey: authKey,
		KeyID:   cfg.KeyID,
		TeamID:  cfg.TeamID,
	}

	// 创建客户端
	var client *apns2.Client
	if cfg.Production {
		client = apns2.NewTokenClient(tok).Production()
	} else {
		client = apns2.NewTokenClient(tok).Development()
	}

	log.Printf("[APNs] 客户端初始化成功 (production=%v, bundleID=%s)", cfg.Production, cfg.BundleID)

	return &APNsClient{
		client:   client,
		bundleID: cfg.BundleID,
		enabled:  true,
	}, nil
}

// Send 发送推送通知
func (c *APNsClient) Send(ctx context.Context, deviceToken string, notification *Notification) error {
	if !c.enabled {
		return nil
	}

	// 构建 payload
	p := payload.NewPayload().
		AlertTitle(notification.Title).
		AlertBody(notification.Body).
		Sound(notification.Sound)

	if notification.Badge > 0 {
		p.Badge(notification.Badge)
	}

	// 添加自定义数据
	for key, value := range notification.Data {
		p.Custom(key, value)
	}

	// 构建通知
	n := &apns2.Notification{
		DeviceToken: deviceToken,
		Topic:       c.bundleID,
		Payload:     p,
	}

	// 发送
	res, err := c.client.PushWithContext(ctx, n)
	if err != nil {
		return fmt.Errorf("APNs 推送失败: %w", err)
	}

	if !res.Sent() {
		// 常见错误：BadDeviceToken, Unregistered, ExpiredToken
		return fmt.Errorf("APNs 推送被拒绝: %s (status=%d)", res.Reason, res.StatusCode)
	}

	return nil
}

// IsEnabled 检查是否启用
func (c *APNsClient) IsEnabled() bool {
	return c.enabled
}

// IsInvalidTokenError 检查是否是无效 Token 错误
func IsInvalidTokenError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "BadDeviceToken") ||
		contains(errStr, "Unregistered") ||
		contains(errStr, "ExpiredToken")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
