# 学生会功能闭环审核（严格）

- 日期：2026-08-15
- 角色：student_union 学生会
- 方法：逐个功能查 前端页面→后端接口→数据表→数据流向，判断闭环是否完整、是否有断链/假数据

---

## 一、学生会能力全景与闭环状态

| 功能 | 前端页面 | 后端接口 | 数据落库 | 闭环结论 |
|------|---------|---------|---------|---------|
| AI 活动策划 | event_plan_page ✅ | /union/event-plan ✅ | ❌ 不落库(仅AI生成文本) | ⚠️ 单向(只出方案,不进活动库) |
| AI 海报文案 | poster_gen_page ✅ | /union/poster-gen ✅ | ❌ | ⚠️ 单向 |
| **活动创建(新建)** | activity_manage_page ✅ | POST /health/activities ✅ | ✅ health_activities | ✅ 有闭环 |
| **活动报名** | 学生端(health) → | POST /health/activities/:id/signup ✅ | ✅ health_activity_signups | ✅ 有闭环 |
| **活动统计** | activity_manage 显示 signup_count | health/activities(List带count) ✅ | ✅ 真实 | ✅ 有闭环 |
| 招新 Recruitment | ❌ 无页面 | /union/recruitment ⚠️AI生成 | ❌ | ❌ 断链+假兜底 |
| 成员管理 MemberManage | ❌ 无页面 | /union/member-manage ⚠️兜底假数据 | ❌ | ❌ 断链+假兜底 |
| 问卷 Questionnaire | ❌ 无页面 | /union/questionnaire ⚠️兜底假数据 | ❌ | ❌ 断链+假兜底 |
| 热点追踪 HotTopicTrack | ❌ 无页面(学生端hot_topics另立) | /union/hot-topic-track ⚠️兜底假数据 | ❌ | ⚠️ 断链(重复实现) |
| 活动分析 ActivityAnalysis | ❌ 无页面 | /union/activity-analysis ⚠️兜底假数据(0.85/0.72) | ❌ | ❌ 断链+假数 |

---

## 二、你问的核心：活动"发起是新建还是导入？蔚小芯如何协助开展与统计？"

### 现状（真实链路，只对"健康活动 health_activities"闭环）
1. **发起**：学生会活动管理页「新建活动」（activity_manage_page，POST /health/activities）——**是"新建"，非导入**
2. **协助开展**：AI 活动策划（event_plan）生成方案**但不会自动变成活动**（断链一：策划→创建是两个独立动作，信息不互通）
3. **报名**：学生端活动列表点报名（health/activities + signup）✅ 真实落库
4. **统计**：活动管理页显示 报名人数/容量（ListHealthActivities 聚合 signup_count）✅ 真实

### 关键缺陷（闭环断点）
- **断链 A**：AI 策划(/union/event-plan) 生成的方案 **不落库、不自动一键转成活动**（health_activities）→ 学生会要手动重新"新建"填一遍
- **断链 B**：5 个后端接口（招新/成员/问卷/热点/分析）**无前端页面、无 ApiConfig 常量** → 功能不可达。且无 LLM 时**返回编造的假数据**（张明/45小时/参与度0.85），违反"准确性第一"
- **断链 C**：活动分析(ActivityAnalysis) 是 mock 假数据（reg_rate 0.85/attend_rate 0.72），**没有从真实 health_activity_signups 聚合覆盖/到场率**
- **设计不一致**：存在两套活动体系——`health_activities`(通用活动) 与 `club_activities`(社团活动)，前端 activity_manage 只接 health；社团活动是另一套、无学生会入口

---

## 三、审核结论与修复建议

### P0（建议尽快）
1. **补齐 5 个断链后端接口的前端**：招新/成员管理/问卷/活动分析 —— 至少给"活动分析"做前端(或接入真实统计)
2. **活动分析改用真实数据**：ActivityAnalysis 从 health_activity_signups 聚合覆盖面/到场(如有签到)，**去掉假兜底**
3. **策划→创建打通**：AI 策划输出一键"转成健康活动"(预填表单)

### P1
4. **清理假兜底**：member-manage/questionnaire 等返回 mock 前，明确标注"演示数据"或接真实表
5. **统一活动体系**：health_activities 与 club_activities 合并或明确二选一，否则活动数据两套、统计分裂

### 已合规的部分（保留）
- 活动 新建→报名→统计（health_activities）链路真实、闭环 ✅
- 角色权限：student_union 可新建活动（UnionEventPlan 能力）✅

---

## 四、风险提示
- mock 假数据兜底在"活动分析/成员管理/问卷"存在，若上线误展示会给学生会误导

---

## 五、修复跟进（2026-08-15 已落地，commit 16f1aa1）

### 完成项
- **P0-1 补齐 5 个断链接口前端** ✅：新增「学生会工作台」页（`/union/workbench`，union_workbench_page.dart）集中呈现 成员活跃/活动分析/招新/问卷/热点 5 功能；profile 学生会服务新增入口；路由 / ApiConfig / HealthProvider 打通。
- **P0-2 活动分析改真实数据** ✅：`UnionService` 注入 `db *sql.DB`（`NewUnionService(db, llmClient)`，app.go 无条件构建，不再依赖 LLM）；`AnalyzeActivity` 从 health_activities + health_activity_signups 聚合真实报名率/到场率（含 attended 字段，迁移 084）；去掉硬编码 0.85/0.72 假兜底。
- **P0-3 成员管理真实数据** ✅：`ManageMembers` 按 health_activity_signups JOIN users 真实聚合（参与次数/到场/performance 评级/建议）。
- **结构统一** ✅：五个 /union/* 接口统一返回 `{code,message,data}` 包装（与原 API 约定一致），前端按 `res.data['data']` 解析。
- **数据来源诚实标注**：真实统计时 `data_source=real`；空库/无对应数据时 `data_source=reference`（返回 0/空，不编造张明/0.85 等假值）。

### 验证
- 本机 SQLite 冒烟：member-manage / activity-analysis 真实路径 `data_source=real` 返回真实聚合（报名3/到场2→到场率66.67%，建议树出）。
- 云端部署（服务器源码 reset 到 16f1aa1 + 本地编译 + systemd 重启）：health healthy（mysql 驱动）；登录 stunion/Wxx@2026 code=0；/union/member-manage、activity-analysis、recruitment、questionnaire、hot-topic-track 全部 HTTP 200 code=0。
- 云端 MySQL 空库时 member/analysis 返回空(reference)属正常，待真实活动+报名产生数据后自动转 real。

### 未做/超出范围
- `club_activities` 社团活动体系仍未合并（P1-5），学生会活动仍统一走 health_activities。
- 活动分析建议解析：后端 Suggestions 为 `[]string`（前端勿 cast<Map>）。
- 两套活动体系长期并存会导致统计口径混乱
