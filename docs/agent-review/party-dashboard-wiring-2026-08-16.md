# 党建育人接线：书记党建聚合看板（2026-08-16）

> 对应蓝图 `secretary-party-closed-loop-2026-08-15.md` 的「汇流」落地第一块：
> 把已有的 `party_progress`(入党漏斗) / `party_study_records`(党课学习) 统计**接到书记侧入口**。

## 口径（用户已确认待确认#1）
- **学院书记**（college_admin，owner_scope=college，owner_id=`cs`）→ 只看**本院**数据
- **学校书记**（school_admin）→ 看**全校**（owner_id 空）

## 后端
新增 `SecretaryOutcomeRepo.PartyDashboard(ownerID)`：
- **① 入党漏斗**：`party_progress.current_stage` 分组（applicant/activist/development/probation/member）+ 总人数
- **② 党员数**：`status IN ('member','probation')` 分组（正式党员 / 预备党员）
- **③ 学习记录**：`party_study_records` 总人次 + 总时长(小时) + 按 `study_type` 分布（theory/practice/meeting/volunteer）
- **本院范围过滤**：通过 `JOIN users u ON u.id = pp.user_id WHERE u.owner_id = ?` —— **关键修复**：不用 `party_progress.college`（存中文学名，与学院书记的短码 `cs` 对不上，导致旧 EducationOutcomeDashboard 的 party 块本院永远查不到）
- **诚实 data_source**：`partyDataSource()` —— 0 行=`not_available`（未接入真实党建数据）；有行=`self_reported`（当前为学生自报/意向登记，非组织确认，符合「不瞎编」红线）

新增端点：`GET /api/v1/college/party-dashboard`（权限复用 `outcome.dashboard`，书记已有）
- handler 自动按角色归属：college_admin 用 `u.OwnerID`，school_admin 全校
- service `PartyDashboard(ctx, ownerID)` 透传

## 前端
- `SecretaryProvider`：新增 `partyDashboard` 状态 + `fetchPartyDashboard()`（调 `/college/party-dashboard`）
- `secretary_outcome_dashboard_page.dart`：`_buildParty` 改用党建聚合看板（优先 `partyDashboard`，回退旧 `d['party']`），展示：入党申请总人数 / 正式党员 / 预备党员 / 阶段漏斗（中文名）/ 党课学习人次 / 学习时长 / 按类型人次；DataSrcBadge 标注 `data_source`（not_available/self_reported）；下拉刷新一并刷新党建看板

## 验证
- gofmt 干净 / go vet 0 / go build 0 / flutter analyze 全项目 0 error 0 warning（我改的两个文件仅 3 条既有 info lint）
- 生产库 SQL 复现查询全部有效（party 表当前空 → 返回 0，`data_source=not_available` 诚实显示「待党建真实数据」）

## 后续（蓝图其余两块的接线，未在本轮做）
- **党课/积极分子活动登记**（教师/教辅侧新增登记，补 `created_by` 维度）—— 蓝图待确认#2：独立新表 vs 复用 `party_study_records` 加列
- **协同育人总览**（谈心/后勤/排课/答疑按学院聚合给书记）—— 蓝图待确认#3 口径
- **入党组织发展推进**（介绍人/支部书记操作痕迹）
