-- 059_user_avatar.sql — 用户头像字段
-- 头像以 base64 存 SQLite（跨实例持久，与 feedback_screenshots 同模式）。
-- 幂等：execSQL 已处理 ADD COLUMN 重复列错误（跳过而不报错）。

ALTER TABLE users ADD COLUMN avatar_base64 TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN avatar_mime TEXT DEFAULT 'image/png';
