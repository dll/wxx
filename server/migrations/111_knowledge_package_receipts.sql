-- 知识包协议级接收账本：package_id 是幂等键，防止签名包重放。
-- 兼容 SQLite/MySQL，由迁移 runner 负责方言转换。
CREATE TABLE IF NOT EXISTS knowledge_package_receipts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    package_id TEXT NOT NULL UNIQUE,
    producer TEXT NOT NULL DEFAULT '',
    signature TEXT NOT NULL DEFAULT '',
    trace_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'processing',
    response_json TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_kpr_status ON knowledge_package_receipts(status);
