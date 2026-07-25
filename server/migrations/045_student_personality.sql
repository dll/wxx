-- 045_student_personality.sql — 学生性格洞察持久化（S1 学生核心功能）
--
-- 设计说明：
--   存储 LLM 推断或问卷得出的性格画像（VARK 学习风格 + 大五人格简版）。
--   每用户一条记录（UNIQUE user_id），定期由 service 重算（结合行为数据）。
--   raw_answers 存原始问卷答案（JSON），供重算或审计。

CREATE TABLE IF NOT EXISTS student_personality (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL UNIQUE REFERENCES users(id),
    -- VARK 学习风格偏好分数（0-100）
    visual_score    REAL    NOT NULL DEFAULT 0,
    auditory_score  REAL    NOT NULL DEFAULT 0,
    reading_score   REAL    NOT NULL DEFAULT 0,
    kinesthetic_score REAL  NOT NULL DEFAULT 0,
    -- 大五人格（0-100）
    openness        REAL    NOT NULL DEFAULT 0,
    conscientiousness REAL  NOT NULL DEFAULT 0,
    extraversion    REAL    NOT NULL DEFAULT 0,
    agreeableness   REAL    NOT NULL DEFAULT 0,
    neuroticism     REAL    NOT NULL DEFAULT 0,
    -- LLM 综合解读
    personality_type TEXT   NOT NULL DEFAULT '',  -- 如 INTJ / 建筑师型
    type_label       TEXT   NOT NULL DEFAULT '',  -- 中文标签
    description      TEXT   NOT NULL DEFAULT '',  -- 一段话描述
    learning_style   TEXT   NOT NULL DEFAULT '',  -- 学习风格建议
    strengths        TEXT   NOT NULL DEFAULT '[]', -- JSON 数组
    weaknesses       TEXT   NOT NULL DEFAULT '[]', -- JSON 数组
    career_suggestions TEXT NOT NULL DEFAULT '[]', -- JSON 数组
    -- 原始数据
    raw_answers      TEXT   NOT NULL DEFAULT '{}', -- 问卷原始回答 JSON
    data_source      TEXT   NOT NULL DEFAULT 'llm', -- llm/questionnaire/fallback
    computed_at      TEXT   NOT NULL DEFAULT (datetime('now')),
    created_at       TEXT   NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT   NOT NULL DEFAULT (datetime('now'))
);
