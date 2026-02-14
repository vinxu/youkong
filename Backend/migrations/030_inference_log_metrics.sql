-- 扩展 inference_logs 表，增加推断质量指标字段
ALTER TABLE inference_logs
    ADD COLUMN latency_ms INT NOT NULL DEFAULT 0 COMMENT '推断耗时(毫秒)',
    ADD COLUMN trigger_source VARCHAR(20) NOT NULL DEFAULT 'unknown' COMMENT 'oneclick|auto_scheduler|sync',
    ADD COLUMN context_sections JSON COMMENT '上下文段填充情况',
    ADD COLUMN thinking_tokens INT NOT NULL DEFAULT 0 COMMENT 'thinking token估算',
    ADD INDEX idx_trigger_source (trigger_source),
    ADD INDEX idx_created_user (user_id, created_at);
