-- 推断日志表
CREATE TABLE IF NOT EXISTS inference_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sensor_data JSON COMMENT '输入的传感器数据',
    tools_used JSON COMMENT '使用的工具列表',
    asked_user BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否问了用户',
    initial_result JSON COMMENT '初始推断结果',
    final_result JSON COMMENT '最终结果（修正后）',
    user_corrected BOOLEAN NOT NULL DEFAULT FALSE COMMENT '用户是否修正',
    confidence VARCHAR(10) NOT NULL DEFAULT 'medium' COMMENT '置信度 high/medium/low',
    iterations INT NOT NULL DEFAULT 1 COMMENT 'Agent循环次数',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_timestamp (timestamp),
    INDEX idx_user_confidence (user_id, confidence)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI状态推断日志';
