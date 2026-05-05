-- 007_seed_users.sql — 种子用户（全部 8 种角色，覆盖多作用域）
-- 开发/测试环境使用，生产环境部署前应移除或替换

-- ── 系统管理员 ──
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('sysadmin', '系统管理员', 'sys_admin', 'school', 'all');

-- ── 学校管理员 ──
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('schooladmin', '学校管理员', 'school_admin', 'school', 'all');

-- ── 学院管理员（信息学院） ──
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('collegeadmin', '信息学院管理员', 'college_admin', 'college', 'cs');

-- ── 辅导员（信息学院 / 数理学院） ──
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('counselor_cs', '李辅导员', 'counselor', 'college', 'cs');
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('counselor_math', '王辅导员', 'counselor', 'college', 'math');

-- ── 学生会 ──
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('stunion', '学生会主席', 'student_union', 'college', 'cs');

-- ── 学生（信息学院 / 数理学院） ──
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('student_cs', '张同学', 'student', 'college', 'cs');
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('student_math', '李同学', 'student', 'college', 'math');

-- ── 扩展角色：教师 / 教辅 ──
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('teacher1', '王老师', 'teacher', 'college', 'cs');
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('assistant1', '赵教辅', 'assistant', 'college', 'cs');

-- ── 修复已存在用户的 owner_scope/owner_id（如果之前是 default） ──
UPDATE users SET owner_scope = 'school', owner_id = 'all'
WHERE username = 'admin' AND owner_id = 'default';

UPDATE users SET owner_scope = 'college', owner_id = 'cs'
WHERE username = 'counselor1' AND owner_id = 'default';
