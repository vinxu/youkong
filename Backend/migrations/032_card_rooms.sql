-- 卡片房间表（每人每天一个房间）
CREATE TABLE IF NOT EXISTS card_rooms (
    id VARCHAR(36) PRIMARY KEY,
    owner_id VARCHAR(36) NOT NULL,
    emoji VARCHAR(20) DEFAULT '',
    status_text VARCHAR(200) DEFAULT '',
    room_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_owner_date (owner_id, room_date)
);

-- 卡片房间成员表
CREATE TABLE IF NOT EXISTS card_room_members (
    room_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (room_id, user_id)
);
