-- 066_ai_briefings.sql — AI 简讯模块
-- 首页「AI简讯」资讯列表 + 管理 CRUD + 来源自动抓取（RSS/Atom）
CREATE TABLE IF NOT EXISTS ai_briefings (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source      TEXT NOT NULL DEFAULT '',              -- 来源名称
    category    TEXT NOT NULL DEFAULT 'ai_teaching',   -- 分类：ai_teaching/ai_tool/ai_version/ai_industry
    topic       TEXT NOT NULL,                          -- 主题
    summary     TEXT NOT NULL DEFAULT '',               -- 摘要
    content     TEXT NOT NULL DEFAULT '',               -- 正文（可选）
    link        TEXT NOT NULL DEFAULT '',               -- 详情链接
    keyword     TEXT NOT NULL DEFAULT '',               -- 关键词（逗号分隔）
    published_at TEXT NOT NULL DEFAULT (datetime('now')), -- 资讯发布时间
    fetched_at  TEXT,                                   -- 自动抓取时间（手动录入为空）
    status      INTEGER NOT NULL DEFAULT 1,             -- 1 上架 0 下架
    created_by  INTEGER,                                -- 录入人用户 ID
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_ai_briefings_category ON ai_briefings(category);
CREATE INDEX IF NOT EXISTS idx_ai_briefings_status ON ai_briefings(status);
CREATE INDEX IF NOT EXISTS idx_ai_briefings_published ON ai_briefings(published_at);
CREATE INDEX IF NOT EXISTS idx_ai_briefings_source ON ai_briefings(source);

-- AI 简讯来源配置（自动抓取 RSS/Atom）
CREATE TABLE IF NOT EXISTS ai_briefing_sources (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT NOT NULL,                        -- 来源名称
    url           TEXT NOT NULL DEFAULT '',             -- RSS/Atom URL
    category      TEXT NOT NULL DEFAULT 'ai_teaching',  -- 抓取后归入的分类
    enabled       INTEGER NOT NULL DEFAULT 1,           -- 1 启用 0 停用
    fetch_enabled INTEGER NOT NULL DEFAULT 1,           -- 是否参与定时抓取
    fetch_time    TEXT NOT NULL DEFAULT '08:00',        -- 定时抓取时间 HH:MM
    last_fetch_at TEXT,                                 -- 上次成功抓取时间
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_ai_briefing_sources_enabled ON ai_briefing_sources(enabled);
