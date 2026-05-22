-- 018_process_records.sql — 办事流程办理记录持久化
-- 每个用户对每种流程（入学/离校/请假等）保留一条记录，
-- 记录当前步骤、已完成步骤集合、状态。

CREATE TABLE IF NOT EXISTS process_records (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    record_id        TEXT    NOT NULL UNIQUE,
    user_id          INTEGER NOT NULL REFERENCES users(id),
    flow_type        TEXT    NOT NULL,                -- enrollment | graduation | leave | major_change | ...
    flow_label       TEXT    NOT NULL DEFAULT '',     -- 流程显示名称
    current_step     INTEGER NOT NULL DEFAULT 0,
    completed_steps  TEXT    NOT NULL DEFAULT '[]',   -- JSON 数组，例如 [0,1,3]
    total_steps      INTEGER NOT NULL DEFAULT 0,
    status           TEXT    NOT NULL DEFAULT 'in_progress', -- in_progress | completed | abandoned
    notes            TEXT    NOT NULL DEFAULT '',
    created_at       TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- 同一用户同一流程类型只保留一条活跃记录（业务约束在 service 层处理）
CREATE INDEX IF NOT EXISTS idx_process_records_user_flow
    ON process_records(user_id, flow_type);
CREATE INDEX IF NOT EXISTS idx_process_records_status
    ON process_records(status);
