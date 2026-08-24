-- vOPC A4 里程碑完整业务门禁：评分量表 / 条件通过 / 豁免 / 甲方结构化证据。
-- 纯数据模型 + 应用层枚举扩展；不改 vopc_milestone_reviews 表结构，不改 pass/return 语义。
-- 评分维度得分挂 review 行（vopc_milestone_reviews 有 UNIQUE(submission_id) 约束，
-- 一份提交一份评审记录，此处多行表示该评审的多个维度得分）；weight 列省略，仅用
-- 整数 max_score + min_pass，避免 REAL/DOUBLE 跨方言精度差异。

-- ① 评分量表定义（每阶段一组维度，阶段级共享维度建议源自 milestoneEvidence）
CREATE TABLE IF NOT EXISTS vopc_rubrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    stage TEXT NOT NULL,
    dimension_key TEXT NOT NULL,
    title TEXT NOT NULL,
    max_score INTEGER NOT NULL,
    min_pass INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(stage, dimension_key)
);
CREATE INDEX IF NOT EXISTS idx_vopc_rubrics_stage ON vopc_rubrics(stage);

-- ② 评审维度得分（挂 review 行）
CREATE TABLE IF NOT EXISTS vopc_review_scores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    review_id INTEGER NOT NULL,
    rubric_id INTEGER NOT NULL,
    score INTEGER NOT NULL,
    comment TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(review_id, rubric_id),
    FOREIGN KEY(review_id) REFERENCES vopc_milestone_reviews(id) ON DELETE CASCADE,
    FOREIGN KEY(rubric_id) REFERENCES vopc_rubrics(id)
);
CREATE INDEX IF NOT EXISTS idx_vopc_review_scores_review ON vopc_review_scores(review_id);

-- ③ 条件通过登记（挂 submission）
CREATE TABLE IF NOT EXISTS vopc_milestone_conditions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    submission_id INTEGER NOT NULL,
    description TEXT NOT NULL,
    due_at TEXT,
    satisfied INTEGER NOT NULL DEFAULT 0,
    satisfied_by INTEGER,
    satisfied_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(submission_id) REFERENCES vopc_milestone_submissions(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_milestone_conditions_sub ON vopc_milestone_conditions(submission_id, satisfied);

-- ④ 豁免申请/审批
CREATE TABLE IF NOT EXISTS vopc_milestone_waivers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    stage TEXT NOT NULL,
    required_evidence TEXT NOT NULL DEFAULT '',
    dimension_key TEXT,
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    requested_by INTEGER NOT NULL,
    reviewed_by INTEGER,
    review_note TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TEXT,
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_milestone_waivers_proj ON vopc_milestone_waivers(project_id, stage, status);

-- ⑤ 甲方结构化证据（区别于自由文本 evidence，S2/S5/S6 甲方项目强制阶段）
CREATE TABLE IF NOT EXISTS vopc_client_evidence (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    stage TEXT NOT NULL,
    submission_id INTEGER,
    client_rep TEXT NOT NULL,
    client_contact TEXT NOT NULL DEFAULT '',
    conclusion TEXT NOT NULL,
    sign_method TEXT NOT NULL DEFAULT '',
    file_ref TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    created_by INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(project_id) REFERENCES vopc_projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vopc_client_evidence_proj ON vopc_client_evidence(project_id, stage);
