# 校园导航「管理端拖动节点坐标无法持久化、刷新回退」重构核对清单

> 核对专员：pm-wxx（只读，未修改任何代码）
> 核对日期：2026-08-27
> 结论：leader 定位的三层根因全部核实属实，修复方案可行。以下为核对确认与验收标准。

---

## 一、根因确认（全部核实属实）

### 根因 1：数据层 —— 迁移 079 删空种子数据，且不回填

- 文件：`server/migrations/079_clear_preloaded_data.sql`
- 第 9 节明确执行 `DELETE FROM campus_checkin_steps;`，注释标注「9. 报到打卡点 campus_checkin_steps（12 条）」。
- 该迁移已执行（迁移 runner 幂等，`_migrations` 表已记录 `079_clear_preloaded_data.sql`），且 DELETE 不会自动回填。
- **后果：生产 MySQL 表 `campus_checkin_steps` 为空。** ✅ 属实

> 附带影响：079 同时删除了 kb_resources、process_steps、ai_briefings、competitions、毕设、career_policies、校历、course_schedules、health_activities 等大量演示数据，属**上线前统一清空的既定行为**，非本次 bug 独有。但 campus 模块因前端存在「假节点降级」而放大为拖拽不可用的用户可见故障。

### 根因 2：前端静默降级陷阱

- 文件：`frontend/lib/pages/campus/campus_map_page.dart`
- `_loadStepsFromServer()` 逻辑核实：
  1. 进入时先 `_remoteIds.clear()`；
  2. 管理员走 admin 接口 `ApiConfig.adminCampusSteps?campus=...`，普通用户走公开接口；
  3. `if (resp.data['code'] == 0)` 后取 `list`，**仅当 `list.isNotEmpty()` 时**才填充 `_remoteIds` 并覆盖 `_campusStepsMap`；
  4. 否则（空列表 / code != 0 / 异常 catch）**静默回退到本地硬编码 `_campus.steps`（前端 `_huifengSteps` 6 个 + `_langyaSteps` 6 个假节点），`_remoteIds` 保持为空**。
- **确认：`_remoteIds` 只在 admin 接口成功且 data 非空时填充。** ✅ 属实
- 关键点：本地假节点坐标与 050 纠正后的权威值**一致**（前端常量已按 OSM 纠正过），所以管理员看到的节点位置是「正确的假点」，更具迷惑性——看似可拖动，实则永远写不进库。

### 根因 3：拖拽保存取 `_remoteIds[index]` 为 null

- `_saveCoordinate()` 首行：`final stepId = _remoteIds[index]; if (stepId == null) return false;`
- 因 `_remoteIds` 为空，`stepId` 恒为 null，`_saveCoordinate` 直接返回 false。
- `_savePendingCoordinates()` 中所有请求「失败」，`_pendingCoordinates` 无法清空。
- `_toggleNodeEditing()` 退出编辑时报错：「保存失败：未加载到后端节点，请检查登录状态和网络后重试」。
- **坐标从未写入数据库，刷新即回退。** ✅ 属实

---

## 二、辅助机制核对（均正常）

### 1. 后端接口 `campus_handler.go`

- `ListAdminSteps`（GET /admin/campus/steps）：调用 `h.repo.ListAll(campus)`，空结果时 `steps == nil → steps = []model.CampusStep{}`，返回 `code:0, data:[]` ✅ 正常（不报错，返回空列表）。
- `UpdateStepCoords`（PATCH /admin/campus/steps/:id/coords）：参数绑定 lat/lng（required），含中国范围校验（lat 3~54、lng 73~136），调用 `h.repo.UpdateCoords`。**接口本身正常**，问题仅在于前端传不到有效 id。

### 2. 迁移机制 `server/cmd/migrate/main.go`

- `_migrations` 表记录已执行文件名，`SELECT COUNT(*) ... WHERE filename=?` 判断，已执行则 skip。
- 按 `sort.Strings(files)` 文件名排序执行；执行成功后 `INSERT INTO _migrations (filename)` 记录。
- **幂等确认 ✅**：同一文件名不重复执行。
- **迁移编号确认 ✅**：目录当前最大为 `109_feedback_repair_tasks.sql`，新增应为 **`110_restore_campus_steps.sql`**。

### 3. 方言转换 `server/internal/db/dialect.go`

- `ToMySQL()` 中 `insertOrIgnoreRe` 把 `INSERT OR IGNORE` → `INSERT IGNORE`（第 7 步）✅。
- `pkAutoRe` 把 `INTEGER PRIMARY KEY AUTOINCREMENT` → `BIGINT PRIMARY KEY AUTO_INCREMENT`（第 2 步）✅。
- 但注意：本次恢复数据的 `INSERT OR IGNORE INTO campus_checkin_steps (...)` 是 **DML 而非 DDL**，转换仅依赖 `insertOrIgnoreRe`；若采用 `INSERT OR IGNORE` 需指定唯一键才能真正「幂等去重」（见下方风险点 R2）。
- `INSERT` 列清单中的保留字列名会被 `insertColsRe` 加反引号处理（status/step_order 均非保留字，不受影响）。

---

## 三、风险点（需 leader/dev 关注）

### R1：恢复数据与 079 清空的语义冲突（高）
079 的意图是「上线前清空演示数据、改由管理员上传真实资源」。110 恢复 campus 种子数据在语义上是**逆向操作**。需确认产品决策：campus 报到节点应视为「系统必需默认流程」（随 080 之后恢复），还是应继续由管理员通过「流程管理」自行创建？
- 若管理员已通过流程管理创建了新的 campus_checkin_steps，110 恢复需避免冲突（用 INSERT IGNORE + 唯一键，或用 WHERE NOT EXISTS 判断「该 campus 无任何节点」才插入）。

### R2：INSERT OR IGNORE 的去重依据（高）
`INSERT OR IGNORE`（MySQL `INSERT IGNORE`）只有在**存在唯一约束命中冲突**时才忽略。当前 048 建表未对 `(campus_id, step_order)` 建唯一索引，若生产表为空则无所谓；但若需真正幂等（防止重复插入 12 条），应：
- 优先方案 A：显式 `WHERE NOT EXISTS` 判断该 campus 无节点才插入；
- 或方案 B：先给 `INSERT IGNORE` 增加唯一约束依赖（迁移补 `UNIQUE(campus_id, step_order)` 或显式指定 id 保证主键冲突）。

### R3：id 策略（中）
048 依赖自增 id（未显式指定）。110 若显式指定 id=1..12，可能与管理员后续追加的节点 id 冲突、或与 AUTO_INCREMENT 计数不一致。建议：**不显式指定 id，依赖 AUTO_INCREMENT 自增**，避免主键冲突；URL 中 `:id` 由 `_remoteIds` 从接口返回动态读取，不依赖固定 id。

### R4：坐标权威取值（中）
050 纠正后的权威坐标 = 前端 `_huifengSteps` / `_langyaSteps` 常量（已核对，二者一致）：
- 会峰：step1~6 = (32.2705,118.3055)(32.2745,118.3070)(32.2735,118.3060)(32.2770,118.3040)(32.2740,118.3090)(32.2720,118.3030)
- 琅琊：step1~6 = (32.2921,118.2988)(32.2932,118.3002)(32.2928,118.2995)(32.2940,118.2976)(32.2926,118.3000)(32.2917,118.2992)
- **注意：048 原始种子是错误坐标（会峰/琅琊写反 + 琅琊偏移 2km），绝不能照抄 048 的 lat/lng，须用 050 纠正后的值。** ✅ 方案中已明确，予以确认。

### R5：前端降级逻辑改动范围（中）
当前 `_loadStepsFromServer` 的静默降级对**普通用户**是合理兜底（离线可用），不能一刀切取消。需区分：
- **管理员（`_canEditNodes`）**：`_remoteIds` 为空时不再允许进入编辑/拖动，明确提示「后端无节点数据，请先在流程管理创建节点」。
- **普通用户**：保持现有静默回退到本地常量（只读展示，不涉及写库，无危害）。
- 改动点聚焦于 `_toggleNodeEditing()` 进入编辑前的守卫，以及（建议）admin 加载结果为空时的提示。

### R6：status 字段（低）
方案要求恢复 `status='published'`，与 048 一致。前端管理员走 admin 接口（含 draft）、普通用户走公开接口（仅 published），恢复 published 可同时服务两端。确认无冲突。

---

## 四、修复范围

| 层 | 文件 | 改动类型 | 说明 |
|---|---|---|---|
| 数据层 | `server/migrations/110_restore_campus_steps.sql`（新增） | 新增迁移 | 用 `INSERT OR IGNORE`（SQLite 写法）恢复会峰6+琅琊6共12节点，坐标取 050 权威值，`status='published'`，依赖自增 id（建议加幂等守卫） |
| 前端 | `frontend/lib/pages/campus/campus_map_page.dart` | 修改逻辑 | 管理员编辑模式下，`_remoteIds` 为空时禁止进入编辑并提示，不再静默降级到可拖动假节点；普通用户保留原兜底 |

**明确不在本次范围**：
- 不改 `campus_handler.go`（接口本身正常）。
- 不改 `dialect.go`（转换逻辑已支持所需语句）。
- 不改 079（保留既有清空语义）。
- 不改 048（历史种子保持不动，仅作为字段结构参照）。

---

## 五、验收标准

### 数据层（110 迁移）
1. 迁移文件存在且命名 `110_restore_campus_steps.sql`，编号为当前最大 +1（110）。
2. MySQL 下执行成功：`INSERT OR IGNORE` 被正确转换为 `INSERT IGNORE`，12 条数据落库。
3. SQLite 下执行成功：12 条数据落库。
4. 重复执行迁移不报错、不产生重复数据（幂等：依赖文件名去重 + 插入去重守卫）。
5. 落库坐标与 050 权威值一致（会峰/琅琊分开，无写反、无偏移）。
6. 12 条 `status` 均为 `published`；id 自增不冲突。

### 后端接口
7. `GET /admin/campus/steps?campus=huifeng` 返回 `code:0, data:[6 条节点]`（含 id）。
8. `GET /admin/campus/steps?campus=langya` 返回 `code:0, data:[6 条节点]`。
9. `PATCH /admin/campus/steps/:id/coords` 传入有效 id + 合法坐标，返回 `code:0`，DB 坐标更新。

### 前端（管理员端到端）
10. 管理员进入报到导航，地图正确加载后端 12 节点（非假节点），`_remoteIds` 非空。
11. 管理员拖动节点 → 松手暂存 → 退出编辑 → 校验坐标与 DB 一致 → 提示「全部节点坐标已保存」。
12. **刷新页面后坐标保持不回退**（对照 DB 值）。
13. 后端无节点（空表）时，管理员进入编辑**被阻止**，明确提示「后端无节点数据，请先在流程管理创建节点」，而非静默显示可拖动的假节点。
14. 普通用户体验不受影响：后端不可用时仍能离线看到只读流程（静默回退本地常量）。

### 回归
15. 流程管理（CRUD）面板增删改查、审核发布流程不受 110 迁移影响。
16. 079 之外的其他模块（知识库/流程/竞赛/毕设等）数据不受本次修复影响。
17. 已完成旧数据无冲突：若管理员已在流程管理手动创建过节点，110 不产生重复（依赖幂等守卫）。

---

## 六、待 leader/dev 决策项

1. **R1 语义决策**：campus 节点是否应随 110 恢复为「系统默认流程」？（若产品要求管理员自建，则应改为前端明确引导 + 不恢复数据，仅修前端提示。）
2. **R2 幂等实现**：选「WHERE NOT EXISTS 守卫」还是「补唯一约束 + INSERT IGNORE」？
3. **R3 id 策略**：是否确认「不显式指定 id、依赖自增」？

以上三项目前方案默认：恢复数据 + INSERT IGNORE（加守卫）+ 依赖自增，建议 dev 在 refactor-notes 中明确落地方案后再动工。

---

# 「一键自动修复增强」重构核对清单（反馈修复闭环 MVP 断点打通）

> 核对专员：pm-wxx（只读，未修改任何代码）
> 核对日期：2026-08-27
> 结论：leader 定位的「自动改代码断点」根因属实。服务端状态机与接口已就绪，唯一实质缺口是**执行端拿不到反馈原文（content）**，且本机脚本缺失「自动改码」动作。以下为核对确认与增强范围建议。

---

## 一、根因确认（对照 leader 5 点，全部核实）

### 根因 1：服务端状态机已就绪 ✅

- 文件：`server/internal/service/feedback_repair_task_service.go`
- `validTransitions` 核实：
  - `approved → running | cancelled`
  - `running → awaiting_acceptance | verify_failed`
  - `verify_failed → running | cancelled`
  - `awaiting_acceptance → deploy_pending | verify_failed`
  - `deploy_pending → deploying | verify_failed`
  - `deploying → deployed`
  - `deployed → closed`
  - `closed → {}`（终态）、`cancelled → {}`（终态）
- **闭环完整**：`approved → running → awaiting_acceptance → deploy_pending → deploying → deployed → closed`，且 `verify_failed` 可重新认领回到 `running`、`cancelled` 为独立终态。✅ 属实，无需改动。

### 根因 2：接口齐全 ✅

- 文件：`server/internal/handler/feedback_repair_task_handler.go`
- 管理端（JWT + 能力门控）：`CreateTask` / `ListTasks` / `GetTask` / `CancelTask` / `AcceptTask` / `RejectTask` / `DeployConfirmTask` / `DeployDoneTask`（8 个）。
- 内部执行端（`WXX_REPAIR_AGENT_TOKEN` token 鉴权）：`NextTask`（claim，对应 service.Claim）/ `VerifyTask`（对应 service.SubmitVerify）。
- ✅ 接口齐全。leader 描述中「AcceptTask/RejectTask/DeployConfirmTask/DeployDoneTask」均存在，另有 `CancelTask` 为补充。无需新增管理端接口。

### 根因 3：本机执行端只有 claim/verify，缺失「自动改码」 ✅

- 文件：`scripts/repair-agent.ps1`
- `param` 中 `[ValidateSet("claim", "verify")]`，仅两个模式。
- claim 模式（默认）：调 `POST /api/v1/internal/repair-tasks/next`，打印 `task.diagnosis.summary`、`task.diagnosis.code_files`、`task.feedback_ids`，然后仅输出「请在隔离分支 repair/xxx 完成修复（人工或 AI 编码工具）」的**提示文本**，并给出 `git worktree add` 示例命令，最后 `exit 0`。
- verify 模式：跑 `go vet/test` + `flutter analyze`，`Get-DiffStat` 收集 diff，`Submit-Verify` 上报。
- **确认：claim 不实际改任何文件，仅打印诊断提示让操作者手动复制粘贴改码。** ✅ 实属「自动改代码断点」。

### 根因 4（核心缺口）：执行端拿不到反馈原文 ✅

- 文件：`server/internal/service/feedback_repair_task_service.go` 的 `taskToPayload`（第 406 行起）
- 返回的 `RepairTaskPayload` 字段仅：`TaskNo / Title / Status / FeedbackIDs / BaseCommit / Branch / LogText / CreatedAt` + `Diagnosis`（`*AIRepairResponse`，含 Summary/CodeFiles/Module/RootCause/RepairHint/OCRText/MatchedFiles/RunID）。
- `FeedbackIDs` 只是 `[]string`（反馈业务 id 列表），**不包含任何一条反馈的 `content`（用户问题原文）**。
- **这是「自动修复无从下手」的根因**：AI 编码工具只拿到「AI 摘要（summary）+ 疑似文件列表」，拿不到用户原始反馈原文与期望效果，无法据此生成精准修复。✅ 属实。

### 根因 5：反馈原文位置 ✅

- 文件：`server/internal/model/entity.go` 第 229 行 `type Feedback struct`
- `Content`（`json:"content"`，对应 `feedback.content` 字段）、`Category`、`Module`、`ScreenshotURL`、`ResourceID` 等字段均存在。✅
- `model.AIRepairResponse`（`server/internal/model/dto.go` 第 582 行）：`Module / Summary / CodeFiles / RootCause / RepairHint / OCRText / MatchedFiles / RunID`。**注意：AIRepairResponse 里没有 `Content` 字段**——只有 AI 摘要/修复建议/文件定位，不含用户原文。✅
- 反馈原文从 `feedback` 表读取：`FeedbackService.Get(feedbackID)` → `feedbackRepo.GetByFeedbackID(feedbackID)` 返回 `*model.Feedback`（含 `Content`）。✅
- 补充核实：`FeedbackService.AIRepair`（`feedback_service.go` 第 182 行）内部已 `GetByFeedbackID` 拿到 `fb`，但仅把 `fb.Content` 写进 `FeedbackRepairJob.Summary`（工单审计字段），**未回传到 `AIRepairResponse`，也未进入 `RepairTaskPayload`**。所以诊断链路上到执行端时 content 已丢失。✅

---

## 二、增强范围建议（供 dev 参考，本清单不改代码）

### A. 服务端：执行端 payload 补充「关联反馈原文」

- **目标**：让 `RepairTaskPayload` 携带每条关联反馈的原文与元数据。
- 推荐做法（二选一或组合）：
  1. **扩展 struct**：在 `model.RepairTaskPayload` 新增字段 `FeedbackContents []FeedbackContentItem \`json:"feedback_contents,omitempty"\``，其中
     - `FeedbackContentItem{ FeedbackID, Category, Module, Content, ResourceID }`。
     - 在 `taskToPayload` 里，遍历 `p.FeedbackIDs`，调用 `s.feedbackSvc.Get(fid)` 逐个取 `Content/Category/Module/ResourceID` 填充。
     - 优点：`claim` 一次拿到全部原文，无需额外请求，脚本最省事。缺点：`NextTask` payload 变大，但反馈条数通常有限（可接受）。
  2. **新增内部接口** `GET /api/v1/internal/repair-tasks/:no/feedbacks`：按 taskNo 返回 `[{feedback_id, module, content, category, resource_id}]`。
     - 优点：payload 不膨胀、按需拉取；缺点：脚本需二次请求，且需保证该接口同样走 `RepairAgentTokenAuth`。
- **已知事实**：`AIRepairResponse` 无 `Content`；原文须用 `FeedbackService.Get(feedbackID)`（底层 `GetByFeedbackID`）读取。
- **变更面**：
  - `model/entity.go`：新增 `FeedbackContentItem` struct + `RepairTaskPayload` 增字段（若选方案 A.1）。
  - `service/feedback_repair_task_service.go`：`taskToPayload` 目前是包级函数，需改为接收 `*FeedbackRepairTaskService` 实例以调用 `feedbackSvc.Get` 填充原文。
  - （若选 A.2）`handler/feedback_repair_task_handler.go` 新增内部 GET 接口 + 路由注册，走 token 鉴权。

### B. 执行端 repair-agent.ps1：新增 auto 模式（一键自动）

- **目标**：`claim 领任务 → 拉取反馈原文 → git worktree 隔离分支 → 调用本机 AI 编码工具自动改码 → verify 上报`，一气呵成。
- 流程建议：
  1. `-Mode auto`：先 claim（复用现有 `next` 逻辑，读取 `feedback_contents`）。
  2. 若 `feedback_contents` 已在 claim payload 中（方案 A.1），直接使用；否则调 A.2 的 feedbacks 接口补齐。
  3. 创建 `git worktree add ../wxx-repair-<taskNo> -b repair/<taskNo>`（隔离，不污染主工作区）。
  4. 构造给 AI 编码工具的 prompt：含 `feedback contents`（用户原文）+ `diagnosis.summary/code_files/repair_hint` + 明确约束（不 commit/push、只在 worktree 内改、保持业务逻辑不变）。
  5. 调用本机编码工具改码（见下方「可用编码工具」）。
  6. 复用现有 `Run-Verification` + `Submit-Verify` 上报（passed→awaiting_acceptance）。
- **本机可用编码工具核对**：本机 OpenClaw 环境已具备 `gemini`（Gemini CLI one-shot prompts/生成/skills）、`gh-issues`、`spike`、`github` 等技能。适合用 **OpenClaw agent（openclaw / gemini CLI）** 作为自动编码执行器，配合上下文提示词（feedback 原文 + 诊断）在 worktree 内修改。
- **git worktree 隔离是否合理** ✅：合理且必要——自动改码必须与主工作区/生产代码物理隔离，契合「服务器不改码不部署、本机隔离」的安全边界；配合既有的 `BaseCommit/Branch` 字段可定位基线。
- **token 机制** ✅：`$env:WXX_REPAIR_AGENT_TOKEN` 已存在（claim/verify 均在用），auto 模式直接复用，无需新机制。

### C. 安全边界（必须明确强调）

1. **「服务器不改码不部署」原则保持不变**（见 `feedback_repair_task_service.go` 包注释与 handler 注释）：服务端只做状态机流转 + 审计。
2. **自动修复只发生在本机 `git worktree` 隔离分支**，绝不触碰主工作区，绝不自动 `commit/push/部署`。
3. **部署仍由管理员人工确认**：`DeployConfirm → DeployDone` 均为人工触发；auto 模式最远只做到「verify 上报 → awaiting_acceptance」，后续验收（`Accept`）、部署确认（`DeployConfirm`）、部署完成（`DeployDone`）全部仍由管理员在管理端完成。
4. 执行端鉴权仍走 `WXX_REPAIR_AGENT_TOKEN`（`middleware.RepairAgentTokenAuth`），token 不硬编码、不入库、不入日志。
5. 若引入新的内部 feedbacks 接口，必须同样挂在 token 鉴权下，不得暴露给前台用户。

---

## 三、接口/字段变更清单（汇总）

| 层 | 文件 | 变更 | 说明 |
|---|---|---|---|
| model | `server/internal/model/entity.go` | 新增 `FeedbackContentItem` | `{feedback_id, category, module, content, resource_id}` |
| model | `server/internal/model/entity.go` | `RepairTaskPayload` 增字段 | `FeedbackContents []FeedbackContentItem \`json:"feedback_contents,omitempty"\`` |
| service | `server/internal/service/feedback_repair_task_service.go` | `taskToPayload` 改签名 | 接收 `*FeedbackRepairTaskService`，遍历 feedback_ids 调 `feedbackSvc.Get` 填原文 |
| handler（可选） | `server/internal/handler/feedback_repair_task_handler.go` | 新增 `GET /api/v1/internal/repair-tasks/:no/feedbacks` | 若选方案 A.2，走 token 鉴权 |
| 路由（可选） | 路由注册处 | 注册上述内部接口 | 仅在选 A.2 时 |
| 脚本 | `scripts/repair-agent.ps1` | 新增 `auto` 模式 | 扩 `ValidateSet("claim","verify","auto")`；claim→拉原文→worktree→AI 改码→verify 上报 |

---

## 四、验收标准（auto 模式）

### 服务端
1. `NextTask`（claim）返回的 `payload.data.feedback_contents` 包含每条关联反馈的 `feedback_id / content / category / module / resource_id`，content 为用户问题原文非空。
2. （若选 A.2）`GET /api/v1/internal/repair-tasks/:no/feedbacks` 无 token 或错误 token 返回 401，正确 token 返回原文列表。
3. 原 `RepairTaskPayload` 既有字段（TaskNo/Diagnosis 等）不回归，diagnosis 仍含 Summary/CodeFiles（可空）。

### 执行端 auto
4. `pwsh -File scripts/repair-agent.ps1 -Mode auto`（token 已设）能：领取任务 → 读取反馈原文 → 在 `../wxx-repair-<taskNo>` 隔离 worktree（分支 `repair/<taskNo>`）内完成改码 → 跑 `go vet/test + flutter analyze` → 上报 verify。
5. 改码在 worktree 内发生，主工作区 `git status` 无残留改动；无 `commit/push` 动作。
6. verify 通过后服务端任务状态变为 `awaiting_acceptance`；verify 失败则 `verify_failed`，且日志含 go vet/test/flutter analyze 结果。

### 安全边界回归
7. 全流程服务器未执行任何源码修改/构建/部署；部署仍是管理员人工 `DeployConfirm/DeployDone`。
8. token 机制不变：无 `WXX_REPAIR_AGENT_TOKEN` 时 auto/claim/verify 均安全退出，不泄漏、不硬编码。
9. 新内部接口不向前台用户暴露（无 token 401）。

### 回归
10. 既有 claim/verify 模式行为不变（脚本向后兼容）。
11. 状态机全链路（approved→…→closed 及 verify_failed/cancelled 分支）不受本次增强影响。

---

## 五、待 leader/dev 决策项

1. **接口形态**：选「A.1 扩展 `RepairTaskPayload` 直接内嵌 feedback_contents」还是「A.2 新增独立内部接口」？建议首选 A.1（一次 claim 拿全、脚本简单、payload 增量可控）。
2. **编码工具**：确认最终调用 `openclaw agent` 还是 `gemini CLI`（本机两者均可用），并明确 prompt 中「不 commit/push/部署、仅限 worktree、保持业务逻辑不变」的硬约束。
3. **feedback_contents 字段命名**：建议 `feedback_contents`（json snake_case，与既有 `feedback_ids` 风格一致）。
