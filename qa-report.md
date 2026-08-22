# wxx vOPC PRD v1.0 回归测试报告

- 测试日期：2026-08-21
- 产品基准：`docs/wxx-vopc-prd-v1.0.md`
- 辅助基准：`pm-checklist.md`、`refactor-notes.md`
- 审查对象：当前未提交工作树（含本轮仅测试侧新增/调整）
- 总体结论：**NO-GO（禁止上线、禁止宣称 P0 验收完成）**

## 1. 执行摘要

开发报告声称已经关闭的以下问题，经代码与自动化回归确认方向正确：

1. 正式 `/api/v1/vopc` 路由已挂学院准入中间件；未登录为 401，guest、inactive、外院、school scope 为 403，合法计算机学院 active 用户为 200。
2. 外院邀请被拒绝，接受邀请时会二次校验当前账号学院归属和状态。
3. S0 草稿已有创建、查看、编辑、保存、补齐、提交闭环；提交后 PUT 返回 409，不能静默修改基线。
4. 项目详情的 `can_manage` 由服务端项目 owner/成员关系计算，前端不再用全局 capability 代替项目关系。
5. 正式应用路由已移除 `POST .../milestones/:stage/advance` 文本直推；阶段可通过里程碑提交、指定评审及 pass 推进。
6. 移动主导航压缩为 5 项，vOPC 保持一键直达。
7. 当前画面中已出现的 vOPC 操作按钮均连接真实 Provider/API；未发现仅改本地状态却提示成功的假按钮。

但仍存在多个 PRD P0 阻断项：

- **前端入口准入未实现**：导航和 `/vopc` 路由只依赖登录状态，外院/guest/未授权用户仍可看到入口并进入页面，再由 API 403；违反 PRD 要求的“前端入口、后端接口、数据查询三层校验”。
- **AI 虚拟员工闭环不存在**：只有创建时固定插入 4 个岗位；无岗位配置、AI task、真实模型调用、项目上下文隔离、版本化 AI 产出、人工接受/修改/退回/否决、Token/成本、额度、超时重试。
- **风险治理与结项复盘不存在**：R2 只是风险字符串，仍无审批门禁；无冻结/解冻/申诉；无继续、转向、暂停、终止、结项及失败复盘 API/UI。
- **里程碑门禁仍不完整**：虽已封闭正式路由文本直推，但提交可仅用任意非空文本且成果版本列表可为空；无评分量表、条件通过、豁免、甲方确认实体、测试/上线审批实体、真实验证与迭代结构化关联。
- **项目公开/展厅与文件安全闭环缺失**：项目仅实现默认私有关系读取；无 college/invite_only/restricted 实际发布策略、publish request/showcase 审批；成果只有引用元数据，无受鉴权的私有上传/下载。
- **任务/决策/成果/邀请只达到部分闭环**：缺任务依赖/评论/附件/完整编辑；决策缺 AI 建议/影响/截止/高风险复核；成员缺搜索、角色变更/移除；普通成员不能按 PRD 提交成果。

因此开发报告中的 `[blocked]` 均按**未覆盖/阻断**处理，没有计为通过。

## 2. 重点验收矩阵

| 验收项 | 结果 | 回归结论与证据 |
|---|---|---|
| 学院准入矩阵（API） | **通过** | 正式路由挂 `CollegeAccess`；专项测试覆盖 401/403/200。学院归属不区分大小写。 |
| 学院准入矩阵（前端入口） | **失败 / P0 阻断** | `router.dart` 明确写“入口和页面访问只依赖登录状态”；`_navItemsForRole` 无学院/access 过滤，外院与 guest 仍看到 vOPC。 |
| 查询层数据隔离 | **通过（私有项目范围）** | 列表仅 owner/active member；非成员猜项目 ID 返回 404；子资源通过项目关系校验。college/invite_only/restricted 尚未实现。 |
| 外院邀请 | **通过** | 邀请 active 外院用户返回 422。 |
| 接受邀请二次校验 | **通过（代码审查）/ 自动化不足** | Handler 在 accept 前重查 status/role/scope/college；缺“邀请后改学院/停用再接受”的专门测试。 |
| S0 创建草稿 | **通过** | 可只填名称创建 draft；服务端创建 owner、4 AI 岗位、S0-S9 里程碑和事件。 |
| S0 查看/编辑/保存 | **通过** | 详情返回完整字段；真实 PUT 持久化并写 `project.draft_updated`；非成员 404。 |
| S0 补齐并提交 | **通过** | 不完整草稿可 PUT 补齐后 submit 进入 S1；缺字段服务端 422。 |
| 提交后禁止编辑 | **通过** | S1 后 PUT 返回 409。尚无正式 change request，故只能判“禁止直接编辑”通过。 |
| owner 的 `can_manage` | **通过** | owner 详情响应 `can_manage:true`。 |
| co_owner/platform_operator/member 的 `can_manage` | **未充分覆盖** | 代码逻辑正确区分 co_owner/platform_operator 与普通成员，但缺完整响应矩阵自动化。 |
| 工作台入口可见性 | **失败 / P0 阻断** | 非学院用户入口仍可见；合法学院用户桌面/移动均可一键进入。 |
| 正式里程碑提交 | **通过（基础）** | 真实 API 落库，验证目标为当前阶段下一阶段、防重复 pending、跨项目成果版本拒绝。 |
| 指定评审 | **通过（基础）** | 指定人必须为项目 mentor/reviewer/platform_operator；非指定普通人 403。 |
| 评审推进 | **通过（基础）** | pass 事务内推进阶段，return 不推进；重复/阶段漂移返回冲突。 |
| 正式路由无文本直推 | **通过** | `routes.go` 已移除 advance 路由；本轮测试路由同步正式配置，并断言该 URL 为 404。 |
| 门禁实质性 | **失败 / P0 阻断** | `evidence` 只需非空，`artifact_version_ids` 可为空；前端固定发送空数组。无法证明阶段交付物、甲方 S2/S5/S6、真实用户验证、交付与迭代。 |
| 项目默认私有 | **通过** | migration 默认 `private`；非成员读不到项目及子资源。 |
| 跨项目隔离 | **通过（现有实体）** | artifact/version/submission 等查询校验 project_id；非成员/跨项目 ID 不泄漏。私有文件 URL 因文件链路不存在而未覆盖。 |
| 任务闭环 | **部分通过** | 创建、真人/AI 岗位负责人校验、验收标准、优先级、截止、状态机、审计存在；缺依赖、评论、附件、字段编辑，AI 不能执行。 |
| 决策闭环 | **部分通过** | 创建、列表、resolve/cancel、理由、审计存在；缺影响、截止、指定决策人、AI 建议和高风险复核。 |
| 成果闭环 | **部分通过** | 成果元数据、版本引用、项目归属和审计存在；缺真实上传/鉴权下载、版本查看 UI、AI 审阅和成员提交权限。 |
| 邀请闭环 | **部分通过** | 邀请、接受/拒绝、接受后成员关系和审计存在；缺用户搜索、角色变更/移除、通知。 |
| AI 默认 4 岗 | **通过（仅初始化）** | 创建项目插入 4 个 enabled 岗位。 |
| AI 配置/执行/审阅/成本/隔离 | **失败 / P0 阻断** | 无表/API/UI/模型执行，不接受“缺密钥”为通过理由。 |
| 移动端导航 | **通过（代码/构建层）** | compact 模式为 5 项并保留 vOPC；Flutter Web release 构建成功。未做 320/360/390dp widget/golden/WebView 真机验证。 |
| 已出现按钮的真实 API | **通过（连接性）** | 创建/编辑/提交、任务、决策、邀请、成果、版本、里程碑提交/评审均调用真实 API；Provider 对失败不返回伪成功。部分按钮缺 mutation 禁用和结果反馈。 |
| 数据库 SQLite 迁移 | **通过** | vOPC Handler 测试在内存 SQLite 执行 097-099；全量 app 迁移测试通过。 |
| 数据库 MySQL 方言转换 | **通过（静态转换）/ 真库未覆盖** | 本轮新增 097-099 `ToMySQL` 回归，确认不残留 AUTOINCREMENT、`INDEX IF NOT EXISTS` 等 SQLite 专有语法。未提供 MySQL 实例，未执行真实 MySQL DDL/业务 SQL。 |
| Turso/libSQL | **未覆盖** | 无可用 Turso 测试环境；097-099 为 SQLite 方言，未做远端 libSQL 迁移实跑。 |
| 风险 R0-R3 | **失败 / P0 阻断** | R3 默认 422、自动升档存在；R1/R2 审批、冻结、解冻、申诉均不存在。 |
| 结项/暂停/转向/终止/复盘 | **失败 / P0 阻断** | 无 close/retrospective API/UI；S9 pass 直接 completed，不能替代 PRD 复盘闭环。 |
| 工作台 PRD 全部信息区块 | **失败** | 缺完成度、AI 状态、风险列表、活动日志、下一里程碑细节等。 |
| 首页八区块 | **失败（P1 域为主）** | 仅我的项目、快速发起、邀请；大厅、伙伴、导师资源、展厅、活动等无真实业务域。 |

## 3. 自动化、静态检查与构建结果

| 命令 | 结果 |
|---|---|
| `go test ./internal/handler -run VOPC -count=1` | **PASS**（含学院准入、S0 编辑、私有隔离、任务、邀请、成果、正式评审及 S0-S9 正式流程） |
| `go test ./internal/auth ./pkg/app ./internal/db -count=1` | **PASS** |
| `go test ./... -count=1` | **PASS** |
| `go vet ./internal/handler ./internal/auth ./pkg/app ./internal/db` | **PASS** |
| `go vet ./...` | **PASS** |
| `go build ./...` | **PASS** |
| `flutter test test/vopc_provider_test.dart` | **PASS：6 项** |
| `flutter test` | **PASS：12 项** |
| 定向 `flutter analyze`（vOPC/router/capability/test） | **PASS：No issues found** |
| 全量 `flutter analyze` | **FAIL：285 个既有 info 级 lint**；输出示例为 `prefer_const_constructors`、`prefer_interpolation_to_compose_strings`、`unnecessary_overrides`，未发现本轮 vOPC 定向分析错误，但全库门禁非零，不能记为通过。 |
| `flutter build web --release` | **PASS**，生成 `build/web` |
| `git diff --check` | **PASS** |

## 4. 本轮测试侧变更

遵守“不修改业务源码”的约束，本轮只调整/新增测试文件：

- 调整 `server/internal/handler/vopc_handler_test.go`
  - 测试路由不再注册已从正式应用移除的文本直推 endpoint；
  - S0-S9 回归改走真实 milestone submission + 指定 reviewer + pass；
  - 明确断言旧 `.../advance` URL 返回 404。
- 新增 `server/internal/db/migration_vopc_test.go`
  - 对 097/098/099 做 SQLite→MySQL 方言转换回归。

未部署，未 commit，未 push。

## 5. 失败项与发布阻断排序

### P0 阻断

1. 前端 vOPC 入口和路由没有学院准入过滤，外院/guest 仍可见并可进入页面。
2. AI 任务、模型调用、项目上下文隔离、版本化产出、四种人工审阅、Token/成本、额度、失败重试全部缺失。
3. R1/R2 审批、风险冻结/解冻/申诉缺失；R2 项目可继续提交和推进。
4. 继续/转向/暂停/终止/结项复盘闭环缺失。
5. 里程碑证据仍可由任意非空文本满足，且前端不能绑定成果版本；评分量表、条件通过、豁免及关键阶段结构化门禁缺失。
6. 私有文件上传/鉴权下载不存在，无法验收“私有文件 URL 不可被未授权用户直接访问”。
7. 项目级 AI 上下文和跨项目复用授权/脱敏不存在。

### 重要失败/未完成

- 工作台信息不完整；首页八区块多数无实现。
- 任务依赖、评论、附件、完整编辑缺失。
- 决策高风险复核及 PRD 字段缺失。
- 成员搜索、角色变更/移除、通知缺失。
- college/invite_only/restricted、展厅申请审核、成果包导出缺失。
- 全量 Flutter analyze 非零；虽为既有 info lint，也应在项目门禁策略中明确是清理还是基线豁免，不能静默写成 PASS。
- 未完成真机/WebView、小屏视觉、真实 MySQL/Turso、P95、备份恢复验收。

## 6. 最终判定

**NO-GO。**

本轮确认学院后端准入、外院邀请封堵、S0 编辑提交、服务端 `can_manage`、移动导航和正式里程碑路由整改有效；但 PRD P0 的 AI 协作、安全治理、结项复盘、实质门禁、私有文件安全，以及前端入口学院准入仍未达到验收标准。不得将 `refactor-notes.md` 中的 `[blocked]` 视作通过，也不得依据现有骨架宣称 vOPC PRD v1.0 已完成。

## 2026-08-22 严格回归追加记录

### 本轮结论
**NO-GO（P0 仍阻断）**。本轮三组指定修复均获得自动化证据支持；但项目整体仍不能上线验收。剩余 P0 阻断至少包括 AI 执行/上下文隔离、风险审批与冻结/解冻、完整里程碑量表/豁免/甲方门禁、私有文件上传下载、结项复盘等缺失域。

### 执行证据

| 检查 | 命令 | 结果 |
|---|---|---|
| Go 全量测试 | `go test ./... -count=1` | **PASS**；所有包通过（含 handler、repository、service、pkg/app） |
| Go 静态检查 | `go vet ./...` | **PASS**（exit 0） |
| Go 构建 | `go build ./...` | **PASS**（exit 0） |
| Flutter 全量测试 | `flutter test` | **PASS**；14 tests passed |
| Flutter vOPC 定向测试 | `flutter test test/vopc_access_test.dart test/vopc_provider_test.dart` | **PASS**；9 tests passed |
| Flutter vOPC 定向 analyze | `flutter analyze lib/pages/vopc/vopc_page.dart lib/config/router.dart lib/providers/vopc_provider.dart lib/utils/vopc_access.dart test/vopc_access_test.dart test/vopc_provider_test.dart` | **PASS**；No issues found |
| Flutter 全量 analyze | `flutter analyze` | **既有 lint/info，NO-GO 门禁**；268 issues，退出码 1，主要为全仓既有 `info` 级提示（含非 vOPC 文件） |
| Flutter Web release | `flutter build web --release` | **PASS**；Built `build/web` |
| 空白检查 | `git diff --check` | **PASS** |
| SQLite/MySQL vOPC 静态兼容 | `go test ./server/internal/db -run TestToMySQLVOPCMigrations -count=1` | **PASS**；vOPC 097-099 转换测试通过 |
| app 路由/迁移专项 | `go test ./server/pkg/app -run 'Test(KeyRoutesReachable\\|RouteRegistrationCount\\|RunMigrationsCreatesSchema\\|RunMigrationsIdempotent)' -count=1` | **PASS** |
| handler vOPC/里程碑/邀请专项 | `go test ./server/internal/handler -run 'Test.*VOPC\\|Test.*Milestone\\|Test.*Invitation' -count=1` | **PASS** |

### 三组修复验收

1. **学院准入入口与 `/vopc` 路由：PASS（自动化覆盖）**
   - 前端 `vopc_access_test.dart` 覆盖 active、guest、inactive、外院、非 college、owner 信息缺失、无 `vopcRead`、计算机学院合法用户。
   - 正式路由已挂载 `CollegeAccess`，并继续要求 JWT/用户存在；`/vopc` 与项目子路由均位于该保护组。
   - 后端专项测试通过，包含准入矩阵与外院邀请/接受复验。

2. **里程碑真实成果版本门禁：PASS（基础硬门禁）**
   - handler 测试覆盖空版本失败；既有逻辑校验版本存在、属于当前项目、状态/内容有效。
   - 前端加载真实 artifact versions，提交必须选择至少一个版本；Provider 测试断言请求携带真实 `artifact_version_ids`。
   - `migration_vopc_test.go` 的 097-099 SQLite→MySQL 转换测试通过；app 完整迁移与幂等专项通过。

3. **S1-S9 旧文本直推禁用、S0 提交正常：PASS**
   - 正式路由不再注册 `POST /milestones/:stage/advance`；handler 回归断言旧端点 404，源码搜索未发现前端 `advanceProject`/旧直推调用。
   - S0 草稿提交及新增编辑/提交调用测试通过；S1-S9 只能进入正式 milestone submission/review 通道。

### 剩余 P0 阻断

- **P0：Flutter 全量 analyze 未清零**，268 条全仓既有 lint/info 导致退出码 1；vOPC 定向 analyze 已通过。需区分并治理既有 lint 后才能将全量分析记为 PASS。
- **P0：AI 任务真实执行、项目上下文隔离、产出审阅/成本/重试仍缺失**。
- **P0：R1/R2 审批、R3 专项流程、风险冻结/解冻/申诉仍无完整模型/API/UI**。
- **P0：里程碑评分量表、条件通过、豁免审批及甲方 S2/S5/S6 结构化证据仍缺失**；当前“真实版本”门禁不等于完整 PRD 阶段验收。
- **P0：成果私有文件上传/鉴权下载与字段级隔离仍缺失**。
- **P0：结项/终止/暂停/转向/复盘及完整审计查询仍缺失**。

本轮未修改业务源码，未部署、未 commit、未 push；仅更新本报告。

## 2026-08-22 第三轮严格回归（结项/异常状态机 + 风险治理最小闭环）

### 本轮结论
**NO-GO（P0 仍阻断）**。本轮专项针对“S9 结项与异常状态机 + 风险治理最小闭环”验证，新增能力方向正确且获得自动化证据支持；但整体仍不能上线验收。剩余 P0 至少包括：AI 真实执行与上下文隔离、里程碑量表/条件通过/豁免/甲方 S2-S5-S6 结构化证据、私有文件上传/鉴权下载、R3 专项审批未与 R2 区分、生产同构（MySQL/Turso）真库验证、全仓 Flutter analyze 既有 lint 门禁。

### 一、逐项 PASS/FAIL 判定

#### A. 结项/异常状态机

| 编号 | 验收点 | 结果 | 证据 |
|---|---|---|---|
| A1 | S9 里程碑 pass 后不再直接 completed，进入可结项状态 | **PASS** | `vopc_handler.go` 将 `stageStatuses[9]` 由 `completed` 改为 `closeable`；`ReviewMilestone` 在 target==9 时以 `stageStatuses[9]`（closeable）作为 `nextStatus`。`completedLike = {completed, closeable}` 使 closeable 下任务/决策/里程碑均被拦截。 |
| A2 | close/retrospective 理由、失败证据、风险处置、成果包要点、人类结项决策缺一不可 | **PASS（有边界）** | `closeInput.normalizeAndValidate`：reason 必填≤4000；close 强制 `human_decision`+`outcome_package`；terminate 强制 `failure_evidence`；pivot 强制 `human_decision`。**边界**：A2 点名的“风险处置”未作为独立必填字段落到结项闭环——`close` 仅校验 human_decision/outcome_package，风险处置未在 close 时强校验“已无未处置风险”或关联风险记录（风险处置在 `vopc_risks`/`vopc_risk_appeals` 独立闭环，未与结项强制串联）。 |
| A3 | pause/resume/pivot/terminate/archive 合法流转，理由/证据/权限/审计正确，非法流转被拒 | **PASS** | `closeTransition` 状态机 + `CloseProject` 单事务写 `vopc_close_records`+`vopc_events`。自动化：`TestVOPCCloseStateMachine` 覆盖非管理 404、非法动作 422、draft-close 409、pause→resume、terminate 缺证据 422、terminate→archive、terminated 禁止 close/pivot 409；`TestVOPCPivotResetsProject` 覆盖 pivot 回 S0/draft 并复位 9 个里程碑。权限由 `projectPolicy(…,"manage")` 收敛到 owner/co_owner/platform_operator。 |
| A4 | 迁移 101 SQLite/MySQL 方言兼容、幂等、建表正确 | **PASS（静态转换 + 幂等，无真 MySQL）** | `101_vopc_close_state_machine.sql` 建 `vopc_close_records` 表 + `completed_at`/`closed_at` 可空列 + 索引。`TestToMySQLVOPCMigrations` 纳入 101，断言无 AUTOINCREMENT/`INDEX IF NOT EXISTS` 残留，且 `INTEGER PRIMARY KEY AUTOINCREMENT`→`BIGINT PRIMARY KEY AUTO_INCREMENT`。`TestRunMigrationsIdempotent`/`TestRunMigrationsCreatesSchema` 通过（`_migrations` filename 去重；`isDuplicateColumnError`/`isDuplicateIndexError` 容错）。**边界**：无真实 MySQL 实例 DDL/业务 SQL 实跑。 |

#### B. 风险治理最小闭环

| 编号 | 验收点 | 结果 | 证据 |
|---|---|---|---|
| B1 | risk/approval/freeze/appeal 四实体表（迁移 102） | **PASS** | `102_vopc_risk_governance.sql` 建 `vopc_risks`/`vopc_risk_approvals`/`vopc_freeze_records`/`vopc_risk_appeals` 四表 + 三索引。 |
| B2 | R2 未审批（含未双人审批）不得推进里程碑 | **PASS** | `milestoneAdvanceAllowed` 在 `SubmitMilestone` 内对 R2/R3 项目校验“存在一条同 `risk_level` 且 `status='approved'` 的风险”。`TestVOPCRiskGovernanceAndGate` 覆盖：R2 未审批提交 S2→409；单人 approve 后仍 `open`；同人重复审批→409；第二人 approve→`approved` 后提交 S2→201。 |
| B3 | R3 专项权限 + 双人审批放行 | **FAIL（语义缺口）** | 当前 R3 与 R2 共用同一 `milestoneAdvanceAllowed`/同一条 `VOPCRiskManage` 能力+`platform_operator` 成员门禁，仅以 `risk_level` 字符串区分；**未实现 PRD 13.1 的“R3 禁止或专项审批——默认禁止，按学校制度专项审批”的独立专项通道**。R3 风险可由任意 project manager 通过 `CreateRisk` 创建，审批门槛与 R2 完全一致（双人 approve），无更高权限/线下专项审批实体。这是本轮最实质的语义缺口。 |
| B4 | 创建风险、审批、冻结/解冻、申诉 API 与审计 | **PASS** | `CreateRisk`/`ListRisks`/`ApproveRisk`/`FreezeProject`/`CreateRiskAppeal`/`ResolveRiskAppeal` 全部落库并 `writeEvent` 审计；路由挂 `VOPCProjectManage`（create/appeal）或 `VOPCRiskManage`（approve/freeze/resolve）。`TestVOPCRiskFreezeAndAppeal` 覆盖 owner 冻结 403、admin 冻结 200、冻结后 submit 409、申诉 201、owner 裁定 403、admin 裁定 200、解冻→pending_review。 |
| B5 | 迁移 102 方言兼容/幂等 | **PASS（静态 + 幂等，无真 MySQL）** | `TestToMySQLVOPCMigrations` 纳入 102，同 A4 断言；幂等测试通过。**边界**：无真 MySQL/Turso 实跑。 |

#### C. 回归不退化

| 验收点 | 结果 | 证据 |
|---|---|---|
| 既有 vOPC 学院准入/S0 草稿/里程碑版本门禁/邀请/任务/决策测试 | **PASS** | `go test ./internal/handler` 全部 PASS，含 `TestVOPCAccessHTTPMatrix`、`TestVOPCS0ToS9FormalMilestoneFlowAndNoTextAdvanceRoute`、`TestVOPCTaskLifecycleAuthorizationAndAudit`、`TestVOPCInvitation…` 等。 |
| 路由计数一致 | **PASS** | `TestRouteRegistrationCount`（静态计数下限 479）、`TestKeyRoutesReachable` 均 PASS。**注**：计数测试的 vOPC 路由锚点列表尚未显式补充 7 条新增治理/结项路由（close/close-records/risks×2/freeze/risk-appeals×2），当前仅靠“总计数下限”兜底，建议后续将 7 条新路由加入锚点断言。 |

#### D. 全量门禁

| 命令 | exit code | 结果 |
|---|---|---|
| `go build ./...` | 0 | PASS |
| `go vet ./...` | 0 | PASS |
| `go test ./... -count=1` | 0 | PASS（全包，含 handler/repository/service/app） |
| `flutter test` | 0 | PASS（14 tests） |
| vOPC 定向 `flutter analyze` | 0 | PASS（No issues found，6 items） |
| `git diff --check` | 0 | PASS |

（全仓 `flutter analyze` 仍为既有 lint/info 非零门禁，本轮未重跑，沿用上轮结论 268 条既有 info 级提示，不计入本轮定向 PASS。）

### 二、命令与逐项 exit code（本次实跑）

- `go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./... -count=1` → exit 0（`ok` 全包，无 FAIL）
- `go test ./internal/handler -run 'Test.*(VOPC|Close|Risk|Freeze|Appeal|Pivot|Governance|StateMachine)' -count=1 -v` → exit 0（TestVOPCCloseStateMachine / TestVOPCPivotResetsProject / TestVOPCRiskGovernanceAndGate / TestVOPCRiskFreezeAndAppeal 均 PASS）
- `go test ./internal/db -run TestToMySQLVOPCMigrations -count=1 -v` → exit 0（097–102 六项 PASS）
- `go test ./pkg/app -run 'Test(RouteRegistrationCount|KeyRoutesReachable|RunMigrationsCreatesSchema|RunMigrationsIdempotent)' -count=1` → exit 0（迁移幂等/建表通过；日志“成功执行 99 个迁移文件”与 `server/migrations/*.sql` 实为 99 个文件一致，101/102 已被 `go:embed migrations/*.sql` 收录执行）
- `flutter test` → exit 0（14 tests passed）
- `flutter analyze <vOPC 定向 6 文件>` → exit 0（No issues found）
- `git diff --check` → exit 0

### 三、语义缺口与可接受性判定

1. **“审批不存在 risk id 返回 404 而非 403”——可接受（并推荐保持）**。`ApproveRisk` 对不存在的风险返回 404“风险不存在或无权操作”，对存在风险但非治理角色返回 403。这是 fail-closed 且不泄露风险存在性的正确姿势；且路由层 `VOPCRiskManage` 能力门控使非治理用户（student）在进入 handler 前已被 403 拦截。结论：**该语义可接受，不构成缺陷**。

2. **R3 专项审批缺口（B3）——不可接受，P0**。R3 与 R2 共用双人审批门槛，未实现 PRD 13.1 的专项审批/默认禁止通道；R3 风险可由任意 manager 创建。建议后续落地独立 R3 专项权限与线下审批录入实体。

3. **结项风险处置未强校验（A2 边界）——建议项，非本轮 P0**。`close` 未强制校验“项目已无未处置风险/申诉”，风险闭环与结项闭环未强制串联。PRD 未将“结项时风险清零”列为硬门槛，故记建议，不阻断。

4. **项目级 vs 风险级 risk_level 分离——提示项**。`milestoneAdvanceAllowed` 以 `vopc_projects.risk_level`（创建时自动升级 R2/R3）为闸，要求存在同等级 `approved` 风险；`CreateRisk` 的风险级 `risk_level` 可独立填 R0–R3。若项目 auto-promote 为 R3 但只登记了 R2 风险，即使 R2 已批准仍无法推进门槛——语义自洽，但需文档说明该“同等级匹配”约束。

### 四、剩余 P0 排序

1. R3 专项审批未实现（默认禁止 + 专项通道 + 线下录入实体缺失）。
2. AI 任务真实执行、项目上下文隔离、版本化产出、四人审阅、Token/成本/重试仍缺失。
3. 里程碑评分量表/条件通过/豁免/甲方 S2-S5-S6 结构化证据仍缺失（结构化版本元数据门禁≠完整阶段验收）。
4. 成果私有文件上传/鉴权下载及字段级隔离仍缺失。
5. 生产同构（真实 MySQL/Turso 升级/并发/备份恢复）与真机/WebView/P95 未验证；全仓 `flutter analyze` 既有 lint 门禁未治理。
6. （新）`TestRouteRegistrationCount` 未将 7 条新增 vOPC 结项/风险治理路由纳入锚点断言（测试覆盖补强，非功能缺陷）。

### 五、最终判定

**NO-GO。**

本轮结项/异常状态机（A）与风险治理最小闭环（B，除 R3 专项）的代码与自动化证据方向正确、质量合格，A1–A4、B1、B2、B4、B5、C、D 全部通过，整体未回归退化，路由计数一致。但由于 R3 专项审批通道缺失（B3）以及上轮已明确的 AI 闭环、完整里程碑业务门禁、私有文件安全、生产同构验证等 P0 阻断项仍存在，**不得上线、部署或宣称 vOPC PRD v1.0 P0 验收完成**。

本轮仅更新本报告（含本追加章节），未修改业务源码、未新增迁移、未部署、未 commit、未 push。
---

## 2026-08-22 第四轮严格回归（R3 独立专项审批通道 + A/B/C/D 全量复核）

- 测试日期：2026-08-22 10:02–10:12（Asia/Shanghai）
- 基准：docs/wxx-vopc-prd-v1.0.md、pm-checklist.md、refactor-notes.md、audit-report.md
- 审查对象：当前未提交工作树（新增 vopc_close.go / vopc_risk.go / vopc_governance_test.go / 101_*.sql / 102_*.sql，及改动 vopc_handler.go / vopc_delivery.go / vopc_decisions.go / routes.go / vopc_handler_test.go / migration_vopc_test.go）
- 运行环境：go 1.x（server 全包）、Windows/pwsh

### 本轮结论

第三轮报告标记的「R3 专项审批未实现（B3）」在本轮已落地并通过自动化验证。A/B/C/D 四项逐项复核全部 PASS，无新增回归、无新增 P0 阻断。**本轮相对第三轮的唯一实质变更是 R3 独立专项通道实现**，其余 A/B/D 结论维持第三轮判定。

### 一、逐项 PASS/FAIL 判定（证据）

**A. 结项与异常状态机 — PASS**
- close/pause/resume/pivot/terminate/archive 合法流转：TestVOPCCloseStateMachine（含 pause→paused、resume→pending_review、terminate 需失败证据 422、terminated 后 close/pivot 409、archive 200）全通过。
- 权限：非管理角色 close → 404（test 断言 on-manager close got 404）on-manager close got 404）。
- 必填校验：terminate 缺失败证据 422、close 缺 human_decision/outcome_package 422（closeInput.normalizeAndValidate）。
- 非法流转：draft 直接 close → 409（draft close got 409）。
- pivot 里程碑重置：TestVOPCPivotResetsProject 断言 stage=S0/status=draft，且非 S0 里程碑 esets=9（9 个非 S0 里程碑复位 pending）。
- S9 pass 不再直接 completed：TestVOPCS0ToS9FormalMilestoneFlowAndNoTextAdvanceRoute 断言 S9 后 status=closeable，再 close → completed。
- close/retro 事件审计：writeEvent 与 vopc_close_records 同事务写入（CloseProject 单 tx、defer Rollback）。
- 方言兼容：101 仅 DDL（AUTOINCREMENT、CURRENT_TIMESTAMP、IF NOT EXISTS、ADD COLUMN），TestToMySQLVOPCMigrations PASS。

**B. 风险治理最小闭环 — PASS**
- 风险创建：TestVOPCRiskGovernanceAndGate 201；未加入项目的平台角色审批 → 404（pprove without membership got 404）。
- R2 双人 gate：单人 approve→风险仍 open、同审批人重复 409、第二人 approve→approved、审批后里程碑 201；未审批里程碑 409（R2 unapproved milestone got 409）。
- freeze/unfreeze 权限：TestVOPCRiskFreezeAndAppeal owner 冻结 403、admin 冻结 200→risk_frozen、冻结后 submit 409、admin 解冻→pending_review。
- 申诉/resolve 权限与原子性：owner resolve 403，admin resolve 200（单 tx + RowsAffected 校验）。
- 方言兼容：102 仅 DDL，TestToMySQLVOPCMigrations PASS。

**C. R3 独立专项通道 — PASS（本轮新增）**
- 普通 manager 创建 R3 → 403（owner create R3 got 403）。
- platform_operator 创建 R3 → 403、platform_operator 审批 R3 → 403（代码 isSpecialRiskGovernance 分流）。
- 专项角色（risk_governance）创建 R3 → 201。
- 缺专项审批里程碑 → 409；单人专项 approve → open 且里程碑仍 409；双人专项 approve → approved 且里程碑 201。
- 任一专项 reject → rejected，后续里程碑 409（ejected R3 milestone got 409）。
- R2 语义不退化：R2 仍走 platform_operator 双人审批（TestVOPCRiskGovernanceAndGate 稳定通过）。

**D. 既有闭环不回归 — PASS**
- 学院准入、S0 草稿、成果版本门禁、任务、决策、邀请、正式里程碑评审、迁移 097-100 全量通过。
- go test ./... -count=1 全包 PASS；历史迁移 097-100 文件与 git 基线无 diff（未误改）。

### 二、命令与逐项 exit code（本次实跑）

| 门禁 | 结果 | exit code |
|---|---|---|
| go build ./... | PASS | 0 |
| go vet ./... | PASS | 0 |
| go test ./... -count=1 | PASS | 0 |
| go test ./internal/db -run TestToMySQLVOPCMigrations -count=1 | PASS | 0 |
| go test ./pkg/app -run 'Test(KeyRoutesReachable\|RouteRegistrationCount\|RunMigrationsCreatesSchema\|RunMigrationsIdempotent)' -count=1 | PASS | 0 |
| git diff --check | PASS | 0 |
| go test ./internal/handler -run 'TestVOPC(CloseStateMachine\|PivotResetsProject\|RiskGovernanceAndGate\|RiskFreezeAndAppeal\|R3SpecialGovernanceChannel\|S0ToS9FormalMilestoneFlowAndNoTextAdvanceRoute)' -count=1 -v | PASS | 0（6/6 用例 PASS）|

- 前端：本批为纯后端改动，flutter 未涉及，记「未涉及」。

### 三、残留/垃圾与安全核查

- 无 createProjectAt/ytesBuffer 等占位残留；ytes.Buffer 仅出现在既有测试 helper（request 序列化 json body）与上传/语音测试，均为合法使用。
- 无伪成功：close/approve/freeze/resolve 均走真实事务 + RowsAffected/committed 标志 + defer Rollback，任一步失败回滚。
- 事务与审计原子：vopc_close_records / vopc_risks / vopc_risk_approvals / vopc_freeze_records / vopc_risk_appeals 与对应 writeEvent 同事务。
- 历史迁移 097-100 未改动；101/102 为新增，方言兼容通过。

### 四、语义缺口（提示/建议项，非本轮阻断）

1. 结项未强制校验“项目已无未处置风险/申诉”（同第三轮建议项）。
2. 项目级 vs 风险级 risk_level 的“同等级匹配”约束仍待文档说明。
3. TestRouteRegistrationCount 锚点断言仍未将 7 条新增 vOPC 结项/风险治理路由计数纳入（测试覆盖补强，非功能缺陷）。

### 五、最终判定

**NO-GO。**（维持整体 NO-GO，非因本轮本批功能缺陷）

本轮 A/B/C/D 全部 PASS，R3 专项通道（第三轮 P0）已实现；相对前轮无新增回归、无新增 P0 阻断。整体仍维持 NO-GO 的原因是上轮及更早已明确的、非本批范围的 P0 阻断项仍未消除：AI 虚拟员工闭环、里程碑完整业务门禁（评分量表/条件通过/豁免/甲方结构化证据）、私有文件安全与字段级隔离、生产同构（真实 MySQL/Turso 并发/备份恢复）、前端入口三层校验、flutter analyze 既有 lint 门禁等。

本轮仅更新本报告（追加本章节），未修改业务源码、未新增迁移、未部署、未 commit、未 push。
---

## 附录 B 三缺口修复回归（2026-08-22，QA 子代理二次回归）

基准：`docs/wxx-vopc-prd-v1.0.md`、`pm-checklist.md`、`refactor-notes.md`、`qa-report.md`、`audit-report.md`。

工作树改动（`git status`/`git diff --stat`，与预期一致）：
- `server/internal/handler/vopc_delivery.go`（+28）
- `server/internal/handler/vopc_risk.go`（+41/-若干）
- `server/internal/handler/vopc_close.go`（+7）
- `server/internal/handler/vopc_governance_test.go`（+198/-39）
- `refactor-notes.md`（+43）

### H-B1 里程碑门禁 TOCTOU 绕过 — PASS

- **证据**：`ReviewMilestone`（`vopc_delivery.go`）`next=="passed"` 分支、在 `stageStatuses[target]` 推进前新增 `milestoneAdvanceAllowed(tx, id)` 复核（`vopc_delivery.go:665-672`），与 `SubmitMilestone`（`:506`）为**同一函数同源**，非复制实现。
- **时序缺口封死**：提交后、评审前若登记未批 R3 风险或项目升档，reviewer pass 会被 409 拒绝且 stage 不变。
- **测试**：`TestVOPCMilestoneGateTOCTOU` 新增——R0 项目提交 S2 → 提交后治理角色登记 R3 → reviewer pass 断言 409 且 project.stage 仍 S1。`go test -run TestVOPCMilestoneGateTOCTOU -count=1 -v` **PASS**。
- **原子性**：门禁拦截时 `defer tx.Rollback()` 全量回滚（review insert 与 submission 状态更新均不落库），submission 保持 pending；无部分提交。

### H-B2 freeze 统一拦截写入 — PASS（并澄清申诉豁免）

- **证据**：新增 `projectBlockedForWrite(tx, id)`（`vopc_delivery.go:762`），命中 `blockedStatuses ∪ completedLike`。接入：
  - `CreateArtifact`（`vopc_delivery.go:306`）→ 409 且不落库不写事件；
  - `CreateArtifactVersion`（`vopc_delivery.go:400`）→ 409；
  - `CreateRisk`（`vopc_risk.go:103`）→ 409；
  - `CloseProject`（`vopc_close.go:120-124`）新增 `status=="risk_frozen"` 前置拦截，close/pause/pivot/terminate/archive 均 409（此前 `pivot`/`pause`/`terminate` 可从 risk_frozen 通过 closeTransition）。
- **申诉豁免**：`CreateRiskAppeal` **有意不加拦截**（走 `manageableProject`），冻结→申诉→裁定→解冻闭环成立。冻结后主理人仍可 201 申诉。
- **测试**：`TestVOPCFreezeBlocksBusinessWrites` 新增——冻结后 CreateArtifact/CreateArtifactVersion/CreateRisk 各 409 且断言不落库（frozenArtifacts/frozenRisks=0），pivot/terminate 各 409，appeal 仍 201。`-run TestVOPCFreezeBlocksBusinessWrites -count=1 -v` **PASS**。

### M-B1 R3 角色可达 — PASS

- **证据**：独立 `risk_governance` project-role **已全仓移除**——`rg risk_governance` 仅命中注释/文件名（`102_vopc_risk_governance.sql`）/文档，无任何 `.go`/`.sql` 的 `project_role='risk_governance'` 判定或授予路径；测试 helper `addRiskGovernance`（db.Exec 直插自证）已删除。
- **判定改写**：`isSpecialRiskGovernance` 为「项目内 `platform_operator` 成员（与 R2 同源）∧ 治理系统角色 `college_admin`/`school_admin`/`sys_admin`」（`vopc_risk.go:20-58`）。三个治理角色均由 `001_init.sql` role CHECK 与 `007_seed_users.sql` 播种，**生产可达**。
- **测试驱动**：`TestVOPCR3SpecialGovernanceChannel` 重写为真实可授角色——治理系统角色+platform_operator 建 R3→201、student 挂 platform_operator 建 R3→403、owner→403、双人专项审批放行、任一 reject 封死。`addCollegeAdmin` 播种的是真实系统角色（非 self-proving 直插 R3 角色）。`-run ... -count=1 -v` **PASS**。
- **R2/R3 差异保留**：R2=`isRiskManager`（任意 platform_operator），R3=`isSpecialRiskGovernance`（platform_operator+治理系统角色），R3 门禁更严（挂未批 R3 风险即拦），未退化。

### 不回归核查 — PASS

既有结项状态机、风险治理、学院准入、S0、成果版本门禁、任务/决策/邀请、正式里程碑、迁移 097-102 方言兼容全部仍绿（见下方命令表）。

### 命令与 exit code（本次实跑，server/ 下）

| 门禁 | 结果 | exit code |
|---|---|---|
| go test ./... -count=1 | PASS | 0 |
| go vet ./... | PASS | 0 |
| go build ./... | PASS | 0 |
| go test ./internal/db -run TestToMySQLVOPCMigrations -count=1 | PASS | 0（097-102 六方言全 PASS）|
| go test ./pkg/app -run 'Test(KeyRoutesReachable\|RouteRegistrationCount\|RunMigrationsCreatesSchema\|RunMigrationsIdempotent)' -count=1 | PASS | 0 |
| git diff --check | PASS | 0 |
| go test ./internal/handler -run 'TestVOPCMilestoneGateTOCTOU\|TestVOPCFreezeBlocksBusinessWrites\|TestVOPCR3SpecialGovernanceChannel' -count=1 -v | PASS | 0（3/3 用例执行且 PASS）|

### 残留阻断项（未在本批消除，记录不修）

- **无测试占位/伪成功残留**：`rg createProjectAt|addRiskGovernance|bytesBuffer|t.Skip(|TODO|FIXME` 在 `internal/handler` 无命中（`bytes.Buffer` 仅存在于既有 `request` helper 与增量里合法使用；`w2` 命中均为 chat/auth/kb 无关测试的合法 `httptest.ResponseRecorder`）。`addRiskGovernance` 已彻底删除。
- **事务审计原子**：本轮新增拦截均在既有 `defer tx.Rollback()` 事务内，未新增任何 bypass；`projectBlockedForWrite` 只读 status，不写库。
- **未改历史迁移**：097-102 migration 文件无 diff；本轮无新增迁移。
- 仍维持 NO-GO 的、非本批范围 P0 阻断不变（见上一轮第五节）：AI 真实执行、私有文件/字段级隔离、rubric/条件通过/甲方与发布审批、真实 MySQL/Turso 同构、前端三层校验等，本轮未触碰。

### 最终判定

**GO（针对本批三缺口修复范围）**：H-B1 / H-B2 / M-B1 三项修复均 PASS、有真实负向测试、无占位/伪成功、未改历史迁移、不回归；全部门禁 exit code 0。

**整体项目维持 NO-GO**（非本批功能缺陷）：上一轮及更早已明确的 P0 阻断未消除。

本轮仅追加 qa-report.md 本章节；未改源码/测试/迁移，未部署、未 commit、未 push。
---

## 回归轮次：私有文件闭环（迁移 103 + vopc_files + 接线 + file 门禁）

- 测试日期：2026-08-22
- 审查对象：已暂存工作树中的 vOPC 私有文件闭环改动（`.gitignore`、迁移 103、`vopc_files.go` + 测试、`vopc_delivery.go` file 门禁、routes/config/app 接线、迁移/路由/回归测试接线）
- 复核基准：`docs/wxx-vopc-prd-v1.0.md`、`pm-checklist.md`、`refactor-notes.md`、`qa-report.md`、`audit-report.md`

### 上传端点 POST /projects/:id/files 核查 — PASS

- `vopc.project.manage` capability（`auth.RequireCapability`）+ `CollegeAccess` 学院准入（外院/非 cs → 403）+ 项目写权限（`projectPolicy(..., "manage")`，非成员 → 404）三层门禁齐备。
- 无权限 403/404、未登录 401：`TestVOPCFileUploadPermissionMatrix` 覆盖非成员 404、外院 403、未登录 401、owner 201。
- 20MB 上限 → 413（`vopcMaxFileBytes=20<<20`，`fh.Size > vopcMaxFileBytes` 判 413）；`TestVOPCFileUploadRejectsSizeTypeAndInjection` 覆盖超限 413。
- MIME 白名单不匹配 → 415（`vopcAllowedMimeTypes`；危险 `application/x-msdownload` 覆盖 415）；声明空/octet-stream 时按扩展名兜底判断。
- 事务+审计原子：`tx.Begin()` 后校验权限 → 落盘成功 → `INSERT vopc_files` → `writeEvent("file.uploaded")` → `tx.Commit()`；任一步失败均 `defer tx.Rollback()` 且 `os.Remove(diskPath)`，保证失败不入库不落盘。
- 返回不可猜 `object_key`（`crypto/rand` 32 字节 hex，64-hex），不含真实磁盘路径；`c.JSON` 仅返回 meta（object_key/file_name/mime_type/size/checksum/storage_status）。
- 文件名净化：`safeFileName` 取 `filepath.Base`、去 `\`→`/`、拒绝 `.`/`..`/空/CRLF/NUL 与超长；注入文件名 `../../etc/passwd` 仍返回净化后的 `file_name` 且 object_key 合规（测试断言 key 合法且 fileName 无 `/`/`..`）。

### 下载端点 GET /projects/:id/files/:key 核查 — PASS

- 成员/owner 可下 200、非成员 404、越权（外院）403、未授权 404：`TestVOPCFileDownloadAuthorization` 覆盖 owner 200、member 200（body 含真实字节）、非成员 404、路径注入 `..%2F..` 404。
- key 严格 64-hex（`validObjectKey` 仅小写 `0-9a-f` 且长度 64），路径注入直接 404，杜绝路径穿越。
- 服务端流式（`io.Copy(c.Writer, f)`）+ `X-Content-Type-Options: nosniff` + `X-Checksum-Sha256` + `Content-Disposition` 附件（RFC 5987 `filename*`，`urlPathEscape` percent-encoding）。
- 不暴露真实路径：磁盘路径由 `resolveUploadDir` + object_key 拼接，仅内部使用；磁盘缺失时返回 404（避免状态泄露）。
- 学院准入复检（defense-in-depth）：`DownloadFile` 内部再校验 `OwnerScope=="college"`、`OwnerID==collegeID`、非 guest、`active`（与路由层 `CollegeAccess` 双保险）。

### 受控文件与里程碑门禁核查 — PASS

- `artifact_types` 增加 `file`；`milestoneArtifactTypes` 的 S2/S3/S5/S9 文档型阶段允许 `file`（与 `document` 等价），S4/S6/S7/S8 保持原状。
- `CreateArtifactVersion` 当 `source_kind=="storage_ref"` 时：校验 `source_ref` 为合法 64-hex object_key，且 `vopc_files` 中存在对应的本项目记录且 `storage_status != "scan_failed"`，否则 422。
- `TestVOPCFileMilestoneGateIntegration` 覆盖：合法 storage_ref 版本 201；非法 key（`not-a-key`）422；格式合法但不存在（`aaa...a`）422；受控文件作里程碑提交（S2）201。

### 不回归核查 — PASS

既有学院准入、S0、成果版本门禁、结项状态机、风险治理/R3、任务/决策/邀请、正式里程碑、迁移 097-102 方言兼容全部仍绿（`go test ./...` 全量 PASS，见下命令表）。本批仅新增 `vopc_files` 表与两条路由，未改历史迁移、未改既有 artifact/version 门禁列定义。

### 无占位/伪成功/路径泄露/越权绕过核查 — PASS

- `.uploads/` 已加入 `.gitignore`（第 17 行，`git check-ignore` 命中），无 `.uploads/` 下文件被暂存；仓库 `.uploads/` 目录为空（测试使用 `t.TempDir()` 隔离，未污染仓库）。
- `.gitignore` 已规范化为 LF（0 处 CRLF），`git diff --cached --check` 与 `git diff --check` 均 0 退出。
- 无硬编码可猜 URL / 磁盘绝对路径对外返回；object_key 由加密安全随机生成，不可猜。

### 命令与 exit code（本次实跑，server/ 下）

| 门禁 | 结果 | exit code |
|---|---|---|
| go test ./... -count=1 | PASS | 0（全量包绿）|
| go vet ./... | PASS | 0 |
| go build ./... | PASS | 0 |
| go test ./internal/db -run TestToMySQLVOPCMigrations -count=1 | PASS | 0（097-103 七方言全 PASS）|
| go test ./pkg/app -run 'Test(KeyRoutesReachable\|RouteRegistrationCount\|RunMigrationsCreatesSchema\|RunMigrationsIdempotent)' -count=1 | PASS | 0（4/4 用例 PASS）|
| git diff --cached --check | PASS | 0 |
| git diff --check | PASS | 0 |

### 迁移 103 SQLite/MySQL 兼容性 — PASS（翻译层验证）

- 迁移 103 仅新增一张表 + 三条普通索引；`INTEGER PRIMARY KEY AUTOINCREMENT` 经 `ToMySQL` 译为 `BIGINT PRIMARY KEY AUTO_INCREMENT`，`TEXT ... DEFAULT CURRENT_TIMESTAMP` 译为 `DATETIME ... DEFAULT CURRENT_TIMESTAMP`；`TestToMySQLVOPCMigrations/103_vopc_private_files.sql` PASS，无 `AUTOINCREMENT`/`ON CONFLICT`/`INDEX IF NOT EXISTS` 等 SQLite-only 残留。
- 未改历史迁移：097-102 migration 文件无 diff。

### [blocked] 边界（本批不做伪造，记录不修）

- **云对象存储** [blocked]：`storage_status` 默认 `pending`，落盘为本地受控目录，未接真实云对象存储 SDK / 签名 URL。
- **病毒扫描** [blocked]：无真实病毒扫描；`scan_failed` 状态仅存在于白名单与门禁判定中，当前无写入该状态的链路。
- **真实 MySQL 同构** [blocked]：MySQL 兼容性仅经 `ToMySQL` 翻译器静态验证，未在真实 MySQL 实例上执行 103 迁移。

### 残留阻断项（仅记录，本批未消除）

- 无新增阻塞缺陷：本轮五大门禁全绿，上传/下载/门禁/不回归均 PASS；未发现占位、伪成功、路径泄露或越权绕过。
- 上述 [blocked] 三项为外部能力边界，非本轮可消除，保持记录。

### 最终判定

**GO（针对本批私有文件闭环范围）**：上传/下载鉴权、file 门禁、里程碑 storage_ref 门禁、迁移 103 方言兼容、不回归全部 PASS，有真实正/负向测试，无占位/伪成功，未改历史迁移。

**整体项目维持 NO-GO（历史 P0 阻断，非本批功能缺陷）**：此前已明确的 P0 阻断（真实 AI 执行、私有文件/字段级隔离、rubric/条件通过与审批、真实 MySQL/Turso 同构、前端三层校验等）仍未消除，本轮未触碰。

本轮仅追加 qa-report.md 本章节；未改源码/测试/迁移，未部署、未 commit、未 push。
