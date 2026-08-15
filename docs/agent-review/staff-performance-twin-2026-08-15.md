# 教辅/教师绩效画像（第一增量：绩效 → 数字孪生画像 → 三方绑定）

日期：2026-08-15 · 提交：见文末 · 审核人：wxx-agent

## 背景与目标

用户核心诉求：**功能必须与教辅/辅导员日常工作强关联才有人用**。关联机制设计为——

> 教辅/教师使用蔚小芯功能 → 每次使用产生真实行为/审计记录 → 绩效汇聚到「数字孪生画像」
> → 画像上绑定 教师 / 学生 / 蔚小芯 三方 → 绩效被看见 → 接着用 → 更多数据进画像（强关联闭环）

用户已确认方向：**在现有独立画像基础上加"帮扶、咨询"等绩效维度，复用现有画像机制，顺序实现**（方案 A），并非新建一页。

## 关键事实核对（不瞎编原则）

1. **现有数字孪生是纯学生侧**：`twin_service.go` 固定五维（学业/能力/思想/情感/社交），
   数据全部来自 `AggregateRawMetrics`（成绩/竞赛/党建/情感/社团），无教辅绩效、无绑定。
2. **可复用真实数据源**（本次核实）：
   - `talk_records`：谈心记录表，含 `counselor_id` / `student_id` / `created_at`——**帮扶/咨询的现成真实数据**，
     按 `counselor_id` 聚合可得帮扶数，按 `student_id` 去重可得「服务学生绑定数」。
   - `audit_logs`：审计中间件（`middleware/audit.go`）在**每个请求后异步写入**，
     存 `user_id / role / action(GET/POST) / resource(路由 fullPath)`——**每次蔚小芯功能调用的现成真实记录**，
     按 `resource LIKE '%/assistant/%'` 可聚合排课/考试/通知/材料的调用数，「使用过的蔚小芯能力去重数」即「蔚小芯绑定」。
   - **无教师协作独立表**：协作教师绑定无真实数据源 → 诚实返回 `DataAvailable=false`，不编造。
3. **性能画像入口此前缺失**：`数字孪生`入口在 profile 里被 `if (role=='student'||role=='student_union')` 门控，
   教辅/教师登录看不到 → 本增量补三个角色块入口。

## 实现内容

### 后端（全部真实数据，无硬编码示例）
- `twin_repo.go`：`StaffTwinMetrics` + `AggregateStaffMetrics(userID)`
  - 帮扶/咨询 ← `talk_records WHERE counselor_id=?`
  - 排课处理 ← `audit_logs resource LIKE '%/assistant/schedule-check'`
  - 考试编排 ← `…'/assistant/exam-arrange'`
  - 通知发布 ← `…'/assistant/notification'`
  - 材料产出 ← `…'/assistant/material-templates' OR …'/assistant/doc-process'`
  - 蔚小芯使用 ← `resource LIKE '%/assistant/%' OR '%/counselor/%'`
  - 服务学生绑定 ← `talk_records` 去重 `student_id>0`
  - 蔚小芯能力绑定 ← `audit_logs` 去重 `resource`
- `twin_service.go`：`computeStaffDimensions` + `GetStaffTwin`
  - 6 绩效维度：帮扶咨询 / 排课处理 / 考试编排 / 通知发布 / 材料产出 / 蔚小芯使用
  - 3 三方绑定：服务学生 / 蔚小芯能力 /（协作教师=DataAvailable=false 诚实兜底）
  - 打分规则（透明、可解释、不编造）：`Score = min(100, 真实次数)`，`DataAvailable = 次数>0`，
    无记录时前端显示「数据积累中」；综合分为有数据维度均值。
- `student_handler.go`：`DigitalTwin` 按 `userCtx.Role` 分流——教辅/教师（counselor/teacher/assistant）走
  `GetStaffTwin`，学生/学生会走原 `GetDigitalTwin`。新增 `isStaffRole` helper。
  - 复用 `/me/digital-twin` 与 `/student/digital-twin` 路由 + `SelfTwinRead` 门控
    （staff 经 `student_union→student` 父链继承 `SelfTwinRead`，**可达性核实通过**）。

### 前端
- `profile_page.dart`：counselor/teacher/assistant 三个角色块各加「绩效画像」入口
  （`/student/digital-twin`，后端按角色返回绩效画像）。
- `digital_twin_page.dart`：
  - 教辅/教师显示「绩效画像」标题 + 角色中文名（辅导员/教师/教辅），副标题说明绩效汇聚逻辑；
  - 雷达图标签随维度数自适应字号/半径（9 维不重叠）。

## 验证
- 后端：`go build -tags fts5 ./...` → **BUILD_EXIT:0**
- 前端：`flutter analyze --no-fatal-infos` → **0 error / 0 warning**（仅既有 info 级提示）
- 可达性：`SelfTwinRead` 经父链（assistant→student_union→student 等）对三类 staff 角色可见，
  profile 入口已加，路由已注册。

## 强关联收益
- 教辅用「排课检测/考试编排/通知/材料」→ 每次调用进 `audit_logs` → 绩效画像「排课处理/通知发布」维度增长；
- 辅导员用「谈心谈话」→ 每次记录进 `talk_records` → 「帮扶咨询」维度 + 「服务学生」绑定增长；
- 画像「蔚小芯使用/蔚小芯能力绑定」反映教辅对系统的使用粘性，形成**用得多→画像满→继续用**的闭环。

## 待办（后续增量，顺序实现）
- [ ] 协作教师绑定：需独立教师协作记录（如协作评教表）落库后在 `AggregateStaffMetrics` 补聚合；
- [ ] 「帮扶/咨询」可再按 `talk_records.status/emotion` 细分（当前合为一个维度，避免编造分行）；
- [ ] 强关联优先接教辅前端（学生信息查询先改真实数据、教学日历、通知批量连学生端双向闭环）；
- [ ] 降级/藏起弱关联教辅能力（music-radio / workflow-automation / activity-register）；
- [ ] 辅导员数据来源标注 / 「不瞎编」整改（后端 ~8 handler 硬编码兜底改诚实）；
- [ ] 云端部署本增量（连同 04e7682 角色管理起的未部署改动）。

## 口径提醒
本增量基于用户确认的**方案 A**（复用现有画像加绩效维度）。三份权威角色需求文档口径仍不一致
（教辅 5 项 vs 12 项、辅导员预警 P0 是否列入），未合并前以「用户口头确认的强关联三档分析」为执行基准。
