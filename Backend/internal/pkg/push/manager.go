package push

import (
	"context"
	"log"
	"sync"

	"youkong/internal/model"
)

// Manager 推送管理器，统一管理 APNs 和 TPNS
type Manager struct {
	apns *APNsClient
	tpns *TPNSClient
}

// NewManager 创建推送管理器
func NewManager(apns *APNsClient, tpns *TPNSClient) *Manager {
	return &Manager{
		apns: apns,
		tpns: tpns,
	}
}

// SendResult 推送结果
type SendResult struct {
	Token   string
	Success bool
	Error   error
}

// Send 向单个设备发送推送
func (m *Manager) Send(ctx context.Context, token *model.DeviceToken, notification *Notification) error {
	switch token.Platform {
	case model.PlatformIOS:
		if m.apns != nil && m.apns.IsEnabled() {
			return m.apns.Send(ctx, token.Token, notification)
		}
	case model.PlatformAndroid:
		if m.tpns != nil && m.tpns.IsEnabled() {
			return m.tpns.Send(ctx, token.Token, notification)
		}
	}
	return nil
}

// SendToTokens 向多个设备发送推送，返回失败的 Token
func (m *Manager) SendToTokens(ctx context.Context, tokens []*model.DeviceToken, notification *Notification) []SendResult {
	if len(tokens) == 0 {
		return nil
	}

	var results []SendResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, token := range tokens {
		wg.Add(1)
		go func(t *model.DeviceToken) {
			defer wg.Done()

			err := m.Send(ctx, t, notification)
			result := SendResult{
				Token:   t.Token,
				Success: err == nil,
				Error:   err,
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()

			if err != nil {
				log.Printf("[Push] 推送失败 (platform=%s, token=%s...): %v",
					t.Platform, truncateToken(t.Token), err)
			}
		}(token)
	}

	wg.Wait()
	return results
}

// IsEnabled 检查是否有任何推送通道可用
func (m *Manager) IsEnabled() bool {
	apnsEnabled := m.apns != nil && m.apns.IsEnabled()
	tpnsEnabled := m.tpns != nil && m.tpns.IsEnabled()
	return apnsEnabled || tpnsEnabled
}

// GetInvalidTokens 从推送结果中提取无效的 Token
func GetInvalidTokens(results []SendResult) []string {
	var invalidTokens []string
	for _, r := range results {
		if !r.Success && r.Error != nil {
			if IsInvalidTokenError(r.Error) {
				invalidTokens = append(invalidTokens, r.Token)
			}
		}
	}
	return invalidTokens
}

func truncateToken(token string) string {
	if len(token) <= 10 {
		return token
	}
	return token[:10]
}
