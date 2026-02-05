-- 016_user_schedule_preferences.sql
-- 用户时刻表偏好设置

CREATE TABLE IF NOT EXISTS user_schedule_preferences (
    user_id VARCHAR(36) PRIMARY KEY COMMENT '用户ID',
    hide_past_events BOOLEAN DEFAULT FALSE COMMENT '是否隐藏过去的日程',
    default_visibility VARCHAR(20) DEFAULT 'all_friends' COMMENT '默认可见性',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户时刻表偏好设置';
