-- vOPC 风险治理最小闭环：风险、审批、冻结/解冻、申诉。
-- 仅服务端可治理元数据；不接入外部审批系统、不涉及真实资金/合同。

CREATE TABLE IF NOT EXISTS vopc_risks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    risk_level TEXT NOT NULL DEFAULT 'R0',
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'open',
    reported_by INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_risks_project ON vopc_risks(project_id, status, created_at);

CREATE TABLE IF NOT EXISTS vopc_risk_approvals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    risk_id INTEGER NOT NULL,
    approver_user_id INTEGER NOT NULL,
    decision TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(risk_id) REFERENCES vopc_risks(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_risk_approvals_risk ON vopc_risk_approvals(risk_id, created_at);

CREATE TABLE IF NOT EXISTS vopc_freeze_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    reason TEXT NOT NULL,
    acted_by INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_freeze_records_project ON vopc_freeze_records(project_id, created_at);

CREATE TABLE IF NOT EXISTS vopc_risk_appeals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    submitted_by INTEGER NOT NULL,
    resolved_by INTEGER,
    resolution TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TEXT,
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_risk_appeals_project ON vopc_risk_appeals(project_id, status, created_at);
