# LLM 网关（A1 AI 运行基座）

> 2026-08-30 发布冲刺引入。所有面向大模型的对话调用统一经 `LLMGateway` 出口。

## 定位

`server/internal/service/llm_gateway.go`。与既有组件的关系：

| 组件 | 职责 | 关系 |
|------|------|------|
| `LLMGateway` | 模型路由 + 调用审计（本表） | ChatService 唯一 LLM 出口 |
| `llm.FailoverClient` | 主备模型容灾（超时切换） | 网关的底层 client 可为 Failover |
| `TokenStatsService` | 配额计量（日/月） | 保持不变，仍由调用点触发 |
| `llm_call_logs` 表 | 调用审计（trace、延迟、用量、成败） | 网关落库 |

## 三条链路已接入

- 同步问答：`ChatService.chatViaGateway`（chat_service_assemble.go / chat_service_temporal.go）
- 流式问答：`ChatService.streamViaGateway`（chat_service_stream.go）
- 用户自备 Key 路由：`LLMGateway.resolveOverride`（原 resolveUserLLMOverrides 逻辑迁移至此）

## 审计字段（migrations/113_llm_call_logs.sql）

`trace_id / user_id / session_id / provider / model / prompt_tokens / output_tokens / latency_ms / status / error_msg`

- 同步调用：latency = 全耗时；流式：latency = 首包耗时，tokens 由消费方统计（流式协议不含 usage）
- 落库失败记 `[WARN]`，不影响对话主链路（与 P0-2 静默丢错治理一致）

## 后续扩展点（A2/A3）

- 降级链配置化（免费 → DeepSeek → 付费）在网关收口，不再散落调用点
- 成本记账（价格表 × tokens）可直接挂 `record()`
