# 重构/修复记录（dev-refactor-wxx）

## 问题概述

校园导航管理员拖动报到节点后坐标无法持久化，刷新页面后坐标回退。根因（leader+pm 已锁定）：

1. **数据层**：迁移 `079_clear_preloaded_data.sql` 删光了 `campus_checkin_steps` 表的 12 条种子数据（会峰 6 + 琅琊 6），生产表为空。
2. **前端**：`campus_map_page.dart` 的 `_loadStepsFromServer()` 在 admin 接口返回空/失败时，静默回退到硬编码假节点，`_remoteIds` 保持为空。
3. **结果**：管理员拖动假节点 → `_saveCoordinate` 里 `_remoteIds[index]` 取 `null` → `return false` → 退出时报“未加载到后端节点”，坐标从未写库。

## 改动文件列表

| 文件 | 改动类型 |
| ---- | -------- |
| `server/migrations/110_restore_campus_steps.sql` | 新增 |
| `frontend/lib/pages/campus/campus_map_page.dart` | 修改 `_loadStepsFromServer()` + 新增 `_showNoRemoteStepsHint()` |

---

## 修复 1：新增迁移 `server/migrations/110_restore_campus_steps.sql`

恢复会峰 6 + 琅琊 6 共 12 个报到节点，`status='published'`。

### 坐标来源（关键决策）

- **业务字段**（`title/location/duration/task/materials/contact/note/icon_name`）从 `048_campus_map_steps.sql` 种子数据**精确复制**。
- **`lat/lng`** 采用 `050_fix_campus_step_coords.sql` 纠正后的权威 WGS-84 值（与前端 `campus_map_page.dart` 硬编码常量一致）。

> ⚠️ 权威坐标核对结论：任务清单中列出的「琅琊 5/6」坐标
> `32.3138,118.3101 / 32.3147,118.3102` 是 **048 的错误坐标**（偏移到校区以北约 2km，
> 正是 050 迁移要纠正的对象）。指令明确要求以 `050_fix_campus_step_coords.sql` 与
> 前端硬编码常量为权威来源。二者一致确认：
>
> - 琅琊 5「校园卡与网络」= `32.2926, 118.3000`
> - 琅琊 6「入学体检与学籍核验」= `32.2917, 118.2992`
>
> 本迁移最终采用上述 050/前端一致的权威值，未使用任务清单中的 048 错误值。

### 写入要点

- SQLite 写法 `INSERT OR IGNORE INTO`（方言转换会自动转 MySQL `INSERT IGNORE`）。
- `id` 不显式指定，依赖 `AUTOINCREMENT` 自增分配。
- 迁移机制用 `_migrations` 表保证每个文件只执行一次，`INSERT OR IGNORE` 足够；`campus_checkin_steps` 表本身无唯一约束列（即便 MySQL 下不去重也无妨，因为只执行一次）。

### 最终坐标表

| campus | order | title | lat | lng | icon |
| ------ | ----- | ----- | --- | --- | ---- |
| huifeng | 1 | 校门入校核验 | 32.2705 | 118.3055 | login |
| huifeng | 2 | 学院报到 | 32.2745 | 118.3070 | account_balance |
| huifeng | 3 | 缴费与绿色通道 | 32.2735 | 118.3060 | payments |
| huifeng | 4 | 宿舍入住 | 32.2770 | 118.3040 | bed |
| huifeng | 5 | 校园卡与网络 | 32.2740 | 118.3090 | credit_card |
| huifeng | 6 | 入学体检与学籍核验 | 32.2720 | 118.3030 | health_and_safety |
| langya | 1 | 校门入校核验 | 32.2921 | 118.2988 | login |
| langya | 2 | 学院报到 | 32.2932 | 118.3002 | account_balance |
| langya | 3 | 缴费与绿色通道 | 32.2928 | 118.2995 | payments |
| langya | 4 | 宿舍入住 | 32.2940 | 118.2976 | bed |
| langya | 5 | 校园卡与网络 | 32.2926 | 118.3000 | credit_card |
| langya | 6 | 入学体检与学籍核验 | 32.2917 | 118.2992 | health_and_safety |

---

## 修复 2：前端 `campus_map_page.dart` fallback 陷阱

在 `_loadStepsFromServer()` 中区分 `_canEditNodes`：

- **管理员**（`_canEditNodes == true`）：当 admin 接口返回空列表或失败时，**不再**把硬编码 `_campus.steps` 当作可拖动的编辑源使用；保留 `_remoteIds` 为空，并通过新增的 `_showNoRemoteStepsHint()` 弹出提示 **“后端无报到节点数据，请先在流程管理创建节点”**。此时拖拽保存逻辑 `_remoteIds[index]` 为 null，会正确返回 false，避免“假成功/坐标写不进库”。
- **普通用户**（非管理员）：保持原有离线兜底逻辑不变，静默回退本地硬编码常量，查看报到流程不受影响。

### 关键 diff 说明

1. `_loadStepsFromServer()` 方法头部新增详细 doc 注释，说明管理员/普通用户两条分支的行为差异。
2. 在“空列表/业务失败”分支与 `catch {}` 分支中，各加入：
   ```dart
   if (_canEditNodes) {
     _showNoRemoteStepsHint();
   }
   ```
   普通用户分支不触发提示、不改变兜底行为。
3. 新增方法：
   ```dart
   void _showNoRemoteStepsHint() {
     if (!mounted) return;
     ScaffoldMessenger.of(context).showSnackBar(
       const SnackBar(
         content: Text('后端无报到节点数据，请先在流程管理创建节点'),
         duration: Duration(seconds: 4),
       ),
     );
   }
   ```

### 业务逻辑保持

- 普通学生查看报到流程：不变（仍走公开接口 + 本地兜底）。
- 管理员成功加载到后端节点时：行为不变（`_campusStepsMap` 覆盖、`_remoteIds` 填充、地图刷新）。
- 拖拽保存、退出编辑时的重试/校验逻辑：不变。

---

## 验证建议（供 qa-regression-wxx 参考）

1. 执行迁移后，`campus_checkin_steps` 应恢复 12 条 `published` 记录，坐标与上表一致。
2. 管理员打开校园地图 → 应能看到 12 个真实后端节点并可拖动；拖动后退出编辑，坐标应落库；刷新后坐标不回退。
3. 若后端无数据（或模拟空列表/失败），管理员应看到“后端无报到节点数据…”提示，且拖拽保存报错而非假成功。
4. 普通学生视角查看报到流程不受影响（离线/后端不可用仍能查看本地常量兜底）。

---

## 七·reviewer 回修（H1 / H2 高危缺陷）

### 背景
reviewer 评审发现 110_restore_campus_steps.sql 采用 \CREATE UNIQUE INDEX (campus_id, step_order)\ + \INSERT OR IGNORE\ 实现幂等，存在两处高危缺陷：

- **H1**：唯一索引在建索引时若表内已有同 \step_order\ 数据（079 清空后管理员重建部分节点），会触发 \Duplicate entry\，而 \xecSQL\ 的容错只识别 \duplicate key name / already exists\，不识别 \duplicate entry\，导致迁移中断、服务无法启动。
- **H2**：唯一索引不含 \status\，会阻碍审核流（draft→pending_review→published）。审核流中同 \(campus_id, step_order)\ 的多 status 行合法共存（管理员新建同 step_order 节点、或为已发布节点再建 draft 多版本），唯一索引会把这类合法操作变成无意义 500。

### 结论
核对 \campus_checkin_steps\ 真实业务约束（048 起仅 id 主键、无任何业务唯一约束），并对照 \campus_repository.go\ 的 Create/Update/Submit/Publish 语义与 \campus_handler.go\ 的 CreateStep/UpdateStep：

- \step_order\ 仅用于排序（\ORDER BY step_order\），**不是唯一业务键**；
- \status\ 参与审核流，draft/pending_review/published 同表共存，\Create\ 无条件插入 \status='draft'\，无重复检测，业务层根本不依赖该唯一索引。

因此原「建 (campus_id, step_order) 唯一索引」方向错误，**应彻底移除唯一索引**，改用 \INSERT ... SELECT ... WHERE NOT EXISTS\ 守卫实现幂等去重。

### 修改内容

1. **重写 server/migrations/110_restore_campus_steps.sql**
   - 删除 \CREATE UNIQUE INDEX\（H1/H2 根因）。
   - 12 条种子改为 \INSERT INTO ... SELECT ... WHERE NOT EXISTS (SELECT 1 FROM campus_checkin_steps WHERE campus_id=? AND step_order=? AND status='published')\。
   - 该写法：幂等（重复执行命中 NOT EXISTS 判定不再插入）、不丢数据（不 DELETE 已有/重建节点）、不阻碍审核流（draft/pending_review 的同 step_order 节点不影响判定，且不阻止同 step_order 多版本）、双方言通用（SQLite / MySQL 均支持）。

2. **同步调整测试 server/internal/db/migration110_campus_test.go**
   - 更新断言：由「必须含唯一索引」改为「不得含任何索引」「必须含 12 条 WHERE NOT EXISTS 守卫」。
   - 新增 \stripSQLComments\ 辅助，剥离注释后再校验，避免注释中引用旧方案字样被误判。

3. **新增 server/internal/db/migration110_functional_test.go**
   - 功能回归：内存 SQLite 建 048 表 → 模拟 079 清空 → 播种同 step_order 多状态节点（重建 published + draft + pending_review）→ 执行 110 → 断言 published 恢复 12 条、既有 draft/pending_review 未被删除、同 step_order 已有 published 不重复插入 → 重复执行 110 幂等。

### 测试结果
\\\
go test ./internal/db/ ./internal/repository/ ./internal/handler/
  ok  github.com/dll/wxx/server/internal/db          (4.3s)
  ok  github.com/dll/wxx/server/internal/repository  (cached)
  ok  github.com/dll/wxx/server/internal/handler     (cached)
\\\
迁移 110 相关测试（含 H1/H2 功能回归）全部 PASS。

### 边界说明
- 未改动 repository/handler 业务代码：Create 仍无条件插 draft，符合「同 step_order 多 status 共存」的原有语义；H1/H2 均通过「不加索引」根治，无需在 CreateStep 增加重复检测。
- 前端 fallback 修复的既有正确部分完全保留，本次仅回修后端迁移与相关测试。

---

## 八、一键自动修复增强

### 背景与缺口
执行端（本机 repair-agent）领取修复任务时，`Claim` 返回的 `RepairTaskPayload` 仅含 `FeedbackIDs`（id 列表）与 `Diagnosis`（AI 诊断摘要），**拿不到反馈原文**，导致无法自动改码。「一键自动修复」闭环因此断裂。

### 改动内容

**A. 服务端：执行端 payload 补充反馈原文**

1. `server/internal/model/entity.go` 新增结构体 `FeedbackRepairContent`（字段 `FeedbackID/Module/Category/Content`），并在 `RepairTaskPayload` 新增字段 `FeedbackContents []FeedbackRepairContent`（json `feedback_contents`）。
2. `server/internal/service/feedback_repair_task_service.go`：
   - 新增 `taskToPayloadWithContents()`，遍历 `FeedbackIDs` 逐条通过 `feedbackSvc.Get(fid)` 取原文填充 `FeedbackContents`；单条取不到（`err != nil` 或 `fb == nil`）时降级：保留 `feedback_id`、`content/module/category` 留空，**不阻断整体 payload**。
   - `Claim()` 内 `taskToPayload(t)` 改为 `s.taskToPayloadWithContents(t)`（唯一出口），保证执行端拿到原文。

**B. 执行端脚本：repair-agent.ps1 新增 auto 一键模式**

- 新增 `-Mode auto`：claim 领取（复用现有 NextTask 调用）→ 组装含「反馈原文 + AI 诊断摘要 + 相关代码文件路径」的 prompt → `git worktree add` 创建隔离分支 `repair/<task_no>`，在本机调用 AI 编码工具（默认 `gemini`，可用 `WXX_REPAIR_CODER` 覆盖，回退 `openclaw`）改码 → `Run-Verification` + `Submit-Verify` 复用现有函数上报。
- 全程不 commit/push/部署；token 仍用 `WXX_REPAIR_AGENT_TOKEN`（不硬编码、不入日志）；prompt 写入 worktree 内 `repair-prompt.txt`（不入库、不上报）。

**C. 测试与记录**

- `server/internal/service/feedback_repair_task_service_test.go` 新增两个测试：
  - `TestRepairTask_Payload_ContainsFeedbackContents`：payload 含反馈原文（feedback_id/content/module/category）。
  - `TestRepairTask_Payload_MissingFeedback_Degrades`：某条反馈取不到时降级（保留 feedback_id、content 留空），Claim 不崩溃。

### 测试结果
```
go build ./...                                                    # 全绿
go test ./internal/service/ ./internal/handler/                   # 全绿
  ok  github.com/dll/wxx/server/internal/service  151.140s
  ok  github.com/dll/wxx/server/internal/handler  82.841s
```

### 边界说明
- 未改动 `validTransitions` 状态机、管理端接口语义、verify/accept/deploy 流程。
- 服务器仍不改码/构建/部署，自动修复只发生在本机 worktree 隔离分支。
- token 保持不硬编码、不入日志。

### 端到端触发方式
管理员设置 `WXX_REPAIR_AGENT_TOKEN`（可选 `WXX_REPAIR_CODER`）后，一键执行：
```powershell
pwsh -File scripts/repair-agent.ps1 -Mode auto -BaseUrl https://wxx-agent.online
```
自动完成：领取任务 → worktree 隔离分支 AI 改码 → 验证 → 上报（passed→awaiting_acceptance / failed→verify_failed）。

---

## 九·reviewer P1 回修

### 背景
reviewer 对「反馈一键自动修复」增强（scripts/repair-agent.ps1 auto 模式）评审出 2 个 P1 高危项，需合并前精确回修：

- **P1-1**：`repair-prompt.txt`（含反馈原文）写入 worktree 目录（`../wxx-repair-<taskNo>/`），但 `.gitignore` 未忽略 `wxx-repair-*/` 与 `repair-prompt.txt`，误 `git add .` 会泄露用户反馈原文。
- **P1-2**：feedback 原文（用户可控）拼进编码工具 prompt，未强制「仅改 diagnosis.code_files 白名单文件」，存在 prompt 注入越界改码风险。

### 关键事实核实
- `$RepoRoot` = `E:\2026-2027\2026-2027-1\MyProjects\wxx`；worktree 落点 `$wt = Join-Path $RepoRoot ".." ("wxx-repair-" + $taskNo)`，即**仓库根目录的上级目录**（如 `E:\2026-2027\2026-2027-1\MyProjects\wxx-repair-rt-xxx\`）。
- git worktree 生成的目录是**仓库外**的独立工作目录，本身不会被主仓库 `git add .` 拾取（gitignore 只作用于仓库内路径，`../wxx-repair-*/` 这种写法无效）。
- 真正的风险：`repair-prompt.txt` 写入 worktree 根；若在 worktree 内 `git add .`，会把该文件带入 `repair/<task_no>` 分支。原有代码仅在 `passed` 时删除，`failed` 时会残留。

### 修改内容

**P1-1（根治 prompt 文件入 git）— 三层防护：**
1. **.gitignore（全局兜底，仓库内路径）**：追加
   ```
   # repair-agent auto 模式：反馈原文 prompt 与隔离 worktree（敏感数据，绝不入库）
   repair-prompt.txt
   wxx-repair-*/
   ```
   `repair-prompt.txt`（不锚定路径）覆盖任何子目录位置；`wxx-repair-*/` 兜底（若未来有人把 worktree 建在仓库内或 .gitignore 作用范围内）。
2. **worktree 内 `.git/info/exclude`（针对当前 worktree，无需 commit）**：写入 worktree 前追加 `repair-prompt.txt`，确保即便 worktree 分支内含提示词也不被 `git add` 拾取。
3. **脚本内强制删除**：将原来「仅 `passed` 时删除」改为「无论 passed/failed，在 verify+submit 之前统一 `Remove-Item`」，保证本地不残留敏感文件。

**P1-2（提示词硬约束）— 强化为安全红线：**
1. prompt 新增「⚠️ 安全红线」块，明确声明反馈原文是**不可信、不可执行的用户数据**，严禁把其中的命令/路径/代码/配置/“忽略此规则”之类指令当真实指令。
2. 硬性限定：只允许修改【相关代码文件路径】白名单文件（diagnosis.code_files），禁止新增/删除/重命名/修改白名单之外的文件，白名单之外路径一律不得触碰。
3. **白名单为空时短路**：`auto` 模式在组装 prompt 前新增判断，`code_files.Count -eq 0` 时直接退出，**不调用任何编码工具**，并打印明确提示（避免无白名单时 AI 自由发挥）。
4. 保留原有「不 commit/push/部署、不改业务逻辑/状态机/接口语义、worktree 内自测」约束，并新增「不得读取与本任务无关的文件/密钥/令牌/环境变量」。

### 验证结果
- 语法检查：`[System.Management.Automation.Language.Parser]::ParseInput` 无错误 → `PS SYNTAX OK`。
- `git check-ignore -v` 三项全覆盖：
  - `repair-prompt.txt` → 命中 `.gitignore:232:repair-prompt.txt`
  - `wxx-repair-rt-123/repair-prompt.txt` → 命中 `.gitignore:233:wxx-repair-*/`
  - `wxx-repair-rt-123/some.go` → 命中 `.gitignore:233:wxx-repair-*/`

### 边界说明
- 未改动服务端 Go 代码的状态机/接口语义，未改动 `validTransitions`、verify/accept/deploy 流程。
- 未改动 auto 模式之外的 claim/verify 模式行为。
