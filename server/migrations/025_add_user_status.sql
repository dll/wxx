-- 025_add_user_status.sql — 用户状态字段（游客审核用）
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
