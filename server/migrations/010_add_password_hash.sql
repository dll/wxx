-- 010_add_password_hash.sql — 用户密码哈希字段
-- 空串表示未设置密码，保留开发环境免密登录

ALTER TABLE users ADD COLUMN password_hash TEXT NOT NULL DEFAULT '';
