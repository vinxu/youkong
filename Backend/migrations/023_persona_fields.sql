-- 023: 个性化 Persona 字段（时间模式 + 人设描述）
ALTER TABLE core_memories
  ADD COLUMN persona_text TEXT COMMENT '由 LLM 生成的个性化人设描述' AFTER social_tendency,
  ADD COLUMN time_pattern_stats JSON COMMENT '时间模式频率表 {weekday:{hour:{activity:count}}}' AFTER persona_text,
  ADD COLUMN persona_generated_at DATETIME COMMENT 'persona 最近生成时间' AFTER time_pattern_stats;
