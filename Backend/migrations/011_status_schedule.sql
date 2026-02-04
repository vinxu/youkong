-- 状态时刻表
-- 存储用户通过语音输入的时刻表，用于定时自动更新状态

CREATE TABLE IF NOT EXISTS status_schedules (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id VARCHAR(36) NOT NULL,
    schedule_date DATE NOT NULL,
    items JSON NOT NULL COMMENT '时刻表条目数组 [{start_time, end_time, emoji, status, executed}]',
    current_index INT DEFAULT 0 COMMENT '当前执行到第几项',
    status ENUM('active', 'completed', 'cancelled') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_user_date (user_id, schedule_date),
    INDEX idx_status (status),
    INDEX idx_active_schedules (status, schedule_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 说明：
-- 1. items JSON 结构示例:
--    [
--      {"start_time": "12:00", "end_time": "13:00", "emoji": "🏨🍚", "status": "在酒店午餐", "executed": false},
--      {"start_time": "13:00", "end_time": "14:00", "emoji": "😴", "status": "午休", "executed": false}
--    ]
-- 2. current_index: 标记当前执行到哪一项，定时任务会检查并执行
-- 3. status: active 表示进行中，completed 表示全部执行完毕，cancelled 表示用户取消
