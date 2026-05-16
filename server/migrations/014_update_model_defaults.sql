-- 014_update_model_defaults.sql
-- 用途：将已有用户的模型默认值更新为最新版本（deepseek-v4-pro / spark-v4.0）
-- 依据：在 013 已部署到生产后，通过修改 013 的默认值不会触发回填，需新增本迁移完成数据回填
-- 仅更新仍停留在旧默认值的记录，避免覆盖用户主动设置的自定义值

-- DeepSeek 模型回填
UPDATE user_model_configs
SET deepseek_model = 'deepseek-v4-pro'
WHERE deepseek_model = 'deepseek-chat';

-- 讯飞星火模型回填
UPDATE user_model_configs
SET xunfei_model = 'spark-v4.0'
WHERE xunfei_model IN ('spark-v3.5', 'spark-v3', 'spark-v2');
