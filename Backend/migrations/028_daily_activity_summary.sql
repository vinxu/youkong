CREATE TABLE IF NOT EXISTS daily_activity_summaries (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    summary_date DATE NOT NULL,
    home_hours DECIMAL(4,1) DEFAULT 0,
    work_hours DECIMAL(4,1) DEFAULT 0,
    transit_hours DECIMAL(4,1) DEFAULT 0,
    screen_active_hours DECIMAL(4,1) DEFAULT 0,
    total_steps INT DEFAULT 0,
    most_active_period VARCHAR(20) DEFAULT '',
    sample_count INT DEFAULT 0,
    text_summary VARCHAR(500) DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_date (user_id, summary_date),
    INDEX idx_user_recent (user_id, summary_date DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
