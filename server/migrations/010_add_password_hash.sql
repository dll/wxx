-- 010_add_password_hash.sql — 用户密码哈希字段
-- 历史迁移曾允许空串；029 起已统一收口为必须使用密码登录

ALTER TABLE users ADD COLUMN password_hash TEXT NOT NULL DEFAULT '';
