-- 041_add_user_consented.sql — 为 users 表新增 consented 隐私授权列
-- 修复审核问题 SEC-02：隐私政策/用户协议同意状态未持久化，RequireConsent 中间件形同虚设
--
-- 机制说明：
--   1. 用户首次调用受保护接口前，前端弹窗引导阅读并同意隐私政策与用户协议。
--   2. POST /api/v1/user/consent 将 consented 置 1 并持久化。
--   3. EnsureUserExists 中以数据库 consented 为权威注入 UserContext.Consented；
--      RequireConsent 中间件据此在数据写入类接口拦截未授权用户。
--
-- 幂等：SQLite 不支持 ADD COLUMN IF NOT EXISTS，execSQL 已处理重复列错误（跳过）。
-- 存量用户默认置 1，避免升级后已在用用户被强制重新授权而中断服务。

ALTER TABLE users ADD COLUMN consented INTEGER NOT NULL DEFAULT 1;
