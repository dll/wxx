-- 038_user_notifications.sql — 用户站内通知系统
-- 用于存储每个用户的站内通知，支持已读状态和关联跳转

CREATE TABLE IF NOT EXISTS user_notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT DEFAULT '',
    type TEXT NOT NULL DEFAULT 'system',  -- system/feedback/knowledge/activity/career
    related_type TEXT DEFAULT '',          -- 关联类型：feedback/resource/plan等
    related_id INTEGER DEFAULT 0,
    is_read INTEGER DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_user_notifications_user ON user_notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_user_notifications_read ON user_notifications(user_id, is_read);

-- 插入测试数据：给 user_id=1 插入 5 条不同类型的通知
INSERT INTO user_notifications (user_id, title, content, type, related_type, related_id, is_read) VALUES
(1, '欢迎使用蔚小芯', '感谢您使用蔚小芯智能助手，祝您使用愉快！', 'system', '', 0, 0),
(1, '您的反馈已收到', '您提交的问题反馈我们已经收到，正在处理中，请耐心等待。', 'feedback', 'feedback', 1, 0),
(1, '新知识推荐', '为您推荐一篇新的知识库文章，快来看看吧！', 'knowledge', 'resource', 1, 0),
(1, '活动提醒', '校园文化活动即将开始，欢迎踊跃参加！', 'activity', '', 0, 1),
(1, '就业资讯更新', '新的校园招聘信息已发布，快来查看适合你的岗位。', 'career', '', 0, 0);
