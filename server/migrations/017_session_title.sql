-- 017_session_title.sql — 会话标题持久化
-- 给 sessions 表加 title 列，支持用户重命名对话；
-- 老数据默认空字符串，前端展示时若空则 fallback 到首条用户消息前 12 字。

ALTER TABLE sessions ADD COLUMN title TEXT NOT NULL DEFAULT '';
