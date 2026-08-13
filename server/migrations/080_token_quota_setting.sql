-- 080_token_quota_setting.sql — 每月 AI Token 限额（管理员可配置）
--
-- 背景：
--   默认每月 100,000 tokens/人（config 默认值同步调整）。管理员可在
--   「系统配置」页修改 monthly_token_quota 覆盖默认值，运行时生效。
--   值 0 表示不限。

INSERT OR IGNORE INTO system_settings (`key`, `value`, description, updated_by, updated_at)
VALUES ('monthly_token_quota', '100000', '每人每月 AI 对话 token 限额（0 表示不限，管理员可调整）', 'system', CURRENT_TIMESTAMP);