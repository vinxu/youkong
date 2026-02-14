package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"youkong/internal/model"
	"youkong/internal/repository"
)

// BookingService 预约业务逻辑
type BookingService struct {
	bookingRepo         *repository.BookingRepository
	scheduleRepo        *repository.ScheduleRepository
	conversationService *ConversationService
	userRepo            *repository.UserRepository
	notificationService *NotificationService
	messageRepo         *repository.MessageRepository
}

// NewBookingService 创建预约服务
func NewBookingService(
	bookingRepo *repository.BookingRepository,
	scheduleRepo *repository.ScheduleRepository,
	conversationService *ConversationService,
	userRepo *repository.UserRepository,
	notificationService *NotificationService,
	messageRepo *repository.MessageRepository,
) *BookingService {
	return &BookingService{
		bookingRepo:         bookingRepo,
		scheduleRepo:        scheduleRepo,
		conversationService: conversationService,
		userRepo:            userRepo,
		notificationService: notificationService,
		messageRepo:         messageRepo,
	}
}

// CreateBooking 创建预约（由 V4 draft_invite confirm 后调用）
func (s *BookingService) CreateBooking(
	ctx context.Context,
	creatorID string,
	title string,
	emoji string,
	date time.Time,
	startTime string,
	endTime string,
	location string,
	participantIDs []string,
	messageID string,
) (*model.Booking, error) {
	if emoji == "" {
		emoji = "🤝"
	}

	booking := &model.Booking{
		ID:           uuid.New().String(),
		CreatorID:    creatorID,
		Title:        title,
		Emoji:        emoji,
		ScheduleDate: date,
		StartTime:    startTime,
		EndTime:      endTime,
		Location:     location,
		Status:       "pending",
		MessageID:    messageID,
	}

	if err := s.bookingRepo.Create(ctx, booking); err != nil {
		return nil, fmt.Errorf("创建预约失败: %w", err)
	}

	// 添加参与者
	for _, pid := range participantIDs {
		if err := s.bookingRepo.AddParticipant(ctx, booking.ID, pid); err != nil {
			fmt.Printf("[BookingService] 添加参与者失败 booking=%s user=%s: %v\n", booking.ID, pid, err)
		}
	}

	// 在创建者的日程中注入条目（状态=待确认）
	participantNames := s.getUserNames(ctx, participantIDs)
	s.injectScheduleItem(ctx, creatorID, date, model.ScheduleItem{
		StartTime: startTime,
		EndTime:   endTime,
		Emoji:     emoji,
		Status:    title + "（待确认）",
		BookingID: booking.ID,
		WithUsers: strings.Join(participantNames, "、"),
	})

	return booking, nil
}

// RespondToBooking 响应预约邀请
func (s *BookingService) RespondToBooking(ctx context.Context, bookingID, userID, action string) (*model.Booking, error) {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("预约不存在: %w", err)
	}

	if booking.Status == "cancelled" {
		return nil, fmt.Errorf("预约已取消")
	}

	// 验证用户是参与者
	isParticipant, err := s.bookingRepo.IsParticipant(ctx, bookingID, userID)
	if err != nil || !isParticipant {
		return nil, fmt.Errorf("你不是该预约的参与者")
	}

	// 更新参与者状态
	statusMap := map[string]string{"accept": "accepted", "decline": "declined"}
	newStatus := statusMap[action]
	if err := s.bookingRepo.UpdateParticipantStatus(ctx, bookingID, userID, newStatus); err != nil {
		return nil, fmt.Errorf("更新状态失败: %w", err)
	}

	// 更新邀请消息的 metadata.status（让双方 UI 从按钮变为已接受/已拒绝）
	s.updateInviteMessageStatus(ctx, booking.CreatorID, userID, bookingID, newStatus)

	if action == "accept" {
		// 在接受者的日程中注入条目
		creatorName := s.getUserName(ctx, booking.CreatorID)
		s.injectScheduleItem(ctx, userID, booking.ScheduleDate, model.ScheduleItem{
			StartTime: booking.StartTime,
			EndTime:   booking.EndTime,
			Emoji:     booking.Emoji,
			Status:    booking.Title,
			BookingID: booking.ID,
			WithUsers: creatorName,
		})

		// 检查是否所有参与者都已响应：任一人接受即 confirmed
		if err := s.bookingRepo.UpdateStatus(ctx, bookingID, "confirmed"); err != nil {
			fmt.Printf("[BookingService] 更新预约状态失败: %v\n", err)
		}

		// 更新创建者日程中的条目状态（显示"已约"）
		s.updateCreatorScheduleItem(ctx, booking, true)

		// 向创建者发送确认消息
		s.notifyCreator(ctx, booking, userID, "accepted")
	} else if action == "decline" {
		// 向创建者发送拒绝消息
		s.notifyCreator(ctx, booking, userID, "declined")

		// 检查是否所有人都拒绝了
		s.checkAllDeclined(ctx, booking)
	}

	// 重新获取最新状态
	booking, _ = s.bookingRepo.GetByID(ctx, bookingID)
	return booking, nil
}

// UpdateMessageID 回填 Booking 关联的消息 ID
func (s *BookingService) UpdateMessageID(ctx context.Context, bookingID, messageID string) {
	if err := s.bookingRepo.UpdateMessageID(ctx, bookingID, messageID); err != nil {
		fmt.Printf("[BookingService] 回填 message_id 失败 booking=%s: %v\n", bookingID, err)
	}
}

// CancelBooking 取消预约
func (s *BookingService) CancelBooking(ctx context.Context, bookingID, userID string) error {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("预约不存在: %w", err)
	}

	if booking.CreatorID != userID {
		return fmt.Errorf("只有创建者可以取消预约")
	}

	if err := s.bookingRepo.UpdateStatus(ctx, bookingID, "cancelled"); err != nil {
		return fmt.Errorf("取消预约失败: %w", err)
	}

	// 从所有参与方的日程中移除 booking 条目
	s.removeBookingFromSchedules(ctx, booking)

	return nil
}

// GetBooking 获取预约详情
func (s *BookingService) GetBooking(ctx context.Context, bookingID, userID string) (*model.BookingDetail, error) {
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("预约不存在: %w", err)
	}

	// 验证用户有权查看
	isParticipant, err := s.bookingRepo.IsParticipant(ctx, bookingID, userID)
	if err != nil || !isParticipant {
		return nil, fmt.Errorf("无权查看该预约")
	}

	// 获取创建者名字
	creatorName := s.getUserName(ctx, booking.CreatorID)

	// 获取参与者信息
	participants, err := s.bookingRepo.GetParticipants(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	var pInfos []model.BookingParticipantInfo
	for _, p := range participants {
		name := s.getUserName(ctx, p.UserID)
		avatar := ""
		if user, err := s.userRepo.GetByID(ctx, p.UserID); err == nil {
			avatar = user.Avatar
		}
		pInfos = append(pInfos, model.BookingParticipantInfo{
			UserID:      p.UserID,
			Nickname:    name,
			Avatar:      avatar,
			Status:      p.Status,
			RespondedAt: p.RespondedAt,
		})
	}

	return &model.BookingDetail{
		Booking:      *booking,
		CreatorName:  creatorName,
		Participants: pInfos,
	}, nil
}

// GetUserBookings 获取用户的预约列表
func (s *BookingService) GetUserBookings(ctx context.Context, userID string, limit int) ([]*model.Booking, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.bookingRepo.GetByUserID(ctx, userID, limit)
}

// GetUserBookingsForDate 获取用户某天的预约
func (s *BookingService) GetUserBookingsForDate(ctx context.Context, userID string, date time.Time) ([]*model.Booking, error) {
	return s.bookingRepo.GetBookingsByDateRange(ctx, userID, date, date)
}

// injectScheduleItem 在用户的日程中注入一条 booking 条目
func (s *BookingService) injectScheduleItem(ctx context.Context, userID string, date time.Time, item model.ScheduleItem) {
	schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, userID, date)
	if err != nil {
		// 没有当天的时刻表，创建一个
		schedule = &model.StatusSchedule{
			UserID:       userID,
			ScheduleDate: date,
			Items:        model.ScheduleItems{item},
			CurrentIndex: 0,
			Status:       model.ScheduleStatusActive,
			Visibility:   model.VisibilityAllFriends,
		}
		if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
			fmt.Printf("[BookingService] 创建日程失败: %v\n", err)
		}
		return
	}

	// 有现有时刻表，插入到合适的位置（按时间排序）
	items := []model.ScheduleItem(schedule.Items)
	inserted := false
	for i, existing := range items {
		if item.StartTime < existing.StartTime {
			// 插入到当前位置
			items = append(items[:i+1], items[i:]...)
			items[i] = item
			inserted = true
			break
		}
	}
	if !inserted {
		items = append(items, item)
	}

	schedule.Items = model.ScheduleItems(items)
	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		fmt.Printf("[BookingService] 更新日程失败: %v\n", err)
	}
}

// updateCreatorScheduleItem 更新创建者日程中的 booking 条目
func (s *BookingService) updateCreatorScheduleItem(ctx context.Context, booking *model.Booking, accepted bool) {
	schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, booking.CreatorID, booking.ScheduleDate)
	if err != nil {
		return
	}

	for i, item := range schedule.Items {
		if item.BookingID == booking.ID {
			if accepted {
				schedule.Items[i].Status = booking.Title + "（已约）"
			} else {
				schedule.Items[i].Status = booking.Title + "（已拒绝）"
			}
			break
		}
	}

	s.scheduleRepo.Update(ctx, schedule)
}

// removeBookingFromSchedules 从所有参与方的日程中移除 booking 条目
func (s *BookingService) removeBookingFromSchedules(ctx context.Context, booking *model.Booking) {
	// 收集所有相关用户
	userIDs := []string{booking.CreatorID}
	participants, _ := s.bookingRepo.GetParticipants(ctx, booking.ID)
	for _, p := range participants {
		userIDs = append(userIDs, p.UserID)
	}

	for _, uid := range userIDs {
		schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, uid, booking.ScheduleDate)
		if err != nil {
			continue
		}

		var filtered []model.ScheduleItem
		for _, item := range schedule.Items {
			if item.BookingID != booking.ID {
				filtered = append(filtered, item)
			}
		}

		if len(filtered) != len(schedule.Items) {
			schedule.Items = model.ScheduleItems(filtered)
			s.scheduleRepo.Update(ctx, schedule)
		}
	}
}

// notifyCreator 通知创建者参与者的响应
func (s *BookingService) notifyCreator(ctx context.Context, booking *model.Booking, respondUserID, action string) {
	if s.conversationService == nil {
		return
	}

	respondName := s.getUserName(ctx, respondUserID)
	var content string
	if action == "accepted" {
		content = fmt.Sprintf("%s 接受了你的邀请「%s」", respondName, booking.Title)
	} else {
		content = fmt.Sprintf("%s 拒绝了你的邀请「%s」", respondName, booking.Title)
	}

	conv, err := s.conversationService.GetOrCreateConversation(ctx, booking.CreatorID, respondUserID)
	if err != nil {
		return
	}

	// metadata 中携带 action，前端通过 metadata.action 判断接受/拒绝
	metadata := fmt.Sprintf(`{"action":"%s","booking_id":"%s"}`, action, booking.ID)

	s.conversationService.SendMessage(ctx, conv.ID, respondUserID, &SendMessageRequest{
		Type:     model.MessageTypeConfirmResponse,
		Content:  content,
		Metadata: json.RawMessage(metadata),
	})
}

// updateInviteMessageStatus 更新邀请消息的 metadata.status
func (s *BookingService) updateInviteMessageStatus(ctx context.Context, creatorID, respondUserID, bookingID, status string) {
	if s.messageRepo == nil || s.conversationService == nil {
		return
	}

	conv, err := s.conversationService.GetOrCreateConversation(ctx, creatorID, respondUserID)
	if err != nil {
		fmt.Printf("[BookingService] 获取会话失败: %v\n", err)
		return
	}

	if err := s.messageRepo.UpdateMessageMetadataStatus(ctx, conv.ID, bookingID, status); err != nil {
		fmt.Printf("[BookingService] 更新邀请消息状态失败: %v\n", err)
	}
}

// checkAllDeclined 检查是否所有参与者都拒绝了
func (s *BookingService) checkAllDeclined(ctx context.Context, booking *model.Booking) {
	participants, err := s.bookingRepo.GetParticipants(ctx, booking.ID)
	if err != nil {
		return
	}

	allDeclined := true
	for _, p := range participants {
		if p.Status != "declined" {
			allDeclined = false
			break
		}
	}

	if allDeclined {
		s.bookingRepo.UpdateStatus(ctx, booking.ID, "cancelled")
		// 更新创建者日程显示"已拒绝"（而非直接移除）
		s.updateCreatorScheduleItem(ctx, booking, false)
	}
}

// getUserName 获取用户名字
func (s *BookingService) getUserName(ctx context.Context, userID string) string {
	if s.userRepo == nil {
		return "未知"
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "未知"
	}
	return user.Nickname
}

// getUserNames 批量获取用户名字
func (s *BookingService) getUserNames(ctx context.Context, userIDs []string) []string {
	names := make([]string, len(userIDs))
	for i, id := range userIDs {
		names[i] = s.getUserName(ctx, id)
	}
	return names
}

// GetPendingBookingsForUser 获取用户的待处理邀请（被邀请方视角）
func (s *BookingService) GetPendingBookingsForUser(ctx context.Context, userID string) ([]*repository.BookingWithCreator, error) {
	return s.bookingRepo.GetPendingForUser(ctx, userID)
}

// FilterScheduleForViewer 根据查看者过滤时刻表中的 booking 信息
// 参与方可以看到完整信息，非参与方只能看到通用描述
func (s *BookingService) FilterScheduleForViewer(ctx context.Context, items []model.ScheduleItem, viewerID string) []model.ScheduleItem {
	filtered := make([]model.ScheduleItem, len(items))
	copy(filtered, items)

	for i, item := range filtered {
		if item.BookingID != "" {
			isParticipant, err := s.bookingRepo.IsParticipant(ctx, item.BookingID, viewerID)
			if err != nil || !isParticipant {
				// 非参与方：隐藏敏感信息
				filtered[i].WithUsers = ""
				filtered[i].BookingID = ""
				filtered[i].Status = "有约"
			}
		}
	}
	return filtered
}
