-- 095_homework.sql — 教师作业信息发布（P2 轻量版，2026-08-17）
-- 蔚小芯侧重教育非教学：作业仅存信息（标题/说明/时间/归属），不做学生提交、批改、内容流转。
-- 归属强约束：course_id 必须对应该教师 approved 授课关系（teacher_courses），对称 R3 成绩强校验。
-- 纯 CREATE TABLE/INDEX，SQLite / MySQL 均兼容（IF NOT EXISTS，无 ON CONFLICT，时间默认 localtime，对齐 092/093/094）。
CREATE TABLE IF NOT EXISTS homework (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    teacher_id    INTEGER NOT NULL,              -- 外键→users.id（发布教师，唯一权威键）
    course_id     TEXT    NOT NULL,              -- 外键语义→courses.course_id（TEXT 稳定键）
    course_name   TEXT    NOT NULL DEFAULT '',   -- 冗余展示名，非权威（对齐 teacher_courses.course_name 口径）
    semester      TEXT    NOT NULL,              -- 非权威展示口径（发布学期，对齐 student_grades/teacher_courses）
    title         TEXT    NOT NULL,              -- 作业标题（信息发布核心字段）
    description   TEXT    NOT NULL DEFAULT '',   -- 作业说明/要求（信息发布用，纯文本）
    publish_at    TEXT,                          -- 发布日期（ISO；空=使用创建时间）
    due_at        TEXT,                          -- 截止日期（ISO；模糊时间，无截止提醒流转）
    status        TEXT    NOT NULL DEFAULT 'active',  -- active/published/archived（轻量状态，非审核流）
    created_at    TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at    TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
    UNIQUE(teacher_id, course_id, semester, title)   -- 幂等键：同教师同课程同学期同标题仅一条
);
CREATE INDEX IF NOT EXISTS idx_homework_teacher ON homework(teacher_id, semester);
CREATE INDEX IF NOT EXISTS idx_homework_course  ON homework(course_id, semester);
