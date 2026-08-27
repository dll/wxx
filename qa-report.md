# 回归测试报告 — 反馈修复闭环 MVP → 一键自动修复

- 测试专员：qa-regression-wxx
- 测试日期：2026-08-27
- 项目根目录：`E:\2026-2027\2026-2027-1\MyProjects\wxx`
- 测试环境：Windows 10.0.26200 (x64)、go 1.26.0、PowerShell 7+、SQLite（内存测试库）
- 改动范围：`entity.go` / `feedback_repair_task_service.go` / `repair-agent.ps1`

## 结论（TL;DR）

**回归通过，未发现 P0/P1 缺陷。** 原有状态机与管理端/执行端接口语义保持不变；新增 `feedback_contents` 字段向后兼容，并为合法 JSON 数组（非 null）；单条反馈取不到时降级不崩溃（已有测试覆盖并通过）。发现 1 个 P2 性能优化建议（N+1 查询，见 §5.1）与若干无法本地实测的未覆盖项。

---

## 1. 代码走查结论

### 1.1 `model/entity.go`
- 新增 `FeedbackRepairContent` 结构体，字段 `feedback_id/module/category/content` 全部为 JSON 可序列化字符串，无敏感字段（不含截图 base64），符合「脱敏载荷」设计预期。
- `RepairTaskPayload` 新增 `FeedbackContents []FeedbackRepairContent`，tag 为 `json:"feedback_contents"`（无 `omitempty`），与 `FeedbackIDs` 并列。
- 状态机常量（approved→running→awaiting_acceptance→deploy_pending→deploying→deployed→closed，verify_failed/cancelled 分支）**未改动**。
- ✅ 通过。

### 1.2 `service/feedback_repair_task_service.go`
- 新增 `taskToPayloadWithContents()`，在 `taskToPayload()` 基础上逐条 `feedbackSvc.Get(fid)` 填充原文。
- **`Claim()` 出口**已改为调用 `taskToPayloadWithContents(t)` 返回 payload（原 taskToPayload 保留，供内部复用）。
- **降级逻辑正确**：单条 `Get` 返回 err 或 `nil` 时，`item` 仅保留 `FeedbackID`（module/category/content 空），且 `continue` 不影响后续条目与其他反馈。
- **JSON 数组非 null 保证**：`taskToPayload` 中 `FeedbackContents` 初始化为 `[]model.FeedbackRepairContent{}`（空切片而非 nil），故即使 FeedbackIDs 为空，序列化后也为 `"feedback_contents":[]`，**不会是 null**。
- `FeedbackService.Get(fid)` 内部走 `feedbackRepo.GetByFeedbackID`（单条主键/唯一键查询，无归属校验），语义与既有调用一致。
- 状态机流转方法（Accept/Reject/DeployConfirm/DeployDone/Cancel/SubmitVerify）逻辑**零改动**。
- ✅ 通过。

### 1.3 `scripts/repair-agent.ps1`
- 新增 `-Mode auto` 分支：claim → worktree 隔离分支 → 组装 prompt（含 feedback_contents）→ 调用编码工具（gemini/openclaw/自定义 WXX_REPAIR_CODER）→ verify 上报。
- 安全边界保持：不 commit/push/部署；token 从 `WXX_REPAIR_AGENT_TOKEN` 读取。
- `ValidateSet("claim","verify","auto")` 参数校验正确。
- `feedback_contents` 为空时组装提示词兜底为「(无反馈原文)」，不会抛空引用。
- PowerShell 解析（AST ParseFile）**无语法错误**。
- ✅ 通过（静态解析）。

---

## 2. 测试用例与执行结果

### 2.1 编译

| 用例 | 命令 | 结果 |
|---|---|---|
| 全量编译 | `go build ./...` | ✅ 通过（exit 0） |

### 2.2 单元/回归测试

| 包 | 命令 | 结果 |
|---|---|---|
| service | `go test -count=1 ./internal/service/` | ✅ ok（155.8s） |
| handler | `go test -count=1 ./internal/handler/` | ✅ ok（89.7s） |
| repository | `go test -count=1 ./internal/repository/` | ✅ ok（83.5s） |
| model | `go test -count=1 ./internal/model/` | ✅ ok（3.1s） |
| 全内部包 | `go test ./internal/...` | ✅ 全部 ok |

### 2.3 状态机/新增功能专项（-run TestRepairTask）

| 用例 | 结果 |
|---|---|
| TestRepairTask_StateMachine_FullChain（全链路） | ✅ PASS |
| TestRepairTask_StateMachine_VerifyFailedLoop（验证失败回路） | ✅ PASS |
| TestRepairTask_StateMachine_Cancel（取消，终态） | ✅ PASS |
| TestRepairTask_StateMachine_Reject（驳回） | ✅ PASS |
| TestRepairTask_IllegalTransition_ReturnsError（非法流转） | ✅ PASS |
| **TestRepairTask_Payload_ContainsFeedbackContents（新增）** | ✅ PASS |
| **TestRepairTask_Payload_MissingFeedback_Degrades（新增）** | ✅ PASS |
| TestRepairTask_ConcurrencyGate（并发闸门） | ✅ PASS |

> 新增测试 `Payload_ContainsFeedbackContents` 与 `Payload_MissingFeedback_Degrades` 已确认真正执行（`go test -count=1 -run TestRepairTask` 非缓存），并验证：
> - `feedback_contents` 包含逐条反馈原文，module/category/content 正确填充；
> - 单条 feedback 取不到时，保留 `feedback_id`、content 留空，**整体 Claim 不失败**。

---

## 3. 重点回归验证

### 3.1 Claim payload 含 `feedback_contents`，且为合法 JSON 数组
- **结论：正确。** `FeedbackContents` 初始化为空切片 `[]model.FeedbackRepairContent{}`（非 nil），`json.Marshal` 产出 `"feedback_contents":[]`，**非 null**。
- 向后兼容：旧执行端 JSON 反序列化时忽略未知/额外字段，`feedback_contents` 仅为新增字段，不破坏既有 `task_no/title/status/feedback_ids/diagnosis/...` 结构。

### 3.2 状态机语义不退化
- validTransitions 状态表未改动，全链路、verify_failed 回路、cancel、reject、非法流转各用例均 PASS。

### 3.3 管理端接口语义不变
- CreateTask/ListTasks/GetTask/Accept/Reject/DeployConfirm/DeployDone 方法体零改动，仅 Claim 出口新增反馈原文填充。handler 层 NextTask 返回值类型仍为 payload（`data` 字段），无破坏。

### 3.4 执行端 NextTask/VerifyTask 语义不变
- NextTask 仍返回 `{code:0, data: payload}`；空任务返回 `data: nil`；并发冲突返回 409。语义不变。

---

## 4. 缺陷分级

| 级别 | 编号 | 缺陷描述 | 状态 |
|---|---|---|---|
| P0 | — | 无（状态机/接口未退化，无崩溃/数据破坏风险） | 通过 |
| P1 | — | 无 | 通过 |
| P2 | #1 | 批量任务时逐条 `FeedbackService.Get` 导致 N+1 查询 | 建议优化，非阻塞 |

---

## 5. 并发 / 性能 / 兼容性隐患分析

### 5.1 N+1 查询（P2，性能建议）
`taskToPayloadWithContents` 对 `p.FeedbackIDs` 逐条调用 `feedbackSvc.Get(fid)`，每条触发一次 `GetByFeedbackID` 查询。批量任务（如 50~200 条反馈）时会产生 N 次查询。
- **影响评估**：Claim 为低频操作（全局仅 1 个 running 闸门），单次领取 N 通常很小（管理员批量创建但单任务反馈数有限），实际风险低。
- **建议**：可新增 `feedbackRepo.GetByFeedbackIDs(ids []string)` 批量 `WHERE feedback_id IN (...)` 一次性取回，再逐条映射，消除 N+1。当前实现非阻塞缺陷，不影响正确性。

### 5.2 并发闸门
`Claim` 先 `CountActiveRunning()`（running + awaiting_acceptance 计数）判断，再 `NextClaimable` → `UpdateClaim`。SQLite 单连接测试下串行安全；生产 MySQL 下 `CountActiveRunning` + `UpdateClaim` 之间非原子，理论上存在极小窗口的并发认领竞态。**与改动无关**（原逻辑即如此），且闸门为「全局仅 1 running」语义，风险可接受，未引入新竞态。

### 5.3 兼容性
- 新增 `feedback_contents` 为**增量字段**，旧执行端忽略即可，向后兼容 ✅。
- `diagnosis` 使用 `omitempty`，`feedback_contents` **无** `omitempty`（保证恒输出 `[]`），符合「为空也是合法 JSON 数组而非 null」要求 ✅。
- 新增 `FeedbackRepairContent` 无指针字段，循环 append 中每次新建 `item` 变量，无共享指针别名问题 ✅。

### 5.4 PowerShell auto 模式
- `gemini -p $prompt` 与 `openclaw $prompt` 依赖本机 CLI 环境；未安装时命令会失败，但脚本不 push/部署，失败仅影响本次 auto 流程（无副作用）。
- `Invoke-Expression` 用于自定义 coder，属受控本机工具，提示词已 `Set-Content` 到 worktree 的 `repair-prompt.txt`（不入库，通过后删除）。

---

## 6. 未覆盖项（如实标注）

| 未覆盖项 | 原因 |
|---|---|
| 真实云端 MySQL 连接 | 测试使用内存 SQLite（`testutil.NewTestDBFull`）；生产 MySQL 的 N+1/并发行为未在真实库复现 |
| 真实触发 gemini CLI | 本机未实际执行 `gemini -p` 编码改码流程（仅静态解析 + AST 语法校验） |
| flutter analyze/test 真实回归 | 前端 Flutter 侧未在本轮验证（改动集中在 Go 后端 + PowerShell） |
| 端到端 HTTP 链路（带 token 中间件）的实际请求 | handler 层通过单测间接覆盖，未发起真实 `POST /api/v1/internal/repair-tasks/next` 请求 |
| `feedback_contents` 大 JSON payload 的实际体积/带宽 | 未压测 |

---

## 7. 附：关键证据

- `go build ./...` exit 0。
- `go test ./internal/...` 全部 `ok`。
- `go test -count=1 -run TestRepairTask` 8 用例全 PASS（含 2 个新增用例）。
- `repair-agent.ps1`：`Parser.ParseFile` 无语法错误（`PARSE OK - no syntax errors`）。

**最终结论：本次「一键自动修复」增强改动正确，原有功能无退化，新增功能符合预期，可进入下一步（reviewer 评审）。**
