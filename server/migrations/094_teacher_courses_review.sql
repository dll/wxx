-- 094_teacher_courses_review.sql — 教师授课关系申报+教辅审核（R3，2026-08-17）
-- 升级方案A「声明即授权」为「申报→教辅/教务审核确认→确认后才能录入对应课程成绩」强校验。
-- 对称于 graduation_outcome 审核流（pending/approved/rejected + reviewed_at/by/name/note）。
-- 纯 CREATE TABLE/INDEX，SQLite / MySQL 均兼容，与 092/093 同级保守（不引入 ON CONFLICT）。
CREATE TABLE IF NOT EXISTS teacher_courses (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    teacher_id    INTEGER NOT NULL,              -- 外键→users.id（教师账号，唯一权威键）
    course_id     TEXT    NOT NULL,              -- 外键语义→courses.course_id（TEXT 稳定键）
    course_name   TEXT    NOT NULL DEFAULT '',   -- 冗余展示名（对齐 student_grades.course_name），非权威
    semester      TEXT    NOT NULL,              -- 对齐 student_grades.semester 口径
    status        TEXT    NOT NULL DEFAULT 'pending',  -- 状态机：pending/approved/rejected
    created_by    INTEGER NOT NULL,              -- 申报人=教师本人 user_id
    reviewed_by   INTEGER NOT NULL DEFAULT 0,    -- 审核人 user_id（0=未审核）
    reviewed_name TEXT    NOT NULL DEFAULT '',   -- 冗余审核人姓名展示
    review_note   TEXT    NOT NULL DEFAULT '',   -- 驳回/通过说明
    reviewed_at   TEXT,                          -- 审核时间（ISO，审核即真实操作留痕）
    created_at    TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at    TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
    UNIQUE(teacher_id, course_id, semester)      -- 幂等键：同教师同课程同学期仅一条
);
CREATE INDEX IF NOT EXISTS idx_teacher_courses_status  ON teacher_courses(status);
CREATE INDEX IF NOT EXISTS idx_teacher_courses_teacher ON teacher_courses(teacher_id, semester);
