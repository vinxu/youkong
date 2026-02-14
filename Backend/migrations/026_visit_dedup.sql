ALTER TABLE learned_places
    ADD COLUMN last_counted_at DATETIME DEFAULT NULL AFTER day_count;

-- 回填：用 last_seen_at 初始化，防止升级后第一次 ping 全部重新计数
UPDATE learned_places SET last_counted_at = last_seen_at;
