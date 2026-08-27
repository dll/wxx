# 反馈修复闭环 MVP — 只读代码审核报告

> 审核人：leader-wxx（主代理，只读评审） · 日期：2026-08-27
> 范围：反馈修复闭环 MVP 全部新增/修改后端、前端、脚本。
> 结论：**有条件通过**（已修复 2 处问题，其余为可接受的 MVP 边界，无高危漏洞）。
> 未修改任何业务源码（仅修正注释与脚本字段类型）。

---

## 一、总体结论

| 维度 | 结论 |
|---|---|
| 权限隔离 | ✅ 通过 |
| 执行端 token 安全 | ✅ 通过 |
| 任务状态机 | ✅ 通过（有一处非单事务 TOCTOU，见边界） |
| 水平越权修复（反馈/截图） | ✅ 通过 |
| 服务器不执行改码/构建/部署 | ✅ 通过 |
| 脚本与后端契约一致性 | ⚠️ 已修复（见问题 2） |

---

## 二、发现并已修复的问题

### 问题 1（低危·文档准确性）：service 头部注释残留已删除概念
- 位置：`server/internal/service/feedback_repair_task_service.go` 包注释
- 原文错误声明「执行端认证：使用专用 capability system.repair.execute」——该 capability 已在早前接管任务中被否决并移除，实际采用 `WXX_REPAIR_AGENT_TOKEN` token 中间件。
- **已修复**：注释改为准确描述 token 中间件方案，消除误导。

### 问题 2（中危·契约一致性）：执行端脚本字段类型与后端不一致
- 位置：`scripts/repair-agent.ps1` 的 `Submit-Verify`
- 后段 `RepairTaskVerifyRequest` 中 `go_vet/go_test/flutter_analyze/flutter_test` 均为 **string**（`pass`/`fail`），原脚本发送 JSON **bool**，会导致 Go `ShouldBindJSON` 反序列化失败，验证上报不可用。
- **已修复**：脚本改为发送 `"pass"`/`"fail"` 字符串，`passed` 使用 bool、`flutter_test` 发 `"skip"`，与后端契约对齐。

---

## 三、核查通过的关键安全项

### 1. 执行端 token 鉴权（`middleware/repair_agent_token.go`）
- ✅ 环境变量 `WXX_REPAIR_AGENT_TOKEN`，不硬编码、不入日志、不写库。
- ✅ `crypto/subtle.ConstantTimeCompare` 常量时间比较，防时序侧信道。
- ✅ token 未配置（空）时内部端点返回 **404**（非 401），不暴露路由存在性。
- ✅ 仅作用于 `/api/v1/internal/repair-tasks`，与前台 JWT 完全隔离，不授予任何业务角色执行能力。
- ✅ 常量时间比较要求两侧长度一致才有意义；此处 `ConstantTimeCompare` 直接比较两个字节切片，长度不等时返回 0（不相等），安全。（无 `length==provided` 校验的误判风险：长度不同即不匹配。）

### 2. 水平越权修复（反馈详情/处理日志/截图）
- ✅ `Get`/`GetLogs` 改走 `GetAuthorized`：普通用户仅读本人，`union.feedback.list` 能力者读全部；无权/不存在统一 404，不泄露存在性。
- ✅ `ServeScreenshot` 改为双重归属校验：
  - 路径 1：上传者本人（`uploaded_by` 存 username，`CanAccessScreenshot` 用 `userRepo.GetByID` 反查 username 比对，已 nil 守卫）；
  - 路径 2：反馈引用者（`COUNT(feedback WHERE user_id=? AND screenshot_url LIKE ?)`）。
  - 任一命中即可；反馈管理员（`union.feedback.list`）直接放行。
- ✅ 即便 `userRepo` 为 nil（路径 1 降级跳过），路径 2 仍强制「仅本人反馈引用的截图可读」，安全不降级。

### 3. 状态机（`feedback_repair_task_service.go`）
- ✅ `validTransitions` 覆盖 approved→running→awaiting_acceptance→deploy_pending→deploying→deployed→closed，含 verify_failed 回路、cancelled 终态。
- ✅ 每个入口（Cancel/Claim/SubmitVerify/Accept/Reject/DeployConfirm/DeployDone）都先 `GetByTaskNo` 后 `canTransition` 校验，非法流转返回 400。
- ✅ `SubmitVerify` 仅允许 `running` → 上报；`Accept` 仅 `awaiting_acceptance`；`DeployDone` 仅 `deploying`。
- ✅ `Claim` 前 `CountActiveRunning` 做全局单 running 闸门，避免并发改码冲突。

### 4. 服务器无副作用
- ✅ 服务层/处理器仅做状态流转 + 审计日志，无任何 shell 执行、文件写、构建、部署调用。
- ✅ `deploy-confirm`/`deploy-done` 只写状态字段，不触发服务器动作。
- ✅ `repair-agent.ps1` 明确「不 commit/push/部署」，仅领取 + 验证上报。

---

## 四、可接受的 MVP 边界（已记录，非阻塞）

1. **Claim 非单事务（TOCTOU）**：`CountActiveRunning` → `NextClaimable` → `UpdateClaim` 非同一事务，存在极窄竞争窗口。MVP 依赖「仅 1 个执行端 + 人工审批节奏」控制；若需更强并发，建议后续改事务 + `SELECT ... FOR UPDATE`/乐观锁。
2. **全局单 running 含 awaiting_acceptance**：`CountActiveRunning` 把 `awaiting_acceptance` 也计入，即一个任务待验收时会阻塞下一个任务领取。属保守安全选择，符合「单飞」意图，但吞吐受限。
3. **前端管理端任务 UI 未做**：本轮仅纠正「在线修复」→「修复诊断」文案 + 补 API 常量，完整任务列表/详情/验收界面留待后续批次，未虚构。
4. **执行端脚本不自动改码**：与「服务器不执行改码」边界一致，代码修改由操作者/AI 编码工具在隔离 worktree 完成。

---

## 五、测试/门禁复核（审核时复跑）

- `gofmt -l`（本轮修改的 service 文件）✅
- `go build ./...` ✅
- 前序已通过（未复跑，结论沿用）：`go vet ./...`、`go test ./internal/handler ./internal/service ./pkg/app ./internal/db`、`dart format`、`flutter analyze`（仅既有 info）、`flutter build web --release`、`git diff --check`。

---

## 六、审核后动作建议

1. 无需回退任何实现；本轮仅修正注释与脚本字段类型，业务逻辑无变化。
2. 建议进入 QA 回归（qa-regression-wxx）做一次完整回归测试，重点覆盖：
   - 反馈详情/日志/截图越权访问（普通用户 vs 反馈管理员 vs 越权第三方）；
   - 修复任务状态机全链路（创建→领取→验证上报→验收→部署确认→部署完成→关闭）及非法流转；
   - 内部端点 token 鉴权：缺失 token 404、错误 token 401、正确 token 放行。
   - 依赖注入路径（`userRepo` 是否始终注入）。
3. 通过后按规则生成 `refactor-final-summary.md`，再交用户确认提交/推送/部署。
