-- 增强用户分析缓存表（添加生活状态描述）
-- 创建时间: 2026-02-02

ALTER TABLE user_analysis_cache
ADD COLUMN life_status_description TEXT COMMENT '生活状态描述（如"在家休息放松中"）' AFTER life_status_label;

-- 为更好地支持新的状态推理系统，添加更多字段
ALTER TABLE user_analysis_cache
MODIFY COLUMN availability_reason VARCHAR(255) COMMENT '简短理由（保留用于兼容）',
ADD COLUMN mood VARCHAR(20) COMMENT '心情状态（relaxed/busy/focused/tired等）' AFTER life_status_description,
ADD COLUMN activity VARCHAR(50) COMMENT '当前活动（watching_tv/working/exercising等）' AFTER mood,
ADD COLUMN context TEXT COMMENT '上下文信息（JSON格式，包含时间、地点等）' AFTER activity;
