package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"youkong/internal/model"
	"youkong/internal/repository"
)

// InteractionService 互动业务逻辑
type InteractionService struct {
	interactionRepo    *repository.InteractionRepository
	friendshipRepo     *repository.FriendshipRepository
	userRepo           *repository.UserRepository
	notificationService *NotificationService
}

// NewInteractionService 创建互动服务
func NewInteractionService(
	interactionRepo *repository.InteractionRepository,
	friendshipRepo *repository.FriendshipRepository,
	userRepo *repository.UserRepository,
	notificationService *NotificationService,
) *InteractionService {
	return &InteractionService{
		interactionRepo:    interactionRepo,
		friendshipRepo:     friendshipRepo,
		userRepo:           userRepo,
		notificationService: notificationService,
	}
}

// SendInteraction 发送互动（5分钟冷却）
func (s *InteractionService) SendInteraction(ctx context.Context, senderID string, req *model.SendInteractionRequest) error {
	// 1. 检查好友关系
	areFriends, err := s.friendshipRepo.AreFriends(ctx, senderID, req.ReceiverID)
	if err != nil {
		return fmt.Errorf("检查好友关系失败: %w", err)
	}
	if !areFriends {
		return fmt.Errorf("你们还不是好友")
	}

	// 2. 检查冷却时间（5分钟）
	last, err := s.interactionRepo.GetLastBetween(ctx, senderID, req.ReceiverID)
	if err == nil && last != nil {
		elapsed := time.Since(last.CreatedAt)
		if elapsed < 5*time.Minute {
			remaining := 5*time.Minute - elapsed
			return fmt.Errorf("互动太频繁，请 %d 秒后再试", int(remaining.Seconds()))
		}
	}

	// 3. 创建互动记录
	interaction := &model.Interaction{
		ID:             uuid.New().String(),
		SenderID:       senderID,
		ReceiverID:     req.ReceiverID,
		ActionEmoji:    req.ActionEmoji,
		ActionLabel:    req.ActionLabel,
		ActionPushText: req.ActionPushText,
	}

	if err := s.interactionRepo.Create(ctx, interaction); err != nil {
		return fmt.Errorf("保存互动失败: %w", err)
	}

	// 4. 发送推送通知
	go func() {
		sender, err := s.userRepo.GetByID(context.Background(), senderID)
		if err != nil {
			fmt.Printf("[互动] 获取发送者信息失败: %v\n", err)
			return
		}

		title := fmt.Sprintf("%s %s", sender.Nickname, req.ActionEmoji)
		body := req.ActionPushText
		if err := s.notificationService.SendPushToUser(context.Background(), req.ReceiverID, title, body); err != nil {
			fmt.Printf("[互动] 推送失败: %v\n", err)
		} else {
			fmt.Printf("[互动] 推送成功 %s → %s: %s\n", senderID, req.ReceiverID, req.ActionLabel)
		}
	}()

	return nil
}

// GetTodayCounts 批量获取今日互动计数
func (s *InteractionService) GetTodayCounts(ctx context.Context, receiverIDs []string) (map[string]int, error) {
	return s.interactionRepo.GetTodayCounts(ctx, receiverIDs)
}
