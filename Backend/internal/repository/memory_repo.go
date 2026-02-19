package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

// MemoryRepository 记忆数据访问层
type MemoryRepository struct {
	db *sqlx.DB
}

// NewMemoryRepository 创建记忆数据访问层
func NewMemoryRepository(db *sqlx.DB) *MemoryRepository {
	return &MemoryRepository{db: db}
}

// ExecRaw 执行原始 SQL（用于数据收集等场景）
func (r *MemoryRepository) ExecRaw(ctx context.Context, query string, args ...interface{}) {
	_, _ = r.db.ExecContext(ctx, query, args...)
}

// ========== CoreMemory 操作 ==========

// GetCoreMemory 获取用户的核心记忆
func (r *MemoryRepository) GetCoreMemory(ctx context.Context, userID string) (*model.CoreMemory, error) {
	var memory model.CoreMemory
	query := `SELECT * FROM core_memories WHERE user_id = ?`
	err := r.db.GetContext(ctx, &memory, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &memory, nil
}

// CreateCoreMemory 创建核心记忆
func (r *MemoryRepository) CreateCoreMemory(ctx context.Context, memory *model.CoreMemory) error {
	query := `INSERT INTO core_memories (user_id, behavior_insights, time_patterns, location_preferences, social_tendency, confidence_score, sample_count, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		memory.UserID,
		memory.BehaviorInsights,
		memory.TimePatterns,
		memory.LocationPreferences,
		memory.SocialTendency,
		memory.ConfidenceScore,
		memory.SampleCount,
		now,
		now,
	)
	return err
}

// UpdateCoreMemory 更新核心记忆
func (r *MemoryRepository) UpdateCoreMemory(ctx context.Context, memory *model.CoreMemory) error {
	query := `UPDATE core_memories SET
              behavior_insights = ?,
              time_patterns = ?,
              location_preferences = ?,
              social_tendency = ?,
              confidence_score = ?,
              sample_count = ?,
              updated_at = ?
              WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query,
		memory.BehaviorInsights,
		memory.TimePatterns,
		memory.LocationPreferences,
		memory.SocialTendency,
		memory.ConfidenceScore,
		memory.SampleCount,
		time.Now(),
		memory.UserID,
	)
	return err
}

// UpsertCoreMemory 插入或更新核心记忆
func (r *MemoryRepository) UpsertCoreMemory(ctx context.Context, memory *model.CoreMemory) error {
	query := `INSERT INTO core_memories (user_id, behavior_insights, time_patterns, location_preferences, social_tendency, confidence_score, sample_count, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
              ON DUPLICATE KEY UPDATE
              behavior_insights = VALUES(behavior_insights),
              time_patterns = VALUES(time_patterns),
              location_preferences = VALUES(location_preferences),
              social_tendency = VALUES(social_tendency),
              confidence_score = VALUES(confidence_score),
              sample_count = VALUES(sample_count),
              updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		memory.UserID,
		memory.BehaviorInsights,
		memory.TimePatterns,
		memory.LocationPreferences,
		memory.SocialTendency,
		memory.ConfidenceScore,
		memory.SampleCount,
	)
	return err
}

// UpdatePersonaFields 更新 persona 相关字段（每日凌晨生成）
func (r *MemoryRepository) UpdatePersonaFields(ctx context.Context, userID, personaText, timePatternStats string) error {
	query := `INSERT INTO core_memories (user_id, persona_text, time_pattern_stats, persona_generated_at, created_at, updated_at)
              VALUES (?, ?, ?, NOW(), NOW(), NOW())
              ON DUPLICATE KEY UPDATE
              persona_text = VALUES(persona_text),
              time_pattern_stats = VALUES(time_pattern_stats),
              persona_generated_at = NOW(),
              updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query, userID, personaText, timePatternStats)
	return err
}

// GetActiveUsersWithRecentHistory 获取近 N 天有 status_histories 记录的活跃用户 ID
func (r *MemoryRepository) GetActiveUsersWithRecentHistory(ctx context.Context, days int) ([]string, error) {
	var userIDs []string
	query := `SELECT DISTINCT user_id FROM status_histories WHERE created_at > DATE_SUB(NOW(), INTERVAL ? DAY)`
	err := r.db.SelectContext(ctx, &userIDs, query, days)
	if err != nil {
		return nil, err
	}
	return userIDs, nil
}

// IncrementSampleCount 增加样本数量并更新置信度
func (r *MemoryRepository) IncrementSampleCount(ctx context.Context, userID string) error {
	// 置信度 = min(100, 样本数 * 2)
	query := `UPDATE core_memories SET
              sample_count = sample_count + 1,
              confidence_score = LEAST(100, (sample_count + 1) * 2),
              updated_at = NOW()
              WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// ========== StatusHistory 操作 ==========

// SaveStatusHistory 保存状态历史
func (r *MemoryRepository) SaveStatusHistory(ctx context.Context, userID string, status *model.ExtendedStatusReportRequest) error {
	// 序列化原始数据
	rawData, err := json.Marshal(status)
	if err != nil {
		return err
	}

	now := time.Now()
	weekday := int(now.Weekday())
	hour := now.Hour()
	isWeekend := weekday == 0 || weekday == 6

	query := `INSERT INTO status_histories (user_id, raw_data, day_of_week, hour_of_day, is_weekend, created_at)
              VALUES (?, ?, ?, ?, ?, ?)`
	_, err = r.db.ExecContext(ctx, query,
		userID,
		rawData,
		weekday,
		hour,
		isWeekend,
		now,
	)
	return err
}

// GetRecentHistory 获取最近的状态历史
func (r *MemoryRepository) GetRecentHistory(ctx context.Context, userID string, limit int) ([]*model.StatusHistory, error) {
	var histories []*model.StatusHistory
	query := `SELECT * FROM status_histories WHERE user_id = ? ORDER BY created_at DESC LIMIT ?`
	err := r.db.SelectContext(ctx, &histories, query, userID, limit)
	if err != nil {
		return nil, err
	}
	return histories, nil
}

// GetHistoryByTimeRange 获取指定时间范围的状态历史
func (r *MemoryRepository) GetHistoryByTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*model.StatusHistory, error) {
	var histories []*model.StatusHistory
	query := `SELECT * FROM status_histories WHERE user_id = ? AND created_at BETWEEN ? AND ? ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &histories, query, userID, start, end)
	if err != nil {
		return nil, err
	}
	return histories, nil
}

// GetHistoryByHourAndWeekday 获取特定小时和星期几的历史数据（用于模式分析）
func (r *MemoryRepository) GetHistoryByHourAndWeekday(ctx context.Context, userID string, hour, weekday int, limit int) ([]*model.StatusHistory, error) {
	var histories []*model.StatusHistory
	query := `SELECT * FROM status_histories WHERE user_id = ? AND hour_of_day = ? AND day_of_week = ? ORDER BY created_at DESC LIMIT ?`
	err := r.db.SelectContext(ctx, &histories, query, userID, hour, weekday, limit)
	if err != nil {
		return nil, err
	}
	return histories, nil
}

// GetHistoryCount 获取用户的历史记录数量
func (r *MemoryRepository) GetHistoryCount(ctx context.Context, userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM status_histories WHERE user_id = ?`
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}

// DeleteOldHistory 删除旧的历史记录（保留最近 N 天）
func (r *MemoryRepository) DeleteOldHistory(ctx context.Context, days int) (int64, error) {
	query := `DELETE FROM status_histories WHERE created_at < DATE_SUB(NOW(), INTERVAL ? DAY)`
	result, err := r.db.ExecContext(ctx, query, days)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ========== 分析缓存操作 ==========

// UserAnalysisCache 用户分析缓存
type UserAnalysisCache struct {
	ID                      int64     `db:"id"`
	UserID                  string    `db:"user_id"`
	AvailabilityStatus      string    `db:"availability_status"`
	AvailabilityProbability int       `db:"availability_probability"`
	AvailabilityReason      string    `db:"availability_reason"`
	AvailabilityConfidence  string    `db:"availability_confidence"`
	LifeStatusEmoji         string    `db:"life_status_emoji"`
	LifeStatusLabel         string    `db:"life_status_label"`
	LifeStatusDescription   *string   `db:"life_status_description"` // 详细描述
	Mood                    *string   `db:"mood"`                    // 心情
	Activity                *string   `db:"activity"`                // 活动
	Context                 *string   `db:"context"`                 // 上下文
	GifURL                  *string   `db:"gif_url"`                 // GIF URL
	GiphyQuery              *string   `db:"giphy_query"`             // Giphy 搜索词
	UseGif                  bool      `db:"use_gif"`                 // 是否使用 GIF 显示模式
	CreatedAt               time.Time `db:"created_at"`
	UpdatedAt               time.Time `db:"updated_at"`
	IsAIGuess               bool      `db:"is_ai_guess"` // 是否为 AI 推测的状态（字段顺序与数据库列顺序一致）
}

// SaveAnalysisCache 保存分析缓存
func (r *MemoryRepository) SaveAnalysisCache(ctx context.Context, userID string, result *model.AnalysisResult) error {
	// 如果是 AI 猜测的结果，需要检查现有缓存
	// 用户手动设置的状态（is_ai_guess=false）不应被 AI 猜测覆盖
	if result.IsAIGuess {
		existing, err := r.GetAnalysisCache(ctx, userID)
		if err == nil && existing != nil && !existing.IsAIGuess {
			// 现有缓存是用户手动设置的，不要用 AI 猜测覆盖
			return nil
		}
	}

	query := `INSERT INTO user_analysis_cache (
		user_id, availability_status, availability_probability, availability_reason, availability_confidence,
		life_status_emoji, life_status_label, life_status_description, mood, activity, context, gif_url, giphy_query, use_gif, is_ai_guess
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
		availability_status = VALUES(availability_status),
		availability_probability = VALUES(availability_probability),
		availability_reason = VALUES(availability_reason),
		availability_confidence = VALUES(availability_confidence),
		life_status_emoji = VALUES(life_status_emoji),
		life_status_label = VALUES(life_status_label),
		life_status_description = VALUES(life_status_description),
		mood = VALUES(mood),
		activity = VALUES(activity),
		context = VALUES(context),
		gif_url = VALUES(gif_url),
		giphy_query = VALUES(giphy_query),
		use_gif = VALUES(use_gif),
		is_ai_guess = VALUES(is_ai_guess),
		updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		userID,
		result.Availability.Status,
		result.Availability.Probability,
		result.Availability.Reason,
		result.Availability.Confidence,
		result.LifeStatus.Emoji,
		result.LifeStatus.Label,
		strPtrOrNull(result.LifeStatus.Description),
		strPtrOrNull(result.Mood),
		strPtrOrNull(result.Activity),
		strPtrOrNull(result.Context),
		strPtrOrNull(result.LifeStatus.GifURL),
		strPtrOrNull(result.LifeStatus.GiphyQuery),
		result.LifeStatus.UseGif,
		result.IsAIGuess,
	)
	return err
}

// strPtrOrNull 将字符串转换为指针，空字符串返回 nil
func strPtrOrNull(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ClearAnalysisCache 清除用户的分析缓存（用户清除日程时调用）
func (r *MemoryRepository) ClearAnalysisCache(ctx context.Context, userID string) error {
	query := `DELETE FROM user_analysis_cache WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// GetAnalysisCache 获取分析缓存
func (r *MemoryRepository) GetAnalysisCache(ctx context.Context, userID string) (*model.AnalysisResult, error) {
	var cache UserAnalysisCache
	query := `SELECT * FROM user_analysis_cache WHERE user_id = ?`
	err := r.db.GetContext(ctx, &cache, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &model.AnalysisResult{
		Availability: model.AvailabilityAnalysis{
			Status:      cache.AvailabilityStatus,
			Probability: cache.AvailabilityProbability,
			Reason:      cache.AvailabilityReason,
			Confidence:  cache.AvailabilityConfidence,
		},
		LifeStatus: model.LifeStatus{
			Emoji:       cache.LifeStatusEmoji,
			Label:       cache.LifeStatusLabel,
			Description: ptrStrToStr(cache.LifeStatusDescription),
			GifURL:      ptrStrToStr(cache.GifURL),
			GiphyQuery:  ptrStrToStr(cache.GiphyQuery),
			UseGif:      cache.UseGif,
		},
		Mood:      ptrStrToStr(cache.Mood),
		Activity:  ptrStrToStr(cache.Activity),
		Context:   ptrStrToStr(cache.Context),
		UpdatedAt: cache.UpdatedAt, // 包含更新时间用于时效检查
		IsAIGuess: cache.IsAIGuess, // 是否为 AI 推测的状态
	}, nil
}

// ptrStrToStr 将字符串指针转换为字符串
func ptrStrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// GetAnalysisCacheByUserIDs 批量获取分析缓存
func (r *MemoryRepository) GetAnalysisCacheByUserIDs(ctx context.Context, userIDs []string) (map[string]*model.AnalysisResult, error) {
	if len(userIDs) == 0 {
		return map[string]*model.AnalysisResult{}, nil
	}

	query, args, err := sqlx.In(`SELECT * FROM user_analysis_cache WHERE user_id IN (?)`, userIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var caches []UserAnalysisCache
	err = r.db.SelectContext(ctx, &caches, query, args...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*model.AnalysisResult)
	for _, cache := range caches {
		result[cache.UserID] = &model.AnalysisResult{
			Availability: model.AvailabilityAnalysis{
				Status:      cache.AvailabilityStatus,
				Probability: cache.AvailabilityProbability,
				Reason:      cache.AvailabilityReason,
				Confidence:  cache.AvailabilityConfidence,
			},
			LifeStatus: model.LifeStatus{
				Emoji:       cache.LifeStatusEmoji,
				Label:       cache.LifeStatusLabel,
				Description: ptrStrToStr(cache.LifeStatusDescription),
				GifURL:      ptrStrToStr(cache.GifURL),
				GiphyQuery:  ptrStrToStr(cache.GiphyQuery),
				UseGif:      cache.UseGif,
			},
			Mood:      ptrStrToStr(cache.Mood),
			Activity:  ptrStrToStr(cache.Activity),
			Context:   ptrStrToStr(cache.Context),
			UpdatedAt: cache.UpdatedAt, // 包含更新时间用于时效检查
			IsAIGuess: cache.IsAIGuess, // 是否为 AI 推测的状态
		}
	}
	return result, nil
}

// ========== ConversationMemory 操作 ==========

// GetConversationMemory 获取会话记忆
func (r *MemoryRepository) GetConversationMemory(ctx context.Context, conversationID string) (*model.ConversationMemory, error) {
	var memory model.ConversationMemory
	query := `SELECT * FROM conversation_memories WHERE conversation_id = ?`
	err := r.db.GetContext(ctx, &memory, query, conversationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &memory, nil
}

// CreateConversationMemory 创建会话记忆
func (r *MemoryRepository) CreateConversationMemory(ctx context.Context, memory *model.ConversationMemory) error {
	query := `INSERT INTO conversation_memories (conversation_id, memories, summary, total_messages)
              VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		memory.ConversationID,
		memory.Memories,
		memory.Summary,
		memory.TotalMessages,
	)
	return err
}

// UpdateConversationMemory 更新会话记忆
func (r *MemoryRepository) UpdateConversationMemory(ctx context.Context, memory *model.ConversationMemory) error {
	query := `UPDATE conversation_memories SET
              memories = ?,
              summary = ?,
              last_summary_at = ?,
              total_messages = ?,
              updated_at = NOW()
              WHERE conversation_id = ?`
	_, err := r.db.ExecContext(ctx, query,
		memory.Memories,
		memory.Summary,
		memory.LastSummaryAt,
		memory.TotalMessages,
		memory.ConversationID,
	)
	return err
}

// UpsertConversationMemory 插入或更新会话记忆
func (r *MemoryRepository) UpsertConversationMemory(ctx context.Context, memory *model.ConversationMemory) error {
	query := `INSERT INTO conversation_memories (conversation_id, memories, summary, total_messages)
              VALUES (?, ?, ?, ?)
              ON DUPLICATE KEY UPDATE
              memories = VALUES(memories),
              summary = VALUES(summary),
              total_messages = VALUES(total_messages),
              updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		memory.ConversationID,
		memory.Memories,
		memory.Summary,
		memory.TotalMessages,
	)
	return err
}

// UpdateConversationSummary 更新对话总结
func (r *MemoryRepository) UpdateConversationSummary(ctx context.Context, conversationID, summary string) error {
	query := `UPDATE conversation_memories SET
              summary = ?,
              last_summary_at = NOW(),
              updated_at = NOW()
              WHERE conversation_id = ?`
	_, err := r.db.ExecContext(ctx, query, summary, conversationID)
	return err
}

// IncrementMessageCount 增加消息计数
func (r *MemoryRepository) IncrementMessageCount(ctx context.Context, conversationID string) error {
	query := `INSERT INTO conversation_memories (conversation_id, total_messages)
              VALUES (?, 1)
              ON DUPLICATE KEY UPDATE
              total_messages = total_messages + 1,
              updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query, conversationID)
	return err
}

// ========== 福尔摩斯分析缓存操作 ==========

// HolmesAnalysisCacheRow 福尔摩斯分析缓存数据库行
type HolmesAnalysisCacheRow struct {
	ID                  int64     `db:"id"`
	UserID              string    `db:"user_id"`
	RawClues            string    `db:"raw_clues"`
	Features            string    `db:"features"`
	ReasoningModel      string    `db:"reasoning_model"`
	ReasoningThinking   string    `db:"reasoning_thinking"`
	ReasoningConclusion string    `db:"reasoning_conclusion"`
	ResultAvailable     bool      `db:"result_available"`
	ResultProbability   int       `db:"result_probability"`
	ResultConfidence    string    `db:"result_confidence"`
	ResultSummary       string    `db:"result_summary"`
	ResultEmoji         string    `db:"result_emoji"`
	InputHash           string    `db:"input_hash"`
	ExpiresAt           time.Time `db:"expires_at"`
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
}

// SaveHolmesCache 保存福尔摩斯分析缓存
func (r *MemoryRepository) SaveHolmesCache(ctx context.Context, userID string, result *model.HolmesResult, inputHash string, ttl time.Duration) error {
	// 序列化 RawData 和 Features
	rawCluesJSON, _ := json.Marshal(result.RawData)
	featuresJSON, _ := json.Marshal(result.Features)

	expiresAt := time.Now().Add(ttl)

	query := `INSERT INTO holmes_analysis_cache (
		user_id, raw_clues, features,
		reasoning_model, reasoning_thinking, reasoning_conclusion,
		result_available, result_probability, result_confidence, result_summary, result_emoji,
		input_hash, expires_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
		raw_clues = VALUES(raw_clues),
		features = VALUES(features),
		reasoning_model = VALUES(reasoning_model),
		reasoning_thinking = VALUES(reasoning_thinking),
		reasoning_conclusion = VALUES(reasoning_conclusion),
		result_available = VALUES(result_available),
		result_probability = VALUES(result_probability),
		result_confidence = VALUES(result_confidence),
		result_summary = VALUES(result_summary),
		result_emoji = VALUES(result_emoji),
		input_hash = VALUES(input_hash),
		expires_at = VALUES(expires_at),
		updated_at = NOW()`

	reasoningModel := ""
	reasoningThinking := ""
	reasoningConclusion := ""
	if result.Reasoning != nil {
		reasoningModel = result.Reasoning.Model
		reasoningThinking = result.Reasoning.Thinking
		reasoningConclusion = result.Reasoning.Conclusion
	}

	_, err := r.db.ExecContext(ctx, query,
		userID,
		string(rawCluesJSON),
		string(featuresJSON),
		reasoningModel,
		reasoningThinking,
		reasoningConclusion,
		result.Result.Available,
		result.Result.Probability,
		result.Result.Confidence,
		result.Result.Summary,
		"", // emoji 可以从 summary 推断
		inputHash,
		expiresAt,
	)
	return err
}

// GetHolmesCache 获取福尔摩斯分析缓存
func (r *MemoryRepository) GetHolmesCache(ctx context.Context, userID string) (*model.HolmesResult, error) {
	var row HolmesAnalysisCacheRow
	query := `SELECT * FROM holmes_analysis_cache WHERE user_id = ? AND expires_at > NOW()`
	err := r.db.GetContext(ctx, &row, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// 反序列化
	var rawData *model.HolmesClue
	var features *model.HolmesFeatures
	if row.RawClues != "" {
		json.Unmarshal([]byte(row.RawClues), &rawData)
	}
	if row.Features != "" {
		json.Unmarshal([]byte(row.Features), &features)
	}

	result := &model.HolmesResult{
		RawData:  rawData,
		Features: features,
		Reasoning: &model.HolmesReasoning{
			Model:      row.ReasoningModel,
			Thinking:   row.ReasoningThinking,
			Conclusion: row.ReasoningConclusion,
		},
		GeneratedAt: row.UpdatedAt,
	}
	result.Result.Available = row.ResultAvailable
	result.Result.Probability = row.ResultProbability
	result.Result.Confidence = row.ResultConfidence
	result.Result.Summary = row.ResultSummary

	return result, nil
}

// GetHolmesCacheByUserIDs 批量获取福尔摩斯分析缓存
func (r *MemoryRepository) GetHolmesCacheByUserIDs(ctx context.Context, userIDs []string) (map[string]*model.HolmesResult, error) {
	if len(userIDs) == 0 {
		return map[string]*model.HolmesResult{}, nil
	}

	query, args, err := sqlx.In(`SELECT * FROM holmes_analysis_cache WHERE user_id IN (?) AND expires_at > NOW()`, userIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var rows []HolmesAnalysisCacheRow
	err = r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*model.HolmesResult)
	for _, row := range rows {
		var rawData *model.HolmesClue
		var features *model.HolmesFeatures
		if row.RawClues != "" {
			json.Unmarshal([]byte(row.RawClues), &rawData)
		}
		if row.Features != "" {
			json.Unmarshal([]byte(row.Features), &features)
		}

		holmesResult := &model.HolmesResult{
			RawData:  rawData,
			Features: features,
			Reasoning: &model.HolmesReasoning{
				Model:      row.ReasoningModel,
				Thinking:   row.ReasoningThinking,
				Conclusion: row.ReasoningConclusion,
			},
			GeneratedAt: row.UpdatedAt,
		}
		holmesResult.Result.Available = row.ResultAvailable
		holmesResult.Result.Probability = row.ResultProbability
		holmesResult.Result.Confidence = row.ResultConfidence
		holmesResult.Result.Summary = row.ResultSummary

		result[row.UserID] = holmesResult
	}
	return result, nil
}

// ========== 用户状态反馈操作（用于学习）==========

// SaveStatusFeedback 保存状态反馈
func (r *MemoryRepository) SaveStatusFeedback(ctx context.Context, userID string, historyID int64, predictedAvailable bool, predictedProbability int, predictedConfidence string, actualAvailable bool, feedbackSource string) error {
	query := `INSERT INTO user_status_feedback (
		user_id, status_history_id,
		predicted_available, predicted_probability, predicted_confidence,
		actual_available, feedback_source
	) VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		userID,
		historyID,
		predictedAvailable,
		predictedProbability,
		predictedConfidence,
		actualAvailable,
		feedbackSource,
	)
	return err
}

// GetFeedbackByTimeSlot 获取特定时间槽的反馈数据（用于 Few-shot 学习）
func (r *MemoryRepository) GetFeedbackByTimeSlot(ctx context.Context, userID string, dayOfWeek, hourOfDay, limit int) ([]map[string]interface{}, error) {
	query := `SELECT f.*, h.raw_data
		FROM user_status_feedback f
		JOIN status_histories h ON f.status_history_id = h.id
		WHERE f.user_id = ?
		AND h.day_of_week = ?
		AND h.hour_of_day = ?
		ORDER BY f.created_at DESC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, userID, dayOfWeek, hourOfDay, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, historyID int64
		var feedbackUserID, confidence, source, rawData string
		var predictedAvailable, actualAvailable bool
		var probability int
		var createdAt time.Time
		var weight float64

		err := rows.Scan(&id, &feedbackUserID, &historyID, &predictedAvailable, &probability, &confidence, &actualAvailable, &source, &weight, &createdAt, &rawData)
		if err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"raw_data":              rawData,
			"predicted_available":   predictedAvailable,
			"predicted_probability": probability,
			"actual_available":      actualAvailable,
			"feedback_source":       source,
		})
	}

	return results, nil
}

// ========== 用户时间槽统计操作 ==========

// UpdateTimeSlotStats 更新时间槽统计
func (r *MemoryRepository) UpdateTimeSlotStats(ctx context.Context, userID string, dayOfWeek, hourOfDay int, available bool, locationType, activityType string, screenDuration int) error {
	availableInt := 0
	if available {
		availableInt = 1
	}

	query := `INSERT INTO user_time_slot_stats (
		user_id, day_of_week, hour_of_day,
		sample_count, available_count, available_rate,
		common_location_type, common_activity_type, avg_screen_duration_mins
	) VALUES (?, ?, ?, 1, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
		sample_count = sample_count + 1,
		available_count = available_count + ?,
		available_rate = (available_count + ?) * 100 / (sample_count + 1),
		common_location_type = COALESCE(?, common_location_type),
		common_activity_type = COALESCE(?, common_activity_type),
		avg_screen_duration_mins = (avg_screen_duration_mins * sample_count + ?) / (sample_count + 1),
		updated_at = NOW()`

	availableRate := availableInt * 100 // 第一条记录的有空率

	_, err := r.db.ExecContext(ctx, query,
		userID, dayOfWeek, hourOfDay,
		availableInt, availableRate, locationType, activityType, screenDuration,
		availableInt, availableInt, locationType, activityType, screenDuration,
	)
	return err
}

// GetTimeSlotStats 获取时间槽统计
func (r *MemoryRepository) GetTimeSlotStats(ctx context.Context, userID string, dayOfWeek, hourOfDay int) (int, int, error) {
	var sampleCount, availableRate int
	query := `SELECT sample_count, available_rate FROM user_time_slot_stats WHERE user_id = ? AND day_of_week = ? AND hour_of_day = ?`
	err := r.db.QueryRowContext(ctx, query, userID, dayOfWeek, hourOfDay).Scan(&sampleCount, &availableRate)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, 50, nil // 默认50%
		}
		return 0, 0, err
	}
	return sampleCount, availableRate, nil
}

// ========== 用户状态记忆操作（训练 AI）==========

// SaveUserStatusMemory 保存用户选择的状态记忆
func (r *MemoryRepository) SaveUserStatusMemory(ctx context.Context, memory *model.UserStatusMemory) error {
	source := memory.Source
	if source == "" {
		source = "unknown"
	}
	query := `INSERT INTO user_status_memory (user_id, emoji, status, context, source, created_at)
              VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		memory.UserID,
		memory.Emoji,
		memory.Status,
		memory.Context,
		source,
		time.Now(),
	)
	return err
}

// GetRecentUserStatusMemory 获取用户最近的状态记忆
func (r *MemoryRepository) GetRecentUserStatusMemory(ctx context.Context, userID string, limit int) ([]*model.UserStatusMemory, error) {
	var memories []*model.UserStatusMemory
	query := `SELECT id, user_id, emoji, status, context, source, created_at
              FROM user_status_memory
              WHERE user_id = ?
              ORDER BY created_at DESC
              LIMIT ?`
	err := r.db.SelectContext(ctx, &memories, query, userID, limit)
	if err != nil {
		return nil, err
	}
	return memories, nil
}

// GetUserStatusMemoryCount 获取用户状态记忆数量
func (r *MemoryRepository) GetUserStatusMemoryCount(ctx context.Context, userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM user_status_memory WHERE user_id = ?`
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}

// UpdateLifeStatus 更新用户的生活状态（缓存表）
func (r *MemoryRepository) UpdateLifeStatus(ctx context.Context, userID, emoji, label string) error {
	query := `INSERT INTO user_analysis_cache (user_id, life_status_emoji, life_status_label)
              VALUES (?, ?, ?)
              ON DUPLICATE KEY UPDATE
              life_status_emoji = VALUES(life_status_emoji),
              life_status_label = VALUES(life_status_label),
              updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query, userID, emoji, label)
	return err
}

// ========== 推断纠正记录操作 ==========

// SaveInferenceCorrection 保存推断纠正记录
func (r *MemoryRepository) SaveInferenceCorrection(ctx context.Context, correction *model.InferenceCorrection) error {
	query := `INSERT INTO inference_corrections (
		user_id, original_emoji, original_activity,
		corrected_emoji, corrected_activity, corrected_place,
		day_of_week, hour_of_day, device_context, location_type
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		correction.UserID,
		correction.OriginalEmoji,
		correction.OriginalActivity,
		correction.CorrectedEmoji,
		correction.CorrectedActivity,
		correction.CorrectedPlace,
		correction.DayOfWeek,
		correction.HourOfDay,
		correction.DeviceContext,
		correction.LocationType,
	)
	return err
}

// GetTimeslotCorrections 获取指定时段的纠正聚合（30天内同时段 GROUP BY）
func (r *MemoryRepository) GetTimeslotCorrections(ctx context.Context, userID string, dayOfWeek, hourOfDay int) ([]model.TimeslotCorrectionSummary, error) {
	query := `SELECT original_emoji, original_activity,
		corrected_emoji, corrected_activity, corrected_place,
		COUNT(*) as count
		FROM inference_corrections
		WHERE user_id = ?
		AND day_of_week = ?
		AND hour_of_day = ?
		AND created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
		GROUP BY original_emoji, original_activity, corrected_emoji, corrected_activity, corrected_place
		ORDER BY count DESC
		LIMIT 10`
	var summaries []model.TimeslotCorrectionSummary
	err := r.db.SelectContext(ctx, &summaries, query, userID, dayOfWeek, hourOfDay)
	if err != nil {
		return nil, err
	}
	return summaries, nil
}

// ========== 日维度活动摘要操作 ==========

// UpsertDailyActivitySummary 插入或更新日维度活动摘要
func (r *MemoryRepository) UpsertDailyActivitySummary(ctx context.Context, summary *model.DailyActivitySummary) error {
	query := `INSERT INTO daily_activity_summaries (
		user_id, summary_date, home_hours, work_hours, transit_hours,
		screen_active_hours, total_steps, most_active_period, sample_count, text_summary
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
		home_hours = VALUES(home_hours),
		work_hours = VALUES(work_hours),
		transit_hours = VALUES(transit_hours),
		screen_active_hours = VALUES(screen_active_hours),
		total_steps = VALUES(total_steps),
		most_active_period = VALUES(most_active_period),
		sample_count = VALUES(sample_count),
		text_summary = VALUES(text_summary)`
	_, err := r.db.ExecContext(ctx, query,
		summary.UserID,
		summary.SummaryDate,
		summary.HomeHours,
		summary.WorkHours,
		summary.TransitHours,
		summary.ScreenActiveHours,
		summary.TotalSteps,
		summary.MostActivePeriod,
		summary.SampleCount,
		summary.TextSummary,
	)
	return err
}

// ========== 推断日志操作 ==========

// SaveInferenceLog 保存推断日志
func (r *MemoryRepository) SaveInferenceLog(ctx context.Context, log *model.InferenceLog) error {
	toolsJSON := "[]"
	if len(log.ToolsUsed) > 0 {
		if data, err := json.Marshal(log.ToolsUsed); err == nil {
			toolsJSON = string(data)
		}
	}

	query := `INSERT INTO inference_logs (
		user_id, timestamp, sensor_data, tools_used, asked_user,
		initial_result, final_result, user_corrected, confidence, iterations,
		latency_ms, trigger_source, context_sections, thinking_tokens
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		log.UserID,
		log.Timestamp,
		log.SensorData,
		toolsJSON,
		log.AskedUser,
		log.InitialResult,
		log.FinalResult,
		log.UserCorrected,
		log.Confidence,
		log.Iterations,
		log.LatencyMs,
		log.TriggerSource,
		log.ContextSections,
		log.ThinkingTokens,
	)
	return err
}

// MarkInferenceLogCorrected 标记最近的推断日志为已纠正
func (r *MemoryRepository) MarkInferenceLogCorrected(ctx context.Context, userID string, finalResult string) error {
	query := `UPDATE inference_logs
		SET user_corrected = TRUE, final_result = ?
		WHERE user_id = ? AND timestamp > DATE_SUB(NOW(), INTERVAL 15 MINUTE)
		ORDER BY timestamp DESC LIMIT 1`
	_, err := r.db.ExecContext(ctx, query, finalResult, userID)
	return err
}

// GetDailyActivitySummary 获取指定日期的活动摘要
func (r *MemoryRepository) GetDailyActivitySummary(ctx context.Context, userID string, date string) (*model.DailyActivitySummary, error) {
	var summary model.DailyActivitySummary
	query := `SELECT id, user_id, summary_date, home_hours, work_hours, transit_hours,
		screen_active_hours, total_steps, most_active_period, sample_count, text_summary
		FROM daily_activity_summaries
		WHERE user_id = ? AND summary_date = ?`
	err := r.db.GetContext(ctx, &summary, query, userID, date)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &summary, nil
}
