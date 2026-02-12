-- 024: 为 user_status_memory 添加 source 字段，标记数据来源
-- user_confirmed: 用户主动设置/确认
-- ai_auto: AI 自动推测
-- ai_oneclick: 一键推断
-- unknown: 历史数据（无来源标记）

ALTER TABLE user_status_memory
ADD COLUMN source VARCHAR(20) NOT NULL DEFAULT 'unknown';
