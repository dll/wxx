-- 033_students_2025_level.sql — 导入2025级测试学生
-- 用于测试学生角色各功能（课程地图、打卡、数字孪生等）

INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id, status, password_hash)
SELECT '20250101001', '张测试', 'student', 'college', 'cs', 'active',
       '$2b$12$YkzWU.X9v6G6l2cPqWGGvuOFLME4Y8Ym16hUkowJ1MSvzFgQCnuUS' -- Wxx@2026
WHERE NOT EXISTS (SELECT 1 FROM users WHERE username = '20250101001');

INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id, status, password_hash)
SELECT '20250101002', '李测试', 'student', 'college', 'cs', 'active',
       '$2b$12$YkzWU.X9v6G6l2cPqWGGvuOFLME4Y8Ym16hUkowJ1MSvzFgQCnuUS'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE username = '20250101002');

INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id, status, password_hash)
SELECT '20250101003', '王测试', 'student', 'college', 'cs', 'active',
       '$2b$12$YkzWU.X9v6G6l2cPqWGGvuOFLME4Y8Ym16hUkowJ1MSvzFgQCnuUS'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE username = '20250101003');

INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id, status, password_hash)
SELECT '20250101004', '赵测试', 'student', 'college', 'cs', 'active',
       '$2b$12$YkzWU.X9v6G6l2cPqWGGvuOFLME4Y8Ym16hUkowJ1MSvzFgQCnuUS'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE username = '20250101004');

INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id, status, password_hash)
SELECT '20250101005', '陈测试', 'student', 'college', 'cs', 'active',
       '$2b$12$YkzWU.X9v6G6l2cPqWGGvuOFLME4Y8Ym16hUkowJ1MSvzFgQCnuUS'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE username = '20250101005');

INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id, status, password_hash)
SELECT '20250201001', '刘测试', 'student', 'college', 'cs', 'active',
       '$2b$12$YkzWU.X9v6G6l2cPqWGGvuOFLME4Y8Ym16hUkowJ1MSvzFgQCnuUS'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE username = '20250201001');

INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id, status, password_hash)
SELECT '20250201002', '黄测试', 'student', 'college', 'cs', 'active',
       '$2b$12$YkzWU.X9v6G6l2cPqWGGvuOFLME4Y8Ym16hUkowJ1MSvzFgQCnuUS'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE username = '20250201002');

INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id, status, password_hash)
SELECT '20250201003', '吴测试', 'student', 'college', 'cs', 'active',
       '$2b$12$YkzWU.X9v6G6l2cPqWGGvuOFLME4Y8Ym16hUkowJ1MSvzFgQCnuUS'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE username = '20250201003');