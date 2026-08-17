-- 091_student_profile_snapshot_history.sql — 数字孪生快照「历史留痕」底座（P1-2 A 路径，2026-08-17）
--
-- 与 student_profile_snapshot（043，按 user_id UNIQUE、每次 Upsert 覆盖）并存的**独立纯追加历史表**：
--   - 主快照表仍为「每学生最近一行」，行为一字不改（UpsertSnapshot 语义不变）；
--   - 本表按 (user_id, computed_at) 每天最多一条，追加记录「每次重算的纵向采样」，
--     供 P1-2 growth_trend（成长归因/趋势）与 P1-1 Trends 共用同一纵向数据底座。
--
-- 设计要点（遵循 pm-check-nurture-wiring.md 方案 a）：
--   1. **不含 AI 长文本**（ai_interpretation/gap_analysis/stage_advice 三长大文本列不落历史，
--      它们按月变化大且看板/纵向归因不用，存此会膨胀；纵向只需要五维分数）。
--   2. `UNIQUE(user_id, computed_at)`：同一学生同一天多次重算只保留一条（按「天」去抖，
--      膨胀≈学生数×天数，可控；也是纵向按天采样的基础）。
--   3. 两索引：
--      - (user_id, computed_at)：纵向取「某学生近 N 周」历史（趋势主路径）；
--      - (owner_scope, owner_id, computed_at)：书记按院聚合历史趋势（沿用越权红线，带 owner 收窄）。
--   4. 方言兼容：仅 CREATE TABLE + CREATE INDEX，走 ToMySQL 双向（SQLite/MySQL/Turso 均成立）；
--      computed_at 用 TEXT（SQLite datetime）→ MySQL DATETIME 自动转换。
--
-- 写出时机：方案 A —— 在 twin_service.GetDigitalTwin 写主快照处「同写历史」（新增 repo 方法，
-- 失败仅告警不阻断主流程）。本表结构不依赖调度器，为后续方案 B（周期快照）预留扩展。

CREATE TABLE IF NOT EXISTS snapshot_history (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    -- 冗余归属，便于按 scope 聚合（与 users 表同步，沿用主快照口径）
    owner_scope     TEXT    NOT NULL DEFAULT 'college',
    owner_id        TEXT    NOT NULL DEFAULT 'default',
    college         TEXT    DEFAULT '',
    major           TEXT    DEFAULT '',
    class_name      TEXT    DEFAULT '',

    -- 五维分数（0-100，越高越好）——与主快照同口径，纵向差分 basis
    academic_score    REAL  NOT NULL DEFAULT 0,   -- 学业
    ability_score     REAL  NOT NULL DEFAULT 0,   -- 能力
    ideological_score REAL  NOT NULL DEFAULT 0,   -- 思想
    emotional_score   REAL  NOT NULL DEFAULT 0,   -- 情感
    social_score      REAL  NOT NULL DEFAULT 0,   -- 社交

    computed_at     TEXT    NOT NULL,             -- 快照计算时间（业务侧归一化到「当天日期 YYYY-MM-DD 00:00:00」以便按天去抖）
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- 跨日去重 + 纵向采样主约束：每学生每天 ≤ 1 条历史
CREATE UNIQUE INDEX IF NOT EXISTS idx_snapshot_history_user_day ON snapshot_history(user_id, computed_at);
-- 院级历史趋势聚合索引
CREATE INDEX IF NOT EXISTS idx_snapshot_history_scope ON snapshot_history(owner_scope, owner_id, computed_at);
-- 全局按时间聚合索引
CREATE INDEX IF NOT EXISTS idx_snapshot_history_date ON snapshot_history(computed_at);
