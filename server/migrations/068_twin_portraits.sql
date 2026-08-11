-- 068_twin_portraits.sql — 数字孪生画像
-- 存储基于用户照片/超星原型生成的蔚小芯风格画像
-- 图片以 base64 存 SQLite（与 users.avatar_base64 / feedback_screenshots 同模式）
CREATE TABLE IF NOT EXISTS twin_portraits (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id        INTEGER NOT NULL,
    prototype_type TEXT NOT NULL DEFAULT 'photo',  -- photo(用户照片) | chao_xing(超星原型)
    prompt_version TEXT NOT NULL DEFAULT '1.0',    -- 提示词版本，升级可重新生成
    image_base64   TEXT NOT NULL,                  -- 生成的画像(base64)
    image_mime     TEXT NOT NULL DEFAULT 'image/png',
    source_photo_base64 TEXT DEFAULT '',           -- 原型照片(base64，仅 photo 模式)
    created_at     TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, prototype_type)
);
CREATE INDEX IF NOT EXISTS idx_twin_portraits_user ON twin_portraits(user_id);
