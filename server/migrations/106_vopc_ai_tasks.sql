-- vOPC B1 AI 任务真实执行闭环：ai 任务执行记录 + 项目级额度。
-- 纯增量：不改既有表，新增两张表。SQLite+MySQL 方言兼容、幂等。
-- 任务状态：status ∈ pending/running/succeeded/failed/superseded；
--   succeeded 后由主理人审阅，final_decision ∈ accepted/revised/rejected/overruled。
-- 额度：vopc_ai_quotas 聚合项目累计 token；未配置时使用应用层默认。

CREATE TABLE IF NOT EXISTS vopc_ai_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    role_key TEXT NOT NULL,
    instruction TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 'deepseek',
    model TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    max_tokens INTEGER NOT NULL DEFAULT 500,
    output_content TEXT NOT NULL DEFAULT '',
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    retry_count INTEGER NOT NULL DEFAULT 0,
    error_msg TEXT NOT NULL DEFAULT '',
    final_decision TEXT,
    decision_by INTEGER,
    decision_note TEXT NOT NULL DEFAULT '',
    revision TEXT NOT NULL DEFAULT '',
    decided_at TEXT,
    created_by INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_ai_tasks_proj ON vopc_ai_tasks(project_id, created_at);
CREATE INDEX IF NOT EXISTS idx_vopc_ai_tasks_status ON vopc_ai_tasks(status);

CREATE TABLE IF NOT EXISTS vopc_ai_quotas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    monthly_limit_tokens INTEGER NOT NULL DEFAULT 500000,
    used_tokens INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id),
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_ai_quotas_proj ON vopc_ai_quotas(project_id);
