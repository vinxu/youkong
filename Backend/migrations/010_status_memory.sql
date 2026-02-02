-- 用户状态记忆表
-- 用于记录用户选择的状态，作为后续 AI 推理的训练数据

CREATE TABLE IF NOT EXISTS user_status_memory (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id VARCHAR(36) NOT NULL COMMENT '用户ID',
    emoji VARCHAR(10) NOT NULL COMMENT '状态emoji',
    status VARCHAR(100) NOT NULL COMMENT '状态描述',
    context JSON COMMENT '选择时的设备上下文数据',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX idx_user_time (user_id, created_at) COMMENT '按用户和时间查询'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户状态记忆表';
