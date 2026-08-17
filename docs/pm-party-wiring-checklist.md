# P0-2 党建接线 · 需求核对清单（pm-wxx）

> 核对人：pm-wxx（只读核对专员，未修改任何源码）
> 核对日期：2026-08-17
> 核对对象：蔚小芯（WXX）智慧学工 AI 智能体 · P0-2 党建接线
> 核对基线：代码实证（`capabilities.go` / `routes.go` / `secretary_outcome_*` / 迁移 024/089 / 前端 `frontend/lib`）+ 既有审核文档
> 诚实声明：以下各项均以「真实代码/表/路由是否存在」为判定，不臆断；凡「数据为空」均如实标注，不视为缺陷。

---

## 〇、一句话结论

**P0-2 党建接线在 2026-08-15/16 已落地并接线完成（后端 + 能力门控 + 前端页 + 双端常量 + 回归测试全部齐备）**，与 `发布审核报告v2-2026-08-16`「党建育人闭环（本轮完成）」结论一致。
- 剩余的**不是「缺接线」，而是「缺真实数据」**：`party_progress` / `party_study_records` 生产当前 0 行（诚实空，红线要求不造数据）。
- 还有 2 项「口径待定夺」（蓝图待确认#3 协同育人口径；辅导员/书记党建推进痕迹为后续扩展），属规划而非阻塞。

> ⚠️ 重要更新：我核对前默认「待接线」，逐项核对后发现**代码已闭合**（`party-dashboard-wiring-2026-08-16.md` 是落地式文档，release v2 已实测冒烟通过）。故本清单以**「核对确认」**为主，「待办」收敛到很小。

---

## 一、目标 / 范围

| 需求陈述 | 状态 | 核实依据 |
|---|---|---|
| 学院书记（college_admin，owner_scope=college，owner_id=角色归属）→ 只看本院党建数据 | ✅ 已落地 | handler 自动按角色回填 ownerID；repo 按 `users.owner_id` 过滤 |
| 学校书记（school_admin）→ 看全校（owner_id 空） | ✅ 已落地 | ownerID 为空 → 全校，无范围过滤 |
| 党建漏斗：申请书→积极分子→发展对象→预备党员→转正，各阶段人数 | ✅ 已落地 | repo 按 `party_progress.current_stage GROUP BY`；前端中文映射全（见下） |
| 党员数：正式党员 / 预备党员 | ✅ 已落地 | `status IN ('member','probation') GROUP BY status` |
| 党课学习：总人次 / 总时长(小时) / 按 study_type 分布 | ✅ 已落地 | repo 对 `party_study_records` COUNT + SUM(duration) + GROUP BY study_type |
| 诚实 data_source 标注（0 行=not_available；有行=self_reported，不伪装组织确认） | ✅ 已落地 | `partyDataSource(stageTotal, studyCount)` + 前端 `DataSrcBadge` |

**五阶段代码 → 中文口径**（`party_stages` 种子数据 + 前端映射一致）：
`applicant`=提交入党申请书 → `activist`=入党积极分子 → `development`=发展对象 → `probation`=预备党员 → `member`=正式党员（前端 L127-131、迁移 024 L259-265 完全对齐）。

---

## 二、数据表字段与缺口

### 2.1 实际表结构（迁移 024 + 089，均已建/加列）

**`party_stages`**（阶段字典）：`code（唯一）/ name / description / required_docs(JSON) / sort_order` — 种子 5 条已插（`INSERT OR IGNORE`），`sort_order` 1-5。

**`party_progress`**（入党漏斗 + 党员数）：
`id / user_id / student_id / student_name / college / current_stage / apply_date / activator_date / development_date / probation_start / conversion_date / status(applicant|activist|development|probation|member) / party_member_id(介绍人) / branch_secretary(支部书记) / notes / created_at / updated_at`
- 索引：`idx_party_progress_user_id`、`idx_party_progress_status`。
- 字段与书记职能一一对应（发展党员/介绍人/支部书记/各阶段日期）。

**`party_study_records`**（党课学习 + 组织登记）：
`id / user_id / study_type(theory|practice|meeting|volunteer) / title / content / duration(分钟) / study_date / certificate / status / created_at`
- **089 追加列**：`created_by BIGINT NULL`、`created_by_role VARCHAR(128) NULL`、`paid INTEGER NULL DEFAULT 0`；索引 `idx_party_study_created_by`。
- 索引：`idx_party_study_user_id`。

### 2.2 学生自报 vs 组织登记 如何区分
- **`party_progress`**：目前仅学生自报/意向登记（SelfPartyRead/Write），无组织侧写入路径；书记看板据此标 `self_reported`。**注意**：`party_progress.college` 存中文学名，与学院书记短码（`cs`）对不上 —— 这正是不用该列做本院范围过滤的原因（`/agent-review/party-dashboard-wiring` 已明示「关键修复」）。
- **`party_study_records`**：`created_by IS NULL` = 学生自报原样兼容；`created_by > 0` + `created_by_role` = 教师/教辅/组织侧登记（党建登记走这条路）。

### 2.3 data_source 诚实标注机制
- `partyDataSource(阶段总, 学习总)`：全 0 → `not_available`（未接入真实党建数据）；有行 → `self_reported`（当前为学生自报/意向登记，非组织确认）。
- 前端 `DataSrcBadge` 三态诚实边界，空态页显示「暂无党建育人数据（数据待充实）」，**不编造数字**。
- Migion 089 决策（待确认#2 已定）：复用 `party_study_records` 加列，**不**独立新表，登记走现有聚合、书记看板立即可见，不新造绩效。

---

## 三、能力门控是否已具备

### 3.1 已注册能力（`server/internal/auth/capabilities.go`，代码源真）
| 能力 | 注册角色 | 说明 |
|---|---|---|
| `self.party.read` / `self.party.write` | 全体 student | 学生自报（党建教育查看/操作） |
| `party.record.write` / `party.record.read` | counselor / teacher / assistant | 党课/活动登记（2026-08-16 新增） |
| `college.collab.dashboard` | college_admin（继承） | 协同育人总览（2026-08-16） |
| `outcome.dashboard` | college_admin（school_admin 继承） | 教育成果/党建看板门控（**复用**，未新增） |
| `outcome.record.write/read/review` | teacher/counselor/assistant + student(自报) | 毕业去向（党建隶属书记教育成果域） |
| `counselor.secondclass.board` | counselor | 第二课堂班级看板（同书记育人域，非本项核心） |

### 3.2 路由接线（`server/pkg/app/routes.go`）
| 方法与路径 | 能力门控 | 版本 |
|---|---|---|
| `GET /api/v1/college/party-dashboard` | `outcome.dashboard` | 书记（本院/全校） |
| `GET /api/v1/college/collab-dashboard` | `college.collab.dashboard` | 书记 |
| `POST /api/v1/teacher/party/register` | `party.record.write` | 教师/教辅 |
| `GET /api/v1/teacher/party/records` | `party.record.read` | 教师/教辅 |
| `DELETE /api/v1/teacher/party/records/:id` | `party.record.write` | 教师/教辅（删本人登记） |
| `GET /api/v1/college/nurture-kpi` | `outcome.dashboard` | 书记（D5-1 联动） |
| `GET /api/v1/college/education-outcome` | `outcome.dashboard` | 书记大屏 |

### 3.3 书记侧还缺什么能力/路由？
- **无阻塞缺口**。书记侧党建看板/协同育人/登记/审核均已有能力 + 路由。
- 可选项（非本项 P0-2 必做）：
  - 「入党组织发展推进」（介绍人确认、支部书记阶段更新带操作痕迹）——蓝图闭环第 3 块，**尚无独立能力/路由**，属后续扩展。
  - 辅导员/学生侧党史教育「党员数确认」无组织侧审批流（`party_progress.status` 无 `approved` 语义），如需「组织确认」需新增登记/审核能力（当前诚实标 self_reported 已覆盖不造假）。

### 3.4 handler 权限派生（`secretary_outcome_handler.go`）
- `PartyDashboard` / `CollabDashboard` / `NurtureKPI` / `OutcomeDashboard`：`owner_id` 缺省时，`college_admin` 取 `u.OwnerID`（本院），`school_admin` 留空（全校）。与蓝图待确认#1 口径一致。

---

## 四、双端同步约束（能力常量 ↔ 前端）

**已满足**。`frontend/lib/utils/capability_utils.dart` 已登记（L98-102）：
- `outcomeDashboard = 'outcome.dashboard'`
- `partyRecordWrite = 'party.record.write'`
- `partyRecordRead = 'party.record.read'`
- `collabDashboard = 'college.collab.dashboard'`

> **约束提示（给 dev）**：今后**凡是后端 `capabilities.go` 新增能力常量，必须在 `capability_utils.dart` 同步登记**，否则前端能力判断漏配。本项四个常量已双端对齐，无遗漏。

---

## 五、明确【禁止触碰】红线（诚实边界）

| 红线 | 现状 | 是否触碰过 |
|---|---|---|
| 不造真实党建数据（party 表 0 行保持诚实空） | 生产 `party_progress=0`、`party_study_records=0`（`079_clear_preloaded_data` 已清测试数据） | 否，符合 |
| 不改变既有 party 数据语义（`party_progress`/`party_study_records` 原表复用，仅加列） | 089 仅 `ADD COLUMN`（created_by/created_by_role/paid），未改既有列类型/含义 | 否 |
| 不引入本地大模型 | 党建看板为纯 SQL 聚合，无本地推理模型 | 否 |
| 不把自报当组织确认 | `data_source=self_reported`/`not_available` 明确区分；毕业去向另有 `approved` 审核态 | 否 |
| 归因不做因果断言 | 协同育人/党建均为「真实记录 + 按学院聚合」，只趋势/相关性 | 否 |

---

## 六、落地动作拆解（给 dev 的核对结论）

> 因接线代码已全部就位，本项从「待开发」降级为「确认 + 收尾」。按优先级排列：

### P0（无，已完成项确认）
- [x] 端点 `GET /college/party-dashboard`（本院/全校范围、五阶段漏斗、党员数、学习人次/时长/类型分布）— 代码实证在 `secretary_outcome_repo.go` `PartyDashboard`。
- [x] 端点 `POST/GET/DELETE /teacher/party/*`（党课/活动登记，落 `party_study_records.created_by`）。
- [x] 端点 `GET /college/collab-dashboard`（协同育人总览：谈心/后勤/党建/排课按学院聚合）。
- [x] 能力门控四常量双端登记。
- [x] 前端 3 页：`party_dashboard_page.dart` / `collab_dashboard_page.dart` / 既有 `secretary_outcome_dashboard_page.dart` 党建块 + `PartyDashboardSection`/`CollabDashboardSection` + `DataSrcBadge` 空态展示。
- [x] 回归测试 `d1_1_party_collab_test.dart`（空态不伪造 + 数据态标 source）。

### P1（建议收尾，非阻塞）
1. **生产部署二次确认**（优先级 P1，依赖：CI/CD）：确认最新二进制已含本项全部路由/能力；`089` 迁移在生产库已应用（release v2 标注已应用，建议抽查 `party_study_records` 是否有 `created_by` 列 + 索引）。
2. **文档回填**（P1）：`docs/蔚小芯待完成.md` 仍停留在 2026-08-10，**未收录党建接线已完成**，建议补注；避免后续 reviewer/次轮 agent 误以为待做。

### P2（后续蓝图扩展，非 P0-2 范围内的既有缺口）
3. **入党组织发展推进痕迹**（P2）：介绍人确认、支部书记阶段更新 + 操作人/操作角色记录（当前 `party_progress` 无操作痕迹列、无审批流）。方案倾向复用 `audit_logs` 或加操作列，**需先定口径**（蓝图「待确认」未定，勿擅造）。
4. **协同育人口径最终定夺**（P2，蓝图待确认#3）：当前按「真实记录 + 按学院聚合」实现（含学生侧？是否只统计教师/教辅真实动作），口径未最终拍板 —— 待用户确认后是否调整过滤。
5. **党建真实数据导入/登记入口**（P2，数据依赖）：`party_progress` 现阶段仅学生自报；若要让书记看到「正式党员/预备党员」真数字，需组织侧登记或导入（**等真实数据，不造假**）。

---

## 七、核实不确/风险提示

- **「030」迁移缺号**：migrations 目录从 029 直接到 031，无 030 —— 已确认与本项无关（党建表在 024/089）。
- **`party_progress.college` 中文学名陷阱**：本院范围过滤已用 `users.owner_id` 规避；**dev 勿改用 `party_progress.college` 做学院过滤**，否则学院书记本院永远查不到。
- **测试数据清理**：production 为 0 行是诚实状态，若未来要演示漏斗需**真实数据**，严禁 seed 虚构党建数字。

---

*生成：pm-party-wiring-checklist.md · 仅供需求核对参考，不override任何源码。*
