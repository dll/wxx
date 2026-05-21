-- 016_fix_seed_users.sql — 补充种子用户，修复 Vercel 多实例 SQLite 问题
-- 背景：Vercel serverless 每实例独立 /tmp/wxx.db，LoginByUsername 自动创建的用户
-- 只在当前实例存在，切换到其他实例时 FOREIGN KEY 约束失败。
-- 解决：在数据库初始化时预置所有常用测试用户，确保跨实例一致。

-- ── 补充学生（student1） ──
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('student1', 'student1', 'student', 'college', 'cs');

-- ── 补充辅导员（counselor1, counselor2） ──
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('counselor1', 'counselor1', 'counselor', 'college', 'cs');

INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('counselor2', 'counselor2', 'counselor', 'college', 'math');

-- ── 补充系统管理员（admin） ──
INSERT OR IGNORE INTO users (username, display_name, role, owner_scope, owner_id)
VALUES ('admin', 'admin', 'sys_admin', 'school', 'all');

-- ── 修复已存在的自动创建用户（owner_id = 'default' 表示是自动创建的） ──
UPDATE users SET role = 'counselor', owner_scope = 'college', owner_id = 'cs'
WHERE username = 'counselor1' AND owner_id = 'default';

UPDATE users SET role = 'counselor', owner_scope = 'college', owner_id = 'math'
WHERE username = 'counselor2' AND owner_id = 'default';

UPDATE users SET role = 'sys_admin', owner_scope = 'school', owner_id = 'all'
WHERE username = 'admin' AND owner_id = 'default';

UPDATE users SET role = 'student', owner_scope = 'college', owner_id = 'cs'
WHERE username = 'student1' AND owner_id = 'default';
