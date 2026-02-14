CREATE TABLE IF NOT EXISTS inference_corrections (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    original_emoji VARCHAR(20) NOT NULL DEFAULT '',
    original_activity VARCHAR(100) NOT NULL DEFAULT '',
    corrected_emoji VARCHAR(20) NOT NULL,
    corrected_activity VARCHAR(100) NOT NULL,
    corrected_place VARCHAR(100) NOT NULL DEFAULT '',
    day_of_week TINYINT NOT NULL,
    hour_of_day TINYINT NOT NULL,
    device_context JSON,
    location_type VARCHAR(20) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_timeslot (user_id, day_of_week, hour_of_day, created_at),
    INDEX idx_user_created (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
