-- 为状态同步系统添加 AI 推测标记
-- 创建时间: 2026-02-05
-- 用于区分 AI 推测的状态和用户主动设置的状态

-- 为 user_analysis_cache 添加 AI 推测标记
ALTER TABLE user_analysis_cache
ADD COLUMN is_ai_guess TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否为 AI 推测的状态（0=用户设置，1=AI推测）';

-- 说明：
-- 1. is_ai_guess = 0: 用户通过语音主动设置的状态
-- 2. is_ai_guess = 1: AI 根据设备数据推测的状态
-- 3. 用户主动设置的状态优先级高于 AI 推测
-- 4. status_schedules 表的 items JSON 中也会包含 is_ai_guess 字段
