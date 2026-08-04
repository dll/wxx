-- 054: 内置种子账号保活 — 用户要求内置账号不可停用
-- 背景：007/016 种子账号用 INSERT OR IGNORE 未显式设置 status；
--      生产库个别内置账号 status 异常（pending/disabled）导致登录报
--      "账号尚未启用或已被停用"（auth_service.go:456 拦截）。
-- 处理：
--  1. 强制启用全部内置账号（幂等）
--  2. admin 强制为 sys_admin（016 仅 OR IGNORE，已存在时 role 可能被自动创建覆盖）
--  3. admin 口令固定为项目指定 Wxx@2026；其余内置账号仅空密码时补齐默认 wxx@2025

UPDATE users SET status = 'active'
WHERE username IN (
  'admin','sysadmin','schooladmin','collegeadmin',
  'counselor_cs','counselor_math','stunion',
  'student_cs','student_math','teacher1','assistant1',
  'student1','counselor1','counselor2'
);

UPDATE users SET role = 'sys_admin', owner_scope = 'school', owner_id = 'all'
WHERE username = 'admin';

-- Wxx@2026 (bcrypt $2a$10$)
UPDATE users SET password_hash = '$2a$10$hTeu3DreEVsQkC2n3zjyEePYGmq0T199B/HNT9fU64fEPfy1STGkW'
WHERE username = 'admin';

-- wxx@2025 (bcrypt $2b$12$，与 029 一致)，仅空密码时补齐
UPDATE users SET password_hash = '$2b$12$YkzWU.X9v6G6l2cPqWGGvuOFLME4Y8Ym16hUkowJ1MSvzFgQCnuUS'
WHERE username IN (
  'sysadmin','schooladmin','collegeadmin',
  'counselor_cs','counselor_math','stunion',
  'student_cs','student_math','teacher1','assistant1',
  'student1','counselor1','counselor2'
)
  AND (password_hash = '' OR password_hash IS NULL);
