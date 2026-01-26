-- 智能记忆系统数据库迁移
-- 创建时间: 2026-01-26

-- 核心记忆表：存储 LLM 提炼的用户行为洞察
CREATE TABLE IF NOT EXISTS core_memories (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,

    -- 行为模式洞察
    behavior_insights TEXT COMMENT '行为模式洞察，如：工作日晚上8-10点通常在家刷手机',

    -- 关键时间规律
    time_patterns TEXT COMMENT '时间规律，如：周五晚上有空概率高',

    -- 地点偏好
    location_preferences TEXT COMMENT '地点偏好，如：周末喜欢去三里屯',

    -- 社交倾向
    social_tendency TEXT COMMENT '社交倾向，如：响应速度快，平均5分钟内回复',

    -- 置信度（数据量越大越高）
    confidence_score INT DEFAULT 0 COMMENT '置信度 0-100',

    -- 样本数量
    sample_count INT DEFAULT 0 COMMENT '累计样本数量',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_user_id (user_id),
    INDEX idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户核心记忆';

-- 状态历史表：保存每次上报的原始数据，用于分析和更新记忆
CREATE TABLE IF NOT EXISTS status_histories (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,

    -- 原始上报数据（JSON 格式存储完整 StatusReport）
    raw_data JSON COMMENT '完整的上报数据',

    -- 上报时的上下文
    day_of_week TINYINT COMMENT '星期几 0-6 (Sunday=0)',
    hour_of_day TINYINT COMMENT '小时 0-23',
    is_weekend BOOLEAN DEFAULT FALSE COMMENT '是否周末',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_user_created (user_id, created_at),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='状态上报历史';

-- 用户实时分析缓存表（可选，主要数据存 Redis，这里做持久化备份）
CREATE TABLE IF NOT EXISTS user_analysis_cache (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,

    -- 最新分析结果
    availability_status VARCHAR(20) COMMENT '有空/忙碌/可能有空',
    availability_probability INT COMMENT '有空概率 0-100',
    availability_reason VARCHAR(50) COMMENT '理由',
    availability_confidence VARCHAR(10) COMMENT 'high/medium/low',

    -- 生活状态
    life_status_emoji VARCHAR(10) COMMENT 'Emoji',
    life_status_label VARCHAR(20) COMMENT '状态标签',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_user_id (user_id),
    INDEX idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户分析缓存';

-- 定期清理旧数据的事件（保留最近 30 天的状态历史）
-- 注意：需要在 MySQL 配置中启用事件调度器 SET GLOBAL event_scheduler = ON;
DELIMITER //
CREATE EVENT IF NOT EXISTS cleanup_old_status_histories
ON SCHEDULE EVERY 1 DAY
STARTS CURRENT_TIMESTAMP
DO
BEGIN
    DELETE FROM status_histories
    WHERE created_at < DATE_SUB(NOW(), INTERVAL 30 DAY);
END//
DELIMITER ;
