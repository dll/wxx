-- 第三方应用接入（external_apps）— 应用中心清单
-- manifest 为完整 JSON（见 docs/external-apps.md v0.1），运行时解析
CREATE TABLE IF NOT EXISTS external_apps (
    id           TEXT PRIMARY KEY,           -- manifest.id
    manifest     TEXT NOT NULL,              -- 完整 JSON
    enabled      INTEGER NOT NULL DEFAULT 1, -- 启停标志（0 停用）
    created_by   INTEGER NOT NULL,           -- sys_admin 用户 ID
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_external_apps_enabled ON external_apps(enabled);