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

// SendInteraction 发送互动（+1 使用同状态仅一次规则，其他用 5 分钟冷却）
func (s *InteractionService) SendInteraction(ctx context.Context, senderID string, req *model.SendInteractionRequest) error {
	// 1. 检查好友关系
	areFriends, err := s.friendshipRepo.AreFriends(ctx, senderID, req.ReceiverID)
	if err != nil {
		return fmt.Errorf("检查好友关系失败: %w", err)
	}
	if !areFriends {
		return fmt.Errorf("你们还不是好友")
	}

	// 2. 冷却/去重检查
	if req.ActionLabel == "+1" {
		// +1 规则：对同一好友的同一状态只能 +1 一次
		// 通过检查 updatedAt 实现：如果最近的 +1 时间 >= 好友状态更新时间，则拒绝
		latestMap, err := s.interactionRepo.GetLatestPlusOneMap(ctx, senderID, []string{req.ReceiverID})
		if err == nil {
			if latestPlusOne, ok := latestMap[req.ReceiverID]; ok {
				// 如果最近 +1 时间在 1 分钟内，大概率是同一状态期间的重复请求
				if time.Since(latestPlusOne) < 1*time.Minute {
					return fmt.Errorf("你已经对这个状态 +1 过了")
				}
			}
		}
	} else {
		// 其他互动：5 分钟冷却
		last, err := s.interactionRepo.GetLastBetween(ctx, senderID, req.ReceiverID)
		if err == nil && last != nil {
			elapsed := time.Since(last.CreatedAt)
			if elapsed < 5*time.Minute {
				remaining := 5*time.Minute - elapsed
				return fmt.Errorf("互动太频繁，请 %d 秒后再试", int(remaining.Seconds()))
			}
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
		data := map[string]string{
			"type":         "interaction",
			"sender_id":    senderID,
			"action_emoji": req.ActionEmoji,
			"action_label": req.ActionLabel,
		}
		if err := s.notificationService.SendInteractionPush(context.Background(), req.ReceiverID, title, body, data); err != nil {
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

// GetLatestPlusOneMap 批量获取某用户对每个好友最近 +1 的时间
func (s *InteractionService) GetLatestPlusOneMap(ctx context.Context, senderID string, receiverIDs []string) (map[string]time.Time, error) {
	return s.interactionRepo.GetLatestPlusOneMap(ctx, senderID, receiverIDs)
}

// GetPlusOnesSince 批量获取每人在指定时间之后收到的所有 +1 互动记录
func (s *InteractionService) GetPlusOnesSince(ctx context.Context, receiverIDs []string, since time.Time) (map[string][]model.Interaction, error) {
	return s.interactionRepo.GetPlusOnesSince(ctx, receiverIDs, since)
}

// EnrichScheduleWithPlusOnes 为时刻表条目填充 +1 用户列表
func (s *InteractionService) EnrichScheduleWithPlusOnes(ctx context.Context, userID string, items []model.ScheduleItem, dateStr string) {
	if len(items) == 0 {
		return
	}

	// 获取该用户当天收到的所有 +1
	plusOnes, err := s.interactionRepo.GetTodayPlusOnes(ctx, userID, dateStr)
	if err != nil || len(plusOnes) == 0 {
		return
	}

	// 收集所有发送者 ID 用于批量查询昵称
	senderIDs := make([]string, 0, len(plusOnes))
	senderIDSet := make(map[string]bool)
	for _, p := range plusOnes {
		if !senderIDSet[p.SenderID] {
			senderIDSet[p.SenderID] = true
			senderIDs = append(senderIDs, p.SenderID)
		}
	}

	// 批量获取用户信息
	users, err := s.userRepo.GetByIDs(ctx, senderIDs)
	if err != nil {
		return
	}
	nicknameMap := make(map[string]string)
	for _, u := range users {
		nicknameMap[u.ID] = u.Nickname
	}

	// 将 +1 分配到对应时段
	for i := range items {
		var usersForSlot []model.PlusOneUser
		for _, p := range plusOnes {
			createdTime := p.CreatedAt.Format("15:04")
			if createdTime >= items[i].StartTime && createdTime < items[i].EndTime {
				nickname := nicknameMap[p.SenderID]
				if nickname == "" {
					nickname = "用户"
				}
				usersForSlot = append(usersForSlot, model.PlusOneUser{
					UserID:   p.SenderID,
					Nickname: nickname,
				})
			}
		}
		if len(usersForSlot) > 0 {
			// 去重（同一用户只显示一次）
			seen := make(map[string]bool)
			var deduped []model.PlusOneUser
			for _, u := range usersForSlot {
				if !seen[u.UserID] {
					seen[u.UserID] = true
					deduped = append(deduped, u)
				}
			}
			items[i].PlusOnes = deduped
		}
	}
}
