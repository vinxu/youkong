-- 025: Add gender and mbti fields to user_profiles
ALTER TABLE user_profiles
  ADD COLUMN gender VARCHAR(10) DEFAULT NULL AFTER user_id,
  ADD COLUMN mbti VARCHAR(4) DEFAULT NULL AFTER gender;
