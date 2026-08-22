# wxx vOPC PRD v1.0 严格兑现核对清单

> 唯一产品基准：`docs/wxx-vopc-prd-v1.0.md`（2026-08-20）
>
> 核对日期：2026-08-21
>
> 核对范围：当前前后端源码、数据库迁移、专项测试及未提交 `git diff`。
>
> 状态定义：**已完成**＝当前代码存在可执行前后端闭环且满足 PRD 边界；**部分完成**＝已有真实实现但缺少必要字段、流程、权限、页面或测试；**未完成**＝无可验收闭环。
>
> 优先级：**P0 阻断**＝违反准入/安全边界或 P0 核心闭环不可验收；**P0**＝首版必需；**P1/P2**＝按 PRD 分期。

## 1. 总体结论

当前 vOPC **已经具备可运行的项目最小骨架**，不再是“未开始”：存在一级入口、项目草稿创建、私有项目关系、S0-S9 顺序推进、真人邀请、任务状态流转、决策、成果版本、里程碑提交/评审和业务事件，并有后端及 Provider 专项测试。

但按 PRD v1.0 严格验收，当前只能判定为 **P0 部分完成，不能上线验收**。最严重问题是：

1. **学院准入边界实际未启用**：`CollegeAccess` 虽已实现，却没有挂到正式 `/api/v1/vopc` 路由；前端导航对所有角色固定显示；测试更明确把 guest、外院和 school scope 期望为 200。该现状直接违反 PRD 4.1、7.1、16.2。
2. **邀请可越过学院边界**：邀请逻辑只检查用户存在且 active，不校验学院归属；测试明确期望外院用户邀请成功。
3. **S0 草稿无法编辑**：可创建草稿，但没有 `PUT /projects/:id`，创建后工作台也没有编辑资料入口；信息不全时只能卡在提交错误，无法补齐。
4. **AI 仅有岗位占位数据**：创建时插入 4 个岗位，缺少岗位读取/配置页面、AI 任务执行、模型调用、版本化产出、人类接受/修改/退回/否决、Token/成本、超时重试和项目上下文隔离。
5. **阶段推进存在绕过正式评审的双通道**：主理人可直接调用 `.../milestones/:stage/advance` 用一段文本逐级推进直至 S9；无需成果版本、评分量表或指定评审，无法满足“阶段门禁与人类评审”的实质要求。
6. **风险治理、结项复盘、展示审批、讨论通知、资源等域缺失**；异常状态仅被识别为阻断值，没有产生/审批/解冻/申诉闭环。

### 当前专项验证

- `go test ./internal/handler -run VOPC -count=1`：**通过**。
- `flutter test test/vopc_provider_test.dart`：**通过（6 项）**。
- `git diff --check`：**通过**。
- 当前未提交 diff 仅涉及：`vopc_page.dart`、`vopc_provider.dart`、`vopc_handler.go`、`vopc_handler_test.go`；主要修正服务端返回 `can_manage` 及任务空状态引导，未解决上述准入与核心 AI 阻断。

## 2. 逐条需求核对

| # | PRD 需求 | 状态 | 源码证据 | 缺口 | 验收标准 | 优先级 |
|---|---|---|---|---|---|---|
| 1 | 仅计算机学院已授权用户可见入口、进入页面、调用接口和查询数据 | **未完成** | `server/internal/handler/vopc_handler.go:48-62` 有 `CollegeAccess`；但 `server/pkg/app/routes.go:134-159` 的 vOPC group 未挂载它，注释反而写“所有已登录系统用户均可进入”；`frontend/lib/config/router.dart:743-758` 对所有角色固定加入 vOPC；`vopc_handler_test.go:126-137` 期望 guest/外院/school scope 均 200 | 三层校验均未兑现；`AccessStatus` 永远返回 allowed=true | 未登录 401；guest、外院、校外、非 college scope、未授权用户前端不显示且全部 vOPC API 403；合法 CS active 用户 200；查询层不得泄漏 | **P0 阻断** |
| 2 | 所有目标角色（本科/研究生/教师/辅导员/教辅/学生会/学院管理者）均可创建参与，系统角色与项目角色分离 | **部分完成** | `server/internal/auth/capabilities.go:193-198,268,308,328,361` 定义并分配部分 vOPC 能力；`097_vopc_p0.sql:33-44` 有项目成员表；`projectPolicy` 依据项目关系 | 未见覆盖所有目标系统角色的准入/创建矩阵测试；Capability 清单缺 mentor/resource/publish/risk/analytics；正式创建路由本身未要求 create capability | 对每种目标系统角色做创建、邀请、接受、跨项目不同项目角色测试；未授权角色拒绝；项目角色不由全局系统角色替代 | **P0** |
| 3 | Web/桌面 NavigationRail 独立一级入口 | **已完成（功能层）** | `router.dart:272-281` 注册 `/vopc` 与工作台；`router.dart:757` 使用 rocket 图标；`router.dart:774-821` 桌面 NavigationRail | 仍受 #1 准入缺口影响 | 合法学院用户桌面一键进入；当前选中态正确；非法用户无入口 | **P0 阻断依赖 #1** |
| 4 | 移动端一级一键直达且避免 6 项拥挤；WebView 同路由 | **部分完成** | `router.dart:743-758` 共 6 项；`router.dart:830-863` 移动 NavigationBar 直接渲染全部 6 项 | 未按 PRD 在 6 项不适配时合并“服务”为“更多”；无小屏/横屏/Flutter WebView 验收证据 | 320/360/390dp 下标签、触控区无拥挤截断，vOPC 一键可达；必要时服务并入更多；WebView 同 `/vopc` | **P0** |
| 5 | vOPC 首页八区块：我的项目、快速发起、大厅、孵化任务、伙伴、导师资源、展厅、活动 | **部分完成** | `vopc_page.dart:21-101` 有我的项目、创建入口和待邀请 | 项目大厅、孵化待办聚合、寻找伙伴、导师资源、成果展厅、活动均无页面/API；列表只查本人/成员项目 | 首页八区块具备真实数据源、权限过滤、空/错/加载状态；无假按钮 | **P1**（我的项目/快速发起为 P0） |
| 6 | 自拟项目创建并保存 S0 草稿 | **部分完成** | `vopc_page.dart:106-231` 表单调用 Provider；`vopc_provider.dart:251-264` POST；`CreateProject` 创建项目、owner、4 AI 岗位、10 里程碑和事件；`097` 迁移字段较完整 | 前端表单只暴露 9 项，风险/数据类型/项目类型固定，未填写导师/资源需求、真实试用、外发、资金；没有 AI 引导/建议；只有名称必填，虽可作为草稿但缺显式“完整创建向导” | 用户可选择全部 PRD 项并保存不完整草稿；风险字段根据选择服务端重算；10 分钟可完成；AI 只建议不代提交 | **P0** |
| 7 | S0 草稿资料查看、编辑、保存、补齐后提交 | **未完成** | `GetProject` 返回资料；`SubmitProject` 在提交时校验必填；工作台有“提交立项”按钮 | 无 `PUT /projects/:id` 路由/Handler/Provider；工作台不展示全部资料、不提供编辑按钮；提交报缺失后用户无处补齐 | owner/co_owner 可反复编辑 S0；普通成员不可改关键资料；保存后刷新可见；缺字段定位并可补齐；提交后按规则限制编辑/走变更 | **P0 阻断** |
| 8 | 创建/提交涉及真实用户、敏感数据、外发、资金、高风险时进入审核 | **部分完成** | `normalizeAndValidate` 自动将相关情况升为 R2/R3；R3 返回 422；测试覆盖资金拒绝 | R2 仅改风险字符串，仍可直接提交和推进；R1 无告知审核；R3 无专项审批录入/审计 API；前端无法选择这些字段 | R0 自动；R1 告知/基础审核；R2 状态进入待审核且未批不可推进/试点/发布；R3 默认禁止并支持授权专项流程；均有理由与事件 | **P0 阻断** |
| 9 | 项目工作台展示阶段、完成度、真人/AI 状态、下一里程碑、待决策、风险、最新产出、活动日志 | **部分完成** | `vopc_page.dart:477-574,658-964` 展示阶段/status/type/risk、下一步、决策、真人、成果、里程碑提交、任务 | 无完成度、AI 状态、下一里程碑细节、风险预警、最新产出摘要、活动日志；没有资料编辑/公司章程等操作入口 | 要求字段均从真实 API 返回；每区块有独立空/错/加载；按项目角色显示操作 | **P0** |
| 10 | S0-S9 顺序状态机及阶段交付物 | **部分完成** | `stageStatuses`、`milestoneEvidence`；`transition` 禁止跳阶段；`TestVOPCS0ToS9StateMachine` 验证顺序到 S9 | 状态映射不完全符合 PRD 项目状态序列（缺 requirement_received/accepted/iterating 等）；S9 直接 status=completed；阶段证据只是一段文本，可与阶段真实成果脱节 | 状态转换契约覆盖正常/异常状态；每阶段所需证据可结构化验证；并发、重复、跳跃均后端拒绝 | **P0** |
| 11 | 甲方项目不得跳过 S2/S5/S6；自拟不得跳过用户验证、交付、迭代 | **部分完成** | `transition` 对 client_requirement 的 target 3/6/7 要求至少 10 字证据；真实试点 S9 要求证据含“用户” | 仅长度/关键字校验，可填任意文本绕过；自拟且 real_user_trial=false 可无用户验证直达 S9；没有交付版本/迭代记录硬关联 | 甲方确认、测试、上线审批须关联指定角色/评审及成果版本；自拟结项必须关联真实验证、可用交付和至少一次迭代/失败证据 | **P0 阻断** |
| 12 | 每阶段必交产出、评分量表、门禁、提交评审、通过/退回、合理豁免 | **部分完成** | `099` 有 milestone submissions/reviews；`SubmitMilestone` 校验下一阶段、证据、版本归属和评审者；`ReviewMilestone` 支持 pass/return 并推进 | 无评分量表/分数/条件通过/豁免；主理人可绕开正式 submission/review，直接调 advance；前端提交时 artifact_version_ids 固定空数组 | 唯一推进通道必须消费已通过提交；量表逐项评分；退回可重提；豁免需理由、审批人、权限及事件；前端可选择成果版本 | **P0 阻断** |
| 13 | 真人成员邀请、接受、分配项目角色 | **部分完成** | `vopc_delivery.go:51-219` 邀请/接受；前端邀请与待邀请按钮均调用真实 API；测试覆盖闭环 | 只按用户 ID 邀请，缺搜索；不校验受邀者学院归属；测试明确邀请外院 user 5 成功；无成员角色变更/移除；资源支持者不在角色枚举 | 只能查找并邀请 CS 已授权 active 用户；接受后项目可见；角色变更/移除有矩阵和审计；外院邀请 403/422 | **P0 阻断** |
| 14 | 申请加入公开招募项目、所需技能/人数/工作量、主理人审批 | **未完成** | 无 join-request 路由、表或页面 | 整个申请加入/招募闭环缺失 | college/invite_only 项目可发布招募；授权用户申请；主理人通过/拒绝并分配角色；全程通知和审计 | **P1** |
| 15 | 默认 4 个 AI 岗位，可配置，最多启用 6 个，按类型推荐 | **部分完成** | `CreateProject` 固定插入 project_manager、market_user、product_solution、execution 四岗位；`vopc_ai_roles` 有 enabled | 无 GET/POST ai-roles API、无前端 AI 团队区、不可调整；无最多 6 个后端约束；不按项目类型推荐 | 创建后可见 4 岗；owner 可在 7 默认岗位中启停/配置且 enabled≤6；推荐可解释；变更有事件 | **P0 阻断** |
| 16 | AI 任务真实执行闭环：输入、上下文、模型、版本、耗时、Token/成本、审阅动作、最终状态 | **未完成** | 任务可把 `assignee_ai_role` 记录为负责人；除此无 ai-task 表/API/模型调用 | AI 岗位只是字符串负责人，没有实际执行；无接受/修改/退回/否决、额度、超时、重试、成本和人工审阅 | 真实调用模型并产生隔离的草稿版本；记录全部 PRD 字段；人工四种审阅动作；失败/重试不阻塞项目；额度受控 | **P0 阻断** |
| 17 | 项目级 AI 上下文隔离、批准事实、跨项目授权脱敏 | **未完成** | 未发现 vOPC Context/知识库接入 | 没有项目上下文、草稿/批准事实边界、AI 文件访问授权 | 项目 A 的成员/AI 无法读取 B；未批准草稿不入事实；跨项目复用需授权与脱敏；归档执行保留策略 | **P0 阻断** |
| 18 | 任务创建、真人/AI 负责人、优先级、截止、验收标准、合法状态流转 | **部分完成** | `taskInput`、`CreateTask`、`UpdateTask`、`validTaskTransition`；前端新建/状态按钮真实调用 API；测试覆盖 todo→in_progress→review→done、越权和审计 | 无依赖、评论、附件；更新 API 只改状态，不能编辑标题/负责人/期限/优先级；AI 不能执行；cancelled 后无恢复策略；任务无详情路由 | 完整字段可 CRUD；依赖无环；真人/AI 负责人有效；评论/附件受权；状态含验收责任和并发控制；关键动作审计 | **P0** |
| 19 | 决策中心：背景、选项、AI 建议、影响、截止、决策人、理由、结果；高风险复核 | **部分完成** | `vopc_decisions.go` 有创建/读取/resolve/cancel、决定人和事件；前端按钮均接 API | 无 AI 建议、影响、截止时间、指定决策人；高风险无导师/平台复核；创建时强制先填“决定内容”与待决策语义冲突 | 字段齐全；pending 可指定决策人和截止；高风险未复核不得生效；所有理由/修改可追溯 | **P0** |
| 20 | 成果仓库与版本：多类型、作者、来源、许可、可见范围 | **部分完成** | `099` 成果/版本表；`vopc_delivery.go:222-393` CRUD；前端登记成果/版本调用真实 API；测试覆盖 | 仅元数据/引用，无实际文件上传/私有下载；前端类型提示少于后端；版本列表没有工作台查看入口；字段级可见性只接受 private/project，无学院展示审批；普通成员不能提交成果（PRD 允许） | 支持 PRD 文件与引用类型；私有 URL 鉴权；成员按权限上传；版本可查看/下载；来源、许可、checksum、作者、可见范围完整 | **P0** |
| 21 | AI 产出可接受、修改、退回、否决并保留版本 | **未完成** | 成果版本是通用人工登记，未关联 AI 调用/审阅 | 无 AI 产出实体和审阅动作 | 四种动作均真实落库；修改形成新版本；保留原始模型产出、审阅人、意见、最终状态和成本 | **P0 阻断** |
| 22 | 里程碑正式提交、指定评审、点评、通过/退回 | **部分完成** | `SubmitMilestone`、`ReviewMilestone` 及前端按钮；测试覆盖指定 reviewer 通过、普通成员 403 | 仅 pass/return，无评分、条件通过、终止建议；导师通用点评无项目/任务/成果层接口；前端只凭全局 capability 显示评审菜单，后端才做指定关系校验 | 指定评审/平台运营可评；普通人不可见动作；评分量表、条件通过、退回整改、终止建议和各层点评可追溯 | **P0** |
| 23 | 风险 R0-R3、冻结/解冻/处置、理由、申诉，真实试点/外发审批 | **未完成** | 项目有 risk_level；`blockedStatuses` 会阻止部分写操作；创建时 R2/R3 粗判 | 无 `vopc_risks` 表、风险 API/页面、`vopc.risk.manage`、冻结/解冻动作、理由/申诉；也没有展示/外发审批 | 风险规则自动触发；授权管理员可冻结/解冻且必须填写理由；申诉与复核留痕；冻结覆盖所有写/AI/发布操作 | **P0 阻断** |
| 24 | 异常状态 rejected/change_pending/paused/terminated/archived/risk_frozen 及继续/转向/暂停/终止/结项复盘 | **未完成** | 后端只把 paused/risk_frozen/terminated/archived 当阻断集合 | 无产生这些状态的 API/UI；缺 rejected/change_pending；无 continue/pivot/close、失败证据、责任判断、归档策略；S9 自动 completed | 每个动作有独立权限、理由、证据和事件；终止需反馈/验证/复盘；结项需成果包；可恢复状态有合法反向流转 | **P0 阻断** |
| 25 | 权限：JWT + Capability + 学院归属 + 项目关系 + 字段可见性 | **部分完成** | 正式路由有 JWT/EnsureUserExists；部分写路由 RequireCapability；`projectPolicy` 限 owner/co_owner/platform_operator 管理，私有项目猜 ID 返回 404；测试覆盖私有隔离 | 缺学院中间件；Create/List/Get/UpdateTask 等路由未要求相应 capability；Capability 本身按系统角色广泛赋予 manage；无字段级可见性、college/invite_only/restricted 查询策略 | 五层策略同时通过；项目角色决定项目写，系统 capability 只授权动作类型；所有资源子 ID 绑定项目；负向矩阵自动化 | **P0 阻断** |
| 26 | 项目默认 private；college/invite_only/restricted；禁止 campus/public；学院展厅须申请审核 | **部分完成** | migration 默认 private；详情/列表目前只对 owner/成员可读；成果默认 private | 无可见性变更 API，未实现 invite_only/college/restricted；无 publish-request/showcase/审核；成果只有 private/project | 默认私有；变更须 owner 申请、审核人批准；college 只展示允许字段；敏感成果可更严；外院/游客始终拒绝 | **P0 默认私有已满足；发布为 P1** |
| 27 | 所有关键操作及 AI 调用可审计，失败不得虚假成功 | **部分完成** | `vopc_events`；项目、任务、决策、邀请、成果版本、里程碑均事务内 writeEvent；失败注入测试验证项目与事件原子回滚；当前 diff 返回 can_manage | 事件表仅 action/detail，无 before/after 独立字段（被拼进 detail）、traceId/IP/结果；无读取界面/API；无 AI 调用和风险/发布审计；部分更新 SQL 忽略错误/RowsAffected | 关键业务均记录 actor、角色、before/after、理由、时间、trace、结果；审计失败业务回滚；授权审计员可查询；AI 调用全量记录 | **P0** |
| 28 | 讨论、任务评论、@成员、里程碑/风险/评审通知 | **未完成** | 无 vOPC discussion/comment/notification 接口或页面 | 完整协作通知闭环缺失 | 评论与 @ 实际落库；邀请、截止、里程碑、风险、评审触发现有通知；权限和已读状态正确 | **P0/P1** |
| 29 | 导师点评、项目评审、路演、资源对接、成果展厅 | **未完成** | 仅有 milestone reviewer 角色和评审；无其余路由/表/页面 | 项目/任务/成果点评、评分评审、Demo/问答、资源意向、展厅申请审核均缺 | 按 F-012~016 分别建立 API/页面/权限/审计；展厅严格限学院授权用户 | **P1** |
| 30 | 一键成果包导出 | **未完成** | 无 export API/button | 无摘要、贡献、决策、里程碑、试点、风险、复盘聚合导出 | 结项可生成可下载成果包，内容完整、权限受控、有操作者/时间/trace/水印 | **P1** |
| 31 | 加载、异常、空状态和重试 | **部分完成** | 首页/工作台/任务有 Linear/CircularProgressIndicator、ErrorCard、重试、空项目/空任务；当前 diff 增加 S0 空任务引导及“新建第一个任务”真实动作 | decisions/members/artifacts/milestones 缺独立 loading/error/empty；Provider 共用一个 `error`，某子请求失败可污染全页；邀请按钮无 mutation 禁用/成功失败反馈；若 detail 子加载失败可能最终仍显示错误混杂 | 每区块独立 loading/error/empty；失败明确且可局部重试；提交防重复；成功/失败有反馈；403/404/409/422 文案正确；不得误报成功 | **P0** |
| 32 | 桌面、移动端及小程序适配 | **部分完成** | 主壳按 >900 切 NavigationRail/NavigationBar；页面大量使用 ListView/Wrap/LayoutBuilder/可滚动 Dialog | 无 widget/golden/真机测试；创建/操作对话框在窄屏、键盘弹出、长文本下未验收；6 导航项风险未处理；无 WebView 证据 | 320dp 到桌面宽度、横竖屏、键盘和长文本无溢出；Web/WebView/Android/iOS 核心流程可操作；有自动/人工验收记录 | **P0** |
| 33 | 常规列表/工作台 P95<2s，AI 超时重试，备份归档恢复 | **未完成** | 查询有限制 100，部分索引已建；无性能/恢复测试 | 无分页、P95 证据；AI 任务不存在；无 vOPC 备份恢复/归档验证 | 代表性 20-30 项目及成果/事件量下 P95<2s；分页稳定；AI 超时重试；数据库和私有文件备份恢复演练通过 | **P0 阻断（AI/安全）** |

## 3. 前端按钮与真实 API 闭环审计

| 页面按钮/动作 | 状态 | 实际调用 | 核对结论 |
|---|---|---|---|
| 首页刷新/下拉刷新 | **已完成** | `loadProjects` + `loadInvitations` → GET projects/invitations | 有真实 API；共用 error 状态仍需拆分 |
| 创建项目/保存草稿 | **部分完成** | `createProject` → POST `/vopc/projects` | 真闭环；字段不全且创建后不可编辑 |
| 项目卡进入工作台 | **已完成** | 路由 `/vopc/projects/:id` → GET detail 及子资源 | 私有关系由后端保护；学院准入缺失 |
| 接受/拒绝邀请 | **部分完成** | POST `/vopc/invitations/:id/respond` | 真闭环；无操作中禁用/结果提示，且外院邀请漏洞 |
| 提交立项 | **部分完成** | POST `/projects/:id/submit` | 真 API；缺草稿编辑入口，资料不足后无法补齐 |
| 推进阶段 | **部分完成/高风险** | POST `/projects/:id/milestones/Sn/advance` | 真 API，但形成绕过正式里程碑评审的捷径 |
| 创建/处理决策 | **部分完成** | POST decisions、PUT decision | 真闭环；字段和高风险复核不足 |
| 邀请成员 | **部分完成** | POST members | 真闭环；手输 ID 且不校验学院 |
| 登记成果/登记版本 | **部分完成** | POST artifacts/versions | 真闭环；仅引用元数据、无私有文件闭环；版本无查看 UI |
| 提交里程碑材料 | **部分完成** | POST milestone-submissions | 真 API；前端永远提交空 artifact_version_ids，不能选择成果证据 |
| 通过/退回里程碑 | **部分完成** | POST milestone review | 真 API；前端动作显示只看全局 capability，指定关系由后端二次拒绝 |
| 新建任务/空态新建第一个任务 | **部分完成** | POST tasks | 真闭环；S1-S8 门禁正确，缺依赖/评论/附件/编辑 |
| 任务状态按钮 | **部分完成** | PUT task status | 真闭环且后端校验状态；仅状态更新 |
| 刷新任务/工作台 | **已完成** | GET tasks / detail + 子资源 | 有真实 API；串行子请求和共享错误状态影响体验 |
| AI 团队/执行/审阅按钮 | **未完成** | 无 | 页面和 API 均不存在 |
| 风险冻结/解冻/申诉 | **未完成** | 无 | 页面和 API 均不存在 |
| 编辑 S0 资料 | **未完成** | 无 PUT project | P0 关键断点 |
| 继续/转向/暂停/终止/结项复盘 | **未完成** | 无 close API | P0 关键断点 |
| 评论/@/附件/通知 | **未完成** | 无 | 协作断点 |
| 展厅申请/发布/成果包导出 | **未完成** | 无 | PRD P1/结项能力缺失 |

**结论：** 当前画面中已经呈现的主要操作按钮大多接入真实 API，不属于纯展示按钮；但 PRD 要求的多个关键动作根本尚未出现。最危险的不是“假按钮”，而是“直接推进阶段”这个真实按钮/API 可以绕开正式提交与评审。

## 4. 数据模型与 API 覆盖

### 已存在

- 表：`vopc_projects`、`vopc_project_members`、`vopc_ai_roles`、`vopc_tasks`、`vopc_milestones`、`vopc_decisions`、`vopc_events`、`vopc_invitations`、`vopc_artifacts`、`vopc_artifact_versions`、`vopc_milestone_submissions`、`vopc_milestone_reviews`。
- API：access、项目列表/创建/详情/提交/阶段推进、任务列表/创建/状态更新、决策列表/创建/处理、成员列表/邀请/响应、成果列表/创建/版本、里程碑提交列表/创建/评审。

### PRD 建议范围中缺失

- `PUT /projects/:id`（S0 编辑/项目变更）。
- join requests、AI roles 配置、AI tasks、通用 reviews/导师点评、resources、close、publish request、showcases、admin analytics。
- 风险、讨论/评论/附件、通知、用户反馈/缺陷/工单、路演、复盘、成果包导出相关表/API。
- PRD 数据表：`vopc_reviews`、`vopc_resource_needs/offers`、`vopc_risks`、`vopc_showcases`；AI task/产出审阅/成本与上下文隔离也无可替代表。

## 5. 测试覆盖评价

### 已覆盖且本轮通过

- JWT 基础访问矩阵（但期望值本身违反学院边界）。
- 私有项目列表/详情猜 ID 隔离、owner 提交与重复提交。
- 创建事务回滚、无效枚举、资金触发 R3 拒绝。
- 项目角色管理边界与 blocked 状态。
- S0→S9 顺序状态机。
- 任务创建、验收必填、负责人授权、状态机、跨项目访问和事件。
- 邀请接受、成果/版本、指定评审里程碑、事件。
- Provider 的详情子资源加载、任务创建、决策处理、成果虚假成功防护、邀请刷新、409 错误透传。

### 必须补充的阻断测试

1. 正式路由挂载学院准入后：guest、外院、school scope、inactive、未登录全部负向；所有目标 CS 角色正向。
2. 邀请外院/未授权用户必须失败，接受邀请时再次校验准入状态。
3. S0 创建不完整草稿→编辑补齐→提交；普通成员编辑/改可见性失败。
4. R1/R2/R3 审核、冻结/解冻、理由与事件。
5. 禁止直接 advance 绕过里程碑提交/评审；甲方 S2/S5/S6 和自拟用户验证/交付/迭代的结构化证据。
6. AI 任务真实调用、项目上下文隔离、四种人工审阅、版本、成本、额度、超时/重试和单任务失败隔离。
7. 成果私有文件直链越权、跨项目 artifact/version/submission ID 混用。
8. 结项/转向/暂停/终止/复盘及归档恢复。
9. 各区块错误/空/加载、重复点击、窄屏/桌面/WebView。
10. 列表与工作台 P95、分页、备份恢复。

## 6. 当前 git diff 专项结论

- `server/internal/handler/vopc_handler.go` 新增 `can_manage`，由 owner 或 active 的 `co_owner/platform_operator` 判定。
- `frontend/lib/providers/vopc_provider.dart` 接收 `can_manage`；`vopc_page.dart` 不再用“owner 或全局 manage capability”推断项目管理权，改信任服务端项目关系结果。这一修改方向正确，避免全局 capability 把非项目成员误显示为管理者。
- `vopc_handler_test.go` 增加 owner `can_manage:true` 验证。
- 任务空状态增加 S0 提交立项提示和真实按钮，或 S1-S8 新建首任务按钮。
- diff 无空白错误，专项测试通过。
- 尚缺 co_owner/platform_operator/member 的 `can_manage` 响应矩阵测试；且 `can_manage` 只影响前端显示，后端策略仍必须作为唯一授权依据（当前项目写接口多数已做，但学院边界仍未挂载）。

## 7. 发布前关键门禁（按顺序）

1. **立即修复学院准入三层边界**：正式路由挂 `CollegeAccess(cfg.VOPCCollegeID)`，入口由真实 access/capability 决定，所有查询和邀请对象再次校验；纠正当前错误测试期望。
2. **补 S0 项目编辑闭环**：`PUT project` + 工作台资料查看/编辑 + 项目角色授权 + 审计。
3. **统一里程碑推进通道**：移除/封闭主理人文本直推捷径，推进必须消费符合门禁的已评审提交；补量表、成果版本和豁免。
4. **实现可验收 AI 最小闭环**：岗位配置、AI task、真实模型调用、项目上下文、产出版本、人工四动作、Token/成本、额度与失败重试。
5. **落实风险与结项治理**：R1/R2/R3 审批，冻结/解冻/申诉；继续/转向/暂停/终止/结项和复盘证据。
6. **补齐任务协作与成果安全**：依赖、评论、附件、成员可提交；私有文件上传/下载鉴权和字段可见性。
7. **完善异常/空/加载与多端测试**，完成 P95、备份恢复和完整负向权限回归后才可判 P0 通过。

## 8. 最关键阻断项

**首要阻断：vOPC 正式路由和前端入口没有执行计算机学院准入，且邀请流程允许外院 active 用户加入。** 这是 PRD 明确要求的访问红线，不是体验缺口；在修复并完成 API/入口/查询/邀请全链路负向测试前，不应进入试点或发布。

紧随其后的核心闭环阻断是：**S0 无法编辑补齐、AI 虚拟员工没有真实执行与人工审阅、阶段可绕过正式评审直推 S9、风险与结项复盘无操作闭环。**

---

本轮除本文件 `pm-checklist.md` 外未修改任何项目文件。
