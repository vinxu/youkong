-- 用户人设表
CREATE TABLE IF NOT EXISTS user_personas (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL UNIQUE,

    -- 基础人设
    personality TEXT,           -- 性格特点
    speaking_style TEXT,        -- 说话风格
    interests TEXT,             -- 兴趣爱好
    social_habits TEXT,         -- 社交习惯

    -- 语言特征
    common_phrases TEXT,        -- 常用语/口头禅
    emoji_usage VARCHAR(50),    -- emoji使用习惯
    message_length VARCHAR(20), -- 消息长度偏好

    -- 置信度
    confidence_score INT DEFAULT 0,
    sample_count INT DEFAULT 0,

    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 关系画像表
CREATE TABLE IF NOT EXISTS relationships (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    friend_id VARCHAR(36) NOT NULL,

    -- 关系定位
    closeness VARCHAR(20) DEFAULT '还不太熟',      -- 亲密度
    interact_style VARCHAR(20) DEFAULT '未知',     -- 互动风格
    common_topics TEXT,                            -- 常聊话题

    -- 互动特征
    tone_with_this TEXT,                           -- 和这个人说话的语气
    joke_level VARCHAR(20) DEFAULT '低',           -- 开玩笑程度
    shared_memory TEXT,                            -- 共同记忆/经历

    -- 数据量
    message_count INT DEFAULT 0,
    confidence_score INT DEFAULT 0,

    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uk_user_friend (user_id, friend_id),
    INDEX idx_user_id (user_id),
    INDEX idx_friend_id (friend_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 对话上下文表（LLM 会话窗口）
CREATE TABLE IF NOT EXISTS conversation_contexts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    conversation_id VARCHAR(36) NOT NULL UNIQUE,

    messages JSON NOT NULL,              -- LLM 消息历史 [{role, content}, ...]
    token_count INT DEFAULT 0,           -- 估算 token 数
    summary TEXT,                        -- 上下文总结（用于重建）
    last_sync_msg_id VARCHAR(36),        -- 最后同步的消息ID

    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_conversation (conversation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
