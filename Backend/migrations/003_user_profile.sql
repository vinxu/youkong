-- 用户画像表（用于更精准的状态推理）
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id VARCHAR(36) PRIMARY KEY,

    -- 职业信息
    occupation_type ENUM(
        'student',          -- 学生
        'office_worker',    -- 上班族（9-5）
        'freelancer',       -- 自由职业
        'shift_worker',     -- 轮班工作
        'entrepreneur',     -- 创业者
        'homemaker',        -- 家庭主妇/夫
        'retired',          -- 退休
        'other'             -- 其他
    ) DEFAULT 'other',

    -- 工作时间规律
    work_schedule ENUM(
        'regular_9to5',     -- 固定 9-5
        'flexible',         -- 弹性工作
        'shift',            -- 轮班
        'irregular',        -- 不规律
        'none'              -- 无固定工作
    ) DEFAULT 'flexible',

    -- 典型工作时间段（JSON 格式，如 {"start": "09:00", "end": "18:00"}）
    typical_work_hours JSON,

    -- 生活方式偏好
    lifestyle_type ENUM(
        'early_bird',       -- 早起型
        'night_owl',        -- 夜猫子
        'balanced'          -- 均衡
    ) DEFAULT 'balanced',

    -- 运动习惯
    exercise_frequency ENUM(
        'daily',            -- 每天
        'regular',          -- 经常（3-5次/周）
        'occasional',       -- 偶尔（1-2次/周）
        'rarely'            -- 很少
    ) DEFAULT 'occasional',

    -- 社交倾向
    social_preference ENUM(
        'very_social',      -- 非常社交
        'moderately_social',-- 适度社交
        'introvert'         -- 内向
    ) DEFAULT 'moderately_social',

    -- 常驻地点（JSON 数组，用于位置判断）
    -- 格式: [{"type": "home", "name": "望京", "lat": 39.998, "lng": 116.474}, ...]
    frequent_locations JSON,

    -- 其他偏好设置
    preferences JSON,

    -- 时间戳
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 索引
CREATE INDEX idx_user_profiles_occupation ON user_profiles(occupation_type);
CREATE INDEX idx_user_profiles_lifestyle ON user_profiles(lifestyle_type);
