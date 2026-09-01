-- 115_120001_counselor.sql
-- 用户反馈 2026-09-01：工号 120001（胡少启）正确身份为「辅导员 + 管理员」，
-- 并非教师。迁移 114 误将其设为 teacher（主）+ college_admin，本迁移修正：
--   1. users.role 主角色：teacher -> college_admin（管理员权限更高，作为主角色更合理）
--   2. user_roles：删除误加的 teacher，改回 counselor（辅导员），保留 college_admin
-- 说明：college_admin 自身继承 counselor/teacher/assistant 全部能力，
--      故即便不保留 counselor 亦具备辅导员能力；此处仍显式登记 counselor 以准确表达身份。
-- 方言兼容：INSERT OR IGNORE 会被迁移 runner 转为 MySQL 的 INSERT IGNORE；
--          DELETE 使用子查询（不用 MySQL 专属的多表 JOIN 语法），SQLite/MySQL 均可执行。

-- 1. 修正主角色（若当前非 sys_admin 则统一为 college_admin）
UPDATE users
SET role = 'college_admin', updated_at = CURRENT_TIMESTAMP
WHERE username = '120001' AND role NOT IN ('sys_admin');

-- 2. 删除误加的 teacher 身份（子查询定位 user_id，兼容双端）
DELETE FROM user_roles
WHERE user_id IN (SELECT id FROM users WHERE username = '120001')
  AND role = 'teacher';

-- 3. 增补 counselor 身份（如缺失则插入，幂等）
INSERT OR IGNORE INTO user_roles (user_id, role, granted_by)
SELECT id, 'counselor', 'migration_115' FROM users WHERE username = '120001';

-- 4. 确保 college_admin 身份存在（幂等）
INSERT OR IGNORE INTO user_roles (user_id, role, granted_by)
SELECT id, 'college_admin', 'migration_115' FROM users WHERE username = '120001';

-- 5. 账户状态确保可用
UPDATE users
SET status = 'active'
WHERE username = '120001' AND status != 'active';
