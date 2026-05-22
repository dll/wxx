-- 019_feedback_screenshot_blob.sql — 反馈截图直接存进 SQLite，跨 Vercel 实例可读
-- Vercel serverless 的 /tmp 文件系统是 ephemeral 且每实例独立，
-- 截图存文件会在实例回收后丢失。改为存入 DB blob（base64 字符串），
-- /uploads/feedback/{filename} 路由从 DB 读出直接返回 PNG。

CREATE TABLE IF NOT EXISTS feedback_screenshots (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    filename    TEXT    NOT NULL UNIQUE,        -- 与原 url 路径中的文件名一致
    mime_type   TEXT    NOT NULL DEFAULT 'image/png',
    size_bytes  INTEGER NOT NULL DEFAULT 0,
    data_base64 TEXT    NOT NULL,               -- base64 编码的图片字节
    uploaded_by TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_feedback_screenshots_filename
    ON feedback_screenshots(filename);
