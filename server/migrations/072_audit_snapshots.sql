-- 072_audit_snapshots.sql — 审计恢复快照
-- 对可恢复的写操作记录操作前后状态，供管理员「恢复操作」回滚。
-- 仅覆盖白名单业务表（用户状态/知识资源状态/反馈状态等），避免通用回滚的语义风险。
CREATE TABLE IF NOT EXISTS audit_snapshots (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    audit_id     INTEGER NOT NULL,                -- 关联 audit_logs.id
    op_table     TEXT NOT NULL,                   -- 目标表：users / kb_resources / feedback
    record_id    TEXT NOT NULL,                   -- 目标记录标识
    operation    TEXT NOT NULL,                   -- 语义操作：user.status / kb.status / feedback.status
    before_json  TEXT NOT NULL DEFAULT '',        -- 操作前状态
    after_json   TEXT NOT NULL DEFAULT '',        -- 操作后状态
    restored     INTEGER NOT NULL DEFAULT 0,      -- 1 已恢复 0 未恢复
    restored_at  TEXT,
    restored_by  TEXT DEFAULT '',                 -- 恢复操作执行人
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_audit_snapshots_audit ON audit_snapshots(audit_id);
CREATE INDEX IF NOT EXISTS idx_audit_snapshots_restored ON audit_snapshots(restored);
CREATE INDEX IF NOT EXISTS idx_audit_snapshots_op ON audit_snapshots(operation);
