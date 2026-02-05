-- 015_prediction_tasks.sql
-- AI 状态推测任务表

CREATE TABLE IF NOT EXISTS prediction_tasks (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    status ENUM('pending', 'processing', 'completed', 'failed', 'cancelled') DEFAULT 'pending',

    -- 推测时间范围
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    target_date DATE NOT NULL,

    -- 推测结果（JSON 格式）
    predicted_items JSON,       -- AI 推测的时段
    existing_items JSON,        -- 用户已有的时段
    merged_preview JSON,        -- 合并后的预览

    -- 推理过程
    reasoning_content TEXT,     -- 推理内容（Markdown 格式）
    reasoning_steps JSON,       -- 推理步骤（数组）
    user_context JSON,          -- 用户上下文数据

    -- 错误信息
    error_message TEXT,
    progress INT DEFAULT 0,     -- 进度 0-100

    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at DATETIME,      -- 完成时间
    confirmed_at DATETIME,      -- 确认时间

    -- 索引
    INDEX idx_user_status (user_id, status),
    INDEX idx_user_created (user_id, created_at DESC),
    INDEX idx_target_date (user_id, target_date)
);
