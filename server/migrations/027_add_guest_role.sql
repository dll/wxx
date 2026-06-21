-- 027_add_guest_role.sql — 在 users 表的 CHECK 约束中添加 guest 角色
-- SQLite 不支持直接修改 CHECK 约束，通过重建表实现

PRAGMA foreign_keys = OFF;

CREATE TABLE IF NOT EXISTS users_new (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    username    TEXT    NOT NULL UNIQUE,
    display_name TEXT   NOT NULL DEFAULT '',
    role        TEXT    NOT NULL DEFAULT 'student'
                CHECK(role IN ('sys_admin','school_admin','college_admin',
                               'counselor','student_union','student',
                               'teacher','assistant','guest')),
    owner_scope TEXT    NOT NULL DEFAULT 'college'
                CHECK(owner_scope IN ('school','college','class')),
    owner_id    TEXT    NOT NULL DEFAULT '',
    password_hash TEXT  NOT NULL DEFAULT '',
    voice_enabled INTEGER NOT NULL DEFAULT 0,
    status      TEXT    NOT NULL DEFAULT 'active',
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO users_new (id, username, display_name, role, owner_scope, owner_id, password_hash, voice_enabled, status, created_at, updated_at)
    SELECT id, username, display_name, role, owner_scope, owner_id, password_hash, voice_enabled, status, created_at, updated_at FROM users;

DROP TABLE users;

ALTER TABLE users_new RENAME TO users;

PRAGMA foreign_keys = ON;
