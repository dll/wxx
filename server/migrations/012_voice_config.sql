-- 012_voice_config.sql — 用户语音功能开关
-- 在 users 表增加 voice_enabled 字段，控制语音功能总开关

ALTER TABLE users ADD COLUMN voice_enabled INTEGER NOT NULL DEFAULT 0;
