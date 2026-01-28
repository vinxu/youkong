-- 福尔摩斯推理框架数据库迁移
-- 创建时间: 2026-01-28

-- 用户状态历史扩展表（用于 Few-shot 学习）
-- 在原有 status_histories 基础上添加反馈数据
CREATE TABLE IF NOT EXISTS user_status_feedback (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    status_history_id BIGINT COMMENT '关联 status_histories 表的 ID',

    -- 预测结果
    predicted_available BOOLEAN COMMENT '预测是否有空',
    predicted_probability INT COMMENT '预测的有空概率 0-100',
    predicted_confidence VARCHAR(10) COMMENT '置信度 high/medium/low',

    -- 真实结果（用户反馈或推断）
    actual_available BOOLEAN COMMENT '实际是否有空',
    feedback_source VARCHAR(20) DEFAULT 'implicit' COMMENT 'implicit/explicit/inferred',

    -- 学习权重
    learning_weight FLOAT DEFAULT 1.0 COMMENT '学习权重（显式反馈权重更高）',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_user_time (user_id, created_at),
    INDEX idx_status_history (status_history_id),
    INDEX idx_feedback_source (feedback_source)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户状态反馈（用于学习）';

-- 福尔摩斯分析缓存表
CREATE TABLE IF NOT EXISTS holmes_analysis_cache (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,

    -- 原始线索（JSON 格式）
    raw_clues JSON COMMENT 'HolmesClue 结构',

    -- 特征层（JSON 格式）
    features JSON COMMENT 'HolmesFeatures 结构',

    -- 推理过程
    reasoning_model VARCHAR(50) COMMENT '使用的模型',
    reasoning_thinking TEXT COMMENT '思考过程（<think>标签内容）',
    reasoning_conclusion TEXT COMMENT '推理结论',

    -- 最终结果
    result_available BOOLEAN COMMENT '是否有空',
    result_probability INT COMMENT '有空概率 0-100',
    result_confidence VARCHAR(10) COMMENT '置信度',
    result_summary VARCHAR(50) COMMENT '简短总结',
    result_emoji VARCHAR(10) COMMENT '状态 Emoji',

    -- 缓存控制
    input_hash VARCHAR(64) COMMENT '输入数据的哈希，用于缓存判断',
    expires_at TIMESTAMP COMMENT '缓存过期时间',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_user_id (user_id),
    INDEX idx_input_hash (input_hash),
    INDEX idx_expires_at (expires_at),
    INDEX idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='福尔摩斯分析缓存';

-- 用户时间槽统计表（用于快速查询用户在特定时间段的历史规律）
CREATE TABLE IF NOT EXISTS user_time_slot_stats (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,

    -- 时间槽定义
    day_of_week TINYINT NOT NULL COMMENT '星期几 0-6',
    hour_of_day TINYINT NOT NULL COMMENT '小时 0-23',

    -- 统计数据
    sample_count INT DEFAULT 0 COMMENT '样本数量',
    available_count INT DEFAULT 0 COMMENT '有空次数',
    available_rate INT DEFAULT 0 COMMENT '有空率 0-100',

    -- 常见特征（JSON 格式）
    common_location_type VARCHAR(20) COMMENT '最常见位置类型',
    common_activity_type VARCHAR(20) COMMENT '最常见活动类型',
    avg_screen_duration_mins INT COMMENT '平均屏幕使用时长',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_user_time_slot (user_id, day_of_week, hour_of_day),
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户时间槽统计';

-- 定期清理过期的福尔摩斯分析缓存
DELIMITER //
CREATE EVENT IF NOT EXISTS cleanup_expired_holmes_cache
ON SCHEDULE EVERY 1 HOUR
STARTS CURRENT_TIMESTAMP
DO
BEGIN
    DELETE FROM holmes_analysis_cache
    WHERE expires_at < NOW();
END//
DELIMITER ;

-- 定期清理旧的反馈数据（保留最近 90 天）
DELIMITER //
CREATE EVENT IF NOT EXISTS cleanup_old_feedback
ON SCHEDULE EVERY 1 DAY
STARTS CURRENT_TIMESTAMP
DO
BEGIN
    DELETE FROM user_status_feedback
    WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY);
END//
DELIMITER ;
