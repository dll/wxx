-- 043_student_profile_snapshot.sql — 个人数字孪生数据底座（S1 5.1 / 7.5）
--
-- 设计说明：
--   数字孪生以五维模型刻画学生：学业(academic)、能力(ability)、思想(ideological)、
--   情感(emotional)、社交(social)。本快照表统一存储五维分数与 AI 解读，
--   供三条线复用：学生本人(self.twin.read)、辅导员看板(counselor.twin.board)、
--   学院大屏(college.twin.screen)，按 scope 收窄读取，避免各功能各自拉数据口径不一。
--
--   五维原始指标从既有业务表实时聚合（成绩/竞赛/党建/情感/社团），
--   本表存的是「最近一次计算的快照」，用于快速读取与趋势对比；
--   明细原始数据仍以各业务表为准。
--
-- 幂等：CREATE IF NOT EXISTS；owner_scope/owner_id 冗余便于按 scope 聚合大屏。

CREATE TABLE IF NOT EXISTS student_profile_snapshot (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL UNIQUE REFERENCES users(id),
    -- 冗余归属，便于辅导员/学院按 scope 聚合（与 users 表同步）
    owner_scope     TEXT    NOT NULL DEFAULT 'college',
    owner_id        TEXT    NOT NULL DEFAULT 'default',
    college         TEXT    DEFAULT '',
    major           TEXT    DEFAULT '',
    class_name      TEXT    DEFAULT '',

    -- 五维分数（0-100，越高越好）
    academic_score    REAL  NOT NULL DEFAULT 0,   -- 学业：GPA/成绩/通过率
    ability_score     REAL  NOT NULL DEFAULT 0,   -- 能力：竞赛/项目/规划完成
    ideological_score REAL  NOT NULL DEFAULT 0,   -- 思想：党建进度/理论学习
    emotional_score   REAL  NOT NULL DEFAULT 0,   -- 情感：情绪稳定度（风险越低分越高）
    social_score      REAL  NOT NULL DEFAULT 0,   -- 社交：社团/活动参与

    -- AI 解读与差距分析（JSON 字符串，兜底为空）
    ai_interpretation TEXT  NOT NULL DEFAULT '',  -- AI 状态解读文本
    gap_analysis      TEXT  NOT NULL DEFAULT '',  -- 差距分析（JSON 数组）
    stage_advice      TEXT  NOT NULL DEFAULT '',  -- 阶段建议（JSON 数组）

    computed_at     TEXT    NOT NULL DEFAULT (datetime('now')),  -- 快照计算时间
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_twin_scope ON student_profile_snapshot(owner_scope, owner_id);
CREATE INDEX IF NOT EXISTS idx_twin_college ON student_profile_snapshot(college);
CREATE INDEX IF NOT EXISTS idx_twin_class ON student_profile_snapshot(class_name);
