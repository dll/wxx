-- 028_fix_feedback_fk.sql — 重建 feedback 表，移除断裂的 FK 约束
-- 应用层已校验 user_id 有效性，无需数据库层 FK 约束
-- 修复 migration 027 重建 users 表后 FK 引用失效的问题

PRAGMA foreign_keys = OFF;

CREATE TABLE IF NOT EXISTS feedback_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    feedback_id     TEXT    NOT NULL UNIQUE,
    user_id         INTEGER NOT NULL,
    username        TEXT    NOT NULL,
    message_id      TEXT    NOT NULL DEFAULT '',
    resource_id     TEXT    NOT NULL DEFAULT '',
    category        TEXT    NOT NULL DEFAULT 'answer_error',
    content         TEXT    NOT NULL,
    screenshot_url  TEXT    NOT NULL DEFAULT '',
    status          TEXT    NOT NULL DEFAULT 'pending',
    resolved_by     TEXT    NOT NULL DEFAULT '',
    resolved_at     TEXT    DEFAULT NULL,
    reply           TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO feedback_new SELECT * FROM feedback;

DROP TABLE feedback;

ALTER TABLE feedback_new RENAME TO feedback;

CREATE INDEX IF NOT EXISTS idx_feedback_status ON feedback(status);
CREATE INDEX IF NOT EXISTS idx_feedback_user ON feedback(user_id);

PRAGMA foreign_keys = ON;
