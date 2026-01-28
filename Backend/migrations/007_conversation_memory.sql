-- 会话记忆表（每个会话一条记录）
-- 存储 Context Curator 提取的关键信息
CREATE TABLE IF NOT EXISTS conversation_memories (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    conversation_id VARCHAR(36) NOT NULL UNIQUE,

    -- 结构化记忆（JSON 数组，每条带时间戳和上下文）
    memories JSON COMMENT '关键记忆列表: [{type, content, timestamp, time_context, sender, importance}]',

    -- 压缩的历史总结
    summary TEXT COMMENT '对话总结（定期更新）',
    last_summary_at TIMESTAMP NULL COMMENT '上次总结时间',

    -- 统计信息
    total_messages INT DEFAULT 0 COMMENT '总消息数（用于判断是否需要总结）',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_conversation_id (conversation_id),
    INDEX idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
