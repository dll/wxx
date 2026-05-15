-- 011_feedback_enhance.sql — 反馈表功能增强
-- 增加截图链接和回复字段，支持在线反馈闭环

ALTER TABLE feedback ADD COLUMN screenshot_url TEXT NOT NULL DEFAULT '';
ALTER TABLE feedback ADD COLUMN reply TEXT NOT NULL DEFAULT '';
