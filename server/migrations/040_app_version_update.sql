-- 应用版本管理表
CREATE TABLE IF NOT EXISTS app_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version_code INTEGER NOT NULL,          -- 版本号（整数，用于比较，如 1, 2, 3）
    version_name TEXT NOT NULL,             -- 版本名称，如 "0.0.5"
    platform TEXT NOT NULL DEFAULT 'all',   -- 平台：android, web, all
    title TEXT NOT NULL DEFAULT '',         -- 更新标题
    changelog TEXT NOT NULL DEFAULT '',     -- 更新日志
    download_url TEXT NOT NULL DEFAULT '',  -- 下载地址（APK/Web等）
    force_update INTEGER NOT NULL DEFAULT 0,-- 是否强制更新：0否 1是
    status INTEGER NOT NULL DEFAULT 1,      -- 状态：0禁用 1启用
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_app_versions_platform ON app_versions(platform);
CREATE INDEX IF NOT EXISTS idx_app_versions_status ON app_versions(status);
CREATE INDEX IF NOT EXISTS idx_app_versions_version_code ON app_versions(version_code);

-- 插入初始版本数据
INSERT OR IGNORE INTO app_versions (version_code, version_name, platform, title, changelog, download_url, force_update, status)
VALUES
    (1, '0.0.1', 'all', '初始版本', '蔚小芯初始发布版本', '', 0, 1),
    (4, '0.0.4', 'all', '功能增强版', '新增知识治理、学习计划、通知中心等功能', '', 0, 1),
    (5, '0.0.5', 'all', '版本更新功能', '新增自动检查更新、版本提示功能', '', 0, 1);
