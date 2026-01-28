package service

import (
	"context"
	"database/sql"
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
	curatorService      *ContextCuratorService
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
	var curator *ContextCuratorService
	if llmClient != nil {
		session = llm.NewChatSession(llmClient)
		curator = NewContextCuratorService(llmClient, memoryRepo)
	}
	return &AgentChatService{
		contextRepo:         contextRepo,
		messageRepo:         messageRepo,
		personaRepo:         personaRepo,
		relRepo:             relRepo,
		userRepo:            userRepo,
		memoryRepo:          memoryRepo,
		chatSession:         session,
		curatorService:      curator,
		wsManager:           wsManager,
		relService:          relService,
		notificationService: notificationService,
	}
}

// GenerateReply 生成 Agent 回复
// 使用显式对话格式：[名字]: 消息，避免 LLM 角色混淆
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

	// 从数据库获取最近 20 条消息（按时间倒序，最新的在前面）
	dbMessages, err := s.messageRepo.GetMessagesByConversationID(ctx, conversationID, 20, 0)
	if err != nil {
		log.Printf("[AgentChat] 获取消息失败: %v", err)
	}

	// 转换为带发送者名字的消息格式
	history := s.buildChatHistory(dbMessages, userID, user.Nickname, partner.Nickname)

	// 分析对话状态
	convState := s.analyzeHistoryState(history)

	// 直接使用原始上下文（Curator 异步处理，不阻塞主请求）
	chatPrompt := s.chatSession.BuildChatPrompt(history, user.Nickname, partner.Nickname)
	log.Printf("[AgentChat] 使用原始上下文，%d 条消息", len(history))

	// 异步调用 Curator 提取关键信息（下次请求时可用）
	if s.curatorService != nil && s.curatorService.NeedsCuration(len(history)) {
		go func() {
			historyCopy := make([]llm.ChatHistoryMessage, len(history))
			copy(historyCopy, history)
			s.curatorService.CurateContext(context.Background(), conversationID, historyCopy, user.Nickname, partner.Nickname)
			log.Printf("[AgentChat] Curator 异步处理完成，%d 条消息", len(historyCopy))
		}()
	}

	// 获取记忆总结（添加到 System Prompt）
	var memorySummary string
	if s.curatorService != nil {
		memorySummary = s.curatorService.GetMemorySummary(ctx, conversationID)
	}

	// 构建 System Prompt（人设 + 关系 + 规则 + 记忆）
	systemPrompt := s.buildSystemPromptWithMemory(ctx, user, partner, convState, memorySummary)

	// 构建 LLM 消息（只有 2 条：system + user）
	messages := []llm.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: chatPrompt},
	}

	// 调用 LLM 生成回复
	reply, err := s.chatSession.GenerateReply(ctx, messages)
	if err != nil {
		log.Printf("[AgentChat] LLM 调用失败: %v", err)
		return nil, fmt.Errorf("你的元婴罢工了")
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

	// 异步更新消息计数（用于 Curator 判断是否需要总结）
	go func() {
		if s.memoryRepo != nil {
			s.memoryRepo.IncrementMessageCount(context.Background(), conversationID)
		}
	}()

	return response, nil
}

// buildChatHistory 从数据库消息构建带发送者名字的对话历史
// dbMessages 是按时间倒序的（最新的在前），需要反转为正序（最早的在前）
func (s *AgentChatService) buildChatHistory(dbMessages []*model.Message, myUserID, myName, partnerName string) []llm.ChatHistoryMessage {
	history := make([]llm.ChatHistoryMessage, 0, len(dbMessages))

	for i := len(dbMessages) - 1; i >= 0; i-- {
		msg := dbMessages[i]

		// 只处理文本消息
		if msg.Type != model.MessageTypeText {
			continue
		}

		content := ""
		if msg.Content.Valid {
			content = strings.TrimSpace(msg.Content.String)
		}

		// 跳过空消息
		if content == "" {
			continue
		}

		isMe := msg.SenderID == myUserID
		senderName := partnerName
		if isMe {
			senderName = myName
		}

		history = append(history, llm.ChatHistoryMessage{
			SenderName: senderName,
			Content:    content,
			IsMe:       isMe,
			Time:       msg.CreatedAt,
		})
	}

	return history
}

// analyzeHistoryState 分析对话历史状态
func (s *AgentChatService) analyzeHistoryState(history []llm.ChatHistoryMessage) *llm.ConversationState {
	state := &llm.ConversationState{}

	if len(history) == 0 {
		return state
	}

	// 1. 统计末尾连续"我"发送的消息数量
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].IsMe {
			state.MyConsecutiveCount++
		} else {
			break
		}
	}

	// 2. 计算对方上次回复距今时间
	var lastPartnerTime time.Time
	for i := len(history) - 1; i >= 0; i-- {
		if !history[i].IsMe {
			lastPartnerTime = history[i].Time
			break
		}
	}

	if !lastPartnerTime.IsZero() {
		elapsed := time.Since(lastPartnerTime)
		switch {
		case elapsed < 5*time.Minute:
			state.PartnerLastReplyAgo = "刚刚"
		case elapsed < 30*time.Minute:
			state.PartnerLastReplyAgo = fmt.Sprintf("%d分钟前", int(elapsed.Minutes()))
		case elapsed < 2*time.Hour:
			state.PartnerLastReplyAgo = fmt.Sprintf("%d小时前", int(elapsed.Hours()))
		default:
			state.PartnerLastReplyAgo = "较长时间前"
		}
	}

	// 3. 检测对话是否已自然结束
	endKeywords := []string{"晚安", "再见", "拜拜", "下次聊", "回头见", "886", "88", "拜", "睡了", "先这样", "改天聊"}
	checkCount := 3
	if len(history) < checkCount {
		checkCount = len(history)
	}
	for i := len(history) - checkCount; i < len(history); i++ {
		content := history[i].Content
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

	// 4. 提取最近话题
	topicCount := 3
	if len(history) < topicCount {
		topicCount = len(history)
	}
	var recentContents []string
	for i := len(history) - topicCount; i < len(history); i++ {
		content := history[i].Content
		// 使用 rune 正确处理中文字符截断
		runes := []rune(content)
		if len(runes) > 20 {
			content = string(runes[:20]) + "..."
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

// buildSystemPromptWithState 构建带状态的 System Prompt（已废弃，使用 buildSystemPromptWithMemory）
func (s *AgentChatService) buildSystemPromptWithState(ctx context.Context, user, partner *model.User, convState *llm.ConversationState) string {
	return s.buildSystemPromptWithMemory(ctx, user, partner, convState, "")
}

// buildSystemPromptWithMemory 构建带记忆的 System Prompt
func (s *AgentChatService) buildSystemPromptWithMemory(ctx context.Context, user, partner *model.User, convState *llm.ConversationState, memorySummary string) string {
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
		Summary:       memorySummary, // 使用记忆总结
		CurrentTime:   time.Now(),
		ConvState:     convState,
	}

	return s.chatSession.BuildSystemPrompt(data)
}

// buildCuratedChatPrompt 使用 Curator 结果构建对话 Prompt
func (s *AgentChatService) buildCuratedChatPrompt(curated *model.CuratedContext, myName, partnerName string) string {
	var sb strings.Builder

	sb.WriteString("## 对话历史（精选）\n")
	sb.WriteString(curated.Context)

	// 添加关键点提示
	if len(curated.KeyPoints) > 0 {
		sb.WriteString("\n## 关键点\n")
		for _, point := range curated.KeyPoints {
			sb.WriteString(fmt.Sprintf("• %s\n", point))
		}
	}

	// 标注对话状态
	switch curated.State {
	case "waiting_for_reply":
		sb.WriteString(fmt.Sprintf("\n（%s 还没回复）\n", partnerName))
	case "ending":
		sb.WriteString("\n（对话可能要结束了）\n")
	}

	sb.WriteString(fmt.Sprintf("\n## 你的任务\n以 %s 的身份，发送下一条消息。直接输出消息内容，不要加 [%s]: 前缀。",
		myName, myName))

	return sb.String()
}
