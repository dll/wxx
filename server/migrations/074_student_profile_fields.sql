-- 074_student_profile_fields.sql — 学生档案扩展字段（2026级新生录取数据）
-- 幂等：execSQL 已处理 ADD COLUMN 重复列错误（跳过而不报错）
ALTER TABLE users ADD COLUMN gender TEXT DEFAULT '';            -- 性别：男/女
ALTER TABLE users ADD COLUMN campus TEXT DEFAULT '会峰校区';     -- 校区（默认会峰校区）
ALTER TABLE users ADD COLUMN education_level TEXT DEFAULT '';    -- 学历层次：本科等
ALTER TABLE users ADD COLUMN study_duration TEXT DEFAULT '';     -- 学制：4 等
ALTER TABLE users ADD COLUMN expected_graduation_date TEXT DEFAULT ''; -- 预期毕业时间
ALTER TABLE users ADD COLUMN study_mode TEXT DEFAULT '';         -- 学习形式：普通全日制等
ALTER TABLE users ADD COLUMN ethnicity TEXT DEFAULT '';          -- 民族：汉族等
ALTER TABLE users ADD COLUMN political_status TEXT DEFAULT '';   -- 政治面貌：共青团员/群众等
ALTER TABLE users ADD COLUMN birth_date TEXT DEFAULT '';         -- 出生年月（隐私字段）
