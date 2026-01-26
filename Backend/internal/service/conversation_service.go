package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"youkong/internal/model"
	"youkong/internal/repository"
)

type ConversationService struct {
	messageRepo *repository.MessageRepository
	userRepo    *repository.UserRepository
}

func NewConversationService(messageRepo *repository.MessageRepository, userRepo *repository.UserRepository) *ConversationService {
	return &ConversationService{
		messageRepo: messageRepo,
		userRepo:    userRepo,
	}
}

func (s *ConversationService) GetConversations(ctx context.Context, userID string) ([]*model.ConversationResponse, error) {
	convs, err := s.messageRepo.GetConversationsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取会话列表失败: %w", err)
	}

	result := make([]*model.ConversationResponse, 0, len(convs))
	for _, conv := range convs {
		// 确定对方用户ID
		partnerID := conv.User1ID
		if partnerID == userID {
			partnerID = conv.User2ID
		}

		partner, err := s.userRepo.GetByID(ctx, partnerID)
		if err != nil || partner == nil {
			continue
		}

		// 获取最后一条消息
		lastMessage, _ := s.messageRepo.GetLastMessage(ctx, conv.ID)
		var lastMsgResp *model.MessageResponse
		if lastMessage != nil {
			sender, _ := s.userRepo.GetByID(ctx, lastMessage.SenderID)
			if sender != nil {
				lastMsgResp = lastMessage.ToResponse(sender)
			}
		}

		// 获取未读数
		unreadCount, _ := s.messageRepo.GetUnreadCount(ctx, conv.ID, userID)

		result = append(result, &model.ConversationResponse{
			ID:          conv.ID,
			Partner:     *partner.ToProfile(),
			LastMessage: lastMsgResp,
			UnreadCount: unreadCount,
			CreatedAt:   conv.CreatedAt,
		})
	}

	return result, nil
}

func (s *ConversationService) GetOrCreateConversation(ctx context.Context, userID, partnerID string) (*model.Conversation, error) {
	conv, err := s.messageRepo.GetConversationByUsers(ctx, userID, partnerID)
	if err != nil {
		return nil, fmt.Errorf("查询会话失败: %w", err)
	}

	if conv != nil {
		return conv, nil
	}

	// 确保user1_id < user2_id以保持一致性
	user1, user2 := userID, partnerID
	if user1 > user2 {
		user1, user2 = user2, user1
	}

	conv = &model.Conversation{
		ID:        uuid.New().String(),
		User1ID:   user1,
		User2ID:   user2,
		CreatedAt: time.Now(),
	}

	if err := s.messageRepo.CreateConversation(ctx, conv); err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}

	return conv, nil
}

func (s *ConversationService) GetOrCreateConversationWithResponse(ctx context.Context, userID, partnerID string) (*model.ConversationResponse, error) {
	conv, err := s.GetOrCreateConversation(ctx, userID, partnerID)
	if err != nil {
		return nil, err
	}

	partner, err := s.userRepo.GetByID(ctx, partnerID)
	if err != nil || partner == nil {
		return nil, fmt.Errorf("获取对方用户信息失败")
	}

	return &model.ConversationResponse{
		ID:          conv.ID,
		Partner:     *partner.ToProfile(),
		LastMessage: nil,
		UnreadCount: 0,
		CreatedAt:   conv.CreatedAt,
	}, nil
}

func (s *ConversationService) GetMessages(ctx context.Context, conversationID, userID string, limit, offset int) ([]*model.MessageResponse, error) {
	// 验证用户是会话参与者
	conv, err := s.messageRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}
	if conv == nil {
		return nil, fmt.Errorf("会话不存在")
	}
	if conv.User1ID != userID && conv.User2ID != userID {
		return nil, fmt.Errorf("无权限访问此会话")
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	messages, err := s.messageRepo.GetMessagesByConversationID(ctx, conversationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("获取消息失败: %w", err)
	}

	// 标记为已读
	_ = s.messageRepo.MarkAllAsRead(ctx, conversationID, userID)

	// 收集发送者ID
	senderIDs := make([]string, 0, len(messages))
	for _, m := range messages {
		senderIDs = append(senderIDs, m.SenderID)
	}

	senders, _ := s.userRepo.GetByIDs(ctx, senderIDs)
	senderMap := make(map[string]*model.User)
	for _, u := range senders {
		senderMap[u.ID] = u
	}

	result := make([]*model.MessageResponse, 0, len(messages))
	for _, m := range messages {
		sender := senderMap[m.SenderID]
		if sender == nil {
			continue
		}
		result = append(result, m.ToResponse(sender))
	}

	return result, nil
}

type SendMessageRequest struct {
	Type     model.MessageType `json:"type" binding:"required"`
	Content  string            `json:"content"`
	Metadata json.RawMessage   `json:"metadata"`
}

func (s *ConversationService) SendMessage(ctx context.Context, conversationID, senderID string, req *SendMessageRequest) (*model.MessageResponse, error) {
	// 验证用户是会话参与者
	conv, err := s.messageRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}
	if conv == nil {
		return nil, fmt.Errorf("会话不存在")
	}
	if conv.User1ID != senderID && conv.User2ID != senderID {
		return nil, fmt.Errorf("无权限发送消息")
	}

	msg := &model.Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		SenderID:       senderID,
		Type:           req.Type,
		Metadata:       req.Metadata,
		CreatedAt:      time.Now(),
	}

	if req.Content != "" {
		msg.Content = sql.NullString{String: req.Content, Valid: true}
	}

	if err := s.messageRepo.CreateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("发送消息失败: %w", err)
	}

	// 更新会话最后消息时间
	_ = s.messageRepo.UpdateConversationLastMessage(ctx, conversationID)

	sender, _ := s.userRepo.GetByID(ctx, senderID)
	return msg.ToResponse(sender), nil
}
