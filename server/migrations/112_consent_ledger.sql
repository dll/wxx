-- 隐私授权账本：保留政策版本、用途、供应商及撤回/授权历史。
CREATE TABLE IF NOT EXISTS consent_ledger (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    policy_version TEXT NOT NULL,
    purpose TEXT NOT NULL,
    vendor TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL DEFAULT 'granted',
    source TEXT NOT NULL DEFAULT 'api',
    trace_id TEXT NOT NULL DEFAULT '',
    granted_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    revoked_at TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_consent_ledger_user_purpose ON consent_ledger(user_id, purpose, id);
