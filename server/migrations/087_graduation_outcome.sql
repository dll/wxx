-- 087: 书记教育成果 —— 毕业去向登记表（全生命周期闭环最后一环：离校→就业/升学）
-- 用途：存每个毕业生的真实毕业去向，点亮就业率/考研率。
-- 登记流：student(学生自报待审) / counselor·teacher·assistant(教辅直录/审核) 两级，
--       教辅录入或学生自报后，均由教辅（counselor/college_admin 等）审核才计入统计。
-- data_source=real，未审核(status!=approved)的记录不进入书记大屏统计。

CREATE TABLE IF NOT EXISTS graduation_outcome (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  student_id    INTEGER NOT NULL,             -- 毕业生用户ID（user_id，生命周期主键）
  student_name  TEXT    NOT NULL DEFAULT '',
  college       TEXT    NOT NULL DEFAULT '',  -- 学院（冗余，便于书记按学院聚合）
  major         TEXT    NOT NULL DEFAULT '',
  graduate_year INTEGER NOT NULL DEFAULT 0,   -- 毕业届别，如 2026
  outcome_type  TEXT    NOT NULL,             -- 去向：employment(就业)/postgrad(升学读研)/study_abroad(出国留学)/flexible(灵活就业)/entrepreneurship(创业)/unemployed(未就业)
  employer_name TEXT    NOT NULL DEFAULT '',  -- 就业单位 / 升学院校 / 出国学校
  position      TEXT    NOT NULL DEFAULT '',  -- 岗位 / 专业方向
  remark        TEXT    NOT NULL DEFAULT '',  -- 备注
  -- 审核流
  status        TEXT    NOT NULL DEFAULT 'pending',  -- pending(待审)/approved(通过计入统计)/rejected(驳回)
  submitted_by  INTEGER NOT NULL DEFAULT 0,   -- 提交人（学生自报 或 教辅录入），0=管理员导入
  submitted_role TEXT   NOT NULL DEFAULT '',  -- 提交人角色：student/counselor/teacher/assistant/admin
  reviewed_by   INTEGER NOT NULL DEFAULT 0,   -- 审核人（教辅）
  reviewed_name TEXT    NOT NULL DEFAULT '',
  review_note   TEXT    NOT NULL DEFAULT '',  -- 审核意见
  reviewed_at   TEXT    DEFAULT '',
  data_source   TEXT    NOT NULL DEFAULT 'real', -- 真实登记
  created_at    TEXT    NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  updated_at    TEXT    NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

CREATE INDEX IF NOT EXISTS idx_outcome_student ON graduation_outcome(student_id);
CREATE INDEX IF NOT EXISTS idx_outcome_college_year ON graduation_outcome(college, graduate_year);
CREATE INDEX IF NOT EXISTS idx_outcome_status ON graduation_outcome(status);
