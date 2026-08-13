-- 078_checkin_makeups.sql — 打卡补签（断签保护）
--
-- 设计说明：
--   学生每月拥有 2 次补签机会（docs/蔚小芯角色功能.md P1 断签保护）。
--   补签本质是往 student_checkins 写入历史日期，并在此表登记一次消耗，
--   以 (user_id, month) 维度统计当月剩余次数。
--   补签日期仅限当月且为过去日期，不允许补签未来日期。

CREATE TABLE IF NOT EXISTS checkin_makeups (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    month       TEXT    NOT NULL, -- 补签日期所在月份 YYYY-MM
    check_date  TEXT    NOT NULL, -- 被补签的日期 YYYY-MM-DD
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, month, check_date)
);

CREATE INDEX IF NOT EXISTS idx_checkin_makeup_user_month
    ON checkin_makeups(user_id, month);