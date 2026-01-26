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
	keyUserAnalysis  = "agent:analysis:%s" // 用户分析结果缓存
	keyLastStatus    = "agent:last:%s"     // 上一次状态数据

	// 缓存过期时间
	analysisTTL     = 10 * time.Minute // 分析结果缓存 10 分钟
	lastStatusTTL   = 30 * time.Minute // 上次状态缓存 30 分钟

	// 变化检测阈值
	significantTimeGap = 30 * time.Minute // 超过 30 分钟视为显著变化
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
	now := time.Now()

	// 1. 检查是否有显著变化（决定是否需要调用 LLM）
	lastStatus, lastTime := s.getLastStatus(ctx, userID)
	hasSignificantChange := s.detectSignificantChange(status, lastStatus, lastTime, now)

	// 2. 如果没有显著变化，尝试返回缓存结果
	if !hasSignificantChange {
		if cached, err := s.GetCachedAnalysis(ctx, userID); err == nil && cached != nil {
			// 保存当前状态（用于下次对比）
			s.saveLastStatus(ctx, userID, status)
			// 保存历史记录
			s.memoryRepo.SaveStatusHistory(ctx, userID, status)
			return cached, nil
		}
		// 缓存不存在，需要调用 LLM
	}

	// 3. 获取现有核心记忆
	memory, err := s.memoryRepo.GetCoreMemory(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get core memory: %w", err)
	}

	// 4. 获取最近历史记录（用于上下文）
	recentHistory, err := s.memoryRepo.GetRecentHistory(ctx, userID, 10)
	if err != nil {
		// 不影响主流程
		recentHistory = nil
	}

	// 5. 构建分析输入
	input := &llm.AnalysisInput{
		CurrentStatus: status,
		CurrentMemory: memory,
		RecentHistory: recentHistory,
		Timestamp:     now,
		Weekday:       getWeekdayName(now.Weekday()),
		Hour:          now.Hour(),
	}

	// 6. 调用分析器（LLM 或规则）
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

	// 7. 保存当前状态（用于下次对比）
	s.saveLastStatus(ctx, userID, status)

	// 8. 保存状态历史
	if err := s.memoryRepo.SaveStatusHistory(ctx, userID, status); err != nil {
		// 记录错误但不影响返回
		fmt.Printf("save status history error: %v\n", err)
	}

	// 9. 更新核心记忆（如果需要）
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

	// 10. 缓存分析结果（Redis + MySQL）
	if err := s.cacheAnalysisResult(ctx, userID, result); err != nil {
		fmt.Printf("cache analysis result error: %v\n", err)
	}

	return result, nil
}

// lastStatusData 上次状态数据（带时间戳）
type lastStatusData struct {
	Status    *model.ExtendedStatusReportRequest `json:"status"`
	Timestamp time.Time                          `json:"timestamp"`
}

// getLastStatus 获取上次状态
func (s *MemoryService) getLastStatus(ctx context.Context, userID string) (*model.ExtendedStatusReportRequest, time.Time) {
	key := fmt.Sprintf(keyLastStatus, userID)
	data, err := s.redisClient.GetBytes(ctx, key)
	if err != nil || len(data) == 0 {
		return nil, time.Time{}
	}

	var last lastStatusData
	if err := json.Unmarshal(data, &last); err != nil {
		return nil, time.Time{}
	}

	return last.Status, last.Timestamp
}

// saveLastStatus 保存当前状态
func (s *MemoryService) saveLastStatus(ctx context.Context, userID string, status *model.ExtendedStatusReportRequest) {
	key := fmt.Sprintf(keyLastStatus, userID)
	data := lastStatusData{
		Status:    status,
		Timestamp: time.Now(),
	}
	if bytes, err := json.Marshal(data); err == nil {
		s.redisClient.Set(ctx, key, bytes, lastStatusTTL)
	}
}

// detectSignificantChange 检测是否有显著变化
func (s *MemoryService) detectSignificantChange(current, last *model.ExtendedStatusReportRequest, lastTime, now time.Time) bool {
	// 首次上报
	if last == nil {
		return true
	}

	// 时间间隔超过阈值
	if now.Sub(lastTime) > significantTimeGap {
		return true
	}

	// 比较关键字段

	// 1. 屏幕活跃状态变化
	if current.Screen != nil && last.Screen != nil {
		if current.Screen.IsActive != last.Screen.IsActive {
			return true
		}
		if current.Screen.ActivityType != last.Screen.ActivityType {
			return true
		}
	} else if (current.Screen == nil) != (last.Screen == nil) {
		// 一个有一个没有
		return true
	}

	// 2. 位置类型变化
	if current.Location != nil && last.Location != nil {
		if current.Location.PlaceType != last.Location.PlaceType {
			return true
		}
	} else if (current.Location == nil) != (last.Location == nil) {
		return true
	}

	// 3. 专注模式变化
	if current.Mode != nil && last.Mode != nil {
		if current.Mode.IsFocusModeOn != last.Mode.IsFocusModeOn {
			return true
		}
	}

	// 4. 耳机连接状态变化
	if current.Connection != nil && last.Connection != nil {
		if current.Connection.IsHeadphonesConnected != last.Connection.IsHeadphonesConnected {
			return true
		}
	}

	// 没有显著变化
	return false
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
