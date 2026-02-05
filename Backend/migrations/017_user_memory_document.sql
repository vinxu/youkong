-- 用户记忆文档表
-- 用于存储跨会话的用户记忆，实现个性化上下文
CREATE TABLE IF NOT EXISTS user_memory_documents (
    user_id VARCHAR(36) PRIMARY KEY,
    version INT NOT NULL DEFAULT 1,

    -- 用户偏好（JSON）
    -- {hide_past_events: bool, default_view_days: int, preferred_timezone: string, language: string}
    preferences JSON,

    -- 日程模式（JSON 数组）
    -- [{pattern: string, confidence: float, learned_from: [session_ids]}]
    schedule_patterns JSON,

    -- 交互风格（JSON）
    -- {prefers_brief: bool, confirm_before_save: bool, show_reasoning: bool}
    interaction_style JSON,

    -- 关键事实（JSON 数组）
    -- [{fact: string, source: string, learned_at: timestamp}]
    key_facts JSON,

    -- 会话摘要（JSON 数组，保留最近 N 次）
    -- [{session_id: string, date: timestamp, summary: string, key_points: [string]}]
    session_summaries JSON,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 索引：按更新时间查询
CREATE INDEX idx_user_memory_updated ON user_memory_documents(updated_at);
