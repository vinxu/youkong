-- YouKong Database Schema
-- Version: 002_wechat_invite
-- Description: 微信登录、邀请系统、好友关系

-- 1. 微信用户绑定表
CREATE TABLE IF NOT EXISTS wechat_users (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    openid VARCHAR(64) NOT NULL UNIQUE,
    unionid VARCHAR(64),
    nickname VARCHAR(100),
    avatar VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_openid (openid),
    INDEX idx_unionid (unionid),
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. 邀请链接表
CREATE TABLE IF NOT EXISTS invitations (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(16) NOT NULL UNIQUE,
    inviter_id VARCHAR(36) NOT NULL,
    circle_id VARCHAR(36),
    max_uses INT DEFAULT 100,
    use_count INT DEFAULT 0,
    expires_at TIMESTAMP NULL,
    status ENUM('ACTIVE', 'DISABLED', 'EXPIRED') DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (circle_id) REFERENCES circles(id) ON DELETE SET NULL,
    INDEX idx_code (code),
    INDEX idx_inviter_id (inviter_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. 邀请记录表
CREATE TABLE IF NOT EXISTS invitation_records (
    id VARCHAR(36) PRIMARY KEY,
    invitation_id VARCHAR(36) NOT NULL,
    inviter_id VARCHAR(36) NOT NULL,
    invitee_id VARCHAR(36) NOT NULL,
    circle_id VARCHAR(36),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (invitation_id) REFERENCES invitations(id) ON DELETE CASCADE,
    FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (invitee_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (circle_id) REFERENCES circles(id) ON DELETE SET NULL,
    UNIQUE KEY uk_invitation_invitee (invitation_id, invitee_id),
    INDEX idx_inviter_id (inviter_id),
    INDEX idx_invitee_id (invitee_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. 好友关系表（双向存储）
CREATE TABLE IF NOT EXISTS friendships (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    friend_id VARCHAR(36) NOT NULL,
    source ENUM('INVITATION', 'SEARCH', 'MANUAL') NOT NULL,
    invitation_id VARCHAR(36),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_friend (user_id, friend_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (friend_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (invitation_id) REFERENCES invitations(id) ON DELETE SET NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_friend_id (friend_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 5. 修改 users 表（允许手机号为空，添加微信绑定标记）
ALTER TABLE users MODIFY COLUMN phone VARCHAR(20) NULL;
ALTER TABLE users MODIFY COLUMN phone_hash VARCHAR(64) NULL;
ALTER TABLE users ADD COLUMN wechat_bound BOOLEAN DEFAULT FALSE AFTER avatar;
ALTER TABLE users DROP INDEX phone;
ALTER TABLE users ADD UNIQUE INDEX idx_phone (phone);
