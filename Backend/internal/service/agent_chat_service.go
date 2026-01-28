package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"youkong/internal/model"
	"youkong/internal/pkg/llm"
	"youkong/internal/pkg/ws"
	"youkong/internal/repository"
)

// AgentChatService Agent 聊天服务
type AgentChatService struct {
	contextRepo         *repository.ContextRepository
	messageRepo         *repository.MessageRepository
	personaRepo         *repository.PersonaRepository
	relRepo             *repository.RelationshipRepository
	userRepo            *repository.UserRepository
	memoryRepo          *repository.MemoryRepository
	chatSession         *llm.ChatSession
	wsManager           *ws.Manager
	relService          *RelationshipService
	notificationService *NotificationService
}

// NewAgentChatService 创建 Agent 聊天服务
func NewAgentChatService(
	contextRepo *repository.ContextRepository,
	messageRepo *repository.MessageRepository,
	personaRepo *repository.PersonaRepository,
	relRepo *repository.RelationshipRepository,
	userRepo *repository.UserRepository,
	memoryRepo *repository.MemoryRepository,
	llmClient *llm.OpenRouterClient,
	wsManager *ws.Manager,
	relService *RelationshipService,
	notificationService *NotificationService,
) *AgentChatService {
	var session *llm.ChatSession
	if llmClient != nil {
		session = llm.NewChatSession(llmClient)
	}
	return &AgentChatService{
		contextRepo:         contextRepo,
		messageRepo:         messageRepo,
		personaRepo:         personaRepo,
		relRepo:             relRepo,
		userRepo:            userRepo,
		memoryRepo:          memoryRepo,
		chatSession:         session,
		wsManager:           wsManager,
		relService:          relService,
		notificationService: notificationService,
	}
}

// GenerateReply 生成 Agent 回复
func (s *AgentChatService) GenerateReply(ctx context.Context, conversationID, userID string) (*model.AgentReplyResponse, error) {
	if s.chatSession == nil {
		log.Printf("[AgentChat] chatSession 为空，LLM 客户端未初始化")
		return nil, fmt.Errorf("你的元婴罢工了")
	}

	// 获取会话信息
	conv, err := s.messageRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}
	if conv == nil {
		return nil, fmt.Errorf("会话不存在")
	}

	// 验证用户是会话参与者
	if conv.User1ID != userID && conv.User2ID != userID {
		return nil, fmt.Errorf("无权限访问此会话")
	}

	// 确定对方用户ID
	partnerID := conv.User1ID
	if partnerID == userID {
		partnerID = conv.User2ID
	}

	// 获取用户和对方信息
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("获取用户信息失败")
	}

	partner, err := s.userRepo.GetByID(ctx, partnerID)
	if err != nil || partner == nil {
		return nil, fmt.Errorf("获取对方用户信息失败")
	}

	// 获取或创建 LLM 上下文
	convCtx, err := s.getOrCreateContext(ctx, conversationID, user, partner)
	if err != nil {
		return nil, fmt.Errorf("获取上下文失败: %w", err)
	}

	// 每次生成回复时更新 system prompt（确保时间和状态是最新的）
	if err := s.updateSystemPrompt(ctx, convCtx, user, partner); err != nil {
		log.Printf("[AgentChat] 更新 system prompt 失败: %v", err)
	}

	// 同步所有新消息到上下文（正确分配角色）
	if err := s.syncMessages(ctx, convCtx, conversationID, userID); err != nil {
		// 同步失败不影响主流程，记录日志
		log.Printf("[AgentChat] 同步消息失败: %v", err)
	}

	// 获取 LLM 消息
	messages, err := convCtx.GetMessages()
	if err != nil {
		return nil, fmt.Errorf("解析上下文失败: %w", err)
	}

	// 分析对话状态并更新 system prompt
	convState := s.analyzeConversationState(messages)
	if err := s.updateSystemPromptWithState(ctx, convCtx, user, partner, convState); err != nil {
		log.Printf("[AgentChat] 更新带状态的 system prompt 失败: %v", err)
	}

	// 调用 LLM 生成回复
	reply, err := s.chatSession.GenerateReply(ctx, toLLMMessages(messages))
	if err != nil {
		log.Printf("[AgentChat] LLM 调用失败: %v", err)
		return nil, fmt.Errorf("你的元婴罢工了")
	}

	// 追加回复到上下文
	if err := convCtx.AddMessage("assistant", reply); err != nil {
		return nil, fmt.Errorf("保存上下文失败: %w", err)
	}

	// 保存上下文
	if err := s.contextRepo.Update(ctx, convCtx); err != nil {
		// 上下文保存失败不影响主流程
	}

	// 保存消息到数据库
	msg := &model.Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		SenderID:       userID,
		Type:           model.MessageTypeText,
		Content:        sql.NullString{String: reply, Valid: true},
		CreatedAt:      time.Now(),
	}

	if err := s.messageRepo.CreateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("保存消息失败: %w", err)
	}

	// 更新会话最后消息时间
	s.messageRepo.UpdateConversationLastMessage(ctx, conversationID)

	// 更新上下文的最后同步消息ID
	s.contextRepo.UpdateLastSyncMsgID(ctx, conversationID, msg.ID)

	// 构建响应
	response := &model.AgentReplyResponse{
		ID:        msg.ID,
		Sender:    *user.ToProfile(),
		Type:      model.MessageTypeText,
		Content:   reply,
		CreatedAt: msg.CreatedAt,
	}

	// 通过 WebSocket 推送给双方
	if s.wsManager != nil {
		msgResp := msg.ToResponse(user)
		// 推送给发送者
		s.wsManager.SendNewMessage(userID, conversationID, msgResp)
		// 推送给对方
		s.wsManager.SendNewMessage(partnerID, conversationID, msgResp)
	}

	// 发送远程推送给对方（如果对方不在线）
	if s.notificationService != nil {
		go func() {
			s.notificationService.NotifyNewMessage(context.Background(), partnerID, msg, user)
		}()
	}

	// 异步触发关系学习
	go func() {
		if s.relService != nil {
			s.relService.LearnFromConversation(context.Background(), conversationID, userID, partnerID)
		}
	}()

	// 检查是否需要总结
	if convCtx.NeedsSummary(4000) {
		go s.summarizeAndReset(context.Background(), convCtx, user, partner)
	}

	return response, nil
}

// getOrCreateContext 获取或创建 LLM 上下文
func (s *AgentChatService) getOrCreateContext(ctx context.Context, conversationID string, user, partner *model.User) (*model.ConversationContext, error) {
	convCtx, err := s.contextRepo.GetByConversationID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if convCtx != nil {
		return convCtx, nil
	}

	// 创建新上下文
	systemPrompt := s.buildSystemPrompt(ctx, user, partner)

	initialMessages := []model.LLMMessage{
		{Role: "system", Content: systemPrompt},
	}

	messagesJSON, err := json.Marshal(initialMessages)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	convCtx = &model.ConversationContext{
		ConversationID: conversationID,
		Messages:       string(messagesJSON),
		TokenCount:     len(string(messagesJSON)) / 4,
		Summary:        "",
		LastSyncMsgID:  "",
		UpdatedAt:      now,
		CreatedAt:      now,
	}

	if err := s.contextRepo.Create(ctx, convCtx); err != nil {
		return nil, err
	}

	return convCtx, nil
}

// updateSystemPrompt 更新上下文中的 system prompt（确保时间和状态是最新的）
func (s *AgentChatService) updateSystemPrompt(ctx context.Context, convCtx *model.ConversationContext, user, partner *model.User) error {
	// 构建新的 system prompt
	newSystemPrompt := s.buildSystemPrompt(ctx, user, partner)

	// 获取当前消息列表
	messages, err := convCtx.GetMessages()
	if err != nil {
		return err
	}

	// 更新第一个 system 消息
	if len(messages) > 0 && messages[0].Role == "system" {
		messages[0].Content = newSystemPrompt
	} else {
		// 如果没有 system 消息，插入一个
		messages = append([]model.LLMMessage{{Role: "system", Content: newSystemPrompt}}, messages...)
	}

	// 保存更新后的消息
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	convCtx.Messages = string(messagesJSON)
	convCtx.TokenCount = len(convCtx.Messages) / 4
	convCtx.UpdatedAt = time.Now()

	return s.contextRepo.Update(ctx, convCtx)
}

// buildSystemPrompt 构建 System Prompt
func (s *AgentChatService) buildSystemPrompt(ctx context.Context, user, partner *model.User) string {
	// 获取用户人设
	persona, _ := s.personaRepo.GetOrCreate(ctx, user.ID)

	// 获取关系画像
	rel, _ := s.relRepo.GetOrCreate(ctx, user.ID, partner.ID)

	// 获取对方状态（从分析缓存）
	var partnerStatus *llm.PartnerStatus
	if s.memoryRepo != nil {
		if cache, err := s.memoryRepo.GetAnalysisCache(ctx, partner.ID); err == nil && cache != nil {
			partnerStatus = &llm.PartnerStatus{
				Emoji:       cache.LifeStatus.Emoji,
				Label:       cache.LifeStatus.Label,
				Probability: cache.Availability.Probability,
			}
		}
	}

	data := &llm.PromptData{
		MyName:        user.Nickname,
		MyPersona:     persona,
		PartnerName:   partner.Nickname,
		PartnerStatus: partnerStatus,
		Relationship:  rel,
		CurrentTime:   time.Now(),
	}

	return s.chatSession.BuildSystemPrompt(data)
}

// syncMessages 同步所有未同步的消息到上下文，正确分配角色
// 从 LLM 视角：
//   - "assistant" = 我（当前用户）说的话（包括 Agent 代说的）
//   - "user" = 对方说的话（包括对方的 Agent 代说的）
func (s *AgentChatService) syncMessages(ctx context.Context, convCtx *model.ConversationContext, conversationID, userID string) error {
	// 获取最后同步消息之后的新消息
	newMessages, err := s.messageRepo.GetMessagesAfterID(ctx, conversationID, convCtx.LastSyncMsgID, 20)
	if err != nil {
		return err
	}

	if len(newMessages) == 0 {
		return nil
	}

	// 将所有新消息添加到上下文，根据发送者分配正确的角色
	for _, msg := range newMessages {
		content := ""
		if msg.Content.Valid {
			content = msg.Content.String
		}

		if msg.SenderID == userID {
			// 这是"我"（包括我的 Agent）发送的消息 → "assistant"
			if err := convCtx.AddMessage("assistant", content); err != nil {
				return err
			}
		} else {
			// 这是对方发送的消息 → "user"
			if err := convCtx.AddMessage("user", content); err != nil {
				return err
			}
		}

		// 更新最后同步消息ID
		convCtx.LastSyncMsgID = msg.ID
	}

	// 保存上下文
	return s.contextRepo.Update(ctx, convCtx)
}

// summarizeAndReset 总结并重置上下文
func (s *AgentChatService) summarizeAndReset(ctx context.Context, convCtx *model.ConversationContext, user, partner *model.User) {
	messages, err := convCtx.GetMessages()
	if err != nil {
		return
	}

	// 调用 LLM 总结
	summary, err := s.chatSession.SummarizeConversation(ctx, toLLMMessages(messages))
	if err != nil {
		return
	}

	// 重建 System Prompt（包含总结）
	persona, _ := s.personaRepo.GetOrCreate(ctx, user.ID)
	rel, _ := s.relRepo.GetOrCreate(ctx, user.ID, partner.ID)

	data := &llm.PromptData{
		MyName:       user.Nickname,
		MyPersona:    persona,
		PartnerName:  partner.Nickname,
		Relationship: rel,
		Summary:      summary,
		CurrentTime:  time.Now(),
	}

	newSystemPrompt := s.chatSession.BuildSystemPrompt(data)

	// 重置上下文
	newMessages := []model.LLMMessage{
		{Role: "system", Content: newSystemPrompt},
	}

	messagesJSON, _ := json.Marshal(newMessages)
	s.contextRepo.ResetContext(ctx, convCtx.ConversationID, string(messagesJSON), summary)
}

// toLLMMessages 转换消息格式
func toLLMMessages(messages []model.LLMMessage) []llm.ChatMessage {
	result := make([]llm.ChatMessage, len(messages))
	for i, msg := range messages {
		result[i] = llm.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}

// analyzeConversationState 分析对话状态
func (s *AgentChatService) analyzeConversationState(messages []model.LLMMessage) *llm.ConversationState {
	state := &llm.ConversationState{}

	// 过滤掉 system 消息
	var chatMessages []model.LLMMessage
	for _, msg := range messages {
		if msg.Role != "system" {
			chatMessages = append(chatMessages, msg)
		}
	}

	if len(chatMessages) == 0 {
		return state
	}

	// 1. 统计末尾连续 assistant 消息数量
	for i := len(chatMessages) - 1; i >= 0; i-- {
		if chatMessages[i].Role == "assistant" {
			state.MyConsecutiveCount++
		} else {
			break
		}
	}

	// 2. 计算对方上次回复距今时间（基于消息位置估算）
	// 找到最后一个 user 消息的位置
	lastUserIdx := -1
	for i := len(chatMessages) - 1; i >= 0; i-- {
		if chatMessages[i].Role == "user" {
			lastUserIdx = i
			break
		}
	}

	if lastUserIdx >= 0 {
		// 根据之后的 assistant 消息数量估算时间
		msgsSinceUser := len(chatMessages) - 1 - lastUserIdx
		if msgsSinceUser == 0 {
			state.PartnerLastReplyAgo = "刚刚"
		} else if msgsSinceUser <= 2 {
			state.PartnerLastReplyAgo = "几分钟前"
		} else if msgsSinceUser <= 5 {
			state.PartnerLastReplyAgo = "一会儿前"
		} else {
			state.PartnerLastReplyAgo = "较长时间前"
		}
	}

	// 3. 检测对话是否已自然结束
	endKeywords := []string{"晚安", "再见", "拜拜", "下次聊", "回头见", "886", "88", "拜", "睡了", "先这样", "改天聊"}
	// 检查最后几条消息
	checkCount := 3
	if len(chatMessages) < checkCount {
		checkCount = len(chatMessages)
	}
	for i := len(chatMessages) - checkCount; i < len(chatMessages); i++ {
		content := chatMessages[i].Content
		for _, kw := range endKeywords {
			if containsKeyword(content, kw) {
				state.ConversationEnded = true
				break
			}
		}
		if state.ConversationEnded {
			break
		}
	}

	// 4. 提取最近话题（简单实现：取最后几条非 system 消息的摘要）
	topicCount := 3
	if len(chatMessages) < topicCount {
		topicCount = len(chatMessages)
	}
	var recentContents []string
	for i := len(chatMessages) - topicCount; i < len(chatMessages); i++ {
		content := chatMessages[i].Content
		// 截取前20个字符作为话题摘要
		if len(content) > 20 {
			content = content[:20] + "..."
		}
		recentContents = append(recentContents, content)
	}
	if len(recentContents) > 0 {
		state.RecentTopics = strings.Join(recentContents, " / ")
	}

	return state
}

// containsKeyword 检查内容是否包含关键词
func containsKeyword(content, keyword string) bool {
	return strings.Contains(strings.ToLower(content), strings.ToLower(keyword))
}

// updateSystemPromptWithState 更新带对话状态的 system prompt
func (s *AgentChatService) updateSystemPromptWithState(ctx context.Context, convCtx *model.ConversationContext, user, partner *model.User, convState *llm.ConversationState) error {
	// 构建新的 system prompt（带状态）
	newSystemPrompt := s.buildSystemPromptWithState(ctx, user, partner, convState)

	// 获取当前消息列表
	messages, err := convCtx.GetMessages()
	if err != nil {
		return err
	}

	// 更新第一个 system 消息
	if len(messages) > 0 && messages[0].Role == "system" {
		messages[0].Content = newSystemPrompt
	} else {
		// 如果没有 system 消息，插入一个
		messages = append([]model.LLMMessage{{Role: "system", Content: newSystemPrompt}}, messages...)
	}

	// 保存更新后的消息
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	convCtx.Messages = string(messagesJSON)
	convCtx.TokenCount = len(convCtx.Messages) / 4
	convCtx.UpdatedAt = time.Now()

	return s.contextRepo.Update(ctx, convCtx)
}

// buildSystemPromptWithState 构建带状态的 System Prompt
func (s *AgentChatService) buildSystemPromptWithState(ctx context.Context, user, partner *model.User, convState *llm.ConversationState) string {
	// 获取用户人设
	persona, _ := s.personaRepo.GetOrCreate(ctx, user.ID)

	// 获取关系画像
	rel, _ := s.relRepo.GetOrCreate(ctx, user.ID, partner.ID)

	// 获取对方状态（从分析缓存）
	var partnerStatus *llm.PartnerStatus
	if s.memoryRepo != nil {
		if cache, err := s.memoryRepo.GetAnalysisCache(ctx, partner.ID); err == nil && cache != nil {
			partnerStatus = &llm.PartnerStatus{
				Emoji:       cache.LifeStatus.Emoji,
				Label:       cache.LifeStatus.Label,
				Probability: cache.Availability.Probability,
			}
		}
	}

	data := &llm.PromptData{
		MyName:        user.Nickname,
		MyPersona:     persona,
		PartnerName:   partner.Nickname,
		PartnerStatus: partnerStatus,
		Relationship:  rel,
		CurrentTime:   time.Now(),
		ConvState:     convState,
	}

	return s.chatSession.BuildSystemPrompt(data)
}
