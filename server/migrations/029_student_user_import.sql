-- 029_student_user_import.sql — 学生账号扩展字段与无密码账号收口

ALTER TABLE users ADD COLUMN college TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN major TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN class_name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN enrollment_date TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN enrollment_year TEXT NOT NULL DEFAULT '';

-- 历史种子账号原本允许免密登录。预发布阶段统一补齐 bcrypt 密码，
-- 初始明文仅用于开发验收，见 docs/user-import.md，上线前必须修改。
UPDATE users
SET password_hash = '$2b$12$YkzWU.X9v6G6l2cPqWGGvuOFLME4Y8Ym16hUkowJ1MSvzFgQCnuUS'
WHERE password_hash = '' AND role <> 'guest';
