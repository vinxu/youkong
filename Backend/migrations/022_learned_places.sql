-- 用户学习地点聚类表
-- 每个用户的常去地点（家、公司、常去场所等）
CREATE TABLE IF NOT EXISTS learned_places (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    centroid_lat DOUBLE NOT NULL,
    centroid_lng DOUBLE NOT NULL,
    place_type VARCHAR(32) NOT NULL DEFAULT 'unknown',  -- home, work, frequent, temporary_stay, first_visit
    confidence VARCHAR(16) NOT NULL DEFAULT 'low',       -- high, medium, low
    visit_count INT NOT NULL DEFAULT 0,
    night_count INT NOT NULL DEFAULT 0,                  -- 22:00-07:00 出现的天数
    day_count INT NOT NULL DEFAULT 0,                    -- 07:00-22:00 出现的天数
    typical_names TEXT,                                   -- JSON 数组: 常见地点名称 ["亚朵酒店", "星巴克"]
    first_seen_at DATETIME NOT NULL,
    last_seen_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_learned_places_user (user_id),
    INDEX idx_learned_places_user_type (user_id, place_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
