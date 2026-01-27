package service

import (
	"context"
	"fmt"

	"youkong/internal/model"
	"youkong/internal/pkg/llm"
	"youkong/internal/repository"
)

// PersonaService 人设服务
type PersonaService struct {
	personaRepo   *repository.PersonaRepository
	memoryRepo    *repository.MemoryRepository
	personaLearner *llm.PersonaLearner
}

// NewPersonaService 创建人设服务
func NewPersonaService(
	personaRepo *repository.PersonaRepository,
	memoryRepo *repository.MemoryRepository,
	llmClient *llm.OpenRouterClient,
) *PersonaService {
	var learner *llm.PersonaLearner
	if llmClient != nil {
		learner = llm.NewPersonaLearner(llmClient)
	}
	return &PersonaService{
		personaRepo:    personaRepo,
		memoryRepo:     memoryRepo,
		personaLearner: learner,
	}
}

// GetPersona 获取用户人设
func (s *PersonaService) GetPersona(ctx context.Context, userID string) (*model.UserPersona, error) {
	return s.personaRepo.GetOrCreate(ctx, userID)
}

// UpdatePersonaFromStatus 从状态数据更新人设（在状态上报时调用）
func (s *PersonaService) UpdatePersonaFromStatus(ctx context.Context, userID string) error {
	if s.personaLearner == nil {
		return nil // LLM 未配置，跳过
	}

	// 获取当前人设
	persona, err := s.personaRepo.GetOrCreate(ctx, userID)
	if err != nil {
		return fmt.Errorf("get persona failed: %w", err)
	}

	// 获取核心记忆（包含行为模式）
	memory, err := s.memoryRepo.GetCoreMemory(ctx, userID)
	if err != nil {
		return fmt.Errorf("get core memory failed: %w", err)
	}

	// 数据不足时跳过
	if memory == nil || memory.SampleCount < 5 {
		return nil
	}

	// 每10次状态上报触发一次人设学习
	if memory.SampleCount%10 != 0 {
		return nil
	}

	// 调用 LLM 学习人设
	input := &llm.LearnInput{
		TimePatterns:        memory.TimePatterns,
		LocationPreferences: memory.LocationPreferences,
		DevicePatterns:      memory.BehaviorInsights,
		CurrentPersona:      persona,
	}

	update, err := s.personaLearner.LearnPersona(ctx, input)
	if err != nil {
		// LLM 失败不影响主流程
		return nil
	}

	if !update.ShouldUpdate {
		return nil
	}

	// 更新人设
	s.applyPersonaUpdate(persona, update)
	persona.SampleCount++

	return s.personaRepo.Update(ctx, persona)
}

// UpdatePersonaFromChat 从聊天记录更新人设
func (s *PersonaService) UpdatePersonaFromChat(ctx context.Context, userID string, messages []string) error {
	if s.personaLearner == nil || len(messages) < 5 {
		return nil
	}

	persona, err := s.personaRepo.GetOrCreate(ctx, userID)
	if err != nil {
		return fmt.Errorf("get persona failed: %w", err)
	}

	update, err := s.personaLearner.LearnFromChat(ctx, messages, persona)
	if err != nil {
		return nil // LLM 失败不影响主流程
	}

	if !update.ShouldUpdate {
		return nil
	}

	s.applyPersonaUpdate(persona, update)
	persona.SampleCount++

	return s.personaRepo.Update(ctx, persona)
}

// applyPersonaUpdate 应用人设更新
func (s *PersonaService) applyPersonaUpdate(persona *model.UserPersona, update *model.PersonaUpdate) {
	if update.Personality != "" {
		persona.Personality = update.Personality
	}
	if update.SpeakingStyle != "" {
		persona.SpeakingStyle = update.SpeakingStyle
	}
	if update.Interests != "" {
		persona.Interests = update.Interests
	}
	if update.SocialHabits != "" {
		persona.SocialHabits = update.SocialHabits
	}
	if update.CommonPhrases != "" {
		persona.CommonPhrases = update.CommonPhrases
	}
	if update.EmojiUsage != "" {
		persona.EmojiUsage = update.EmojiUsage
	}
	if update.MessageLength != "" {
		persona.MessageLength = update.MessageLength
	}

	// 更新置信度
	if persona.SampleCount > 50 {
		persona.ConfidenceScore = 100
	} else {
		persona.ConfidenceScore = persona.SampleCount * 2
	}
}
