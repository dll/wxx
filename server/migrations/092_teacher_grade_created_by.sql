-- 092_teacher_grade_created_by.sql — 教师成绩录入审计（P0-1，2026-08-17）
-- 方案 A：教师自主声明授课关系并录入真实成绩，created_by 记录声明人，便于审计可追溯。
-- 幂等键仍为 UNIQUE(user_id,course_id,semester,grade_type)，新增列仅作审计，不改幂等语义。
ALTER TABLE student_grades ADD COLUMN created_by INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_student_grades_created_by ON student_grades(created_by);
