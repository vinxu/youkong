-- 020_city_fields.sql
-- 城市字段架构：常驻城市 + show_city 补全

-- 1. users 表新增常驻城市字段
ALTER TABLE users ADD COLUMN city VARCHAR(50) DEFAULT NULL COMMENT '常驻城市' AFTER avatar;

-- 2. user_schedule_preferences 表补全 show_city 字段（016 遗漏）
ALTER TABLE user_schedule_preferences ADD COLUMN show_city BOOLEAN DEFAULT TRUE COMMENT '是否显示城市';
