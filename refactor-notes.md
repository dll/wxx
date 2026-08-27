# 反馈修复闭环 MVP — 重构记录

> 日期：2026-08-26 · 目标：把现有「在线修复」（实为 AI 诊断 + 复制报告，不改代码）升级为
> **管理员审核后单条/批量创建修复任务 → 本机受控修复 → 自动验证 → 管理员验收 → 人工确认部署** 的安全闭环。
>
> 前提约束：服务器绝不执行源码修改、构建或部署；一切改码只发生在受控本机执行端（开发机）。
> 未提交、未推送、未部署。

---

## 一、安全修复（越权读取，P1）

审计发现反馈详情、处理记录、截图接口缺少归属校验，任意登录用户可读他人反馈全文与截图。本轮修复：

- `server/internal/handler/feedback_handler.go`
  - `Get`（GET /feedback/:id）：改为 `GetAuthorized`，普通用户仅可读本人反馈，持 `union.feedback.list` 能力的反馈管理员可读全部。
  - `GetLogs`（GET /feedback/:id/logs）：同一归属校验。
  - `ServeScreenshot`（GET /uploads/feedback/:filename）：仅允许截图上传者本人 / 引用该截图的反馈提交者、以及反馈管理员访问；未授权返回 403。
- `server/internal/service/feedback_service.go`
  - 新增 `GetAuthorized(feedbackID, userID, canManageAll)`。
  - 新增 `CanAccessScreenshot(filename, userID)`（上传者归属 + 反馈引用归属）。
- `server/internal/repository/feedback_repo.go`、`feedback_screenshot_repo.go`
  - 补充归属查询：`CountScreenshotRefsByUser`、`OwnerByFilename`。

保留既有 `Get`/`ListLogs` 仅为内部兼容，对外 HTTP 入口一律走带归属校验的新方法。

---

## 二、修复任务实体与状态机（MVP）

### 迁移 `109_feedback_repair_tasks.sql`

新建 `feedback_repair_tasks` 表（SQLite/MySQL 双方言，长文本列已纳入 `db/dialect.go` 转换为 LONGTEXT）：

- `task_no`（rt-xxxxxxxx）、`creator`（创建即审核）、`feedback_ids`（JSON，支持单/批量）
- `title`、`diagnosis`（合并后的 AI 诊断 JSON）
- `status`、`worker_host`、`base_commit`、`branch`
- `verify_result`（JSON：go_test/go_vet/flutter_analyze/flutter_test）、`diff_stat`、`log_text`
- `accept_note/accepted_by/reject_reason/rejected_by`
- `deploy_confirmed_by/deploy_ref`
- 审计时间戳 `created_at/updated_at`

### 状态机（服务层硬编码校验）

```
approved → running → awaiting_acceptance → deploy_pending → deploying → deployed → closed
                                              ↓ reject                       ↑
                                          verify_failed ← 驳回              (deploy-done)
approved/verify_failed → cancelled（终态）
```

关键约束：
- MVP 全局同时仅允许 1 个 `running` 任务（`Claim` 前 `CountActiveRunning` 闸门），避免并发改码冲突。
- 服务器只写状态与审计，不执行任何改码/构建/部署动作。

---

## 三、后端实现

### 新增文件

- `internal/repository/feedback_repair_task_repo.go`：Create/GetByTaskNo/List/NextClaimable/CountActiveRunning/AppendLog/UpdateStatus/UpdateClaim/UpdateVerifyReport/UpdateAccept/UpdateReject/UpdateDeployConfirm/UpdateDeployDone；`FeedbackIDsToJSON`。
- `internal/service/feedback_repair_task_service.go`：`FeedbackRepairTaskService` + 状态机 `validTransitions`；Create（单/批量，逐条复用现有 `AIRepair` 诊断合并 code_files）、List/Get/Cancel/Claim/SubmitVerify/Accept/Reject/DeployConfirm/DeployDone（可选联动批量 `Resolve` 反馈并触发既有站内通知）。
- `internal/handler/feedback_repair_task_handler.go`：管理端 + 内部执行端两组入口。
- `internal/middleware/repair_agent_token.go`：执行端专用 token 鉴权中间件。
- `server/migrations/109_feedback_repair_tasks.sql`。

### 执行端鉴权（关键安全设计）

内部路由 `/api/v1/internal/repair-tasks/*` 不依赖 JWT 或业务角色：

- 使用独立环境变量 `WXX_REPAIR_AGENT_TOKEN`，**绝不硬编码、不授予任何业务角色（含 sys_admin）**。
- 采用 `crypto/subtle.ConstantTimeCompare` 常量时间比较，防时序侧信道。
- **token 未配置（空）时，内部端点一律返回 404**，不暴露路由存在性。
- 与交互式前台用户 JWT 完全隔离。

### 路由（`pkg/app/routes.go`）

管理端（JWT + UnionFeedbackWrite/List 能力门控）：

```
POST /admin/feedback/repair-tasks              CreateTask
GET  /admin/feedback/repair-tasks              ListTasks
GET  /admin/feedback/repair-tasks/:no          GetTask
POST /admin/feedback/repair-tasks/:no/cancel   CancelTask
POST /admin/feedback/repair-tasks/:no/accept   AcceptTask  (验收 → deploy_pending)
POST /admin/feedback/repair-tasks/:no/reject   RejectTask  (驳回 → verify_failed)
POST .../:no/deploy-confirm   DeployConfirmTask  (仅标记，不触发服务器动作)
POST .../:no/deploy-done      DeployDoneTask     (部署完成记录 + 可选联动解决反馈)
```

内部执行端（token 鉴权）：

```
POST /internal/repair-tasks/next        NextTask    (原子领取)
POST /internal/repair-tasks/:no/verify  VerifyTask  (验证结果上报)
```

### 装配

`pkg/app/app.go`、`pkg/app/deps.go` 增加 `feedbackRepairTaskSvc` / `feedbackRepairTaskH` 并注入 `deps`。

---

## 四、本机执行端脚本 `scripts/repair-agent.ps1`

- `claim`（默认）：调用 `POST /internal/repair-tasks/next` 领取任务，打印诊断报告（摘要/相关文件/反馈 ID），给出隔离 worktree 与 verify 上报指引。**不 commit、不 push、不部署**。
- `verify`：对指定工作区执行 `go vet ./...`、`go test ./...`、`flutter analyze`，收集 diff stat，调用 `POST /internal/repair-tasks/:no/verify` 上报。
- token 从 `WXX_REPAIR_AGENT_TOKEN` 读取，缺失则直接退出（与中间件一致）。

> 说明：脚本不自动改码、不自动创建分支提交；实际代码修改由操作者（或操作者驱动的 AI 编码工具）在隔离 worktree 内完成，本脚本只负责受控领取与验证上报。

---

## 五、前端（最小可用，纠正语义）

- `lib/pages/admin/feedback_page.dart`：详情页按钮「在线修复」→「修复诊断」，注释说明仅做 AI 诊断与定位，实际修复走受控任务流程。
- `lib/widgets/feedback_repair.dart`：面板标题「在线修复助手」→「修复诊断助手」。
- `lib/config/api_config.dart`：补充 `adminRepairTasks` 与 accept/reject/deploy-confirm/deploy-done/cancel 端点常量（供后续任务 UI 接入）。

> 本批先纠正「在线修复」这一误导性文案，避免向用户暗示已自动修复。完整的管理端任务列表/详情/验收 UI 属后续批次，本轮未虚构。

---

## 六、验证（全部通过）

- `gofmt -l`（目标 Go 文件）：通过（已 `gofmt -w` 修复一处）。
- `go vet ./...`：通过。
- `go test ./internal/handler ./internal/service ./pkg/app ./internal/db -count=1`：通过。
- `dart format`（3 个前端文件）：通过。
- `flutter analyze`（定向 3 文件）：仅项目既有 info（`surfaceVariant` 已弃用、`unnecessary_to_list_in_spreads`），无 error/warning。
- `flutter build web --release`：通过（见下方构建记录）。
- `git diff --check`：通过。

---

## 六·补 QA 回归修复（2026-08-27）

QA 回归（qa-regression-wxx）发现 1 项 P1 缺陷，本轮修复：

- **P1 非法状态流转返回 500（应返回 400）**
  - 根因：`feedback_repair_task_handler.go` 的 `badStateOrErr` 用 `switch err` 直接相等比较，而 service 层返回的是 `fmt.Errorf("%w: ...")` 包装后的错误，导致 `ErrRepairTaskBadState` 匹配不上、落入 default 返回 500。
  - 修复：`badStateOrErr` 改用 `errors.Is` 链式判断；同步将 `NextTask`、`VerifyTask`、`notFoundOrErr` 中的 `err == service.ErrXxx` 直接比较统一改为 `errors.Is`（防御未来包装，消除同类隐患）。
  - 验证：`Accept illegal state -> status=500` 已变为 `status=400`，body 正确返回状态机提示；`go build ./...`、`go vet`、5 个核心包 `go test` 全部通过。

---

## 七、已知边界（未伪造，后续批次）

1. 前端尚未提供完整的「修复任务列表/详情/验收/部署」管理界面，当前仅接入诊断语义纠正与 API 常量。
2. `Claim` 的并发闸门（`CountActiveRunning` → `NextClaimable` → `UpdateClaim`）非单事务，存在极窄 TOCTOU 窗口；MVP 依赖「全局仅 1 个执行端 + 人工审批节奏」控制，若需更严并发可后续改为事务 + 行锁。
3. 执行端脚本不出自动改码（与「服务器不执行改码」边界一致），代码修改仍由操作者/AI 编码工具在隔离 worktree 完成。
4. 部署动作完全交由管理员通过既有 GitHub Actions / `make deploy-release` 通道手动执行，本闭环只记录状态，不自动部署。
5. 反馈所属范围（学院/全校）未在本任务内细化；反馈管理员能力沿用既有 `union.feedback.*`。
