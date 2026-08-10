-- 反馈 AI 自动修复工单
-- 执行自动修复（改码→构建→健康检查→热部署→失败回滚）时记录状态，供前端轮询与审计
CREATE TABLE IF NOT EXISTS feedback_repair_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL UNIQUE,              -- 内部运行 ID，如 rp-xxxxxxxx
    feedback_id TEXT NOT NULL,                -- 关联反馈反馈ID
    operator TEXT NOT NULL,                   -- 触发人用户名
    status TEXT NOT NULL DEFAULT 'running',   -- running | succeeded | failed | rolled_back
    stage TEXT NOT NULL DEFAULT 'init',       -- init/diagnose/apply/build/deploy/healthcheck/done/failed
    log_text TEXT DEFAULT '',                 -- 分段日志
    edited_files TEXT DEFAULT '',             -- JSON 数组：被修改的文件（相对仓库根）
    summary TEXT DEFAULT '',
    detail TEXT DEFAULT '',                   -- 修复说明/错误描述
    created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_feedback_repair_jobs_feedback ON feedback_repair_jobs(feedback_id);