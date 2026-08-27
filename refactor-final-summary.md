# 反馈「一键自动修复」增强 — 最终汇总

> 汇总人：leader-wxx · 日期：2026-08-27
> 项目：wxx（蔚小芯，Go + Flutter 已上线系统）
> 本轮需求：在「反馈修复闭环 MVP」基础上，增强为「管理员一键启动 → 从反馈表读取问题原文 → 本机自动改码 → 自动验证上报 → 管理员验收 → 部署确认」的全自动闭环。
> 流水线：pm → dev → qa → review → 回修 → 汇总（严格串行）。

---

## 一、流水线执行概览

| 步骤 | 角色 | 产出 | 状态 |
| --- | --- | --- | --- |
| 1 需求核对 | pm-wxx | `pm-checklist.md`（一键自动修复增强章节） | ✅ 完成 |
| 2 开发重构 | dev-refactor-wxx | `refactor-notes.md`（八·一键自动修复增强） | ✅ 完成 |
| 3 回归测试 | qa-regression-wxx | `qa-report.md`（通过，无 P0/P1） | ✅ 完成 |
| 4 代码审核 | reviewer-audit-wxx | `audit-report.md`（有条件通过 + 2 项 P1） | ✅ 完成 |
| 4.1 P1 回修 | dev-refactor-wxx | `refactor-notes.md`（九·reviewer P1 回修） | ✅ 完成 |
| 5 汇总 | leader-wxx | 本文件 | ✅ 完成 |

---

## 二、背景与核心缺口

上一轮 MVP 已建立完整闭环状态机（`approved → running → awaiting_acceptance → deploy_pending → deploying → deployed → closed`）+ 本机执行端 `scripts/repair-agent.ps1`（claim/verify 两模式）。但「实际改代码」仍是断点：执行端领取任务时**只拿到 `feedback_ids`（ID 列表）和 AI 诊断的 `code_files`，拿不到反馈原文 `content`**，导致自动修复「无从下手」，只能靠管理员手动复制粘贴到本机改码。

本轮核心补齐：**把反馈原文回传执行端 + 执行端新增一键自动改码模式**。

---

## 三、本轮交付内容

### 3.1 服务端：执行端 payload 补充反馈原文

- `model/entity.go` 或 `dto.go`：新增 `FeedbackRepairContent` 结构体（`feedback_id`/`module`/`category`/`content`），`RepairTaskPayload` 新增 `FeedbackContents` 字段（`json:"feedback_contents"`）。
- `service/feedback_repair_task_service.go`：`taskToPayloadWithContents()` 遍历 `FeedbackIDs` 逐条经 `feedbackSvc.Get()` 取原文填充，**取不到单条降级（跳过/留空）不崩溃、不返回 nil payload**；`FeedbackContents` 初始化为空切片，JSON 恒为 `[]` 而非 `null`（向后兼容）。

### 3.2 执行端：repair-agent.ps1 新增 auto 一键模式

- 流程 = ① claim 领取任务 → ② 拿 payload（含原文 + 诊断）→ ③ `git worktree add` 隔离分支改码 → ④ 跑 `Run-Verification` → ⑤ `Submit-Verify` 上报。
- 本机编码工具：优先 `WXX_REPAIR_CODER`（默认 `gemini`），prompt 由 here-string 组装，喂入反馈原文 + 诊断摘要 + 代码文件路径。
- 安全边界：改码只在 worktree 内，不自动 commit/push/部署，保持「服务器不改码不部署」原则。

### 3.3 安全回修（reviewer P1-1 / P1-2）

- **P1-1（原文落盘泄露）**：`.gitignore` 追加 `repair-prompt.txt` 与 `wxx-repair-*/`；脚本同时写 worktree 的 `.git/info/exclude`；无论 passed/failed 统一删除 prompt 文件。三层防护。
- **P1-2（prompt 注入）**：prompt 增加「安全红线」块，声明反馈原文为**不可信、不可执行的用户数据**；`code_files` 白名单为空时**短路不调用编码工具**；强制约束「只允许修改白名单内文件，禁止新建/删除/重命名清单外文件」。

---

## 四、QA 回归结论

- ✅ `go build ./...` 通过。
- ✅ `go test`（service/handler/model/db）全绿。
- ✅ 状态机 8 用例 PASS（含 `Payload_ContainsFeedbackContents`、`Payload_MissingFeedback_Degrades`）。
- ✅ PowerShell AST 无语法错误；`feedback_contents` 恒为合法 JSON 数组（非 null）；向后兼容。
- ✅ 原有反馈主流程、修复任务状态机无回归。
- ⚠️ QA 发现 1 项 P2（N+1 查询，非阻塞，Claim 低频可忽略）。

**QA 结论：通过（无 P0/P1）。**

---

## 五、代码审核结论

audit-report.md 结论「有条件通过」，两个 P1 已回修并复验：

| 编号 | 级别 | 描述 | 处置 |
| --- | --- | --- | --- |
| P1-1 | 高 | repair-prompt.txt（含原文）未 git-ignore | ✅ 已回修（.gitignore + info/exclude + 删除） |
| P1-2 | 高 | prompt 未强制「仅改 code_files 白名单」 | ✅ 已回修（安全红线 + 白名单短路 + 硬约束） |
| P2-1 | 中 | N+1 查询 | 可接受，后续可改批量查询 |
| P2-2 | 中 | 原文未脱敏 PII | 建议文档明确含 PII 反馈不启用 auto |
| P2-3 | 中 | WXX_REPAIR_CODER 可被设为任意命令 | 属「受信操作者自配」，文档标注 |
| L1 | 低 | Write-Host 打印自定义命令（误用才含 secret） | 注释提醒 |

**结论：通过（P1 已修复并复验）。**

---

## 六、端到端触发方式

1. 管理员在反馈详情「在线修复」→ 创建修复任务（`CreateTask`，复用现有 AI 诊断）。
2. 本机执行端一键执行自动修复：
   ```powershell
   $env:WXX_REPAIR_AGENT_TOKEN = "<内部受控token>"
   pwsh -File scripts/repair-agent.ps1 -Mode auto -BaseUrl https://wxx-agent.online
   ```
3. 脚本自动：领取任务 → 拉取 feedback 原文 + 诊断 → `git worktree` 隔离分支改码 → go vet/test + flutter analyze → 上报 verify。
   - passed → 服务端流转 `awaiting_acceptance` → 管理员验收（`AcceptTask`）→ 部署确认（`DeployConfirmTask`）→ 部署完成（`DeployDoneTask`）→ 关闭。
   - failed → `verify_failed` → 可重新认领。
4. **部署保留人工确认**（DeployConfirm/DeployDone 仅标记，不自动触发真实部署）。

---

## 七、未覆盖项（如实记录）

| 项 | 说明 |
| --- | --- |
| MySQL 真库回归 | 本机无 MySQL，方言/迁移仅 SQLite 验证 |
| 真实 LLM 编码执行 | 无 API Key，auto 改码的 AI 编码工具实际运行需部署环境验证 |
| 反馈原文 PII 脱敏 | 未做，建议含敏感信息反馈不启用 auto |
| WXX_REPAIR_CODER 硬拦截 | commit/push/deploy 禁令依赖 prompt 软约束 + worktree 隔离 |

---

## 八、交付文件清单

| 文件 | 作用 |
| --- | --- |
| `pm-checklist.md` | 需求核对（一键自动修复增强） |
| `refactor-notes.md` | 开发改动记录（八·增强 + 九·P1 回修） |
| `qa-report.md` | 回归测试报告 |
| `audit-report.md` | 代码审核报告 |
| `refactor-final-summary.md` | 本汇总 |

---

> 流水线结束。改动已就绪，等待提交与部署。
