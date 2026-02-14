-- 019_bookings.sql: 社交预约（Booking）功能
-- 用于支持"约人"流程：邀请 → 接受 → 双向 Book → 行程提醒

CREATE TABLE IF NOT EXISTS bookings (
    id VARCHAR(36) PRIMARY KEY,
    creator_id VARCHAR(36) NOT NULL,
    title VARCHAR(100) NOT NULL,
    emoji VARCHAR(10) DEFAULT '🤝',
    schedule_date DATE NOT NULL,
    start_time VARCHAR(5) NOT NULL,
    end_time VARCHAR(5) NOT NULL,
    location VARCHAR(200) DEFAULT '',
    note TEXT,
    status ENUM('pending','confirmed','cancelled') DEFAULT 'pending',
    message_id VARCHAR(36) DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_bookings_creator (creator_id),
    INDEX idx_bookings_date (schedule_date),
    INDEX idx_bookings_status (status)
);

CREATE TABLE IF NOT EXISTS booking_participants (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    booking_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    status ENUM('pending','accepted','declined') DEFAULT 'pending',
    responded_at TIMESTAMP NULL,
    notified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uk_booking_user (booking_id, user_id),
    INDEX idx_bp_user_status (user_id, status),
    FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE
);
