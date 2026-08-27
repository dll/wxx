-- 109_feedback_repair_tasks.sql
-- 反馈修复任务闭环（MVP）：审核通过后的修复任务实体，承载
--   创建(审核) -> 执行端领取 -> 验证上报 -> 管理员验收/驳回 -> 部署确认/完成
-- 服务器绝不执行源码修改、构建或部署；一切改码动作发生在受控本机执行端。
-- 本迁移兼容 SQLite/MySQL（由 internal/db 方言层在 MySQL 下自动转换）。
CREATE TABLE IF NOT EXISTS feedback_repair_tasks (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    task_no             TEXT NOT NULL UNIQUE,          -- rt-xxxxxxxx
    creator             TEXT NOT NULL,                 -- 创建(审核)管理员用户名
    feedback_ids        TEXT NOT NULL,                 -- JSON 数组，支持单条/批量
    title               TEXT NOT NULL DEFAULT '',
    diagnosis           TEXT NOT NULL DEFAULT '',      -- 合并后的 AI 诊断(JSON)
    status              TEXT NOT NULL DEFAULT 'approved', -- 见下方状态机注释
    worker_host         TEXT NOT NULL DEFAULT '',      -- 执行端主机标识(领取时写入)
    worker_token_note   TEXT NOT NULL DEFAULT '',      -- 执行端认证说明(不存 token 本身)
    base_commit         TEXT NOT NULL DEFAULT '',
    branch              TEXT NOT NULL DEFAULT '',
    verify_result       TEXT NOT NULL DEFAULT '',      -- JSON: go_test/go_vet/flutter_analyze/flutter_test passed+摘要
    diff_stat           TEXT NOT NULL DEFAULT '',      -- git diff --stat 摘要
    log_text            TEXT DEFAULT '',               -- 任务分段日志(多行)
    accept_note         TEXT NOT NULL DEFAULT '',      -- 管理员验收备注
    accepted_by         TEXT NOT NULL DEFAULT '',
    reject_reason       TEXT NOT NULL DEFAULT '',
    rejected_by         TEXT NOT NULL DEFAULT '',
    deploy_confirmed_by TEXT NOT NULL DEFAULT '',
    deploy_ref          TEXT NOT NULL DEFAULT '',      -- 部署方式记录(GH run id / 手工命令)
    created_at          TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_frt_status ON feedback_repair_tasks(status);

-- 状态机注释（服务层硬编码校验，此处仅文档）：
--   approved            -> running（执行端领取 / 管理员取消）
--   running             -> verifying（执行端开始验证）/ verified_awaiting_acceptance（验证通过上报）/ verify_failed
--   verify_failed       -> running（重新认领）/ cancelled（管理员取消）
--   awaiting_acceptance -> deploy_pending（管理员验收 accept）/ verify_failed（管理员驳回 reject）
--   deploy_pending      -> deploying（管理员 deploy-confirm）/ verify_failed（管理员驳回）
--   deploying           -> deployed（管理员 deploy-done）
--   deployed            -> closed（管理员 close 或 deploy-done 可选联动）
--   cancelled / closed 为终态。
-- 说明：为降低 MVP 状态数，awaiting_acceptance 与 deploy_pending 均已归一为
--   上述字符串常量；verifying 阶段归入 running（verify_result 非空即已上报）。
