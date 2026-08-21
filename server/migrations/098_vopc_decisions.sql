-- vOPC 决策中心兼容迁移：不修改 097，补充状态、决定人和决定时间
ALTER TABLE vopc_decisions ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE vopc_decisions ADD COLUMN decided_at TEXT;
CREATE INDEX IF NOT EXISTS idx_vopc_decisions_project_status ON vopc_decisions(project_id, status, created_at);
