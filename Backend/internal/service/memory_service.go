package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/llm"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

const (
	// Redis key 前缀
	keyUserAnalysis = "agent:analysis:%s" // 用户分析结果缓存

	// 缓存过期时间
	analysisTTL = 10 * time.Minute // 分析结果缓存 10 分钟
)

// MemoryService 记忆服务
type MemoryService struct {
	memoryRepo     *repository.MemoryRepository
	redisClient    *tencent.RedisClient
	memoryAnalyzer *llm.MemoryAnalyzer
}

// NewMemoryService 创建记忆服务
func NewMemoryService(
	memoryRepo *repository.MemoryRepository,
	redisClient *tencent.RedisClient,
	llmClient *llm.OpenRouterClient,
) *MemoryService {
	var analyzer *llm.MemoryAnalyzer
	if llmClient != nil {
		analyzer = llm.NewMemoryAnalyzer(llmClient)
	}
	return &MemoryService{
		memoryRepo:     memoryRepo,
		redisClient:    redisClient,
		memoryAnalyzer: analyzer,
	}
}

// AnalyzeAndUpdateMemory 分析状态并更新记忆
func (s *MemoryService) AnalyzeAndUpdateMemory(ctx context.Context, userID string, status *model.ExtendedStatusReportRequest) (*model.AnalysisResult, error) {
	// 1. 获取现有核心记忆
	memory, err := s.memoryRepo.GetCoreMemory(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get core memory: %w", err)
	}

	// 2. 获取最近历史记录（用于上下文）
	recentHistory, err := s.memoryRepo.GetRecentHistory(ctx, userID, 10)
	if err != nil {
		// 不影响主流程
		recentHistory = nil
	}

	// 3. 构建分析输入
	now := time.Now()
	input := &llm.AnalysisInput{
		CurrentStatus: status,
		CurrentMemory: memory,
		RecentHistory: recentHistory,
		Timestamp:     now,
		Weekday:       getWeekdayName(now.Weekday()),
		Hour:          now.Hour(),
	}

	// 4. 调用分析器（LLM 或规则）
	var result *model.AnalysisResult
	if s.memoryAnalyzer != nil {
		result, err = s.memoryAnalyzer.Analyze(ctx, input)
		if err != nil {
			// 分析失败，使用默认结果
			result = s.getDefaultAnalysisResult()
		}
	} else {
		// 无分析器，使用默认结果
		result = s.getDefaultAnalysisResult()
	}

	// 5. 保存状态历史
	if err := s.memoryRepo.SaveStatusHistory(ctx, userID, status); err != nil {
		// 记录错误但不影响返回
		fmt.Printf("save status history error: %v\n", err)
	}

	// 6. 更新核心记忆（如果需要）
	if result.MemoryUpdate != nil && result.MemoryUpdate.ShouldUpdate {
		if err := s.updateCoreMemory(ctx, userID, memory, result.MemoryUpdate); err != nil {
			fmt.Printf("update core memory error: %v\n", err)
		}
	} else {
		// 只增加样本数量
		if memory != nil {
			if err := s.memoryRepo.IncrementSampleCount(ctx, userID); err != nil {
				fmt.Printf("increment sample count error: %v\n", err)
			}
		} else {
			// 首次创建记忆
			if err := s.createInitialMemory(ctx, userID); err != nil {
				fmt.Printf("create initial memory error: %v\n", err)
			}
		}
	}

	// 7. 缓存分析结果（Redis + MySQL）
	if err := s.cacheAnalysisResult(ctx, userID, result); err != nil {
		fmt.Printf("cache analysis result error: %v\n", err)
	}

	return result, nil
}

// GetCoreMemory 获取用户核心记忆
func (s *MemoryService) GetCoreMemory(ctx context.Context, userID string) (*model.CoreMemoryResponse, error) {
	memory, err := s.memoryRepo.GetCoreMemory(ctx, userID)
	if err != nil {
		return nil, err
	}
	if memory == nil {
		return nil, nil
	}

	return &model.CoreMemoryResponse{
		BehaviorInsights:    memory.BehaviorInsights,
		TimePatterns:        memory.TimePatterns,
		LocationPreferences: memory.LocationPreferences,
		SocialTendency:      memory.SocialTendency,
		ConfidenceScore:     memory.ConfidenceScore,
		SampleCount:         memory.SampleCount,
		UpdatedAt:           memory.UpdatedAt,
	}, nil
}

// GetCachedAnalysis 获取缓存的分析结果
func (s *MemoryService) GetCachedAnalysis(ctx context.Context, userID string) (*model.AnalysisResult, error) {
	// 先从 Redis 获取
	key := fmt.Sprintf(keyUserAnalysis, userID)
	data, err := s.redisClient.GetBytes(ctx, key)
	if err == nil && len(data) > 0 {
		var result model.AnalysisResult
		if err := json.Unmarshal(data, &result); err == nil {
			return &result, nil
		}
	}

	// Redis 没有，从 MySQL 获取
	return s.memoryRepo.GetAnalysisCache(ctx, userID)
}

// GetCachedAnalysisByUserIDs 批量获取缓存的分析结果
func (s *MemoryService) GetCachedAnalysisByUserIDs(ctx context.Context, userIDs []string) (map[string]*model.AnalysisResult, error) {
	result := make(map[string]*model.AnalysisResult)

	// 先从 Redis 批量获取
	for _, userID := range userIDs {
		key := fmt.Sprintf(keyUserAnalysis, userID)
		data, err := s.redisClient.GetBytes(ctx, key)
		if err == nil && len(data) > 0 {
			var analysis model.AnalysisResult
			if err := json.Unmarshal(data, &analysis); err == nil {
				result[userID] = &analysis
			}
		}
	}

	// 找出 Redis 中没有的
	missingIDs := make([]string, 0)
	for _, userID := range userIDs {
		if _, ok := result[userID]; !ok {
			missingIDs = append(missingIDs, userID)
		}
	}

	// 从 MySQL 批量获取缺失的
	if len(missingIDs) > 0 {
		dbResults, err := s.memoryRepo.GetAnalysisCacheByUserIDs(ctx, missingIDs)
		if err == nil {
			for userID, analysis := range dbResults {
				result[userID] = analysis
			}
		}
	}

	return result, nil
}

// updateCoreMemory 更新核心记忆
func (s *MemoryService) updateCoreMemory(ctx context.Context, userID string, existing *model.CoreMemory, update *model.MemoryUpdate) error {
	var memory *model.CoreMemory
	if existing != nil {
		memory = existing
	} else {
		memory = &model.CoreMemory{UserID: userID}
	}

	// 增量更新（只更新非空字段）
	if update.BehaviorInsights != "" {
		memory.BehaviorInsights = mergeInsight(memory.BehaviorInsights, update.BehaviorInsights)
	}
	if update.TimePatterns != "" {
		memory.TimePatterns = mergeInsight(memory.TimePatterns, update.TimePatterns)
	}
	if update.LocationPreferences != "" {
		memory.LocationPreferences = mergeInsight(memory.LocationPreferences, update.LocationPreferences)
	}
	if update.SocialTendency != "" {
		memory.SocialTendency = mergeInsight(memory.SocialTendency, update.SocialTendency)
	}

	// 更新样本数和置信度
	memory.SampleCount++
	memory.ConfidenceScore = min(100, memory.SampleCount*2)

	return s.memoryRepo.UpsertCoreMemory(ctx, memory)
}

// createInitialMemory 创建初始记忆
func (s *MemoryService) createInitialMemory(ctx context.Context, userID string) error {
	memory := &model.CoreMemory{
		UserID:          userID,
		SampleCount:     1,
		ConfidenceScore: 2,
	}
	return s.memoryRepo.UpsertCoreMemory(ctx, memory)
}

// cacheAnalysisResult 缓存分析结果
func (s *MemoryService) cacheAnalysisResult(ctx context.Context, userID string, result *model.AnalysisResult) error {
	// 1. 缓存到 Redis
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	key := fmt.Sprintf(keyUserAnalysis, userID)
	if err := s.redisClient.Set(ctx, key, data, analysisTTL); err != nil {
		// Redis 失败不影响 MySQL
		fmt.Printf("cache to redis error: %v\n", err)
	}

	// 2. 持久化到 MySQL
	return s.memoryRepo.SaveAnalysisCache(ctx, userID, result)
}

// getDefaultAnalysisResult 获取默认分析结果
func (s *MemoryService) getDefaultAnalysisResult() *model.AnalysisResult {
	return &model.AnalysisResult{
		Availability: model.AvailabilityAnalysis{
			Status:      "可能有空",
			Probability: 50,
			Reason:      "数据分析中",
			Confidence:  "low",
		},
		LifeStatus: model.LifeStatus{
			Emoji: "🤔",
			Label: "状态未知",
		},
	}
}

// 辅助函数

func getWeekdayName(w time.Weekday) string {
	names := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return names[w]
}

func mergeInsight(existing, new string) string {
	if existing == "" {
		return new
	}
	// 简单策略：如果新的洞察更长或更详细，使用新的
	// 实际场景可以使用更复杂的合并逻辑
	if len([]rune(new)) > len([]rune(existing)) {
		return new
	}
	return existing
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
