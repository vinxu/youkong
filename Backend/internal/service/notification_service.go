package service

import (
	"context"
	"log"

	"youkong/internal/model"
	"youkong/internal/pkg/push"
	"youkong/internal/pkg/ws"
	"youkong/internal/repository"
)

// NotificationService 推送通知服务
type NotificationService struct {
	deviceTokenRepo *repository.DeviceTokenRepository
	pushManager     *push.Manager
	wsManager       *ws.Manager
}

// NewNotificationService 创建推送通知服务
func NewNotificationService(
	deviceTokenRepo *repository.DeviceTokenRepository,
	pushManager *push.Manager,
	wsManager *ws.Manager,
) *NotificationService {
	return &NotificationService{
		deviceTokenRepo: deviceTokenRepo,
		pushManager:     pushManager,
		wsManager:       wsManager,
	}
}

// NotifyNewMessage 发送新消息通知
func (s *NotificationService) NotifyNewMessage(ctx context.Context, recipientID string, message *model.Message, sender *model.User) error {
	// 1. 检查用户是否在线（WebSocket 已连接）
	if s.wsManager != nil && s.wsManager.IsOnline(recipientID) {
		log.Printf("[Notification] 用户 %s 在线，跳过远程推送", recipientID)
		return nil
	}

	// 2. 检查推送管理器是否可用
	if s.pushManager == nil || !s.pushManager.IsEnabled() {
		log.Println("[Notification] 推送服务未启用")
		return nil
	}

	// 3. 获取用户的设备 Token
	tokens, err := s.deviceTokenRepo.GetActiveByUserID(ctx, recipientID)
	if err != nil {
		log.Printf("[Notification] 获取设备 Token 失败: %v", err)
		return err
	}

	if len(tokens) == 0 {
		log.Printf("[Notification] 用户 %s 没有注册设备 Token", recipientID)
		return nil
	}

	// 4. 构建通知内容
	notification := s.buildMessageNotification(message, sender)

	// 5. 发送推送
	results := s.pushManager.SendToTokens(ctx, tokens, notification)

	// 6. 处理无效 Token
	invalidTokens := push.GetInvalidTokens(results)
	if len(invalidTokens) > 0 {
		s.deactivateInvalidTokens(ctx, invalidTokens)
	}

	return nil
}

// NotifyNewMessageToUser 向指定用户发送消息通知（简化版）
func (s *NotificationService) NotifyNewMessageToUser(ctx context.Context, recipientID, title, body string, data map[string]string) error {
	// 1. 检查用户是否在线
	if s.wsManager != nil && s.wsManager.IsOnline(recipientID) {
		return nil
	}

	// 2. 检查推送管理器
	if s.pushManager == nil || !s.pushManager.IsEnabled() {
		return nil
	}

	// 3. 获取设备 Token
	tokens, err := s.deviceTokenRepo.GetActiveByUserID(ctx, recipientID)
	if err != nil || len(tokens) == 0 {
		return err
	}

	// 4. 构建通知
	notification := push.NewNotification(title, body).WithBadge(1)
	for k, v := range data {
		notification.WithData(k, v)
	}

	// 5. 发送
	results := s.pushManager.SendToTokens(ctx, tokens, notification)

	// 6. 清理无效 Token
	invalidTokens := push.GetInvalidTokens(results)
	if len(invalidTokens) > 0 {
		s.deactivateInvalidTokens(ctx, invalidTokens)
	}

	return nil
}

// SendPushToUser 向指定用户发送推送通知
func (s *NotificationService) SendPushToUser(ctx context.Context, userID, title, body string) error {
	return s.NotifyNewMessageToUser(ctx, userID, title, body, nil)
}

// SendInteractionPush 发送互动推送（不检查在线状态，始终发送）
func (s *NotificationService) SendInteractionPush(ctx context.Context, recipientID, title, body string, data map[string]string) error {
	if s.pushManager == nil || !s.pushManager.IsEnabled() {
		return nil
	}

	tokens, err := s.deviceTokenRepo.GetActiveByUserID(ctx, recipientID)
	if err != nil || len(tokens) == 0 {
		return err
	}

	notification := push.NewNotification(title, body).WithBadge(1)
	for k, v := range data {
		notification.WithData(k, v)
	}

	results := s.pushManager.SendToTokens(ctx, tokens, notification)

	invalidTokens := push.GetInvalidTokens(results)
	if len(invalidTokens) > 0 {
		s.deactivateInvalidTokens(ctx, invalidTokens)
	}

	return nil
}

// buildMessageNotification 构建消息通知
func (s *NotificationService) buildMessageNotification(message *model.Message, sender *model.User) *push.Notification {
	title := sender.Nickname
	body := truncateString(message.Content.String, 50)

	// 根据消息类型调整内容
	switch message.Type {
	case model.MessageTypeAvailabilityCard:
		body = "分享了有空状态"
	case model.MessageTypeConfirmRequest:
		body = "发起了确认请求"
	case model.MessageTypeConfirmResponse:
		body = "回复了确认请求"
	}

	notification := push.NewNotification(title, body).
		WithBadge(1).
		WithData("type", "new_message").
		WithData("conversation_id", message.ConversationID).
		WithData("sender_id", sender.ID).
		WithData("message_id", message.ID)

	return notification
}

// deactivateInvalidTokens 停用无效的 Token
func (s *NotificationService) deactivateInvalidTokens(ctx context.Context, tokens []string) {
	for _, token := range tokens {
		if err := s.deviceTokenRepo.Deactivate(ctx, token); err != nil {
			log.Printf("[Notification] 停用 Token 失败: %v", err)
		} else {
			log.Printf("[Notification] 已停用无效 Token: %s...", truncateString(token, 10))
		}
	}
}

// RegisterDeviceToken 注册设备 Token
func (s *NotificationService) RegisterDeviceToken(ctx context.Context, userID string, req *model.RegisterDeviceTokenRequest) error {
	token := &model.DeviceToken{
		UserID:     userID,
		Token:      req.Token,
		Platform:   model.Platform(req.Platform),
		DeviceName: req.DeviceName,
		AppVersion: req.AppVersion,
	}

	return s.deviceTokenRepo.Upsert(ctx, token)
}

// UnregisterDeviceToken 注销设备 Token
func (s *NotificationService) UnregisterDeviceToken(ctx context.Context, token string) error {
	return s.deviceTokenRepo.Deactivate(ctx, token)
}

// UnregisterAllDeviceTokens 注销用户所有设备 Token（用户登出时）
func (s *NotificationService) UnregisterAllDeviceTokens(ctx context.Context, userID string) error {
	return s.deviceTokenRepo.DeactivateByUserID(ctx, userID)
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
