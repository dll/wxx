-- 073_ai_briefings_refactor.sql — AI 简讯参考 AIHOT 重构
-- 1. 新增热度值与推荐理由字段
ALTER TABLE ai_briefings ADD COLUMN heat INTEGER NOT NULL DEFAULT 0;   -- 热度值（管理端维护，无自动算法）
ALTER TABLE ai_briefings ADD COLUMN reason TEXT NOT NULL DEFAULT '';   -- 推荐理由

-- 2. 用户收藏表
CREATE TABLE IF NOT EXISTS ai_briefing_favorites (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,          -- 收藏用户 ID
    briefing_id INTEGER NOT NULL,          -- 资讯 ID
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (user_id, briefing_id)
);
CREATE INDEX IF NOT EXISTS idx_ai_briefing_favorites_user ON ai_briefing_favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_briefing_favorites_briefing ON ai_briefing_favorites(briefing_id);
