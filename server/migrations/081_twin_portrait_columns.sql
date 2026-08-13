-- 081_twin_portrait_columns.sql — 数字孪生画像列扩展为 LONGTEXT
--
-- 背景：
--   image_base64 / source_photo_base64 原为 TEXT（MySQL 转 VARCHAR(128) 或 TEXT，
--   上限 64KB）。画像尺寸提升到 256x256 后 base64 约 100KB+，超出 TEXT 上限，
--   保存报 Error 1406 (22000): Data too long for column。
--
-- 处理：
--   MySQL：ALTER TABLE ... MODIFY COLUMN ... LONGTEXT（迁移执行器自动跳过
--   SQLite，因 SQLite TEXT 无长度限制）。
--   注意：MySQL 的 LONGTEXT 列不允许 DEFAULT 子句（Error 1101），故省略。
--   已在 dialect.go longLongTextColumns 登记，新库建表直接 LONGTEXT。

ALTER TABLE twin_portraits MODIFY COLUMN image_base64 LONGTEXT NOT NULL;
ALTER TABLE twin_portraits MODIFY COLUMN source_photo_base64 LONGTEXT;
ALTER TABLE users MODIFY COLUMN avatar_base64 LONGTEXT;