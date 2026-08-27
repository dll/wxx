# 反馈修复闭环 MVP — QA 回归测试报告

> 测试人：qa-regression-wxx（回归测试专员）· 日期：2026-08-27
> 范围：反馈修复闭环 MVP 重构后的回归验证。
> 前置输入：`pm-checklist.md`（缺口 G1-G10、改造 M1-M5）、`refactor-notes.md`（验证章节）、`audit-report.md`（第 2、5 节建议）。
> 工具链：Go 1.26.0 / Flutter 3.22.0（Dart 3.4.0）。
> 说明：本轮仅测试，未修改任何业务源码；新增 4 个 `*_test.go` 测试文件用于佐证，可随仓库保留或清理。

---

## 一、改动影响面

| 层 | 新/改文件 | 影响 |
|---|---|---|
| 模型 | `internal/model/entity.go` | 新增 `feedback_repair_tasks` 实体、状态常量（approved/running/…/closed/cancelled）、DTO/请求/载荷结构 |
| 迁移 | `migrations/109_feedback_repair_tasks.sql` | 新建 `feedback_repair_tasks` 表 + 状态索引（SQLite/MySQL 双方言） |
| 仓库 | `internal/repository/feedback_repair_task_repo.go` | 任务 CRUD + 状态更新 + 并发计数 |
| 仓库 | `internal/repository/feedback_repo.go` | 新增 `CountScreenshotRefsByUser`（截图归属校验） |
| 仓库 | `internal/repository/feedback_screenshot_repo.go` | 新增 `OwnerByFilename` |
| 服务 | `internal/service/feedback_repair_task_service.go` | 状态机 + Create/List/Get/Cancel/Claim/SubmitVerify/Accept/Reject/DeployConfirm/DeployDone |
| 服务 | `internal/service/feedback_service.go` | 新增 `GetAuthorized`、`CanAccessScreenshot`，`Get`/`ListLogs` 保留为内部兼容 |
| Handler | `internal/handler/feedback_handler.go` | `Get`/`GetLogs` 改走 `GetAuthorized`；`ServeScreenshot` 加双重归属校验 |
| Handler | `internal/handler/feedback_repair_task_handler.go` | 管理端 + 内部执行端两组入口 |
| 中间件 | `internal/middleware/repair_agent_token.go` | 执行端 token 鉴权（ConstantTimeCompare） |
| 装配 | `pkg/app/app.go` / `pkg/app/deps.go` / `pkg/app/routes.go` | 注入 task 服务/handler，注册管理端与内部路由 |
| 前端 | `frontend/lib/...`（文案/常量） | 「在线修复」→「修复诊断」语义纠正 + API 常量（本轮无完整任务 UI） |
| 脚本 | `scripts/repair-agent.ps1` | 执行端领取/验证上报（不自动改码/部署） |

**核心边界（复核确认）：** 服务器只做状态机与审计，无任何 shell 执行、文件写、构建、部署调用；`deploy-confirm`/`deploy-done` 仅写状态字段。

---

## 二、测试项清单与结果

> 判定：✅ 通过 / ❌ 失败（含真实证据）/ ⏭ 跳过（注明原因）。

### 2.1 工具链门禁

| 项 | 命令 | 结果 | 证据 |
|---|---|---|---|
| Go 编译 | `go build ./...` | ✅ | EXIT 0 |
| Go vet（核心包） | `go vet ./internal/handler ./internal/service ./pkg/app ./internal/db ./internal/middleware` | ✅ | EXIT 0 |
| 既有测试（核心包） | `go test ./internal/handler ./internal/service ./pkg/app ./internal/db ./internal/middleware -count=1` | ✅ | 5 包全部 `ok`（handler 90.454s/service 163.708s/app 77.322s/db 11.928s/middleware 13.574s） |
| Flutter analyze | `cd frontend && flutter analyze` | ✅（无 error/warning） | 312 条 **info**-level 既有 hint（`prefer_const_constructors`/`avoid_web_libraries_in_flutter` 等），抓取 `error -`/`warning -` 为空 |
| Flutter test | `cd frontend && flutter test` | ✅ | `+14: All tests passed!` EXIT 0 |
| 格式检查 | `gofmt -l`（4 个新增测试文件） | ✅ | 空输出 |

### 2.2 越权访问控制（audit 建议①）

新增 `internal/handler/feedback_access_test.go`（9 用例），全部 ✅：

| 场景 | 期望 | 结果 |
|---|---|---|
| 普通用户读本人反馈详情/日志 | 200 | ✅ |
| 越权第三方读他人反馈详情/日志 | 404（不泄露存在性） | ✅ |
| 反馈管理员（student_union，union.feedback.list）读他人反馈 | 200 | ✅ |
| 读不存在反馈 | 404 | ✅ |
| 越权第三方读他人截图 | 403 | ✅ |
| 截图上传者本人读 | 200 | ✅ |
| 引用截图的反馈提交者读（非上传者） | 200 | ✅ |
| 反馈管理员读任意截图 | 200 | ✅ |
| 未认证访问 | 401 | ✅ |

命令：`go test ./internal/handler -run 'TestFeedbackAccess|TestScreenshotAccess' -count=1` → `ok`。

### 2.3 修复任务状态机全链路（audit 建议②）

新增 `internal/service/feedback_repair_task_service_test.go`（6 用例），全部 ✅：

| 用例 | 覆盖流转 | 结果 |
|---|---|---|
| FullChain | approved→running→awaiting_acceptance→deploy_pending→deploying→deployed→closed | ✅ |
| VerifyFailedLoop | running→verify_failed→running→awaiting_acceptance | ✅ |
| Cancel | approved→cancelled（终态，再次取消报错） | ✅ |
| Reject | awaiting_acceptance→verify_failed | ✅ |
| IllegalTransition | approved 下非法验收/部署确认/验证上报均报 `ErrRepairTaskBadState` | ✅ |
| ConcurrencyGate | 存在 running 时第二个任务不可领取（`ErrRepairTaskConcurrency`） | ✅ |

命令：`go test ./internal/service -run 'TestRepairTask' -count=1` → 6 用例 PASS。

### 2.4 内部端点 token 鉴权（audit 建议③）

新增 `internal/middleware/repair_agent_token_test.go`（5 用例），全部 ✅：

| 场景 | 期望 | 实际 |
|---|---|---|
| token 未配置（空环境变量） | 404 | ✅ 404 |
| 已配置但缺失 Authorization 头 | 401 | ✅ 401 |
| 错误 token | 401 | ✅ 401 |
| 缺 Bearer 前缀 | 401 | ✅ 401 |
| 正确 token | 200（放行） | ✅ 200 |

命令：`go test ./internal/middleware -run 'TestRepairAgentToken' -count=1` → 5 用例 PASS。
源码复核：使用 `crypto/subtle.ConstantTimeCompare`，token 取 `WXX_REPAIR_AGENT_TOKEN`，未配置返回 404。

### 2.5 依赖注入路径（audit 建议④）

`pkg/app/app.go` 装配复核 → ✅：

```
feedbackSvc := service.NewFeedbackService(feedbackRepo, userRepo, feedbackScreenshotRepo)  // userRepo 恒注入
```

`userRepo` 由 `repository.NewUserRepo(db)` 始终构造并传入；`CanAccessScreenshot` 中即便 `userRepo` 为 nil 也走「路径 2：反馈引用归属」兜底，不会降级为 `allowed=true`（安全不退化）。→ **无需新增测试，静态装配检查通过。**

### 2.6 原有反馈主流程未退化（audit 建议⑤）

| 功能 | 验证方式 | 结果 |
|---|---|---|
| 提交反馈（成功/用户不存在/未认证/非法分类） | 既有 `feedback_handler_test.go` 4 用例 | ✅（`go test ./internal/handler` 含于 2.1 已 `ok`） |
| AI 诊断（本地兜底/LLM/JSON 解析/工单持久化/无仓库） | 既有 `feedback_air_repair_test.go` 5 用例 | ✅ |
| 我的反馈/评分/状态流转/日志时间线/AI 诊断 | Service 层逻辑本轮未改动（`Submit/Resolve/Rate/ListLogs/ListMine` 保持原实现，仅 `Get` 增加 `GetAuthorized` 包装） | ✅（代码 diff 复核，无行为变更） |

---

## 三、发现的回归 / 缺陷

### 🔴 P1 — 非法状态流转返回 500 而非 400（`badStateOrErr` 错误比较缺陷）

- **位置**：`internal/handler/feedback_repair_task_handler.go` 的 `badStateOrErr`。
- **根因**：
  ```go
  func (h *FeedbackRepairTaskHandler) badStateOrErr(c *gin.Context, err error) {
      switch err {   // ← 直接等于比较
      case service.ErrRepairTaskNotFound: ...
      case service.ErrRepairTaskBadState: ...
      ...
  ```
  但 service 层返回的是 `fmt.Errorf("%w: ...", ErrRepairTaskBadState, ...)` **包装后的 error**，`switch err` 无法命中，最终落入 `default` 返回 **500**。违反需求「非法流转应返回 400」（pm-checklist M2 验收口径），也违反 audit-report 第 3 节「每个入口都先 `GetByTaskNo` 后 `canTransition` 校验，非法流转返回 400」的结论。
- **复现证据**（真实命令输出）：
  ```
  $ go test ./internal/handler -run 'TestRepairTaskHandler_AcceptIllegalState_StatusCode' -count=1 -v
  feedback_repair_task_handler_test.go:78: Accept illegal state -> status=500 body={"code":500,"message":"修复任务当前状态不允许此操作: 仅待验收任务可验收，当前 approved","trace_id":""}
  --- FAIL: TestRepairTaskHandler_AcceptIllegalState_StatusCode (0.71s)
  ```
- **影响**：所有非法状态流转（accept/reject/deploy-confirm/deploy-done/cancel 的错态调用）都会被前端当作服务端错误（500），而非可理解的"当前状态不允许"业务错误（400），前端无法给出正确提示。`Cancelled/closed` 等终态的错误操作同样受影响。
- **修复建议**：`badStateOrErr` 改用 `errors.Is(err, service.ErrRepairTaskBadState)` / `errors.Is(err, service.ErrRepairTaskNotFound)` 进行链式判断（或 service 层返回时用 errors.Is 语义对齐）。

### 🟡 P2 — `RepairTaskVerifyRequest.Passed` 标记 `binding:"required"` 无法阻断 `false`

- **位置**：`internal/model/entity.go` `RepairTaskVerifyRequest.Passed bool \`json:"passed" binding:"required"\``。
- **说明**：`bool` 类型 + `binding:"required"` 在 gin 中只会校验「字段是否存在」，`false` 仍是合法值（不会被判定为缺失）。因此「验证失败上报」能正常入参，语义上这是**期望行为**（`passed=false → verify_failed`），功能正确。此条仅作为提示：若未来希望强制区分"未提供 passed"与"passed=false"，需改用 `*bool`。**不影响当前功能，非阻塞。**（此条为代码审查提示，非实测回归，已明确标注。）

---

## 四、未覆盖的环境依赖项（不虚构结果）

| 项 | 原因 | 影响 |
|---|---|---|
| MySQL 方言真库回归 | 本机无 MySQL 实例 | 迁移 109 与 repo 层仅以 SQLite（`modernc.org/sqlite` 内存库）验证；MySQL 方言转换（`AdaptForDriver`/`LONGTEXT`、`TIMESTAMPDIFF`）未在真实 MySQL 上跑。依据 refactor-notes 第 5 节 R5，上线前需在真实 MySQL 环境演练迁移 109。 |
| Redis | 本机无 Redis 实例 | 缓存降级链路未验证（与本次反馈闭环改动无直接耦合） |
| 真实 LLM/视觉（智谱 GLM-4V）调用 | 无 API Key | AI 诊断的本地兜底路径已测；真实 OCR/文本模型仅在既有 mock 测试中覆盖 |
| 全链路 http 集成（gin 路由 + JWT + token 同时挂载） | 本次以 `httptest` 最小路由分层验证 | `routes.go` 完整路由树（含静态文件挂载、SPA 回退）未做端到端启动回归（`app.New()` 需真实 DB/迁移，未启动） |
| 前端任务管理 UI | refactor-notes 明确"本轮未虚构"，仅语义纠正 + API 常量 | 前端无对应任务列表/验收界面，无法做 UI 级回归 |

---

## 五、结论

**结论：❌ 有条件不通过 —— 存在 1 项 P1 缺陷需在提交前修复。**

新增安全能力（越权修复、token 鉴权）与状态机核心逻辑**功能正确**，均通过真实测试佐证：

1. ✅ 反馈详情/日志/截图越权访问已正确修复（本人/管理员/越权第三方/未认证四象限均验证通过）。
2. ✅ 修复任务状态机全链路（含 verify_failed 回路、cancelled 终态、并发闸门）在 Service 层全部通过。
3. ✅ 内部端点 token 鉴权（未配置 404 / 错误 401 / 正确放行 / ConstantTimeCompare）全部通过。
4. ✅ `userRepo` 恒注入，越权判断不存在 nil 降级泄露。
5. ✅ 原有反馈主流程（提交/我的反馈/评分/状态流转/AI 诊断）回归无退化（既有 + 新增测试全绿）。

**但**非法状态流转的 HTTP 状态码映射存在 P1 缺陷（返回 500 而非 400），与需求「非法流转应返回 400」及 audit-report「非法流转返回 400」的结论相悖，需将 `badStateOrErr` 的比较方式从 `switch err` 改为 `errors.Is` 链式判断后方可判定通过。

其余为可接受的 MVP 边界（见第四节），不构成阻塞。

---

## 附：本轮新增测试文件（可保留或清理）

- `server/internal/middleware/repair_agent_token_test.go`
- `server/internal/service/feedback_repair_task_service_test.go`
- `server/internal/handler/feedback_access_test.go`
- `server/internal/handler/feedback_repair_task_handler_test.go`

均未改动任何业务源码。
