-- 085: 用户职务字段
-- 角色管理增强：在「角色」之外补充「职务」（职务）。
-- 学生会成员可为 主席/副主席/部长/副部长/干事 等；
-- 教师/教辅/辅导员可填职称或职务；学生可填班干部等。
-- 与 role 正交：role 决定权限，position 描述岗位/职责（展示用）。

ALTER TABLE users ADD COLUMN position TEXT NOT NULL DEFAULT '';
