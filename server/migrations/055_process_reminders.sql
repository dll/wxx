-- 055_process_reminders.sql — 办事流程提醒配置
-- 提醒按流程资源维护，可绑定到具体步骤；学生端按 remind_at 和办理进度计算状态。

CREATE TABLE IF NOT EXISTS process_reminders (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    process_id  TEXT    NOT NULL REFERENCES kb_resources(resource_id),
    step_order  INTEGER NOT NULL DEFAULT 0,
    remind_at   TEXT    NOT NULL DEFAULT '',
    title       TEXT    NOT NULL DEFAULT '',
    content     TEXT    NOT NULL DEFAULT '',
    is_enabled  INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_process_reminders_process
    ON process_reminders(process_id, remind_at);
