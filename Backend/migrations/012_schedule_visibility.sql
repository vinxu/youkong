-- 时刻表可见性控制
-- 允许用户选择谁可以看到他们的状态时刻表

-- 为 status_schedules 表添加可见性字段
ALTER TABLE status_schedules
ADD COLUMN visibility ENUM('all_friends', 'circles', 'private') DEFAULT 'all_friends' COMMENT '可见性: all_friends=所有好友, circles=指定圈子, private=仅自己',
ADD COLUMN circle_ids JSON DEFAULT NULL COMMENT '可见圈子ID列表，visibility=circles时使用';

-- 创建时刻表-圈子关联表（用于快速查询"我能看到谁的状态"）
CREATE TABLE IF NOT EXISTS schedule_circles (
    schedule_id BIGINT NOT NULL,
    circle_id VARCHAR(36) NOT NULL,
    PRIMARY KEY (schedule_id, circle_id),
    INDEX idx_circle (circle_id),
    FOREIGN KEY (schedule_id) REFERENCES status_schedules(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户时刻表偏好设置（默认可见性）
CREATE TABLE IF NOT EXISTS user_schedule_preferences (
    user_id VARCHAR(36) PRIMARY KEY,
    default_visibility ENUM('all_friends', 'circles', 'private') DEFAULT 'all_friends',
    default_circle_ids JSON DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 说明：
-- 1. visibility 字段：
--    - all_friends: 所有好友可见
--    - circles: 只有指定圈子的好友可见
--    - private: 仅自己可见
-- 2. circle_ids JSON 结构示例: ["circle-id-1", "circle-id-2"]
-- 3. schedule_circles 表用于快速查询特定圈子能看到的时刻表
-- 4. user_schedule_preferences 存储用户的默认可见性设置
