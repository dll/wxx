-- 009_feedback_and_settings.sql
-- 反馈系统 + 系统配置表

-- 用户反馈表
CREATE TABLE IF NOT EXISTS feedback (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    feedback_id     TEXT    NOT NULL UNIQUE,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    username        TEXT    NOT NULL,
    message_id      TEXT    NOT NULL DEFAULT '',
    resource_id     TEXT    NOT NULL DEFAULT '',
    category        TEXT    NOT NULL DEFAULT 'answer_error',
    content         TEXT    NOT NULL,
    status          TEXT    NOT NULL DEFAULT 'pending',
    resolved_by     TEXT    NOT NULL DEFAULT '',
    resolved_at     TEXT    DEFAULT NULL,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_feedback_status ON feedback(status);
CREATE INDEX IF NOT EXISTS idx_feedback_user ON feedback(user_id);

-- 系统配置表
CREATE TABLE IF NOT EXISTS system_settings (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    key             TEXT    NOT NULL UNIQUE,
    value           TEXT    NOT NULL DEFAULT '',
    description     TEXT    NOT NULL DEFAULT '',
    updated_by      TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- 默认配置项
INSERT OR IGNORE INTO system_settings (key, value, description) VALUES
    ('model_temperature', '0.7', 'LLM 默认温度参数'),
    ('max_tokens', '2048', 'LLM 默认最大 token 数'),
    ('rate_limit_student', '20', '学生每日提问上限'),
    ('rate_limit_counselor', '50', '辅导员每日提问上限'),
    ('emotion_alert_enabled', 'true', '是否启用情感预警'),
    ('content_filter_enabled', 'true', '是否启用内容安全过滤');
