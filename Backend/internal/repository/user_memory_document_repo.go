package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"youkong/internal/model"
)

// UserMemoryDocumentRepository 用户记忆文档数据访问层
type UserMemoryDocumentRepository struct {
	db *sqlx.DB
}

// NewUserMemoryDocumentRepository 创建用户记忆文档数据访问层
func NewUserMemoryDocumentRepository(db *sqlx.DB) *UserMemoryDocumentRepository {
	return &UserMemoryDocumentRepository{db: db}
}

// GetByUserID 根据用户 ID 获取记忆文档
func (r *UserMemoryDocumentRepository) GetByUserID(ctx context.Context, userID string) (*model.UserMemoryDocument, error) {
	var doc model.UserMemoryDocument
	query := `SELECT user_id, version, preferences, schedule_patterns, interaction_style, key_facts, session_summaries, structured_schedule, created_at, updated_at
              FROM user_memory_documents WHERE user_id = ?`
	err := r.db.GetContext(ctx, &doc, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &doc, nil
}

// Create 创建记忆文档
func (r *UserMemoryDocumentRepository) Create(ctx context.Context, doc *model.UserMemoryDocument) error {
	query := `INSERT INTO user_memory_documents (user_id, version, preferences, schedule_patterns, interaction_style, key_facts, session_summaries, structured_schedule, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		doc.UserID,
		1,
		doc.Preferences,
		doc.SchedulePatterns,
		doc.InteractionStyle,
		doc.KeyFacts,
		doc.SessionSummaries,
		doc.StructuredSchedule,
		now,
		now,
	)
	return err
}

// Update 更新记忆文档
func (r *UserMemoryDocumentRepository) Update(ctx context.Context, doc *model.UserMemoryDocument) error {
	query := `UPDATE user_memory_documents SET
              version = version + 1,
              preferences = ?,
              schedule_patterns = ?,
              interaction_style = ?,
              key_facts = ?,
              session_summaries = ?,
              structured_schedule = ?,
              updated_at = ?
              WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query,
		doc.Preferences,
		doc.SchedulePatterns,
		doc.InteractionStyle,
		doc.KeyFacts,
		doc.SessionSummaries,
		doc.StructuredSchedule,
		time.Now(),
		doc.UserID,
	)
	return err
}

// Upsert 插入或更新记忆文档
func (r *UserMemoryDocumentRepository) Upsert(ctx context.Context, doc *model.UserMemoryDocument) error {
	query := `INSERT INTO user_memory_documents (user_id, version, preferences, schedule_patterns, interaction_style, key_facts, session_summaries, structured_schedule, created_at, updated_at)
              VALUES (?, 1, ?, ?, ?, ?, ?, ?, NOW(), NOW())
              ON DUPLICATE KEY UPDATE
              version = version + 1,
              preferences = VALUES(preferences),
              schedule_patterns = VALUES(schedule_patterns),
              interaction_style = VALUES(interaction_style),
              key_facts = VALUES(key_facts),
              session_summaries = VALUES(session_summaries),
              structured_schedule = VALUES(structured_schedule),
              updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		doc.UserID,
		doc.Preferences,
		doc.SchedulePatterns,
		doc.InteractionStyle,
		doc.KeyFacts,
		doc.SessionSummaries,
		doc.StructuredSchedule,
	)
	return err
}

// UpdatePreferences 更新用户偏好
func (r *UserMemoryDocumentRepository) UpdatePreferences(ctx context.Context, userID string, prefs model.UserMemoryPreferences) error {
	// 先检查记录是否存在
	existing, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if existing == nil {
		// 创建新记录
		doc := &model.UserMemoryDocument{
			UserID:      userID,
			Preferences: prefs,
		}
		return r.Create(ctx, doc)
	}

	// 更新偏好
	query := `UPDATE user_memory_documents SET
              preferences = ?,
              version = version + 1,
              updated_at = NOW()
              WHERE user_id = ?`
	_, err = r.db.ExecContext(ctx, query, prefs, userID)
	return err
}

// AddSchedulePattern 添加日程模式
func (r *UserMemoryDocumentRepository) AddSchedulePattern(ctx context.Context, userID string, pattern model.SchedulePattern) error {
	doc, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if doc == nil {
		// 创建新记录
		doc = &model.UserMemoryDocument{
			UserID:           userID,
			SchedulePatterns: model.SchedulePatternList{pattern},
		}
		return r.Create(ctx, doc)
	}

	// 检查是否已存在相似模式
	for i, existing := range doc.SchedulePatterns {
		if existing.Pattern == pattern.Pattern {
			// 更新置信度（取更高值）
			if pattern.Confidence > existing.Confidence {
				doc.SchedulePatterns[i].Confidence = pattern.Confidence
			}
			// 合并学习来源
			doc.SchedulePatterns[i].LearnedFrom = mergeStringSlices(existing.LearnedFrom, pattern.LearnedFrom)
			return r.Update(ctx, doc)
		}
	}

	// 添加新模式
	doc.SchedulePatterns = append(doc.SchedulePatterns, pattern)

	// 限制最大数量
	if len(doc.SchedulePatterns) > model.MaxSchedulePatterns {
		// 按置信度排序，保留高置信度的
		doc.SchedulePatterns = sortAndTruncatePatterns(doc.SchedulePatterns, model.MaxSchedulePatterns)
	}

	return r.Update(ctx, doc)
}

// AddKeyFact 添加关键事实
func (r *UserMemoryDocumentRepository) AddKeyFact(ctx context.Context, userID string, fact model.KeyFact) error {
	doc, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if doc == nil {
		// 创建新记录
		doc = &model.UserMemoryDocument{
			UserID:   userID,
			KeyFacts: model.KeyFactList{fact},
		}
		return r.Create(ctx, doc)
	}

	// 检查是否已存在相同事实
	for _, existing := range doc.KeyFacts {
		if existing.Fact == fact.Fact {
			return nil // 已存在，不重复添加
		}
	}

	// 添加新事实
	doc.KeyFacts = append(doc.KeyFacts, fact)

	// 限制最大数量
	if len(doc.KeyFacts) > model.MaxKeyFacts {
		// 保留最新的
		doc.KeyFacts = doc.KeyFacts[len(doc.KeyFacts)-model.MaxKeyFacts:]
	}

	return r.Update(ctx, doc)
}

// AddSessionSummary 添加会话摘要
func (r *UserMemoryDocumentRepository) AddSessionSummary(ctx context.Context, userID string, summary model.SessionSummary) error {
	doc, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if doc == nil {
		// 创建新记录
		doc = &model.UserMemoryDocument{
			UserID:           userID,
			SessionSummaries: model.SessionSummaryList{summary},
		}
		return r.Create(ctx, doc)
	}

	// 添加新摘要到列表开头
	doc.SessionSummaries = append(model.SessionSummaryList{summary}, doc.SessionSummaries...)

	// 限制最大数量
	if len(doc.SessionSummaries) > model.MaxSessionSummaries {
		doc.SessionSummaries = doc.SessionSummaries[:model.MaxSessionSummaries]
	}

	return r.Update(ctx, doc)
}

// UpdateInteractionStyle 更新交互风格
func (r *UserMemoryDocumentRepository) UpdateInteractionStyle(ctx context.Context, userID string, style model.UserInteractionStyle) error {
	doc, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if doc == nil {
		// 创建新记录
		doc = &model.UserMemoryDocument{
			UserID:           userID,
			InteractionStyle: style,
		}
		return r.Create(ctx, doc)
	}

	// 更新交互风格
	query := `UPDATE user_memory_documents SET
              interaction_style = ?,
              version = version + 1,
              updated_at = NOW()
              WHERE user_id = ?`
	_, err = r.db.ExecContext(ctx, query, style, userID)
	return err
}

// Delete 删除记忆文档
func (r *UserMemoryDocumentRepository) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM user_memory_documents WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// ========== 辅助函数 ==========

// mergeStringSlices 合并两个字符串切片，去重
func mergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(a)+len(b))

	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}

// sortAndTruncatePatterns 按置信度排序并截断
func sortAndTruncatePatterns(patterns model.SchedulePatternList, maxCount int) model.SchedulePatternList {
	if len(patterns) <= maxCount {
		return patterns
	}

	// 简单冒泡排序（数量小，性能不是问题）
	for i := 0; i < len(patterns)-1; i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].Confidence > patterns[i].Confidence {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}

	return patterns[:maxCount]
}
