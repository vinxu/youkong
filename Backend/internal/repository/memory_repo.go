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
	CreatedAt               time.Time `db:"created_at"`
	UpdatedAt               time.Time `db:"updated_at"`
}

// SaveAnalysisCache 保存分析缓存
func (r *MemoryRepository) SaveAnalysisCache(ctx context.Context, userID string, result *model.AnalysisResult) error {
	query := `INSERT INTO user_analysis_cache (user_id, availability_status, availability_probability, availability_reason, availability_confidence, life_status_emoji, life_status_label)
              VALUES (?, ?, ?, ?, ?, ?, ?)
              ON DUPLICATE KEY UPDATE
              availability_status = VALUES(availability_status),
              availability_probability = VALUES(availability_probability),
              availability_reason = VALUES(availability_reason),
              availability_confidence = VALUES(availability_confidence),
              life_status_emoji = VALUES(life_status_emoji),
              life_status_label = VALUES(life_status_label),
              updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		userID,
		result.Availability.Status,
		result.Availability.Probability,
		result.Availability.Reason,
		result.Availability.Confidence,
		result.LifeStatus.Emoji,
		result.LifeStatus.Label,
	)
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
			Emoji: cache.LifeStatusEmoji,
			Label: cache.LifeStatusLabel,
		},
	}, nil
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
				Emoji: cache.LifeStatusEmoji,
				Label: cache.LifeStatusLabel,
			},
		}
	}
	return result, nil
}
