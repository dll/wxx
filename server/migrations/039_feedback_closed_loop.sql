-- 039_feedback_closed_loop.sql — 反馈系统闭环功能增强
-- 增加满意度评分、关联知识资源、处理记录等字段

-- 反馈表增加评分字段
ALTER TABLE feedback ADD COLUMN rating INTEGER NOT NULL DEFAULT 0;
ALTER TABLE feedback ADD COLUMN rating_comment TEXT NOT NULL DEFAULT '';
ALTER TABLE feedback ADD COLUMN rated_at TEXT;

-- 关联知识库资源（resource_id 已有，这里增加关联备注字段）
ALTER TABLE feedback ADD COLUMN linked_resource_note TEXT NOT NULL DEFAULT '';
ALTER TABLE feedback ADD COLUMN linked_at TEXT;
ALTER TABLE feedback ADD COLUMN linked_by TEXT NOT NULL DEFAULT '';

-- 反馈处理记录表（时间线）
CREATE TABLE IF NOT EXISTS feedback_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    feedback_id TEXT NOT NULL,
    action TEXT NOT NULL,
    operator TEXT NOT NULL DEFAULT '',
    detail TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_feedback_logs_feedback ON feedback_logs(feedback_id);
