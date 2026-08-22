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
