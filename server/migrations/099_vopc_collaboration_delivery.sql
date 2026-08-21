-- vOPC P1 协作、成果版本仓库与正式里程碑评审闭环（仅元数据/引用，不执行任意文件）
CREATE TABLE IF NOT EXISTS vopc_invitations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    invitee_user_id INTEGER NOT NULL,
    project_role TEXT NOT NULL DEFAULT 'member',
    status TEXT NOT NULL DEFAULT 'pending',
    message TEXT NOT NULL DEFAULT '',
    invited_by INTEGER NOT NULL,
    responded_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, invitee_user_id, status),
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE,
    FOREIGN KEY(invitee_user_id) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_vopc_invitee_status ON vopc_invitations(invitee_user_id,status,created_at);

CREATE TABLE IF NOT EXISTS vopc_artifacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    artifact_type TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    visibility TEXT NOT NULL DEFAULT 'private',
    license TEXT NOT NULL DEFAULT '',
    created_by INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id,name),
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS vopc_artifact_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    artifact_id INTEGER NOT NULL,
    version TEXT NOT NULL,
    source_kind TEXT NOT NULL,
    source_ref TEXT NOT NULL,
    checksum TEXT NOT NULL DEFAULT '',
    release_notes TEXT NOT NULL DEFAULT '',
    created_by INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(artifact_id,version),
    FOREIGN KEY(artifact_id) REFERENCES vopc_artifacts(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_artifacts_project ON vopc_artifacts(project_id,updated_at);

CREATE TABLE IF NOT EXISTS vopc_milestone_submissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    stage TEXT NOT NULL,
    evidence TEXT NOT NULL,
    artifact_version_ids TEXT NOT NULL DEFAULT '[]',
    reviewer_user_id INTEGER,
    status TEXT NOT NULL DEFAULT 'pending',
    submitted_by INTEGER NOT NULL,
    submitted_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TEXT,
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS vopc_milestone_reviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    submission_id INTEGER NOT NULL,
    reviewer_user_id INTEGER NOT NULL,
    result TEXT NOT NULL,
    note TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(submission_id),
    FOREIGN KEY(submission_id) REFERENCES vopc_milestone_submissions(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_milestone_submission_project ON vopc_milestone_submissions(project_id,stage,status);
