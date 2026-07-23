-- 032_notifications.sql — 通知推送系统
-- 支持创建通知并通过 webhook 推送到 QQ 群/微信群等外部渠道

CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,                            -- 通知标题
    content TEXT NOT NULL,                          -- 通知正文
    author_id INTEGER NOT NULL,                     -- 创建者用户ID
    author_name TEXT NOT NULL,                      -- 创建者姓名
    audience_type TEXT NOT NULL DEFAULT 'all',       -- 受众类型：all/student/counselor/teacher/admin
    status TEXT NOT NULL DEFAULT 'draft',            -- draft/published
    push_qq INTEGER NOT NULL DEFAULT 0,             -- 是否推送到QQ群
    push_wechat INTEGER NOT NULL DEFAULT 0,         -- 是否推送到微信群
    push_result TEXT,                               -- 推送结果（JSON）
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    published_at TEXT,                              -- 发布时间
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS notification_webhooks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,                             -- 名称（如"QQ群-2024级"）
    channel TEXT NOT NULL CHECK(channel IN ('qq','wechat','dingtalk')),  -- 渠道
    webhook_url TEXT NOT NULL,                      -- Webhook 地址
    is_active INTEGER NOT NULL DEFAULT 1,           -- 是否启用
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 插入默认 webhook 占位（实际 URL 通过 .env 或管理界面配置）
INSERT OR IGNORE INTO notification_webhooks (name, channel, webhook_url)
VALUES ('默认QQ群推送', 'qq', '${QQ_WEBHOOK_URL}');

INSERT OR IGNORE INTO notification_webhooks (name, channel, webhook_url)
VALUES ('默认微信群推送', 'wechat', '${WECHAT_WEBHOOK_URL}');

CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_author ON notifications(author_id);
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at);