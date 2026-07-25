-- 046_chat_metrics.sql
-- 问答质量指标表（运维评测体系）
-- 每次问答成功后写入一条记录，用于 admin/metrics 实时聚合

CREATE TABLE IF NOT EXISTS chat_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL DEFAULT '',
    user_id INTEGER NOT NULL DEFAULT 0,
    question TEXT NOT NULL DEFAULT '',
    intent TEXT NOT NULL DEFAULT '',          -- 意图分类
    confidence REAL NOT NULL DEFAULT 0,       -- 模型置信度 0~1
    fallback INTEGER NOT NULL DEFAULT 0,      -- 是否兜底（1=是）
    sources_count INTEGER NOT NULL DEFAULT 0, -- 命中来源数
    duration_ms INTEGER NOT NULL DEFAULT 0,   -- 问答耗时（毫秒）
    trace_id TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT (datetime('now'))
);

-- 常用查询索引
CREATE INDEX IF NOT EXISTS idx_chat_metrics_created ON chat_metrics(created_at);
CREATE INDEX IF NOT EXISTS idx_chat_metrics_fallback ON chat_metrics(fallback, created_at);
CREATE INDEX IF NOT EXISTS idx_chat_metrics_intent ON chat_metrics(intent);
