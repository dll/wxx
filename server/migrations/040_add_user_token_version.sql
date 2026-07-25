-- 040_add_user_token_version.sql — 为 users 表新增 token_version 令牌版本列
-- 修复审核问题 S-01：旧 JWT 复活 / 无法吊销令牌
--
-- 机制说明：
--   1. JWT 签发时写入当前 token_version（tv 声明）。
--   2. 每次请求在 EnsureUserExists 中以数据库 token_version 为权威比对；
--      不一致（旧于数据库）则判定令牌已被吊销，拒绝访问。
--   3. 管理员停用/降权/重置密码等敏感操作时对 token_version +1，
--      即可让该用户此前签发的所有 JWT 立即失效，无需等待自然过期。
--
-- 幂等：SQLite 不支持 ADD COLUMN IF NOT EXISTS，execSQL 已处理重复列错误（跳过）。

ALTER TABLE users ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;
