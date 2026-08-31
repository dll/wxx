-- 114_user_roles.sql — 多角色模型（2026-09-01）
--
-- 背景：
--   原 users.role 为单角色字段，权限沿 auth 角色继承树解析。
--   业务需要"教师 + 管理员"等多重身份（如工号 120001 胡少启），
--   单角色无法表达，且单靠角色继承链会把"管理员"误升为全部低阶身份。
--
-- 设计（最小侵入）：
--   1) 新增 user_roles 关联表：一个用户可挂多个角色。
--   2) users.role 保留为主角色（兼容存量代码/JWT 单 role 字段），
--      取 user_roles 中"权限最高"的角色；user_roles 为空时行为与旧版完全一致。
--   3) 权限判定：主角色能力 + user_roles 各角色能力的并集。
--      （并集逻辑由 auth 包 HasAnyRole/CapabilitiesOfAny 实现，见 capabilities.go）
--
-- 兼容性：
--   - 老用户（user_roles 无记录）：登录/权限/统计与旧版完全一致。
--   - JWT 仍只带主角色 role，另新增 roles 数组；旧 token 无 roles 字段时
--     中间件回退为 [role]，零破坏。

-- 用户多角色关联表
CREATE TABLE IF NOT EXISTS user_roles (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    role       TEXT    NOT NULL,
    granted_by TEXT    NOT NULL DEFAULT '',
    created_at TEXT    NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    UNIQUE(user_id, role)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_user ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role);

-- 任务2：工号 120001（胡少启）升级为 教师+学院管理员 多重身份
-- 说明：
--   - users.role 主角色置 college_admin（权限最高，兼容旧代码/JWT 单角色展示）；
--   - user_roles 补记 teacher + college_admin 两身份；
--   - 不改动 owner_scope/owner_id/college，沿用该账号原有数据范围。
UPDATE users SET role = 'college_admin', updated_at = CURRENT_TIMESTAMP
WHERE username = '120001' AND role NOT IN ('college_admin', 'school_admin', 'sys_admin');

INSERT INTO user_roles (user_id, role, granted_by)
SELECT id, 'college_admin', 'migration_114'
FROM users WHERE username = '120001'
ON CONFLICT(user_id, role) DO NOTHING;

INSERT INTO user_roles (user_id, role, granted_by)
SELECT id, 'teacher', 'migration_114'
FROM users WHERE username = '120001'
ON CONFLICT(user_id, role) DO NOTHING;
