-- 071_user_portal_credentials.sql — 学校门户登录凭证（加密存储）
-- 仅存 AES-GCM 加密后的密码（WXX_ENCRYPTION_KEY），绝不存明文；
-- 前端与查询接口均不回显密码明文。
CREATE TABLE IF NOT EXISTS user_portal_credentials (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id           INTEGER NOT NULL UNIQUE,
    portal_url        TEXT NOT NULL DEFAULT 'https://my0.chzu.edu.cn/',
    portal_account    TEXT NOT NULL DEFAULT '',   -- 学校门户账号（学号/工号）
    portal_password_enc TEXT NOT NULL DEFAULT '', -- AES-GCM 加密后的密码
    updated_at        TEXT NOT NULL DEFAULT (datetime('now')),
    created_at        TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_portal_cred_user ON user_portal_credentials(user_id);
