# vOPC PRD v1.0 重构记录

> 唯一产品基准：`docs/wxx-vopc-prd-v1.0.md`
>
> 核对清单：`pm-checklist.md`（33 项）
>
> 日期：2026-08-21
>
> 约束：未部署、未提交、未推送；保留并完善原工作树中的服务端 `can_manage` 与任务空状态引导。

## 本轮结论

本轮关闭了最先应修复的学院准入红线、外院邀请漏洞、S0 草稿不可补齐和前端文本直推入口，并完善 capability 与小屏导航。项目仍**不能按 PRD v1.0 宣称全部完成或上线**。PRD 要求的 AI 执行、风险治理、全协作域、文件存储、展示和运营等是多个完整子系统；当前代码库缺少相应模型/迁移/API/UI，其中真实 LLM 联调还缺运行环境密钥，私有文件闭环缺已确定的存储接入约定。以下未实现内容均明确标为 `[blocked]`，不冒充完成。

## 已完成的代码变更

- `server/pkg/app/routes.go`
  - 正式 `/api/v1/vopc` 路由挂载 `CollegeAccess`。
  - 创建项目增加 `vopc.project.create` capability。
  - 新增 `PUT /projects/:id`。
  - 从正式路由移除 `POST .../milestones/:stage/advance` 文本直推通道。
- `server/internal/handler/vopc_handler.go`
  - 学院准入拒绝 guest、inactive、非 college scope、非计算机学院用户。
  - 新增 S0 草稿更新：仅项目 owner/co_owner/platform_operator 且仅 `S0/draft`；服务端风险重算；事务与审计原子提交；并发条件校验。
  - 保留并使用服务端 `can_manage` 项目关系结果。
- `server/internal/handler/vopc_delivery.go`
  - 邀请对象查询强制校验 active、非 guest、college scope、计算机学院 ID。
  - 接受邀请时再次校验学院准入，防止邀请后账号归属变化。
- `server/internal/auth/capabilities.go`、`frontend/lib/utils/capability_utils.dart`
  - 补齐 PRD capability：mentor review、resource offer、publish request、risk manage、analytics read。
  - assistant 补齐 vOPC 基础能力；学院管理角色获得治理能力。
- `frontend/lib/providers/vopc_provider.dart`
  - 项目详情模型补齐来源、数据类型、导师/资源需求和风险开关。
  - 新增真实 `PUT project` 草稿保存。
  - S1-S9 客户端不再调用文本直推，只允许 S0 立项提交。
- `frontend/lib/pages/vopc/vopc_page.dart`
  - 创建/编辑向导补充项目类型、来源、数据类型、导师/资源需求、真实用户、外发、资金字段。
  - 工作台提供“编辑草稿”，可查看后补齐并保存。
  - 移除 S1-S9 “推进阶段”按钮；提示正式里程碑评审是唯一通道。
  - 保留原任务空状态真实引导。
- `frontend/lib/config/router.dart`
  - 桌面保留独立 vOPC NavigationRail。
  - 移动端压缩为 5 项，移出“服务”，确保 vOPC 一键直达，避免 6 项拥挤。
- `server/internal/handler/vopc_handler_test.go`
  - 准入矩阵修正为 guest/外院/school scope 403。
  - 外院邀请期望改为 422；既有私有隔离、事务、状态机等测试继续执行。

## 迁移

- 复用并验证现有：
  - `097_vopc_p0.sql`
  - `098_vopc_decisions.sql`
  - `099_vopc_collaboration_delivery.sql`
- 本轮没有新增迁移。本轮已落地功能可由现有字段支持。
- `[blocked]` AI tasks/context/cost/review、risk/appeal、join requests、comments/attachments、resources/offers、showcases、close retrospectives 等后续实现必须新增迁移；这些表不存在，不能用 JSON 假字段冒充。

## 33 项逐条映射

1. **学院准入三层边界 — 已完成（后端入口/API/查询前置）**：正式路由挂学院中间件；guest/外院/school scope 拒绝；项目查询只能在已准入请求中发生。前端导航目前仍主要依赖本地 capability，非法用户即使旧缓存显示入口也会被 API 403；要做到登录态变化即时隐藏仍需统一 shell access provider。
2. **目标系统角色与项目角色分离 — 部分完成**：assistant 补齐基础能力；项目写仍按成员关系。`[blocked]` 缺研究生等真实 role 枚举/种子定义以及全角色 fixture，无法构造完整产品角色矩阵。
3. **桌面独立入口 — 已完成**：NavigationRail 保留 vOPC。
4. **移动一键直达/6 项拥挤 — 已完成（Flutter shell）**：小屏 5 项且 vOPC 一键直达；`[blocked]` 缺小程序 WebView 运行环境，无法做真机证明。
5. **首页八区块 — [blocked]**：我的项目和快速发起已有；大厅、伙伴、导师资源、展厅、活动无后端聚合域/运营数据，不添加假按钮。
6. **完整项目创建/S0 草稿 — 已完成核心字段**：前端可填写 PRD 创建字段，服务端保存并重算风险；AI 引导建议 `[blocked]` 于 AI 服务闭环。
7. **S0 查看/编辑/保存/补齐/提交 — 已完成**：真实 PUT、持久化、项目角色权限、审计、提交校验。
8. **R0-R3 创建审核 — 部分完成**：服务端自动升档，R3 默认拒绝；`[blocked]` R1/R2 审批表和审批人工作台不存在。
9. **工作台完整区块 — 部分完成**：阶段、状态、成员、决策、成果、里程碑、任务存在；`[blocked]` 完成度、AI 状态、风险列表、事件读取 API 不存在。
10. **S0-S9 状态机/结构化证据 — 部分完成**：顺序和并发保护存在；正式路由已禁止文本直推；`[blocked]` 当前里程碑 schema 没有结构化量表字段。
11. **甲方 S2/S5/S6 与自拟验证门禁 — [blocked]**：需甲方确认实体、试点/发布审批和迭代记录表；当前文本字段不能可靠证明。
12. **唯一里程碑评审、量表/条件通过/豁免 — 部分完成**：唯一正式路由为 submission/review；成果版本可绑定；`[blocked]` 迁移只支持 pass/return，无 rubric、conditional_pass、waiver。
13. **真人邀请/角色 — 已完成准入红线与接受复验**；`[blocked]` 用户搜索、角色变更/移除 API 尚无。
14. **加入申请/招募 — [blocked]**：无 join request/recruitment 表、通知和页面。
15. **默认4/最多6 AI 岗位配置 — 部分完成**：创建默认4；`[blocked]` 无岗位配置 API/UI/数据库约束。
16. **AI 任务真实执行/版本/Token/成本/重试 — [blocked]**：Handler 未注入现有 `llm.ChatClient`，无 ai task schema；运行环境也未提供可用于验收的模型密钥。禁止伪造模型返回或费用。
17. **项目 AI 上下文隔离 — [blocked]**：现有 Context Engine 面向聊天/知识库，未定义 vOPC 项目 namespace 与批准事实协议。
18. **任务 CRUD/依赖/评论附件 — 部分完成**：创建、负责人、验收、状态和审计已有；`[blocked]` 依赖图、完整编辑/删除、评论附件 schema/API 不存在。
19. **决策/高风险复核 — 部分完成**：真实决策 API/持久化/处理/审计已有；`[blocked]` 现表缺影响、截止、AI 建议、复核状态字段。
20. **成果私有文件与版本 — 部分完成**：元数据与版本闭环已有；`[blocked]` 没有确定的私有对象存储/本地文件服务接入和签名下载协议，不能把公开 URL 当私有文件。
21. **AI 四种人工审阅 — [blocked]**：依赖 #16 的 AI task/output schema。
22. **正式评审/点评 — 部分完成**：指定评审 pass/return 可推进；`[blocked]` 条件通过、评分和多层点评 schema 不存在。
23. **风险冻结/解冻/申诉 — [blocked]**：只有 risk_level 和阻断状态，没有 risks/appeals/approvals 表及 API/UI。
24. **继续/转向/暂停/终止/结项复盘 — [blocked]**：没有 close/retrospective 表和路由，不能仅改 status 冒充复盘。
25. **JWT+Capability+学院+项目+字段可见性 — 部分完成**：前三层及私有项目关系已执行；`[blocked]` artifact 字段级策略和 restricted scope schema 不完整。
26. **可见性/展厅审批 — 部分完成**：默认 private；`[blocked]` publish request/showcase/审核表和路由不存在。
27. **关键审计 — 部分完成**：现有写操作事务内写事件且失败回滚；新草稿更新亦如此；`[blocked]` trace/IP/result 独立字段及审计查询 API 不存在。
28. **讨论/评论/@/通知 — [blocked]**：vOPC discussion/comment schema 不存在；现有通知服务没有 vOPC 事件适配。
29. **导师/评审/路演/资源/展厅 — [blocked]**：仅里程碑 reviewer 已有，其余 P1 域没有迁移/API/UI。
30. **成果包导出 — [blocked]**：缺复盘、风险、贡献和试点数据源；无法生成 PRD 完整包。
31. **独立加载/错误/空态 — 部分完成**：已有全页、任务、决策状态及真实重试，保留任务空态引导；`[blocked]` Provider 仍共享部分 error，需各域重构状态对象。
32. **桌面/移动/WebView — 部分完成**：响应式导航已修；`[blocked]` 无连接设备/WebView harness，本轮无法执行真机/golden 验收。
33. **P95/AI 超时/备份恢复 — [blocked]**：AI 尚无闭环；未提供代表性数据集、生产同构数据库/文件存储和备份恢复沙箱，不能伪造 P95 或恢复演练结果。

## 门禁结果

- `gofmt`：通过。
- `go test ./internal/handler -run VOPC -count=1`：通过。
- `go test ./internal/auth ./pkg/app -count=1`：通过。
- `go vet ./internal/handler ./internal/auth ./pkg/app`：通过。
- `dart format`（vOPC Provider/UI/router/capability）：通过。
- `flutter test test/vopc_provider_test.dart`：通过，6 项。
- `flutter analyze` 定向：首次发现 1 个 nullable access，修复后复跑通过（No issues found）。
- `git diff --check`：通过。
- 未部署、未 commit、未 push。

## 剩余最高风险

1. AI 岗位仍不能真实执行，P0 AI 验收不成立。
2. R1/R2 审批、冻结/申诉和结项复盘无数据模型，安全治理不成立。
3. 里程碑虽然已关闭文本直推，但评分、条件通过和豁免仍缺 schema。
4. 私有文件仍只有引用元数据，没有受鉴权的上传/下载链路。
5. P1/P2 首页、招募、资源、导师、路演、展厅、成果包均缺完整业务域。

因此当前状态应判定为：**学院准入/S0 草稿阻断已修复，P0 仍部分完成；禁止上线验收。**


## 本轮审计阻断修复（2026-08-22）

- **前端学院准入入口/路由：已完成**：`Storage` 持久化 `owner_scope/owner_id/status`，登录资料同步；新增 `VopcAccess`，桌面/移动导航和 `/vopc` redirect 均要求 active、非 guest、`college/cs` scope 及 `vopcRead` 能力，缓存缺失按拒绝处理。新增准入矩阵单测；后端 `CollegeAccess` 仍是最终边界。
- **里程碑成果版本实质门禁：已完成基础硬门禁**：正式 `SubmitMilestone` 拒绝空 `artifact_version_ids`；前端加载真实成果版本并强制至少选择一个后提交；新增 provider 测试及 handler 空版本回归测试。版本归属/状态/内容等既有后端校验继续生效。
- **旧直推残留：已完成**：移除前端 `advanceProject`/旧 URL；`SubmitProject` 仅接受 S0 草稿立项，S1-S9 不再存在通用文本直推 handler；正式阶段只能走 milestone submission/review。
- 新增真实草稿编辑/提交调用测试覆盖；未伪造 AI、对象存储或外部审批。
- 本轮验证：`go test ./internal/handler -run VOPC -count=1`、`go test ./pkg/app -run 'Test(KeyRoutesReachable|RouteRegistrationCount)' -count=1`、`flutter test test/vopc_provider_test.dart test/vopc_access_test.dart`、定向 `flutter analyze`、`gofmt`、`dart format`、`git diff --check` 均通过。
- **[blocked]** rubric/评分、conditional pass、waiver、甲方审批、AI/对象存储等仍无真实 schema/运行能力，本轮未伪造；整体仍禁止上线验收。

## 2026-08-22 安全门禁后续整改（本轮）

- 普通项目邀请角色白名单移除 `platform_operator`；owner/co_owner 不能再借邀请授予平台治理角色。指定评审规则保持不变，普通成员/导师不能越过指定评审。
- 新增 `100_vopc_artifact_version_gates.sql`：成果版本增加 `status`（active/invalid）与 `intended_stage`。历史版本默认 invalid/无阶段，避免迁移后自动成为可信证据；迁移纳入 SQLite→MySQL 方言静态回归。
- 创建版本现要求受控 `source_kind`、非空安全引用、64 位十六进制 SHA-256 checksum、S2-S9 阶段绑定；这只是可由服务端验证的结构化元数据门禁，**不声称验证外部仓库 commit 或对象存储对象真实存在**。
- 提交事务内重查版本同项目、active、checksum、目标阶段和阶段要求成果类型，并限制 1-20 个、不允许重复。S2-S9 回归改为逐阶段创建独立版本，不再使用同一个 `commit:test` 贯穿。
- 修复邀请计数、邀请 CAS、artifact 派生更新时间、submission CAS、项目/里程碑推进的 SQL error/RowsAffected 忽略；失败会回滚。
- 新增 owner 邀请 platform_operator、普通成员越级评审、重复/失效/跨阶段版本，以及邀请后变外院/guest/inactive 时接受失败且 invitation/member/event 原子一致的负向测试。

边界：本轮仅收紧已有协作/成果元数据和里程碑事务门禁；**AI 真实执行与上下文隔离、私有文件上传/鉴权下载、风险审批/冻结/申诉、S9 后结项复盘等域仍未实现，继续是 NO-GO 阻断项，不能写成完成。**

## vOPC 结项状态机 + 风险治理最小闭环 + 测试修复（2026-08-22）

- **结项与异常状态机（迁移 101）**：新增 vopc_close_records 审计表、vopc_projects 增加 completed_at/closed_at 可空列；S9 里程碑通过后项目进入 closeable（不再直接 completed），必须由项目管理角色发起 close 才落为 completed。合法流转 close(closeable→completed)/pause/resume/pivot(回到 S0 草案并复位里程碑)/terminate(需失败证据)/archive(仅 completed/terminated)；所有动作在单事务内写记录与事件，失败整体回滚。resume 从 pause 记录恢复 previous_status，历史丢失回退 pending_review。
- **风险治理最小闭环（迁移 102）**：新增 vopc_risks / vopc_risk_approvals / vopc_freeze_records / vopc_risk_appeals 四张表。风险创建后 open；R2/R3 项目里程碑推进要求同等级风险双人（两名不同审批人）approve 才解除拦截；冻结/解冻与审批只允许项目内 platform_operator 角色；申诉由主理人发起、平台治理角色裁定（upheld/dismissed），全部审计入 vopc_events。仅服务端可治理元数据，不接入外部审批系统、不涉及真实资金/合同。
- **权限边界**：freeze/approve/resolve 路由挂 vopc.risk.manage capability（college_admin 具备、student 不具备），并在 handler 内二次校验 platform_operator 成员身份；重复审批、非 open 审批、非 pending 裁定、重复冻结均返回 409，无治理成员关系返回 403。
- **测试修复**：清理 vopc_governance_test.go 中无法编译的占用占位（非法签名 createProjectAt、nil 返回的 w2()、bytesBuffer 接口、未使用的 appeal 结构体）；新增结项状态机、pivot 复位、R2 双人审批 gate、freeze/申诉端到端测试；vopcTestDB 迁移列表补入 101/102。migration_vopc_test.go 纳入 101/102 的 SQLite→MySQL 方言静态回归。
- **验证（通过）**：gofmt；go vet ./internal/handler/；go test ./internal/handler -run 'Test.*(VOPC|Close|Risk|Freeze|Appeal|Pivot|Governance)' -count=1；go test ./internal/db -run TestToMySQLVOPCMigrations -count=1；go test ./pkg/app -count=1；git diff --check。未部署、未 commit、未 push。
- **[blocked]** 仍缺：R1/R2 立项审批实体与工作台、rubric/条件通过/豁免、AI 真实执行与上下文隔离、私有文件上传/鉴权下载、甲方审批（S2/S5/S6）等域，本轮未伪造。整体仍禁止上线验收。

## QA 第三轮回归 B3：R3 独立专项审批通道（2026-08-22）

- **背景**：QA-report 第三轮唯一实质缺口 B3——R3 风险与 R2 共用同一双人审批门槛，未实现 PRD 13.1 的独立专项通道，且 R3 风险可由任意 manager 创建。
- **PRD 13.1 口径选择（明确记录）**：PRD 13.1 对 R3 表述为「禁止或专项审批 → 默认禁止，按学校制度专项审批」。本批选择**专项审批**口径，而非纯禁止：任务要求实现可执行的最小专项通道（创建限制 + 双人专项审批 + 拒绝路径 + 里程碑门禁），纯「禁止」会把 R3 风险完全惰性化、与要求的审批流矛盾。落地为「默认禁止推进，专项审批通过后放行」。
- **实现（server/internal/handler/vopc_risk.go）**：
  - 新增 `risk_governance` 专项角色（R3 治理），与 `platform_operator`（R2 一般治理）分离，互不越权；两者均不开放项目侧自助授予（与 platform_operator 一致）。
  - `isSpecialRiskGovernance` 判定函数；`CreateRisk` 对 R3 改用 `readableProject` 边界 + 专项角色二次校验（避免把 risk_governance 放大为完整项目管理权），普通 manager 创建 R3 返回 403；R0/R1/R2 维持既有 `manageableProject` 语义。
  - `ApproveRisk` 按风险等级分流：R3 走 `risk_governance` 专项通道（专项权限 + 两名不同审批人 approve → approved；任一 reject → rejected）；R2 及以下走 `platform_operator` 双人审批（不退化）。
  - `milestoneAdvanceAllowed` 升级：项目为 R3 时需专项审批通过的 R3 风险；**即使项目本体是 R0/R1/R2，只要挂有未专项审批的 R3 风险也一律阻断里程碑**（“禁止推进”落地）。R2 门禁保持原语义。
- **不改迁移**：`risk_governance` 是 `vopc_project_members.project_role` 的既有字符串值，无 DDL 变化；未新增 103，不触碰 101/102 或历史迁移。SQLite/MySQL 方言兼容不受影响。
- **测试（vopc_governance_test.go 新增 TestVOPCR3SpecialGovernanceChannel）**：普通 manager 创建 R3 被拒(403)、platform_operator 越权创建被拒(403)、专项角色创建成功(201)、R3 缺专项审批时里程碑被拦(409)、单人专项审批仍 open 且里程碑仍拦、双专项审批后放行(201)、任一专项 reject 即 rejected 且里程碑再拦。既有 R2 双人审批 gate 测试继续通过，确认 R2 语义不退化。
- **未扩展/仍阻断**：本批未伪造外部审批/模型/对象存储；试点审批、发布审批、文件外发等其他 R3 相关联的域仍缺 schema/运行环境，继续 [blocked]。里程碑门禁为本批实际落地的最小门禁点。
- **验证（全部通过）**：gofmt -w；go vet ./internal/handler/；go test ./internal/handler -run 'Test.*(VOPC|Close|Risk|Freeze|Appeal|Pivot|Governance|R3)' -count=1；go test ./internal/db -run TestToMySQLVOPCMigrations -count=1；go test ./pkg/app -count=1；go test ./... -count=1；git diff --check。未部署、未 commit、未 push（leader 统一提交）。
