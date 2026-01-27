package service

import (
	"context"
	"fmt"
	"strings"

	"youkong/internal/model"
	"youkong/internal/pkg/llm"
	"youkong/internal/repository"
)

// RelationshipService 关系服务
type RelationshipService struct {
	relRepo     *repository.RelationshipRepository
	messageRepo *repository.MessageRepository
	relLearner  *llm.RelationshipLearner
}

// NewRelationshipService 创建关系服务
func NewRelationshipService(
	relRepo *repository.RelationshipRepository,
	messageRepo *repository.MessageRepository,
	llmClient *llm.OpenRouterClient,
) *RelationshipService {
	var learner *llm.RelationshipLearner
	if llmClient != nil {
		learner = llm.NewRelationshipLearner(llmClient)
	}
	return &RelationshipService{
		relRepo:     relRepo,
		messageRepo: messageRepo,
		relLearner:  learner,
	}
}

// GetRelationship 获取关系画像
func (s *RelationshipService) GetRelationship(ctx context.Context, userID, friendID string) (*model.Relationship, error) {
	return s.relRepo.GetOrCreate(ctx, userID, friendID)
}

// LearnFromConversation 从对话中学习关系（每次对话后调用）
func (s *RelationshipService) LearnFromConversation(ctx context.Context, conversationID, userID, friendID string) error {
	if s.relLearner == nil {
		return nil
	}

	// 获取当前关系
	rel, err := s.relRepo.GetOrCreate(ctx, userID, friendID)
	if err != nil {
		return fmt.Errorf("get relationship failed: %w", err)
	}

	// 增加消息计数
	if err := s.relRepo.IncrementMessageCount(ctx, userID, friendID); err != nil {
		// 不影响主流程
	}

	// 每5条消息触发一次关系学习
	if (rel.MessageCount+1)%5 != 0 {
		return nil
	}

	// 获取最近的消息
	messages, err := s.messageRepo.GetByConversationID(ctx, conversationID, 20)
	if err != nil || len(messages) < 3 {
		return nil
	}

	// 构建对话记录文本
	var chatLines []string
	for _, msg := range messages {
		role := "对方"
		if msg.SenderID == userID {
			role = "我"
		}
		chatLines = append(chatLines, fmt.Sprintf("%s: %s", role, msg.Content.String))
	}

	input := &llm.RelationshipLearnInput{
		RecentMessages:      strings.Join(chatLines, "\n"),
		CurrentRelationship: rel,
	}

	update, err := s.relLearner.LearnRelationship(ctx, input)
	if err != nil {
		return nil // LLM 失败不影响主流程
	}

	if !update.ShouldUpdate {
		return nil
	}

	// 应用更新
	s.applyRelationshipUpdate(rel, update)

	return s.relRepo.Update(ctx, rel)
}

// applyRelationshipUpdate 应用关系更新
func (s *RelationshipService) applyRelationshipUpdate(rel *model.Relationship, update *model.RelationshipUpdate) {
	if update.Closeness != "" {
		rel.Closeness = update.Closeness
	}
	if update.InteractStyle != "" {
		rel.InteractStyle = update.InteractStyle
	}
	if update.CommonTopics != "" {
		rel.CommonTopics = update.CommonTopics
	}
	if update.ToneWithThis != "" {
		rel.ToneWithThis = update.ToneWithThis
	}
	if update.JokeLevel != "" {
		rel.JokeLevel = update.JokeLevel
	}
	if update.SharedMemory != "" {
		// 累加共同记忆
		if rel.SharedMemory != "" {
			rel.SharedMemory = rel.SharedMemory + "; " + update.SharedMemory
		} else {
			rel.SharedMemory = update.SharedMemory
		}
	}

	// 更新置信度
	if rel.MessageCount > 100 {
		rel.ConfidenceScore = 100
	} else {
		rel.ConfidenceScore = rel.MessageCount
	}
}
