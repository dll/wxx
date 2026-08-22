# vOPC — 软件审核报告（Software Acceptance Report / SAR）

> 报告方：`leader-wxx`（流水线协调 / SAR 作者）
> 产品：WXX「蔚小芯」智慧学工 AI 智能体 — vOPC（虚拟 OPC 教学软件实训）模块
> 版本：vOPC PRD v1.0
> 报告日期：2026-08-22
> 依据：`docs/wxx-vopc-prd-v1.0.md`、`pm-checklist.md`、`refactor-notes.md`、`qa-report.md`、`audit-report.md`、`refactor-final-summary.md`
> 代码基线：`main@7b905c9`（最近 vOPC 相关推送后的 HEAD）

---

## 0. 重要结论（Executive Summary）

**审核结论：NO-GO（不通过验收；可继续开发，禁止上线宣称 vOPC P0 验收完成）。**

vOPC 已从最初「学院准入缺失、S0 草稿不可补齐、里程碑可文本直推、成果版本无实质门禁、无结项/风险治理」的状态，迭代为包含三层学院准入、S0 全闭环、正式里程碑评审、成果版本安全门禁、结项/异常状态机、风险治理及 R3 专项通道、私有文件鉴权下载的完整后端闭环。流水线（pm → dev → qa → review → leader）多轮串行，全部自动化门禁通过。

但 vOPC PRD v1.0 的 **P0 阶段仍有多项未实现**，且明确 **依赖真实运行环境 / 外部凭据**（AI 真实模型执行、云对象存储、病毒扫描、真实 MySQL/Turso 升级、WebView/真机、P95、备份恢复演练）。根据项目负责人决定：**不为缺运行环境项提供运行时环境，相关实现一律推迟到后期**。因此本报告以「驳回验收、待条件满足后复审」签署。

---

## 1. 审核对象与范围（Audit Scope）

| 项 | 内容 |
|---|---|
| 目标模块 | vOPC（虚拟 OPC 教学软件实训）：S0–S9 项目阶段管理与协作交付 |
| 业务基线 | `docs/wxx-vopc-prd-v1.0.md`（33 项 PRD 需求） |
| 代码范围 | `server/`（Go/Gin，MySQL 8 + Redis 生产语义，开发/测试用 SQLite）+ `frontend/`（Flutter） |
| 交付物基线 | `pm-checklist.md`（33 项核对）、`refactor-notes.md`（重构记录）、`qa-report.md`（回归报告）、`audit-report.md`（代码审核，含附录 A–D）、`refactor-final-summary.md`（流水线汇总） |
| 最近相关提交 | `3e0fd76`→`2aae19d`（学院准入/成果门禁）→`65dbaef`/`ff13e07`/`a239d9a`（结项/风险/R3 + 三缺口）→`ca05196`/`f930bd6`/`f66254b`/`7b905c9`（私有文件闭环 + scan_failed 补丁） |
| 红线 | 不改业务对外契约、能力双端同步、数据库历史迁移、部署/CI、密钥等 |

**明确排除（推迟到后期，需运行环境/凭据）：** AI 真实模型调用与执行、云对象存储、真实病毒扫描、真实 MySQL/Turso 实例升级与并发、WebView/真机、P95 压测、备份恢复演练、Flutter 全量既有 lint 清零。

---

## 2. 需求覆盖摘要（Requirement Coverage）

以 PRD v1.0 33 项 + 后续批次为维度，当前状态如下（完整逐条见 `pm-checklist.md`）：

### 2.1 已完成（通过/基础闭环）

- R 学院准入三层边界（后端中间件 + 查询前置）
- R 外院邀请拒绝 + 接受时二次复核（学院/状态/身份，原子）
- R S0 草稿创建/查看/编辑/补齐/提交，提交后禁止直接编辑
- R 正式 `/api/v1/vopc` 全路由挂载学院准入
- R 移除 S1–S9 文本直推；正式里程碑提交 + 指定评审 + 平台评审推进
- R 成果版本安全门禁：active/intended_stage/SHA-256/来源类型/数量(1-20)/去重/同项目/阶段成果类型匹配
- R 普通邀请角色白名单去除 `platform_operator`（防项目内提权）
- R 关键 SQL error/RowsAffected 检查与事务+审计原子
- R 移动端导航 5 项、vOPC 一键直达
- R 桌面/移动 fail-closed 入口准入 + `/vopc` deep-link（后端为最终安全边界）
- R 任务闭环（创建/负责人校验/验收标准/优先级/截止/状态机/审计）
- R 决策闭环（创建/列表/resolve/cancel/理由/审计）
- R 邀请闭环（创建/接受/拒绝/成员关系/审计）
- R 成果元数据 + 版本闭环
- R **结项与异常状态机**（close/pause/resume/pivot/terminate/archive，CAS+审计+权限，S9 pass→closeable，pivot 重置里程碑）
- R **风险治理最小闭环**（创建/列表、R2 双人审批 gate、freeze/unfreeze、appeal/resolve、统一写门禁、R2/R3 未批不可推进里程碑，TOCTOU 修复）
- R **R3 独立专项通道**（platform_operator ∧ 治理系统角色，双专项审批，任一 reject 封死）
- R **私有文件受控上传 + 鉴权下载**（不可猜 object_key、20MB/MIME 白名单、字段级复检、路径穿越封堵、流式+nosniff+checksum、scan_failed 拦截）

### 2.2 未完成（阻断或推迟）

- **P0-1（推迟·需模型环境）** AI 虚拟员工真实执行、项目上下文隔离、产出版本化、人工四态审阅、Token/成本/额度/超时/重试
- **P0-3（推迟·需对象存储/凭据）** 云对象存储、私有字段级隔离之上的受控上传扩展、真病毒扫描、签名 URL
- **P0-5（待实现）** 里程碑评分量表/条件通过(conditional_pass)/豁免(waiver)/甲方 S2/S5/S6 结构化证据/试点与发布审批实体
- **P0-6（推迟·需生产同构）** 真实 MySQL/Turso 升级与并发、WebView/真机、P95、备份恢复、全量 Flutter lint 清零
- 工作台信息完整区块、首页八区块、项目 AI 上下文/跨项目复用授权等（依赖上述 P0）
- 残留小项：`platform_operator` 项目角色 provisioning API、正式里程碑 TOCTOU 回滚断言的 submission=pending（均已记录，非 P0 阻断）

---

## 3. 代码质量与结构（Code Quality & Architecture）

- 分层戒律保持：handler → service → repository（新增 close/risk/file 均遵循既有分层/事务模式）。
- 新模块文件划分清晰：`vopc_handler.go`（项目/S0）、`vopc_delivery.go`（协作/交付/里程碑）、`vopc_decisions.go`、`vopc_close.go`（结项状态机）、`vopc_risk.go`（风险治理）、`vopc_files.go`（私有文件）。
- 迁移：`097`（P0 骨架）、`098`（决策）、`099`（协作交付）、`100`（成果版本门禁列）、`101`（结项状态机）、`102`（风险治理）、`103`（私有文件表）。均 SQLite/MySQL 方言兼容、幂等、未回退历史迁移。
- 安全实践：fail-closed 权限、事务+审计原子（失败回滚+删盘）、IDOR/路径穿越/枚举封堵、密钥不可猜、未授权一律 404（不泄漏状态）。

---

## 4. 测试与验收门禁（Test & Gate Results）

### 4.1 自动化与静态门禁（最近已执行，真实 exit 0）

| 门禁 | 结果 |
|---|---|
| `go build ./...`（server） | PASS |
| `go vet ./...`（server） | PASS |
| `go test ./... -count=1`（server 全量） | PASS |
| `go test ./internal/handler -run 'Test.*VOPC' -count=1` | PASS |
| `go test ./internal/db -run TestToMySQLVOPCMigrations -count=1`（097–103 方言） | PASS |
| `go test ./pkg/app`（路由/迁移幂等/建 schema） | PASS |
| `flutter test`（前端，含新增 vOPC access/provider） | PASS |
| vOPC 定向 `flutter analyze` | PASS（No issues found） |
| `flutter build web --release` | PASS |
| `git diff --check` / `git diff --cached --check` | PASS |

### 4.2 核心回归覆盖（按模块）

- 学院准入矩阵（401/403/200）、外院邀请、接受二次复核原子
- S0 全流程 + 提交后禁止编辑
- 正式里程碑逐阶段（S2–S9）独立成果版本 + 评审 pass/return
- 成果版本负向（空/重复/跨阶段/失效/数量上限/跨项目/成果类型不符）
- 结项状态机（pause/resume/pivot/terminate/archive、非法流转、越权）
- 风险治理（R2 双人、重复审批 409、未批 409、freeze/unfreeze、appeal/resolve、R3 专项、TOCTOU）
- 私有文件（上传权限/413/415、下载 404/403、scan_failed 409、里程碑 file 集成、路径注入）

---

## 5. 代码审核结论（Code Review Summary）

来源：`audit-report.md`（严格只读，独立复核，附录 A–D）。多轮 reviewer 独立验证，非采信 QA/开发自证。

**已关闭的高危/关键及安全项（经独立证据确认）：**
1. platform_operator 可由普通邀请授予的项目内提权（旧 H-1）
2. 同一占位成果版本贯穿 S2–S9（版本门禁范围）
3. 关键 SQL 忽略 error/RowsAffected
4. TOCTOU：里程碑 pass 推进前未复检风险门禁（H-B1）
5. `risk_frozen` 未统一拦截各业务写（H-B2）
6. R3 专项角色不可达（M-B1）
7. 私有文件 download 不拦 `scan_failed`（本轮已补 patch `7b905c9`）

**审核判定：** 已实现范围质量合格、无高危绕过；整体 NO-GO 仅因未实现/推迟的 P0 存在，不因已实现代码有阻断性缺陷。

**残留（非 P0 阻断，已记录）：**
- `platform_operator` 项目角色 provisioning API 缺失（R2/R3 共享既有角色边界）
- 里程碑 TOCTOU 回滚断言未覆盖 submission=pending；新路由未纳入 `TestRouteRegistrationCount` 锚点
- 测试中 checksum/object 真实性依赖客户端声明（需真实对象存储后才能真验证）

---

## 6. 风险与待办（Residual Risks & Backlog）

### 6.1 风险清单

| 等级 | 项 |
|---|---|
| P0（推迟/未实现） | AI 真实执行与上下文隔离；里程碑完整业务门禁（评分/条件通过/豁免/甲方证据）；私有文件真云对象存储与病毒扫描；生产同构与多端验收 |
| 中 | `platform_operator` provisioning API 缺失；测试对「外部真实性」的自证边界 |
| 低 | scan_failed 由外部扫描器写入（当前无写入方）；MIME 依赖声明非内容嗅探（nosniff 已缓解）；全量 Flutter lint 未清零 |

### 6.2 建议下一批（按减小风险排序，均不依赖运行环境）

1. `platform_operator` 项目角色 provisioning API（消除 R2/R3 共享边界）+ 补里程碑 TOCTOU 回滚断言 + 新路由纳入路由计数锚点
2. 里程碑评分量表/conditional pass/waiver/甲方结构化证据实体（纯数据模型+API+测试）
3. 待项目提供运行环境后：AI 闭环、云对象存储/病毒扫描、真实 MySQL/Turso、多端/WebView、P95、备份恢复

---

## 7. 签署（Sign-off）

| 审核人 | 结论 |
|---|---|
| PM（pm-wxx） | 需求核对完成；P0 未全实现 |
| 开发（dev-refactor-wxx） | 已实现范围全绿；`[blocked]` 如实标注 |
| QA（qa-regression-wxx） | 最近批次 GO；整体 NO-GO |
| 审核（reviewer-audit-wxx） | 多轮只读独立复核；整体 NO-GO |
| 审批（leader-wxx） | **NOT APPROVED / NO-GO** |

**原因：** vOPC PRD v1.0 的 P0 中，AI 虚拟员工、里程碑完整业务门禁、私有文件真云存储与生产同构验收尚未实现；其中依赖真实运行环境的项已按负责人决定推迟到后期。已实现范围质量合格、门禁全绿。

**复审核验条件（届时需满足）：**
1. 提供模型/对象存储运行环境后，完成 AI 执行与私有文件受控上传/鉴权下载的真实验收
2. 补齐里程碑评分/条件通过/豁免/甲方结构化证据
3. 完成真实 MySQL/Turso 升级、并发、备份恢复、多端/WebView、P95 与全量 Flutter lint 治理
4. 关闭 residual 中/低风险之后，由流水线重新执行 pm→dev→qa→review 并复签

---

*本 SAR 由 `leader-wxx` 汇总 pm-checklist/refactor-notes/qa-report/audit-report/final-summary 及 git 变更而成；不新增技术判断，仅整合结论与待办。*
