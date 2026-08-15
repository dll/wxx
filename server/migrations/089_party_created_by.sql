-- 089 党课/活动登记：支撑"教师/教辅登记党课与积极分子活动"
-- 决策(待确认#2)：复用 party_study_records(不独立新表)，加 created_by/created_by_role 列，
-- 登记内容直接进现有 party 聚合(书记看板立即可见)，不新造虚构绩效。
-- created_by=NULL   => 学生自报(现有记录，原样兼容)
-- created_by>0      => 教师/教辅/书记登记(组织侧录入)
ALTER TABLE party_study_records ADD COLUMN created_by BIGINT NULL;
ALTER TABLE party_study_records ADD COLUMN created_by_role VARCHAR(128) NULL;
ALTER TABLE party_study_records ADD COLUMN paid INTEGER NULL DEFAULT 0;

-- 协同育人总览索引(按登记人检索)
CREATE INDEX idx_party_study_created_by ON party_study_records(created_by);
