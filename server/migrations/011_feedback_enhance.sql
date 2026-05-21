-- 011_feedback_enhance.sql — 反馈表功能增强
-- 增加截图链接和回复字段，支持在线反馈闭环
--
-- 注意：009 建表时已包含 screenshot_url 和 reply 列（新数据库直接有），
-- 此处 ALTER TABLE 仅对旧数据库生效。若列已存在，execSQL 自动跳过重复列错误。

ALTER TABLE feedback ADD COLUMN screenshot_url TEXT NOT NULL DEFAULT '';
ALTER TABLE feedback ADD COLUMN reply TEXT NOT NULL DEFAULT '';
