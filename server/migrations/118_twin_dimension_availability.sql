-- 118_twin_dimension_availability.sql — 数字孪生五维可用性
--
-- 分数 0 既可能是真实低分，也可能是旧链路中的“无记录占位”。二者不可由分数反推，
-- 因此主快照与历史快照都显式保存每一维是否有真实数据支撑。
-- 旧记录无法可靠还原，统一默认 unavailable；学生下次重算画像后会写入准确标记。

ALTER TABLE student_profile_snapshot ADD COLUMN academic_available INTEGER NOT NULL DEFAULT 0;
ALTER TABLE student_profile_snapshot ADD COLUMN ability_available INTEGER NOT NULL DEFAULT 0;
ALTER TABLE student_profile_snapshot ADD COLUMN ideological_available INTEGER NOT NULL DEFAULT 0;
ALTER TABLE student_profile_snapshot ADD COLUMN emotional_available INTEGER NOT NULL DEFAULT 0;
ALTER TABLE student_profile_snapshot ADD COLUMN social_available INTEGER NOT NULL DEFAULT 0;

ALTER TABLE snapshot_history ADD COLUMN academic_available INTEGER NOT NULL DEFAULT 0;
ALTER TABLE snapshot_history ADD COLUMN ability_available INTEGER NOT NULL DEFAULT 0;
ALTER TABLE snapshot_history ADD COLUMN ideological_available INTEGER NOT NULL DEFAULT 0;
ALTER TABLE snapshot_history ADD COLUMN emotional_available INTEGER NOT NULL DEFAULT 0;
ALTER TABLE snapshot_history ADD COLUMN social_available INTEGER NOT NULL DEFAULT 0;
