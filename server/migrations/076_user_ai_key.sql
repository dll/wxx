-- 076_user_ai_key.sql — 用户个人 AI API Key（AES-GCM 加密存储，仅供额度耗尽时用户自带 Key 使用）
ALTER TABLE users ADD COLUMN ai_api_key_enc TEXT DEFAULT '';   -- 加密后的 API Key（智谱/DeepSeek）
ALTER TABLE users ADD COLUMN ai_key_provider TEXT DEFAULT '';   -- zhipu / deepseek
