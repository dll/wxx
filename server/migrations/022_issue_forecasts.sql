-- 022_issue_forecasts.sql — 问题预案功能数据表
-- 支持 sys_admin、college_admin 角色查看问题预案分析

-- 1. 问题预案主表
CREATE TABLE IF NOT EXISTS issue_forecasts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    forecast_id     TEXT    NOT NULL UNIQUE,          -- 预案ID（UUID）
    college_id      TEXT    NOT NULL DEFAULT '',       -- 学院ID（空=全校）
    category        TEXT    NOT NULL,                  -- 问题分类：academic|emotion|attendance|complaint|process|discipline
    subcategory     TEXT    NOT NULL DEFAULT '',       -- 子分类
    title           TEXT    NOT NULL,                  -- 问题标题
    risk_level      TEXT    NOT NULL DEFAULT 'low',   -- 风险等级：low|medium|high|urgent
    status          TEXT    NOT NULL DEFAULT 'pending', -- 状态：pending|processing|resolved|archived
    affected_count  INTEGER NOT NULL DEFAULT 0,       -- 影响人数
    root_cause      TEXT    NOT NULL DEFAULT '',       -- 原因分析
    suggested_actions TEXT  NOT NULL DEFAULT '[]',    -- 建议措施（JSON数组）
    data_summary    TEXT    NOT NULL DEFAULT '{}',     -- 数据摘要（JSON）
    sources         TEXT    NOT NULL DEFAULT '[]',    -- 数据来源（JSON数组）
    ai_analysis     TEXT    NOT NULL DEFAULT '',       -- AI分析结果
    created_by      INTEGER,                          -- 创建人ID
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    resolved_at     TEXT,                             -- 解决时间
    resolved_by     INTEGER                           -- 解决人ID
);

-- 2. 问题详情表（存储具体受影响的学生/教师）
CREATE TABLE IF NOT EXISTS issue_details (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    forecast_id     TEXT    NOT NULL REFERENCES issue_forecasts(forecast_id),
    user_id         INTEGER,                          -- 用户ID
    user_type       TEXT    NOT NULL DEFAULT 'student', -- 用户类型：student|teacher|staff
    username        TEXT    NOT NULL DEFAULT '',       -- 用户名
    display_name    TEXT    NOT NULL DEFAULT '',       -- 显示名称
    college         TEXT    NOT NULL DEFAULT '',       -- 学院
    class_name      TEXT    NOT NULL DEFAULT '',       -- 班级
    detail_type     TEXT    NOT NULL,                  -- 详情类型：grade|attendance|emotion|leave|discipline
    detail_data     TEXT    NOT NULL DEFAULT '{}',     -- 详情数据（JSON）
    risk_score      REAL    NOT NULL DEFAULT 0.0,     -- 风险分数（0-1）
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- 3. 问题预案历史表（归档已解决的问题）
CREATE TABLE IF NOT EXISTS issue_forecast_history (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    forecast_id     TEXT    NOT NULL,
    action          TEXT    NOT NULL,                  -- 操作类型：create|update|resolve|archive
    operator_id     INTEGER,                          -- 操作人ID
    operator_name   TEXT    NOT NULL DEFAULT '',       -- 操作人名称
    detail          TEXT    NOT NULL DEFAULT '{}',     -- 操作详情（JSON）
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_issue_forecasts_college ON issue_forecasts(college_id);
CREATE INDEX IF NOT EXISTS idx_issue_forecasts_category ON issue_forecasts(category);
CREATE INDEX IF NOT EXISTS idx_issue_forecasts_risk_level ON issue_forecasts(risk_level);
CREATE INDEX IF NOT EXISTS idx_issue_forecasts_status ON issue_forecasts(status);
CREATE INDEX IF NOT EXISTS idx_issue_forecasts_created_at ON issue_forecasts(created_at);

CREATE INDEX IF NOT EXISTS idx_issue_details_forecast ON issue_details(forecast_id);
CREATE INDEX IF NOT EXISTS idx_issue_details_user ON issue_details(user_id);
CREATE INDEX IF NOT EXISTS idx_issue_details_type ON issue_details(detail_type);

CREATE INDEX IF NOT EXISTS idx_issue_forecast_history_forecast ON issue_forecast_history(forecast_id);
