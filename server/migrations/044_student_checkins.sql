-- 044_student_checkins.sql — 学生每日打卡持久化（S1 学生核心功能）
--
-- 设计说明：
--   每日打卡支持连续天数、断签保护（每月 1 次）、里程碑成就。
--   每用户每天仅一条记录（UNIQUE(user_id, check_date)），幂等。
--   streak 由 service 层实时计算（基于 check_date 连续性），不冗余存储。

CREATE TABLE IF NOT EXISTS student_checkins (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    check_date  TEXT    NOT NULL,  -- 格式 YYYY-MM-DD
    mood        TEXT    DEFAULT '', -- 可选：打卡时的心情标签（happy/normal/tired/sad）
    note        TEXT    DEFAULT '', -- 可选：一句话打卡感想
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, check_date)
);

CREATE INDEX IF NOT EXISTS idx_checkin_user_date ON student_checkins(user_id, check_date DESC);
