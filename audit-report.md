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

---

# 附录 B：vOPC 结项/异常状态机 + 风险治理最小闭环 + R3 专项通道 独立只读复审（2026-08-22 第二批）

- 复审对象：本批新增 `vopc_close.go` / `vopc_risk.go` / `vopc_governance_test.go` / 迁移 101/102，及改动 `vopc_handler.go` / `vopc_delivery.go` / `vopc_decisions.go` / `routes.go` / `vopc_handler_test.go` / `migration_vopc_test.go`。
- 前置材料：已读 `pm-checklist.md`、`refactor-notes.md`、`qa-report.md`（含第三/四轮）、旧 `audit-report.md`、`refactor-final-summary.md`。
- 约束：只读；未修改源码/测试/迁移；未部署、未 commit、未 push。
- 独立验证：`go build ./...` PASS；`go vet ./internal/handler/` PASS；定向 6 个治理用例 `go test` PASS（10.4s）。

## B.1 最终判定

# **NO-GO（本批功能方向正确、质量合格，但与上一轮 P0 缺失项叠加，整体仍不满足上线）**

本批实现了 S9→closeable 分离、close/pause/resume/pivot/terminate/archive 状态机、风险四实体表、R2 双人审批、R3 独立专项通道、里程碑里程碑门禁，方向正确，核心安全语义基本成立，自动化测试为真实断言（非占位）。但存在 **1 个 Medium（R3 专项角色无 provisioning API，通道不可达生产）** 与 **2 个 High 语义缺口（里程碑门禁 TOCTOU 绕过；freeze 未统一拦截全部写操作）**，且不改变上一轮已确认的 AI/私有文件/完整里程碑门禁/生产同构等 P0 阻断。

## B.2 逐项复审结论（含风险等级 + 证据行号）

### B.2.1 结项/异常状态机（vopc_close.go、迁移 101、vopc_handler.go）

| 检查点 | Risk | 证据 | 结论 |
|---|---|---|---|
| S9 pass→closeable（真正分离结项） | Low | `vopc_handler.go:22`（stageStatuses[9]="closeable"）、`vopc_delivery.go:659`（nextStatus=stageStatuses[target]） | **通过**。S9 不再直接 completed，需 `CloseProject` 才落 completed。 |
| close/retro 必填（human_decision+outcome_package） | Low | `vopc_close.go:43-60`（normalizeAndValidate） | **通过**。close 缺两项 422，terminate 缺 failure_evidence 422，pivot 缺 human_decision 422。 |
| 非法流转绕过（close/pause/resume/pivot/terminate/archive） | Medium | `vopc_close.go:196-234`（closeTransition） | **通过（有边界）**。draft 不能 close；terminated/archived 不能 close/pivot；archive 仅 completed/terminated；resume 仅 paused。**边界**：`pause` 未拒绝 `risk_frozen` 状态（见 B.2.3），且 `CloseProject` 未检查 `blockedStatuses`，故冻结项目仍可被 owner 直接 pivot/terminate/pause，构成治理状态与调度状态的交叉但非冻结绕过（resume 会恢复 risk_frozen）。 |
| 权限 fail-closed（非管理 404/403） | Medium | `vopc_close.go:108-115`（projectPolicy "manage"） | **通过**。非 owner/co_owner/platform_operator 返回 404，不泄露项目存在性。 |
| pivot 里程碑重置一致性 | Medium | `vopc_close.go:145-150`（UPDATE vopc_milestones stage<>'S0'→pending） | **通过**。测试断言 resets=9。 |
| resume 恢复 previous_status | Medium | `vopc_close.go:127-138` | **通过**。读 pause 记录 previous_status，历史丢失回退 pending_review。 |
| 状态 CAS（RowsAffected==1 再推进） | Medium | `vopc_close.go:141-144,154-158,168-172` | **通过**。所有状态 UPDATE 检查 RowsAffected，失败 409。 |
| 审计同事务 | High | `vopc_close.go:174-183`（INSERT close_records + writeEvent + Commit，defer Rollback） | **通过**。committed 标志 + 单事务。 |
| 迁移 101 兼容/幂等/不改历史 | Medium | `101_vopc_close_state_machine.sql`；`git status` 仅 101/102 未跟踪，097-100 无 diff；`migration_vopc_test.go:13` 纳入 101 | **通过（静态转换，无真 MySQL）**。IF NOT EXISTS 幂等；AUTOINCREMENT 经 ToMySQL 转换测试。 |

### B.2.2 风险治理最小闭环（vopc_risk.go、迁移 102）

| 检查点 | Risk | 证据 | 结论 |
|---|---|---|---|
| R2 双人审批 = 两名**不同**审批人 | High | `vopc_risk.go:249`（COUNT(DISTINCT approver_user_id)>=2） | **通过**。真正 distinct，非累计两次。 |
| 重复拒绝/重复审批拦截 | Medium | `vopc_risk.go:232-238`（dup>0→409）、`vopc_risk.go:240-241`（非 open→409） | **通过**。同一审批人重复（含 approve 后 reject）被 409。 |
| 未审批确实拦截里程碑 | High | `vopc_risk.go:505-548`（milestoneAdvanceAllowed）+ `vopc_delivery.go:506-509` | **通过**。R2 项目无 approved R2 风险 → SubmitMilestone 409。 |
| freeze/unfreeze 权限与状态一致 | High | `vopc_risk.go:283-351`（isRiskManager + freeze 强制理由 + 状态 CAS） | **通过（有边界）**。非 platform_operator 403；freeze 进 risk_frozen；重复 freeze/unfreeze 409；可冻结状态集排除 completed/terminated/archived。**边界见 B.2.3**：freeze 未覆盖全部写路径。 |
| appeal/resolve 权限一致 | High | `vopc_risk.go:353-501`（CreateRiskAppeal 走 manageableProject；Resolve 走 isRiskManager + status=pending CAS） | **通过**。申诉由主理人发起、治理角色裁定；非治理 resolve 403；重复裁定 409。 |
| 审批/冻结/申诉审计同事务 | High | `vopc_risk.go:260-270,344-350,481-499` | **通过**。ApproveRisk/FreezeProject/ResolveRiskAppeal 均 committed 标志 + defer Rollback + writeEvent。 |

### B.2.3 新发现语义缺口（本批复审新增）

#### H-B1 里程碑门禁 TOCTOU 绕过（High，需整改）

`milestoneAdvanceAllowed` **仅在 `SubmitMilestone` 内调用一次**（`vopc_delivery.go:506`），`ReviewMilestone`（`vopc_delivery.go:597-699`）在 pass 前**不再复检** R2/R3 门禁。时序漏洞：

1. R0/R1 项目（无未处置 R3 风险）提交 S2 里程碑 → 门禁通过，submission 进入 pending；
2. 此时 `risk_governance` 登记一条 R3 风险（r3Outstanding>0），或项目被 promote 为 R2/R3；
3. 指定 reviewer（或 platform_operator）直接 pass 该 pending submission → 项目经 `stageStatuses[target]` 推进到下一阶段，**绕过 R3「禁止推进」/R2「未审批不得推进」门禁**。

证据：`vopc_delivery.go:597-699` 的 ReviewMilestone 全函数无 `milestoneAdvanceAllowed` 调用；`vopc_risk.go:505-548` 仅被 `vopc_delivery.go:506` 引用（`rg milestoneAdvanceAllowed` 仅 2 处命中）。

**整改建议**：在 `ReviewMilestone` 的 `next=="passed"` 分支、执行 `stageStatuses[target]` 推进前，复用 `milestoneAdvanceAllowed(tx, id)` 复核；或对 R2/R3 项目禁止在风险未结清时进入 review pass。

#### H-B2 freeze 未统一拦截全部写操作（High，需整改/澄清口径）

`blockedStatuses = {paused, risk_frozen, terminated, archived}` 仅被以下写路径引用：SubmitProject（`vopc_handler.go:667`）、CreateTask/UpdateTask（`:475/:553`）、SubmitMilestone（`vopc_delivery.go:502`）、CreateDecision/ActDecision（`vopc_decisions.go:127/:205`）。但 **`risk_frozen` 项目仍可**：

- `CreateArtifact`（`vopc_delivery.go:296-324`，无 status 检查）；
- `CreateArtifactVersion`（`vopc_delivery.go:386-436`，无 status 检查）；
- `CreateRisk` / `CreateRiskAppeal`（`vopc_risk.go`），`ApproveRisk` / `FreezeProject`（治理动作，可接受）；
- `CloseProject`（`vopc_close.go`）。

QA 报告与 refactor-notes 声称「冻结后**所有写操作**被 blockedStatuses 拦截」与代码不符。冻结至少未拦截成果/版本/风险登记，也未拦项目结构性流转（pivot/terminate）。

**整改建议**：明确「冻结」的语义边界（是仅拦里程碑/AI/项目推进，还是拦全部业务写），并将其落地到所有业务写 handler，或显式在文档中收缩为「治理冻结仅拦截项目推进与任务/决策写」，避免自称全量拦截。

#### M-B1 R3 专项角色无 provisioning API（Medium，需澄清）

`risk_governance` 与 `platform_operator` 均不在邀请白名单 `projectRoles`（`vopc_delivery.go:17`），且全仓**无任何 API/迁移种子**可授予 `risk_governance`（`rg risk_governance` 仅命中 `vopc_risk.go` 判定函数与测试 helper `addRiskGovernance` 的直接 `db.Exec`）。这意味着：

- 生产环境中没有任何 in-system 流程能把某用户设为 `risk_governance`，R3 专项通道**实际不可达**，只能通过越权直接写库（测试的 `db.Exec`）触发；
- 测试自证无法证明「专项审批」是一条可用的产品流程，仅证明代码分支在手工造数下的行为。

**整改建议**：落地平台治理侧 provisioning 实体（如 admin 授予 risk_governance 的 API/迁移/种子），或将专项审批与现有 `college_admin` 系统角色显式绑定，使通道可达、可验收。否则 R3 专项通道应判定为「代码存在但产品流程缺失」。

#### Low（记录）

- `CreateRisk`/`CreateRiskAppeal` 用 `res.LastInsertId()` 丢弃 error（`vopc_risk.go:118/:413`），`CreateArtifact`/`CreateArtifactVersion` 同（`vopc_delivery.go:315/:413`）。LastInsertId 失败概率极低，但 auditor 侧应统一检查，与旧 H-3「忽略 error/RowsAffected」的收紧口径保持一致。
- `close/terminate/archive` 三个分支统一写 `closed_at=CURRENT_TIMESTAMP`（`vopc_close.go:154`），terminate/archive 语义上并非「结项」，`closed_at` 含义被污染；`completed_at` 仅在 close 写入（`:160`）。建议区分 `terminated_at`/`archived_at` 或改列名。
- 项目级 vs 风险级 `risk_level`「同等级匹配」约束（`vopc_risk.go:535-546`）无文档：项目 auto-promote 为 R3 但只登记 R2 风险时，即使 R2 已 approved，R3 门禁仍拦（语义自洽，需文档说明）。

### B.2.4 R3 专项审批通道专项核查（对应任务第 3 点）

| PRD 13.1 要求 | Risk | 证据 | 结论 |
|---|---|---|---|
| 普通 manager 创建 R3 是否 403 | High | `vopc_risk.go:77-103`（R3 走 readableProject + isSpecialRiskGovernance 二次校验） | **通过**。owner/co_owner 创建 R3 403；`TestVOPCR3SpecialGovernanceChannel:307-311` 断言 403。 |
| 专项审批是否双人 | High | `vopc_risk.go:220-260`（R3 走 isSpecialRiskGovernance + COUNT DISTINCT approve>=2） | **通过**。两名不同 risk_governance approve 才 approved。 |
| 任一 reject 是否封死推进 | High | `vopc_risk.go:240-246`（reject→rejected）+ `milestoneAdvanceAllowed:523`（status<>'approved' 的 R3 未决计数 >0 即拦） | **通过**。rejected 后 r3Outstanding 仍>0，里程碑继续 409。 |
| platform_operator 越权是否被拒 | High | `vopc_risk.go:224-230`（riskLevel=="R3" 走 isSpecialRiskGovernance，非 risk_governance 403） | **通过**。platform_operator 建/审批 R3 均 403；测试 `:318-323` 断言。 |
| 是否与 R2 实质差异而非照抄 | High | `vopc_risk.go:19-41`（isRiskManager vs isSpecialRiskGovernance 分离）+ `CreateRisk` R3 用 readableProject 而 R2 用 manageableProject + 门禁 `milestoneAdvanceAllowed` R3 分支更严（挂有未批 R3 风险即拦） | **通过**。权限角色、创建/审批分流、门禁严格度均与 R2 不同，非照抄。 |

**R3 通道综合结论**：代码层面的权限分流与双人专项审批语义正确、测试为真实断言；但受 M-B1（无 provisioning API）与 H-B1（TOCTOU）制约，专项通道尚不能视为产品可达、端到端安全闭环。

## B.3 事务与审计核查（任务第 4 点）

- `CloseProject`/`ApproveRisk`/`FreezeProject`/`ResolveRiskAppeal`：`committed` 标志 + `defer` 回滚 + `writeEvent`（内部 `execOne` 检查 RowsAffected==1）+ `tx.Commit` 后置 committed=true。**无伪成功**。
- `CreateRisk`/`CreateRiskAppeal`：`defer tx.Rollback()`（无 committed 标志，Commit 后回滚为无害 no-op）；但 `LastInsertId` error 被丢弃（Low）。
- 所有状态 UPDATE 均检查 RowsAffected，失败走 409/500，无静默成功。
- 迁移 101/102 均 `CREATE TABLE IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS`，`ALTER TABLE ... ADD COLUMN`（101），幂等；`migration_vopc_test.go` 将 101/102 纳入 SQLite→MySQL 方言静态转换。

## B.4 测试可信度核查（任务第 6 点）

- `vopc_governance_test.go` 无占位（`rg createProjectAt|func w2|bytesBuffer` 均 0 命中；`bytes.Buffer` 仅存在于既有 helper），6 个用例均为真实 HTTP 请求 + DB 状态断言（status/风险 status/里程碑 count），非伪断言。
- **局限**：治理角色成员全部经测试 helper `addPlatformOperator`/`addRiskGovernance` 用 `db.Exec` 直接插 `vopc_project_members`，绕过任何 provisioning API（因无此 API，见 M-B1）。故测试证明「分支代码正确」，但**不能**证明「专项审批在真实产品可被授予与执行」。QA 若将「R3 专项通道 PASS」理解为产品闭环完成，即被测试自证误导的风险——需在 B.2.3/M-B1 口径下修正 QA 结论。
- `TestVOPCCloseStateMachine`/`TestVOPCPivotResetsProject` 覆盖正/负向流转与 pivot 复位，断言精确到 code 与 status，可信度高。

## B.5 旧审计关闭项与本批不冲突确认（任务第 7 点）

- `platform_operator` 邀请提权封堵：本批未改 `projectRoles`（仍 `{co_owner,member,mentor,reviewer}`，`vopc_delivery.go:17`），`platform_operator`/`risk_governance` 仍不可自助授予。**不冲突**。
- 同一占位版本贯穿 S2-S9、关键 SQL error/RowsAffected：本批未触碰 097-100 迁移及版本门禁核心逻辑，仅 `SubmitMilestone` 追加 `milestoneAdvanceAllowed` 门禁（`:506-509`）。**不冲突**。
- 三项旧关闭继续成立。

## B.6 下一批最小可执行整改顺序

1. **H-B1（最高优先）**：`ReviewMilestone` pass 分支推进前补 `milestoneAdvanceAllowed` 复核，封死提交后登记风险的 TOCTOU 绕过。
2. **M-B1**：落 `risk_governance` provisioning（admin 授角色 API 或绑定 college_admin 系统角色），使 R3 专项通道产品可达；补充「经真实 API 授予专项角色后走通 R3 创建→双人审批→放行」的端到端测试。
3. **H-B2**：明确 freeze 语义边界并统一落地——要么 freeze 拦全部业务写、要么收缩文档口径为「治理冻结仅拦项目推进与任务/决策」，并补对应负向测试。
4. **Low 清理**：统一 `LastInsertId` error 处理；区分 `closed_at` vs `terminated_at`/`archived_at`。
5. 继续推进上一轮既定 P0（AI 闭环、私有文件、rubric/conditional/waiver/甲方/试点发布审批、真实 MySQL/Turso）。

## B.7 签署

本批新增的结项/异常状态机与风险治理/专项审批代码**方向正确、实现质量合格、测试为真实断言**，可继续迭代；但因 H-B1（TOCTOU）、H-B2（freeze 未全覆盖）、M-B1（专项角色不可达）与上一轮 P0 叠加，**不得上线、部署或宣称 vOPC PRD v1.0 P0 验收完成**。

# **本批结论：NO-GO（可继续开发，禁止上线）。**


---

# 附录 C：审计附录 B 三缺口（H-B1/H-B2/M-B1）修复的独立只读复审（2026-08-22 第二批复审）

- 复审方：reviewer-audit-wxx（独立复核，不采信 QA/refactor 的 PASS 结论）
- 复审对象：仅截至本复审时刻的未提交工作树，重点 4 个文件（vopc_delivery.go / vopc_risk.go / vopc_close.go / vopc_governance_test.go）
- 前置材料：pm-checklist、refactor-notes、qa-report（含附录 B 三缺口回归）、audit-report（附录 B）、refactor-final-summary
- 约束：只读；仅更新本文件追加本章；不部署、不 commit、不 push

## C.1 最终判定

# 三缺口修复范围：GO（逐项关闭，代码与测试均真实达标）

针对附录 B 点名 H-B1 / H-B2 / M-B1 三个缺口的最小增量修复，经独立核查属实：不是"只改测试没改代码"、不是"断言自证"、不是"TOCTOU 仍可绕"。三项均在其原始口径下关闭。

但需明确两条**未在本批消除的残留**（均非本批引入、为既有口径边界），且整体项目仍 **NO-GO**（AI 真实闭环、私有文件、完整里程碑 rubric/条件通过/豁免/甲方审批、真实 MySQL/Turso 同构、前端三层准入、flutter analyze 既有 lint 等上一轮 P0 未变）。故：

- **对本批三缺口：GO（关闭）**
- **对整体项目：NO-GO（维持，禁止上线）**

## C.2 逐项独立复核

### H-B1 里程碑门禁 TOCTOU（High）— CLOSED，真实同函数复核 + 全量回滚

| 检查点 | Risk | 证据（本复核实测行号） | 独立结论 |
|---|---|---|---|
| 是否真实同函数（非复制） | High | `rg milestoneAdvanceAllowed` 仅 3 处：定义 `vopc_risk.go:531`，调用 `vopc_delivery.go:514`（SubmitMilestone）与 `:670`（ReviewMilestone） | **通过**。两处调用同一 `milestoneAdvanceAllowed`，无第二个实现。 |
| 复核位置是否封死时序 | High | `vopc_delivery.go:668` `if next=="passed"` → `:670` 调门禁，位于 `:681` 项目阶段推进之前 | **通过**。提交后评审前登记 R3/升档 → pass 被拦。 |
| 409 是否全量回滚（review+submission+stage 均不变） | Critical | `:641` `tx, err := h.db.Begin()` + `:646` `defer tx.Rollback()`；门禁拦截于 `:673` `return`，在此之前 `:655` INSERT review、`:661` UPDATE submission→passed 均在同一 tx 内，由 defer 回滚 | **通过**。三步（review insert / submission 状态 / 项目 stage）同 tx，拦截即全量回滚，submission 保持 pending，项目 stage 不变。 |
| 是否与 SubmitMilestone 同源 | High | `:514` 与 `:670` 均 `milestoneAdvanceAllowed(tx, id)` | **通过**。同函数同签名。 |
| 无其它推进旁路 | High | `rg "SET stage="` 仅 3 处推进/复位：`vopc_delivery.go:681`（ReviewMilestone，已门禁）、`vopc_handler.go:681`（S0→S1 立项，非里程碑）、`vopc_close.go:140`（pivot 回 S0 复位，已加 risk_frozen 拦截）；`rg "advance"` 正式路由无里程碑 advance 端点 | **通过**。正向推进唯一通道已门禁。 |
| 测试可信度 | High | `TestVOPCMilestoneGateTOCTOU`（vopc_governance_test.go:406-461）：R0 提交 S2→提交后治理角色登记 R3→reviewer pass 断言 409 且 `stage` 仍 S1 | **通过**。`go test -run TestVOPCMilestoneGateTOCTOU -count=1 -v` 实测 PASS（0.06s），断言含 stage 不变的真实 DB 读回，非伪断言。 |

**结论：H-B1 关闭。** 但注意测试未显式断言"submission 状态回滚为 pending"（仅断言 response 409 + stage 不变）。经代码审查确认回滚成立（同 tx + defer Rollback），建议后续补一条 submission.status 仍 pending 的断言以固化原子性（测试覆盖补强，非功能缺陷）。

### H-B2 freeze 统一拦截业务写（High）— CLOSED，覆盖完整、申诉豁免合理

| 检查点 | Risk | 证据 | 独立结论 |
|---|---|---|---|
| 统一门禁 helper | Medium | `projectBlockedForWrite`（`vopc_delivery.go:762-770`）判 `blockedStatuses ∪ completedLike` | **通过**。只读 status，不写库。 |
| CreateArtifact 接入 | High | `vopc_delivery.go:306` 门禁，位于 INSERT 之前 | **通过**。409 且不落库不写事件。 |
| CreateArtifactVersion 接入 | High | `vopc_delivery.go:400` 门禁，位于版本归属校验前 | **通过**。409。 |
| CreateRisk 接入 | High | `vopc_risk.go:103` 门禁 | **通过**。409 且不落库（测试断言 frozenRisks=0）。 |
| CloseProject risk_frozen 前置拦截 | High | `vopc_close.go:120-124` 显式 `status=="risk_frozen"` 拦截 | **通过且必要**。经查 `closeTransition`（`:211-234`）对 pause/pivot/terminate 均**未排除 risk_frozen**（仅排除 draft/completedLike/terminated/archived/paused），故 pivot/terminate/pause 可从 risk_frozen 直达，此显式拦截确为必需。 |
| 申诉豁免是否合理且不破坏闭环 | High | `CreateRiskAppeal`（`vopc_risk.go:357-394`）走 `manageableProject`，**未**加 `projectBlockedForWrite`；冻结后主理人仍可申诉（remedy 路径） | **通过/合理**。冻结→申诉→裁定→解冻闭环需要冻结态下仍可申诉；否则冻结即死锁。若无此豁免，冻结项目将无法发起 remedy，闭环断裂。豁免口径正确。 |
| 测试可信度 | High | `TestVOPCFreezeBlocksBusinessWrites`（vopc_governance_test.go:463-527）：冻结后 CreateArtifact/CreateArtifactVersion/CreateRisk 各 409 + frozenArtifacts/frozenRisks=0、pivot/terminate 各 409、appeal 201 | **通过**。`go test -run TestVOPCFreezeBlocksBusinessWrites -count=1 -v` 实测 PASS（0.05s），断言含真实 DB 读回（不落库验证），非伪断言。 |

**结论：H-B2 关闭。** 治理动作（ApproveRisk/FreezeProject/ResolveRiskAppeal）作为 remedy/治理操作**有意不加**门禁，与审计 B.2.3"治理动作可接受"口径一致。

### M-B1 R3 专项角色可达性（Medium）— CLOSED（核心）；附带一条既有限制残留

| 检查点 | Risk | 证据 | 独立结论 |
|---|---|---|---|
| 独立 risk_governance 项目角色全仓移除 | High | `rg risk_governance` 仅命中：迁移文件名 `102_vopc_risk_governance.sql`、`migration_vopc_test.go:13`/`vopc_handler_test.go:42` 的迁移名列表、`vopc_risk.go:42` 注释。**无任何 `.go`/`.sql` 的 `project_role='risk_governance'` 判定或授予路径** | **通过**。判定与授予路径均移除，残留仅为命名/注释。 |
| isSpecialRiskGovernance 复用生产可达角色 | High | `vopc_risk.go:30` `platformGovernanceRoles = setOf("college_admin","school_admin","sys_admin")`；`vopc_risk.go:47-50` `SELECT m.project_role,u.role FROM ... JOIN users u ON u.id=m.user_id` `return role=="platform_operator" && platformGovernanceRoles[sysRole]` | **通过**。三个治理系统角色经 `001_init.sql:10` role CHECK 定义 + `007_seed_users.sql:6/10/14` 播种，生产可达。 |
| 测试不再 db.Exec 直插自证 | High | 删除 `addRiskGovernance` helper；`TestVOPCR3SpecialGovernanceChannel` 改为 `addCollegeAdmin`（真实系统角色）+ `addPlatformOperator`（platform_operator 成员）驱动 | **部分通过（附边界，见下）**。伪造 `risk_governance` 直插已删除；但 `platform_operator` 项目成员仍经 `addPlatformOperator` 用 `db.Exec` 直插（因 platform_operator 亦无 provisioning API，见残留）。 |
| R2/R3 差异保留、无误降 R3 | High | R2=`isRiskManager`（`vopc_risk.go:19-27`，任意 platform_operator）；R3=`isSpecialRiskGovernance`（platform_operator ∧ 治理系统角色）；`milestoneAdvanceAllowed`（`:531-564`）R3 分支（`:538-551`）挂有未批 R3 风险即拦，严于 R2 | **通过**。R3 未降为与 R2 等同：角色更严、创建走 readableProject+二次校验、门禁更严。 |
| 测试可信度 | High | `TestVOPCR3SpecialGovernanceChannel`：owner→403、非治理系统角色( student )挂 platform_operator→403、治理系统角色+platform_operator→201、双人专项审批放行、任一 reject 封死 | **通过**。`go test -run TestVOPCR3SpecialGovernanceChannel -count=1 -v` 实测 PASS（0.09s）。plainOp 使用 DB 内 `users.role='student'`（`:55` 预置 user2 student），与 `token(2,"student",...)` 一致，DB 层角色为权威判据，非 JWT 声明自证。 |

**结论：M-B1 核心关闭。** 独立不可授的 `risk_governance` 已移除，R3 复用与 R2 同源的 `platform_operator` + 治理系统角色，专项通道不再比 R2 更不可达。

**残留（非本批引入，记录不修）：** `platform_operator` 项目角色本身仍无 in-system 授予 API（`projectRoles` 邀请白名单已排除它，`vopc_handler_test.go:480-481` 断言 owner 不得授予；无 admin 项目成员授予端点），其授予依赖"平台治理侧数据分配"（refactor-notes 已明确记录）。因此"治理系统角色 ∧ platform_operator 成员"这条组合的端到端可达性，仍受 platform_operator 授予路径这一既有限制约束——但这是 R2 与 R3 共享的同一限制，R3 不再额外叠加一条不可授角色，M-B1 原缺口（R3 相对 R2 多出的那条不可授 `risk_governance`）已消除。

## C.3 事务审计原子性 / 无伪成功 / 无占位 / 未改历史迁移（任务第 4 点）

- **原子性**：H-B1 门禁位于 ReviewMilestone 单事务内（defer Rollback），拦截时 review/submission/stage 全量回滚；H-B2 的 `projectBlockedForWrite` 只读不写；新增拦截均在既有事务内，未新增任何 bypass。
- **无伪成功**：`go build ./...`、`go vet ./internal/handler/` 实测均 exit 0；三缺口测试均为真实 HTTP 请求 + DB 状态读回断言（frozenArtifacts/frozenRisks=0、stage 不变、风险 status 流转），非占位/非 `t.Skip`。
- **无占位残留**：`rg createProjectAt|addRiskGovernance|bytesBuffer|func w2` 在 `internal/handler` 无命中（`addRiskGovernance` 已彻底删除；`bytes.Buffer` 仅存在于既有 request helper 合法使用）。
- **未改历史迁移**：`git diff --check -- server/` exit 0；097-102 迁移文件无 diff；本批无新增迁移（risk_governance 复用既有字符串值，无 DDL）。
- **101/102 语义未变**：`milestoneAdvanceAllowed` 的 R3 分支逻辑（挂未批 R3 风险即拦，比 R2 严）未被 H-B1 改动，仅新增了调用点；102 的表结构未变。

## C.4 QA 结论与本复核的一致性（任务第 5 点）

- **一致**：QA 附录 B 判定 H-B1/H-B2/M-B1 三项 PASS，本复核独立核查后同意三项在其口径下关闭，证据一致。
- **发现 QA 报告一处轻微失实**：QA"附录 B 三缺口修复回归"章节声称 `git diff --check` PASS，但本复核实测 `git diff --check` 在 `qa-report.md:406` 报 trailing whitespace（`qa-report.md` 自身追加段落引入的空格）。这是 QA 自报门禁与实跑不一致的轻微缺陷，**非代码缺陷**（`server/` 目录 `git diff --check` 为 0）。建议 QA 修正该尾随空白并重跑。
- **QA 措辞需收紧**：QA 称 M-B1 测试"改用真实可授角色"——严格说测试仍以 `addPlatformOperator`（db.Exec 直插）授予 platform_operator 项目成员（因该角色亦无 provisioning API）。准确表述应为"改用真实系统角色（college_admin）取代伪造的 risk_governance 项目角色；platform_operator 成员仍无自助授予路径（既有边界）"。这不推翻 M-B1 关闭结论，但应避免写成"端到端可授闭环已完成"。

## C.5 下一批最小可执行顺序（在整体 P0 之内，除本批外的新增建议）

1. （补强）在 `TestVOPCMilestoneGateTOCTOU` 增加"submission 状态回滚为 pending"断言，固化 H-B1 全量回滚的原子性证明。
2. （补强）将 7 条新增治理/结项路由（close/close-records/risks×2/freeze/risk-appeals×2）纳入 `TestRouteRegistrationCount` 锚点断言（上轮已记建议，仍未做）。
3. （残留）若要将 R3 专项通道做成真正端到端可达，需落 platform_operator 授予的治理侧 provisioning 端点（同解 R2 通行性问题），而非仅为 R3 单独造端点。
4. （Low 清理）统一 `LastInsertId` error 处理；区分 `closed_at` vs `terminated_at`/`archived_at`。
5. 继续推进上一轮既定 P0（AI 闭环、私有文件、rubric/conditional/waiver/甲方/试点发布审批、真实 MySQL/Turso）。
6. （QA）修正 qa-report.md 尾随空白，重跑 `git diff --check` 后如实更新门禁表。

## C.6 签署

H-B1（TOCTOU 真实同函数复核 + 全量回滚）、H-B2（freeze 统一拦截业务写 + 申诉豁免合理）、M-B1（移除不可授 risk_governance，R3 复用 platform_operator+治理系统角色，R2/R3 差异保留）三项修复经独立代码审查 + 定向测试实跑 + 全量 build/vet/diff-check 验证，**均可按其原始口径关闭**。

# 本批复审结论：三缺口 GO（关闭）；整体项目维持 NO-GO（禁止上线、部署或宣称 vOPC PRD v1.0 P0 验收完成）。
