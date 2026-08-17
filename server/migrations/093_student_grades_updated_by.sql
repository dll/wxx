-- 093_student_grades_updated_by.sql — 成绩最后写入人审计（R1，2026-08-17）
-- 在 created_by（首声明人）基础上补充 updated_by：记录最后一次写入/修改成绩的教师，增强审计可追溯性。
-- 幂等键仍为 UNIQUE(user_id,course_id,semester,grade_type)，新增列仅作审计，不改幂等语义。
-- 方言：ADD COLUMN 在 SQLite / MySQL 均支持；此处仅加列 + 索引，不做其他 DDL。
ALTER TABLE student_grades ADD COLUMN updated_by INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_student_grades_updated_by ON student_grades(updated_by);
