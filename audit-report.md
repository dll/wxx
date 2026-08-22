# vOPC 安全门禁增量独立只读复审报告

- 复审日期：2026-08-22
- 复审对象：项目根目录最新未提交工作树，重点覆盖上一轮 `audit-report.md` 之后的迁移 100、协作/成果/里程碑安全门禁、测试及前端准入增量
- 前置材料：已读取 `pm-checklist.md`、`refactor-notes.md`、`qa-report.md`、旧 `audit-report.md`、`refactor-final-summary.md`
- 约束：只读核查；除本报告外未修改源码/测试；未部署、未 commit、未 push

## 1. 最终判定

# **NO-GO**

本轮增量本身方向正确，且旧审计中以下三项已获得足够代码与自动化证据，可以按其原始范围关闭：

1. **旧 H-1：普通邀请授予 `platform_operator` 的项目内提权——已关闭。**
2. **同一占位成果版本贯穿 S2-S9——已关闭（结构化元数据门禁范围内）。**
3. **旧 H-3：本次点名的邀请计数/CAS、artifact 更新时间、submission CAS、项目及里程碑推进关键 SQL 忽略 error/RowsAffected——已关闭。**

但这些关闭不等于 PRD P0 已完成。当前仍无 AI 真实执行与项目上下文隔离、风险审批/冻结/申诉、私有文件受控上传下载、独立结项复盘与异常状态机；里程碑也仍缺 rubric、条件通过、豁免及甲方/试点/发布审批实体。SHA-256 仅验证格式，`source_ref` 仍只是客户端声明，不能证明外部 commit/对象真实存在。因此整体继续 **NO-GO**。

## 2. 指定安全增量逐项复审

| 增量项 | Risk Level | 证据行号 | 结论 |
|---|---|---|---|
| 迁移 `100_vopc_artifact_version_gates.sql` | **Medium** | `server/migrations/100_vopc_artifact_version_gates.sql:1-5`；`server/internal/handler/vopc_handler_test.go:39-47`；`server/internal/db/migration_vopc_test.go:12-40` | **通过（有边界）**。新增 `status`、`intended_stage` 和组合索引；历史行默认 `invalid`/空阶段，未把旧版本自动升级为可信证据；Handler 测试实际执行迁移 100，方言测试纳入 100。真实 MySQL/Turso 升级仍未实跑。 |
| 普通邀请角色移除 `platform_operator` | **High** | `server/internal/handler/vopc_delivery.go:17-21,73-77`；`server/internal/handler/vopc_handler_test.go:462-466` | **通过，旧 H-1 关闭**。普通邀请白名单仅余 co_owner/member/mentor/reviewer；owner 邀请 platform_operator 自动化断言 422。平台运营仍可按既有受治理数据成为成员并执行平台评审，但不能由普通项目邀请授予。 |
| 接受邀请二次学院复核 | **High** | `vopc_delivery.go:197-215`；邀请 CAS/成员插入/事件同事务 `:216-243`；`vopc_handler_test.go:544-588` | **通过**。accept 时重查 active、非 guest、college scope、学院 ID；外院/guest/inactive 三类身份变化均返回 403，邀请保持 pending、成员与事件均为 0。注意 `:210` 将数据库查询错误和资格不符合并成 403，属于 fail-closed 可观测性问题，不构成放行。 |
| `CreateArtifactVersion` 的 SHA-256/阶段/来源门禁 | **High** | `vopc_delivery.go:21-27,359-389,396-428,741-750` | **通过（结构化元数据范围）**。来源类型受控、引用非空/≤2000/禁换行、checksum 规范化后必须 64 位小写十六进制、阶段仅 S2-S9；新版本显式 active 并绑定 intended_stage；artifact 派生更新时间检查 error 和精确 RowsAffected。**未验证外部 commit/对象真实存在**。 |
| `SubmitMilestone` 版本去重/数量/同项目/active/阶段/成果类型 | **Critical** | `vopc_delivery.go:468-557`；正式流程 `vopc_handler_test.go:281-355`；负向 `:495-516` | **通过（基础硬门禁）**。限制 1-20，拒绝非正数/重复；查询绑定 version+artifact.project_id；要求 active、合法 SHA-256、intended_stage 等于目标阶段；至少一个版本命中阶段成果类型。逐阶段流程为 S2-S9 创建独立版本，不再复用占位版本。测试已覆盖空、重复、跨阶段、失效；源码覆盖跨项目和成果类型，当前新增测试未单独断言错误成果类型、21 个版本及跨项目版本，属于测试覆盖缺口。 |
| `ReviewMilestone` RowsAffected 与项目推进一致性 | **Critical** | `vopc_delivery.go:603-693`，尤其 submission CAS `:639-654`、项目 CAS `:655-674`、milestone 更新 `:675-683`、事件/commit `:685-693` | **通过**。review、submission CAS、项目阶段 CAS、当前 milestone 状态、审计事件均在同一事务；各关键更新检查 SQL error 和 RowsAffected，任一步失败由 defer rollback，不会返回虚假成功。普通成员越权 403 见测试 `:526-531`。尚缺并发双评审/故障注入自动化，但实现已封住旧报告指出的忽略错误路径。 |
| 正式 S0→S9 逐阶段里程碑流程 | **High** | `vopc_handler_test.go:281-355` | **通过**。S0 独立 submit；旧 `/advance` 断言 404；S2-S9 每阶段创建对应成果类型、独立 checksum/阶段版本、提交给指定 reviewer 并 pass，最终 S9/completed。该测试证明当前结构化门禁可运行，不证明外部成果真实性或结项复盘完成。 |
| 越权、原子性和负向测试 | **High** | `vopc_handler_test.go:435-542,544-588` | **通过但非穷尽**。覆盖外院邀请、platform_operator 提权、空/重复/跨阶段/失效版本、普通成员评审越权、邀请身份变化后的 invitation/member/event 原子一致性。未新增并发 accept/review、错误成果类型、数量上限、跨项目版本及 SQL 故障注入专测。 |
| 前端 `vopc_access.dart` 准入 | **Medium** | `frontend/lib/utils/vopc_access.dart:4-32`；`frontend/lib/utils/storage.dart:46-68,87-93,119-129`；`frontend/lib/providers/auth_provider.dart:147-170`；`frontend/test/vopc_access_test.dart:6-39` | **通过（客户端体验门禁）**。要求登录、非 guest、active、college scope、学院匹配、`vopc.read`；缓存缺失 fail-closed，身份与能力在 profile 拉取后持久化并触发 router refresh。学院 ID仍在客户端默认硬编码 `cs`，后端可配置，部署改值时可能误拒绝合法用户。 |
| `router.dart` 导航与 deep-link 准入 | **Medium** | `frontend/lib/config/router.dart:172-202,273-283,738-762,780-843`；后端最终边界 `server/pkg/app/routes.go:127-155`、`vopc_handler.go:49-61` | **通过（安全最终边界由后端保证）**。所有 `/vopc` 路由前缀被 redirect；桌面/移动导航仅在 VopcAccess.allowed 时显示；后端正式 group 继续挂 CollegeAccess。现有前端测试只测纯函数，未 pump 真实 GoRouter/deep-link/退出刷新生命周期。 |

## 3. 旧审计发现关闭确认

### 3.1 旧 H-1 / `platform_operator` 提权

**状态：CLOSED。**

- 原因已消除：`projectRoles` 不再包含 `platform_operator`（`vopc_delivery.go:17-20`）。
- HTTP 负向回归：owner 尝试邀请该角色必须 422（`vopc_handler_test.go:465-466`）。
- 未发现其他普通邀请路径可写入该角色。

### 3.2 同一占位版本贯穿 S2-S9

**状态：CLOSED（仅指旧绕过方式）。**

- 版本创建必须绑定唯一 intended_stage、合法 SHA-256 和 active 状态（`vopc_delivery.go:381-401`）。
- 提交事务内要求 intended_stage 精确等于目标阶段，并检查成果类型（`:530-556`）。
- 正式流程测试改为逐阶段新建对应类型成果和独立版本（`vopc_handler_test.go:308-347`）。

**保留边界：**测试中的 `repo:commit:stage-N` 和格式正确 checksum 仍是客户端声明；系统没有访问代码仓库或对象存储验证内容真实性。因此“同一占位 ID 贯穿”已关闭，“伪造元数据代表真实交付物”尚未关闭，继续归入私有文件/外部对象真实性 P0。

### 3.3 关键 SQL 忽略 error/RowsAffected

**状态：CLOSED（旧报告点名路径）。**

- 邀请成员/待处理邀请计数检查 Query error：`vopc_delivery.go:113-125`。
- invitation CAS 检查 error 与 RowsAffected：`:216-227`。
- artifact 更新时间检查 error 与 RowsAffected：`:411-419`。
- reviewer 查询异常不再静默：`:619-626`。
- submission CAS、project CAS、milestone update 全部检查 error/RowsAffected：`:643-683`。
- 上述更新与 review/event 在单事务提交：`:603-693`。

未发现这些路径继续存在 `_ =` 或忽略 RowsAffected 后返回成功的情况。并发/故障注入测试仍值得补齐，但不阻止关闭原具体发现。

## 4. 新增/残余风险

### Critical

#### C-1 外部成果真实性与私有文件安全仍不存在（P0，未通过）

- `CreateArtifactVersion` 只验证 `source_ref` 字符串形态与 checksum 格式（`vopc_delivery.go:381-389`），不会读取仓库 commit、对象 key 或文件内容并重算 SHA-256。
- 迁移 100 仅加状态/阶段（`100_vopc_artifact_version_gates.sql:1-5`），没有受控对象 key、上传状态、扫描状态、大小/MIME、归档/删除状态。
- 影响：攻击者仍可提交虚构引用及任意格式正确的 checksum；当前硬门禁不能作为“成果确实存在”的证据。

#### C-2 AI、风险治理、结项复盘仍缺 P0 闭环（未通过）

- AI task/output/review/context/usage/cost 与真实调用、四种人工审阅、项目上下文隔离仍不存在。
- R1/R2 审批、R3 专项流程、冻结/解冻/申诉及统一风险 gate 仍不存在。
- S9 pass 仍直接形成 `completed`（正式流程测试 `vopc_handler_test.go:348-354`），没有独立 close/retrospective、pause/pivot/terminate/archive 闭环。

### High

#### H-1 完整里程碑业务门禁仍不足（P0，未通过）

当前阶段→成果类型映射（`vopc_delivery.go:23-27`）关闭了通用版本乱用，但仍缺 rubric/逐项评分、conditional pass、waiver、甲方 S2/S5/S6 确认实体、试点与上线审批实体。任意被指定 reviewer 仍可用非空 note 直接 pass。基础元数据门禁通过，不应扩张为 PRD 完整阶段验收通过。

### Medium

#### M-1 测试覆盖尚未覆盖全部新增分支

新增负向测试没有逐项断言：错误成果类型、21 个版本、跨项目版本、非法 checksum/source/stage、并发双 accept/review、关键 SQL 故障注入。实现源码已覆盖前五类输入分支中的大部分，但并发与故障回滚仍缺强自动化证据。

#### M-2 前端准入配置与生命周期证据不足

- 客户端学院 ID 默认硬编码 `cs`（`vopc_access.dart:15-21`），后端取可配置 `VOPCCollegeID`（`routes.go:134`），部署改值可能造成前后端误拒绝不一致。
- `vopc_access_test.dart` 仅测 evaluate 纯函数，没有真实 GoRouter/widget、deep link、profile/capability 延迟加载、退出及身份变化刷新测试。
- 这不会绕过后端 CollegeAccess，但可能产生入口错误显示/误拒绝。

#### M-3 生产同构数据库验证不足

迁移 100 已通过 SQLite 执行和 SQLite→MySQL 静态转换测试，但没有真实 MySQL/Turso 从 099 升级、幂等、锁/事务、备份恢复验证。

## 5. 独立只读验证

| 命令 | 结果 |
|---|---|
| `go test ./internal/handler -run "Test.*VOPC|Test.*Milestone|Test.*Invitation" -count=1` | **PASS**（20.398s） |
| `go test ./internal/db -run TestToMySQLVOPCMigrations -count=1` | **PASS** |
| `go test ./pkg/app -run "Test(KeyRoutesReachable|RouteRegistrationCount|RunMigrationsCreatesSchema|RunMigrationsIdempotent)" -count=1` | **PASS**（78.536s） |
| `flutter test test/vopc_access_test.dart test/vopc_provider_test.dart` | **PASS：8 tests** |
| `flutter analyze lib/utils/vopc_access.dart lib/config/router.dart lib/utils/storage.dart lib/providers/auth_provider.dart test/vopc_access_test.dart test/vopc_provider_test.dart` | **PASS：No issues found** |
| `git diff --check` | **PASS** |

以上测试证明当前代码和断言成立；不替代真实 MySQL/Turso、对象存储/仓库真实性、模型调用、并发故障注入和多端真机验收。

## 6. 仍阻断上线的 P0

1. **AI 真实闭环缺失**：无真实模型执行、项目上下文隔离、输出版本、人工接受/修改/退回/否决、usage/cost、额度、超时重试。
2. **风险治理缺失**：R1/R2 审批、R3 专项流程、冻结/解冻/申诉及统一写/AI/试点/发布/外发 gate 未落地。
3. **私有文件和外部成果真实性缺失**：无受控上传、对象 key、扫描、服务端 checksum 重算、鉴权下载、短时签名及跨项目下载负向测试。
4. **结项与异常状态机缺失**：S9 pass 直接 completed；无独立 close/retro 及 continue/pivot/pause/terminate/archive。
5. **完整里程碑业务审批不足**：缺 rubric、conditional pass、waiver、甲方确认、试点/上线审批实体；结构化版本门禁只是基础层。
6. **生产同构与多端验收不足**：真实 MySQL/Turso 升级/并发/备份恢复、WebView/真机/小屏、P95 未完成；全量 Flutter analyze 既有非零门禁仍未治理。

## 7. 签署

本轮可正式关闭：

- 旧 H-1 普通邀请授予 `platform_operator`；
- 同一占位版本 ID 无差别贯穿 S2-S9；
- 旧审计点名的关键 SQL error/RowsAffected 静默忽略路径。

本轮指定增量总体实现质量合格，基础安全门禁通过；但产品 P0 仍有多个完整业务域缺失，且成果真实性与完整阶段审批不能由格式化 SHA-256 和 `source_ref` 代替。

# **最终结论：NO-GO；禁止上线、部署或宣称 vOPC PRD v1.0 P0 验收完成。**
