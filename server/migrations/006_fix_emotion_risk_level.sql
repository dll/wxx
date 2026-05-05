-- 006_fix_emotion_risk_level.sql — 修复 emotion_logs 风险等级 CHECK 约束
-- 问题：001_init.sql 中 CHECK(risk_level IN ('low','medium','high')) 缺少 'urgent'
-- SQLite 不支持 ALTER TABLE 修改 CHECK，需重建表

-- 1. 创建新表（含完整 CHECK + 所有 004 增强字段）
CREATE TABLE IF NOT EXISTS emotion_logs_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    session_id      TEXT    NOT NULL,
    message_text    TEXT    NOT NULL DEFAULT '',
    score           REAL    NOT NULL DEFAULT 0,
    risk_level      TEXT    NOT NULL DEFAULT 'low'
                    CHECK(risk_level IN ('low','medium','high','urgent')),
    analysis_json   TEXT    NOT NULL DEFAULT '{}',
    notified        INTEGER NOT NULL DEFAULT 0,
    status          TEXT    NOT NULL DEFAULT 'pending'
                    CHECK(status IN ('pending','acknowledged','resolved')),
    acknowledged_by TEXT    NOT NULL DEFAULT '',
    acknowledged_at TEXT    DEFAULT NULL,
    alert_id        TEXT    NOT NULL DEFAULT '',
    username        TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- 2. 迁移现有数据
INSERT INTO emotion_logs_new
    (id, user_id, session_id, message_text, score, risk_level,
     analysis_json, notified, status, acknowledged_by, acknowledged_at,
     alert_id, username, created_at)
SELECT
    id, user_id, session_id, message_text, score, risk_level,
    analysis_json, notified, status, acknowledged_by, acknowledged_at,
    alert_id, username, created_at
FROM emotion_logs;

-- 3. 替换旧表
DROP TABLE emotion_logs;
ALTER TABLE emotion_logs_new RENAME TO emotion_logs;

-- 4. 重建索引
CREATE INDEX IF NOT EXISTS idx_emotion_user ON emotion_logs(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_emotion_alert_id ON emotion_logs(alert_id);
CREATE INDEX IF NOT EXISTS idx_emotion_risk_status ON emotion_logs(risk_level, status, created_at);
