-- 075_must_change_password.sql — 首次登录强制修改密码
-- 学生导入后密码为学号，首次登录必须修改（幂等：重复列错误由 execSQL 跳过）
ALTER TABLE users ADD COLUMN must_change_password INTEGER NOT NULL DEFAULT 0;
