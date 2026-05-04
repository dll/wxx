-- 蔚小芯 SQLite 初始化脚本
-- 执行: make migrate

-- ===== 用户与角色 =====
CREATE TABLE IF NOT EXISTS users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    username    TEXT    NOT NULL UNIQUE,
    display_name TEXT   NOT NULL DEFAULT '',
    role        TEXT    NOT NULL DEFAULT 'student'
                CHECK(role IN ('sys_admin','school_admin','college_admin',
                               'counselor','student_union','student',
                               'teacher','assistant')),
    owner_scope TEXT    NOT NULL DEFAULT 'college'
                CHECK(owner_scope IN ('school','college','class')),
    owner_id    TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ===== 会话 =====
CREATE TABLE IF NOT EXISTS sessions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT    NOT NULL UNIQUE,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ===== 对话消息 =====
CREATE TABLE IF NOT EXISTS messages (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT    NOT NULL REFERENCES sessions(session_id),
    role        TEXT    NOT NULL CHECK(role IN ('user','assistant','system')),
    content     TEXT    NOT NULL,
    trace_id    TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ===== 知识资源（Context Engine 结构化库）=====
CREATE TABLE IF NOT EXISTS kb_resources (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    resource_id     TEXT    NOT NULL UNIQUE,
    resource_type   TEXT    NOT NULL CHECK(resource_type IN ('Policy','Process','FAQ','Activity')),
    owner_scope     TEXT    NOT NULL CHECK(owner_scope IN ('school','college','class')),
    owner_id        TEXT    NOT NULL DEFAULT '',
    role_scope      TEXT    NOT NULL DEFAULT '["student"]',   -- JSON 数组
    version         TEXT    NOT NULL,
    status          TEXT    NOT NULL DEFAULT 'draft'
                    CHECK(status IN ('draft','pending','published','retired')),
    title           TEXT    NOT NULL,
    summary         TEXT    NOT NULL DEFAULT '',
    content         TEXT    NOT NULL DEFAULT '',
    source_link     TEXT    NOT NULL DEFAULT '',
    source_version  TEXT    NOT NULL DEFAULT '',
    effective_at    TEXT    DEFAULT NULL,
    expired_at      TEXT    DEFAULT NULL,
    tags            TEXT    NOT NULL DEFAULT '[]',            -- JSON 数组
    updated_by      TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- FTS5 全文检索索引（BM25）
CREATE VIRTUAL TABLE IF NOT EXISTS kb_fts USING fts5(
    resource_id,
    title,
    summary,
    content,
    content=kb_resources,
    content_rowid=id,
    tokenize='unicode61'
);

-- FTS 同步触发器
CREATE TRIGGER IF NOT EXISTS kb_fts_insert AFTER INSERT ON kb_resources BEGIN
    INSERT INTO kb_fts(rowid, resource_id, title, summary, content)
    VALUES (new.id, new.resource_id, new.title, new.summary, new.content);
END;

CREATE TRIGGER IF NOT EXISTS kb_fts_update AFTER UPDATE ON kb_resources BEGIN
    INSERT INTO kb_fts(kb_fts, rowid, resource_id, title, summary, content)
    VALUES ('delete', old.id, old.resource_id, old.title, old.summary, old.content);
    INSERT INTO kb_fts(rowid, resource_id, title, summary, content)
    VALUES (new.id, new.resource_id, new.title, new.summary, new.content);
END;

CREATE TRIGGER IF NOT EXISTS kb_fts_delete AFTER DELETE ON kb_resources BEGIN
    INSERT INTO kb_fts(kb_fts, rowid, resource_id, title, summary, content)
    VALUES ('delete', old.id, old.resource_id, old.title, old.summary, old.content);
END;

-- ===== 流程节点（结构化优先查询）=====
CREATE TABLE IF NOT EXISTS process_steps (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    resource_id     TEXT    NOT NULL REFERENCES kb_resources(resource_id),
    step_order      INTEGER NOT NULL,
    title           TEXT    NOT NULL,
    materials       TEXT    NOT NULL DEFAULT '[]',    -- JSON 数组
    entry_url       TEXT    NOT NULL DEFAULT '',
    deadline        TEXT    NOT NULL DEFAULT '',
    location        TEXT    NOT NULL DEFAULT '',
    notes           TEXT    NOT NULL DEFAULT ''
);

-- ===== 审计日志 =====
CREATE TABLE IF NOT EXISTS audit_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER,
    username    TEXT    NOT NULL DEFAULT '',
    role        TEXT    NOT NULL DEFAULT '',
    action      TEXT    NOT NULL,
    resource    TEXT    NOT NULL DEFAULT '',
    detail      TEXT    NOT NULL DEFAULT '',
    trace_id    TEXT    NOT NULL DEFAULT '',
    ip          TEXT    NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    result_code INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ===== 情感评估记录 =====
CREATE TABLE IF NOT EXISTS emotion_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    session_id  TEXT    NOT NULL,
    score       REAL    NOT NULL DEFAULT 0,
    risk_level  TEXT    NOT NULL DEFAULT 'low'
                CHECK(risk_level IN ('low','medium','high')),
    notified    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ===== 导出记录 =====
CREATE TABLE IF NOT EXISTS export_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,
    role        TEXT    NOT NULL,
    format      TEXT    NOT NULL,
    answer_id   TEXT    NOT NULL DEFAULT '',
    has_sensitive INTEGER NOT NULL DEFAULT 0,
    trace_id    TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ===== 同步游标 =====
CREATE TABLE IF NOT EXISTS sync_cursors (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    target      TEXT    NOT NULL UNIQUE,              -- e.g. 'weiyuan_zhida'
    cursor_val  TEXT    NOT NULL DEFAULT '',
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ===== 索引 =====
CREATE INDEX IF NOT EXISTS idx_kb_status ON kb_resources(status);
CREATE INDEX IF NOT EXISTS idx_kb_type ON kb_resources(resource_type);
CREATE INDEX IF NOT EXISTS idx_kb_owner ON kb_resources(owner_scope, owner_id);
CREATE INDEX IF NOT EXISTS idx_audit_time ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_emotion_user ON emotion_logs(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
