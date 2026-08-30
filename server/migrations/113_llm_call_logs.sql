-- A1 AI 运行基座：LLM 调用审计日志
-- 每次经 LLMGateway 的调用（同步/流式）记录一条：路由结果、用量、延迟、成败。
-- 与 token_usage（配额计量）互补：本表面向可观测与排障（trace_id 贯穿）。

CREATE TABLE IF NOT EXISTS llm_call_logs (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    trace_id      TEXT NOT NULL,
    user_id       INTEGER,
    session_id    TEXT,
    provider      TEXT NOT NULL,            -- 客户端名（如 deepseek / zhipu / failover(...)）
    model         TEXT,                     -- 实际使用的模型（含用户覆盖后的结果）
    prompt_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    latency_ms    INTEGER DEFAULT 0,        -- 同步=全耗时；流式=首包耗时
    status        TEXT NOT NULL,            -- ok / error
    error_msg     TEXT,
    created_at    TEXT DEFAULT (datetime('now','localtime'))
);

CREATE INDEX IF NOT EXISTS idx_llm_call_logs_user  ON llm_call_logs(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_llm_call_logs_trace ON llm_call_logs(trace_id);
CREATE INDEX IF NOT EXISTS idx_llm_call_logs_status ON llm_call_logs(status, created_at);
