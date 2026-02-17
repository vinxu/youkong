package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"youkong/internal/model"
	"youkong/internal/pkg/ws"
	"youkong/internal/repository"
)

// CardRoomService 卡片房间服务
type CardRoomService struct {
	cardRoomRepo        *repository.CardRoomRepository
	friendshipRepo      *repository.FriendshipRepository
	userRepo            *repository.UserRepository
	notificationService *NotificationService
	wsManager           *ws.Manager
}

// NewCardRoomService 创建卡片房间服务
func NewCardRoomService(
	cardRoomRepo *repository.CardRoomRepository,
	friendshipRepo *repository.FriendshipRepository,
	userRepo *repository.UserRepository,
	notificationService *NotificationService,
	wsManager *ws.Manager,
) *CardRoomService {
	return &CardRoomService{
		cardRoomRepo:        cardRoomRepo,
		friendshipRepo:      friendshipRepo,
		userRepo:            userRepo,
		notificationService: notificationService,
		wsManager:           wsManager,
	}
}

// JoinCard 加入卡片房间
func (s *CardRoomService) JoinCard(ctx context.Context, userID string, req *model.JoinCardRequest) (*model.JoinCardResponse, error) {
	// 1. 不能加入自己（但 owner 会被自动加入房间）
	if userID == req.OwnerID {
		return nil, fmt.Errorf("不能加入自己的卡片")
	}

	// 2. 检查好友关系
	areFriends, err := s.friendshipRepo.AreFriends(ctx, userID, req.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("检查好友关系失败: %w", err)
	}
	if !areFriends {
		return nil, fmt.Errorf("你们还不是好友")
	}

	// 3. 获取或创建房间（状态变了则清除旧房间建新的）
	room, err := s.cardRoomRepo.GetTodayRoomByOwner(ctx, req.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("获取房间失败: %w", err)
	}

	// 如果房间存在但状态已变（emoji 或 status_text 不同），清除旧房间
	if room != nil && req.Emoji != "" && (room.Emoji != req.Emoji || room.StatusText != req.StatusText) {
		_ = s.cardRoomRepo.DeleteRoom(ctx, room.ID)
		room = nil
	}

	if room == nil {
		// 创建房间，记录当前状态
		room = &model.CardRoom{
			ID:         uuid.New().String(),
			OwnerID:    req.OwnerID,
			Emoji:      req.Emoji,
			StatusText: req.StatusText,
			RoomDate:   time.Now().Format("2006-01-02"),
			CreatedAt:  time.Now(),
		}
		if err := s.cardRoomRepo.CreateRoom(ctx, room); err != nil {
			// 并发创建，重新获取
			if isDuplicateKeyError(err) {
				room, err = s.cardRoomRepo.GetTodayRoomByOwner(ctx, req.OwnerID)
				if err != nil || room == nil {
					return nil, fmt.Errorf("创建房间失败")
				}
			} else {
				return nil, fmt.Errorf("创建房间失败: %w", err)
			}
		}
		// Owner 自动加入
		_ = s.cardRoomRepo.AddMember(ctx, room.ID, req.OwnerID)
	}

	// 4. 加入房间（幂等）
	if err := s.cardRoomRepo.AddMember(ctx, room.ID, userID); err != nil {
		return nil, fmt.Errorf("加入房间失败: %w", err)
	}

	// 5. 获取成员列表
	members, err := s.cardRoomRepo.GetMembers(ctx, room.ID)
	if err != nil {
		return nil, fmt.Errorf("获取成员列表失败: %w", err)
	}

	// 6. 推送通知给 owner
	go func() {
		sender, err := s.userRepo.GetByID(context.Background(), userID)
		if err != nil || sender == nil {
			return
		}
		title := fmt.Sprintf("%s 加入了你的状态", sender.Nickname)
		body := "点击查看群聊"
		if s.notificationService != nil {
			_ = s.notificationService.SendPushToUser(context.Background(), req.OwnerID, title, body)
		}

		// WebSocket 通知所有房间成员有新成员加入
		if s.wsManager != nil {
			memberIDs := make([]string, 0, len(members))
			for _, m := range members {
				memberIDs = append(memberIDs, m.UserID)
			}
			s.wsManager.Broadcast(memberIDs, ws.WSMessage{
				Type: "ROOM_MEMBER_JOINED",
				Data: map[string]interface{}{
					"room_id": room.ID,
					"user":    sender.ToProfile(),
				},
			})
		}
	}()

	return &model.JoinCardResponse{
		RoomID:  room.ID,
		Members: members,
	}, nil
}

// GetRoom 获取房间详情
func (s *CardRoomService) GetRoom(ctx context.Context, userID, roomID string) (*model.CardRoomDetail, error) {
	room, err := s.cardRoomRepo.GetRoomByID(ctx, roomID)
	if err != nil || room == nil {
		return nil, fmt.Errorf("房间不存在")
	}

	// 检查是否为成员
	isMember, err := s.cardRoomRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查成员失败: %w", err)
	}
	if !isMember {
		return nil, fmt.Errorf("你不是该房间的成员")
	}

	members, err := s.cardRoomRepo.GetMembers(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("获取成员失败: %w", err)
	}

	return &model.CardRoomDetail{
		ID:         room.ID,
		OwnerID:    room.OwnerID,
		Emoji:      room.Emoji,
		StatusText: room.StatusText,
		RoomDate:   room.RoomDate,
		Members:    members,
		CreatedAt:  room.CreatedAt,
	}, nil
}

// RoomMessageResponse 房间消息响应
type RoomMessageResponse struct {
	ID        string             `json:"id"`
	Sender    *model.UserProfile `json:"sender"`
	Type      string             `json:"type"`
	Content   string             `json:"content"`
	CreatedAt time.Time          `json:"createdAt"`
	IsRead    bool               `json:"isRead"`
}

// GetRoomMessages 获取房间消息
func (s *CardRoomService) GetRoomMessages(ctx context.Context, userID, roomID string, limit, offset int) ([]*RoomMessageResponse, error) {
	// 检查是否为成员
	isMember, err := s.cardRoomRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查成员失败: %w", err)
	}
	if !isMember {
		return nil, fmt.Errorf("你不是该房间的成员")
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	messages, err := s.cardRoomRepo.GetRoomMessages(ctx, roomID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("获取消息失败: %w", err)
	}

	// 批量获取发送者信息
	senderIDMap := make(map[string]bool)
	for _, m := range messages {
		senderIDMap[m.SenderID] = true
	}
	senderIDs := make([]string, 0, len(senderIDMap))
	for id := range senderIDMap {
		senderIDs = append(senderIDs, id)
	}

	senders, err := s.userRepo.GetByIDs(ctx, senderIDs)
	if err != nil {
		return nil, fmt.Errorf("查询发送者信息失败: %w", err)
	}
	senderMap := make(map[string]*model.User)
	for _, u := range senders {
		senderMap[u.ID] = u
	}

	result := make([]*RoomMessageResponse, 0, len(messages))
	for _, m := range messages {
		sender := senderMap[m.SenderID]
		if sender == nil {
			sender = &model.User{ID: m.SenderID, Nickname: "未知用户"}
		}
		result = append(result, &RoomMessageResponse{
			ID:        m.ID,
			Sender:    sender.ToProfile(),
			Type:      "text",
			Content:   m.Content,
			CreatedAt: m.CreatedAt,
			IsRead:    true,
		})
	}

	return result, nil
}

// SendRoomMessage 发送房间消息
func (s *CardRoomService) SendRoomMessage(ctx context.Context, userID, roomID string, req *model.SendRoomMessageRequest) (*RoomMessageResponse, error) {
	// 检查是否为成员
	isMember, err := s.cardRoomRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查成员失败: %w", err)
	}
	if !isMember {
		return nil, fmt.Errorf("你不是该房间的成员")
	}

	// 创建消息
	msgID := uuid.New().String()
	now := time.Now()
	if err := s.cardRoomRepo.CreateRoomMessage(ctx, msgID, roomID, userID, req.Content, now); err != nil {
		return nil, fmt.Errorf("发送消息失败: %w", err)
	}

	sender, _ := s.userRepo.GetByID(ctx, userID)
	if sender == nil {
		sender = &model.User{ID: userID, Nickname: "未知用户"}
	}

	msgResp := &RoomMessageResponse{
		ID:        msgID,
		Sender:    sender.ToProfile(),
		Type:      "text",
		Content:   req.Content,
		CreatedAt: now,
		IsRead:    true,
	}

	// WebSocket 广播给所有成员
	go func() {
		members, err := s.cardRoomRepo.GetMembers(context.Background(), roomID)
		if err != nil {
			return
		}
		memberIDs := make([]string, 0, len(members))
		for _, m := range members {
			memberIDs = append(memberIDs, m.UserID)
		}

		if s.wsManager != nil {
			s.wsManager.Broadcast(memberIDs, ws.WSMessage{
				Type: ws.MessageTypeNewMessage,
				Data: ws.NewMessageData{
					ConversationID: roomID,
					Message:        msgResp,
				},
			})
		}

		// 推送给不在线的成员
		if s.notificationService != nil {
			for _, memberID := range memberIDs {
				if memberID == userID {
					continue
				}
				title := fmt.Sprintf("%s 在群聊中说", sender.Nickname)
				body := req.Content
				_ = s.notificationService.SendPushToUser(context.Background(), memberID, title, body)
			}
		}
	}()

	return msgResp, nil
}
