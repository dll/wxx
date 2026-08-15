# 强关联教辅前端 + 弱关联降级 + 辅导员「不瞎编」整改

日期：2026-08-15 · 提交：见文末 · 审核人：wxx-agent
前置：staff-performance-twin-2026-08-15.md（绩效画像第一增量，cb4f78e）

## 用户确认
- **保留内置演示账号**（便于演示/验收/回退；属数据/运维层，代码不强制覆盖）。
- 完成 2/3/4 三项。

## 2️⃣ 强关联优先接教辅前端（后端已注册、此前缺前端页）
新增 3 个强关联教辅前端页 + profile 入口 + 路由：

| 页面 | 路由 | 数据源 |
|---|---|---|
| 教学日历 `teaching_calendar_page.dart` | `/assistant/teaching-calendar` | 后端 `reference`（学期节点为教务性参照，真实接入后自动更新）+ DataSrcBadge |
| 学生信息查询 `student_info_page.dart` | `/assistant/student-info` | **真实学生账号**（后端改为 users 表，`data_source=real`，去掉硬编码张明/李华）|
| 通知批量 `notification_draft_page.dart` | `/assistant/notification-draft` | 后端 AI 草稿（`reference`）+ DataSrcBadge + 「发布前人工核对」提示 |

配套：`api_config.dart` 加 3 个端点常量；`router.dart` 加 3 条路由；`profile_page.dart` 教辅块加 3 个入口。

## 3️⃣ 弱关联降级/藏起
music-radio / workflow-automation / activity-register **在 profile 教辅块从未出现**（profile 仅列排课/毕业/考试 + 本次新增强关联 + 绩效画像），无前端页、无导航入口 → 天然「藏起」。审核文档明确其定位为弱关联，暂不建前端，避免占用教辅主界面。

## 4️⃣ 辅导员「不瞎编」整改（后端硬编码示例 → 诚实兜底）
| 功能 | 整改前 | 整改后 |
|---|---|---|
| 谈话跟进提醒 `GenerateFollowUpReminders` | 硬编码张明/李华/王芳/赵强 + 伪造逾期 | **接真实 talk_records**（status=following 的学生生成真实跟进；逾期按记录时间推算；无记录则诚实空 + 提示）|
| 风险预测 `fallbackPredictions` | 伪造张明 dropout 0.35 | 返回**空**（无真实预警不虚构学生，前端「暂无预警」）|
| 数字孪生看板 `fallbackTwinBoard` | 伪造张明/李华/王芳 | 返回**空**（无真实画像不虚构）|
| 班级打卡统计 `GenerateCheckinStats` | 伪造 45 人/93%/张明李华中断 | 返回**空 + 诚实说明**（无班级级真实聚合不虚构）|
| 学生信息查询 `QueryStudentInfo` | 硬编码张明/李华 | **查真实 users 表**（role=student，按姓名/学号/班级/专业 LIKE），无命中返回空 |

关键原则：**预测/看板/统计类功能在无真实数据时返回空 + 前端空态（「暂无/数据积累中」），绝不用示例学生伪装成真实风险/画像**。学生信息查询接真实账号，是全流程真实化样板。

## 验证
- 后端 `go build -tags fts5 ./...` → BUILD_EXIT:0
- 前端 `flutter analyze --no-fatal-infos` → **0 error / 0 warning**
- 修复：`DropdownButtonFormField` 3.22 用 `value:`（非 `initialValue:`）、`_api.post` 参数名 `data`（非 `body`）

## 待办
- [ ] 云端部署（04e7682 角色管理起含本次全部未部署改动）
- [ ] 协作教师绑定（需独立教师协作记录表）
- [ ] 教学日历/通知接真实教务数据（需校方 API 或导入）
- [ ] 班级打卡统计类做真实班级级聚合（CheckinRepo 加按 class_name 聚合）
