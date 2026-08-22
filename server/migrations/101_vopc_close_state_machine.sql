-- vOPC 结项与异常状态机：把 S9 里程碑通过与项目结项分离，并落地
-- pause/resume/pivot/terminate/archive/close 的合法流转与审计。
-- 本迁移不与历史迁移冲突，仅新增表与可空时间列。

CREATE TABLE IF NOT EXISTS vopc_close_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    reason TEXT NOT NULL,
    failure_evidence TEXT NOT NULL DEFAULT '',
    outcome_package TEXT NOT NULL DEFAULT '',
    human_decision TEXT NOT NULL DEFAULT '',
    decided_by INTEGER NOT NULL,
    previous_status TEXT NOT NULL DEFAULT '',
    new_status TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_close_records_project ON vopc_close_records(project_id, created_at);

ALTER TABLE vopc_projects ADD COLUMN completed_at TEXT;
ALTER TABLE vopc_projects ADD COLUMN closed_at TEXT;
