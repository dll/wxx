# 辅导员 / 教辅 角色功能审核报告

> 日期：2026-08-15
> 审核对象：counselor（辅导员）、assistant（教辅）角色的功能完整性、数据真实性、入口可达性
> 状态：审核完成；教辅数据来源标注已落地（本增量）；辅导员口头规划见「后续」

## 一、入口可达性 ✅（无缺口）

- **辅导员**：profile「辅导员服务」含 13 项（今日关注/班级学情日报/数字孪生看板/预测性预警/AI干预/谈心谈话/AI话术/思想档案/班级画像/社区管理/热点感知/办事管理/学生列表），路由全部注册，均有入口。
- **教辅**：profile「教辅服务」3 项（排课冲突检测/毕业资格审核/考试编排），路由全部注册，均有入口。

> 与学生会（此前 5 个 union 接口无前端入口）不同，辅导员/教辅入口**均已存在**，非断链问题。

## 二、数据真实性（核心问题 ⚠️）

沿用「不瞎编、真实数据标注」原则逐端点核查：

### 无问题的部分 ✅
- **assistant** `ScheduleCheck` / `GradAudit` / `ExamArrange`：优先走 `phase3.GetScheduleConflicts/GetGraduationSummaries/GetExams` 真实课表/成绩/考试数据（已导入 10870 条课表+491 学生），空数据兜底为 `reference` 且**不再硬编码假学生**（2026-08-15 前曾硬编码「示例学生/王老师」伪数据，现兜底已诚实）。
- **counselor** `GetStudentList`：走 `userRepo.List(scope/ownerID)` 真实学生 + 姓名脱敏 + 范围锁定。
- **counselor** `TalkRecord`：LLM 从真实谈话输入中提取。

### 仍有编造/误导风险的部分 ⚠️
以下 counselor 接口在无真实数据时**返回硬编码示例人物/数据**且前端**无来源标注**（辅导员会误以为是真实学生）：
- `GenerateTwinBoard`（数字孪生看板）：硬编码 张明/王芳/李华 及学术/心理分数。
- `GeneratePredictions`（预测性预警）：硬编码 张明(dropout 0.35)/王芳 风险预测。
- `GenerateMonthlyBrief` / `GenerateFollowUpReminders`：硬编码 张明(学业风险)/李华(情感关注)/逾期提醒。
- `GenerateCheckinStats`：硬编码 张明 打卡中断。
- `GenerateCommunityManage` / `GenerateHotTopicSense`：硬编码假帖子/热点。
- `GenerateDailyFocus` fallback。

> 风险等级：**高**。这些是「人 + 决策」数据，若展示为真会造成辅导员对并不存在的学生做干预（如"跟进张明学业帮扶"）。

## 三、本增量实现（教辅数据来源标注）

- 新增通用组件 `frontend/lib/widgets/data_src_badge.dart`：`DataSrcBadge(src)`，「真实数据」（绿，real）/「参考/AI」（琥珀，其余），点击弹窗说明来源（对齐学生会工作台 `_srcBadge` 语义）。
- 应用到教辅 3 页：`schedule_check_page.dart` / `exam_arrange_page.dart` / `grad_audit_page.dart` 列表头展示后端 `data_source`。
- 验证：`flutter analyze` 项目级 **0 error / 0 warning**（仅新增 info 级 prefer_const，CI `--no-fatal-infos` 通过）。

## 四、后续（辅导员 补齐规划，待确认）

辅导员数据真实性标注与「不瞎编」整改涉及 provider 层 + 后端兜底改造，改动面较大，单列下一增量：
1. `twin_board` / `prediction` / `followup` / `monthly_brief` / `checkin` / `community` / `hot_topic`：把硬编码示例人物改为诚实兜底（空/参考），或注入真实数据，并统一标 `data_source`。
2. 辅导员页统一展示 `DataSrcBadge`。
3. 教辅 `GradAudit` 增加**学生选择器**（当前默认取 `summaries[0]`，需新增学生列表能力给 assistant 角色）。

> 是否按此继续辅导员增量？（改动涉及后端 ~8 个 handler/服务方法 + 前端 ~8 页，一次性完成约需一次跨端提交。）
