CREATE TABLE IF NOT EXISTS interactions (
    id VARCHAR(36) PRIMARY KEY,
    sender_id VARCHAR(36) NOT NULL,
    receiver_id VARCHAR(36) NOT NULL,
    action_emoji VARCHAR(32) NOT NULL,
    action_label VARCHAR(50) NOT NULL,
    action_push_text VARCHAR(200) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_receiver_created (receiver_id, created_at DESC),
    INDEX idx_pair_created (sender_id, receiver_id, created_at DESC)
);
