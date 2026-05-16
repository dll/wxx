-- 013_user_model_config.sql — 用户 AI 模型配置表
-- 每个用户可为 DeepSeek/智谱/讯飞分别配置 API Key 和模型参数

CREATE TABLE IF NOT EXISTS user_model_configs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL UNIQUE REFERENCES users(id),
    -- DeepSeek 配置
    deepseek_key    TEXT    NOT NULL DEFAULT '',
    deepseek_model  TEXT    NOT NULL DEFAULT 'deepseek-v4-pro',
    deepseek_temp   REAL    NOT NULL DEFAULT 0.7,
    deepseek_max_tokens INTEGER NOT NULL DEFAULT 2048,
    -- 智谱清言配置
    zhipu_key       TEXT    NOT NULL DEFAULT '',
    zhipu_model     TEXT    NOT NULL DEFAULT 'glm-4',
    zhipu_temp      REAL    NOT NULL DEFAULT 0.7,
    zhipu_max_tokens INTEGER NOT NULL DEFAULT 2048,
    -- 讯飞星火配置
    xunfei_app_id   TEXT    NOT NULL DEFAULT '',
    xunfei_key      TEXT    NOT NULL DEFAULT '',
    xunfei_secret   TEXT    NOT NULL DEFAULT '',
    xunfei_model    TEXT    NOT NULL DEFAULT 'spark-v4.0',
    xunfei_temp     REAL    NOT NULL DEFAULT 0.7,
    xunfei_max_tokens INTEGER NOT NULL DEFAULT 2048,
    -- 默认模型选择
    default_provider TEXT    NOT NULL DEFAULT 'deepseek',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);
