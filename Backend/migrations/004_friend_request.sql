-- YouKong Database Schema
-- Version: 004_friend_request
-- Description: 好友请求功能

-- 好友请求表
CREATE TABLE IF NOT EXISTS friend_requests (
    id VARCHAR(36) PRIMARY KEY,
    from_user_id VARCHAR(36) NOT NULL,      -- 发起请求的用户
    to_user_id VARCHAR(36) NOT NULL,        -- 接收请求的用户
    message VARCHAR(200),                    -- 附言（可选）
    status ENUM('PENDING', 'ACCEPTED', 'REJECTED', 'CANCELLED') DEFAULT 'PENDING',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (from_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (to_user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY uk_pending_request (from_user_id, to_user_id, status),
    INDEX idx_from_user (from_user_id),
    INDEX idx_to_user (to_user_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 更新 friendships 表的 source 枚举，增加 CONTACTS 类型
ALTER TABLE friendships MODIFY COLUMN source ENUM('INVITATION', 'SEARCH', 'MANUAL', 'CONTACTS', 'REQUEST') NOT NULL;
