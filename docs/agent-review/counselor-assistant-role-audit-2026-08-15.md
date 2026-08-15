# 辅导员 / 教辅 角色功能审核报告

> 日期：2026-08-15
> 审核对象：counselor（辅导员）、assistant（教辅）角色
> 对齐依据：`docs/蔚小芯角色功能.md` v5.2 / `docs/蔚小芯待完成.md`（P1/P2 清单）+ **`docs/蔚小芯角色功能所需材料.md` v2.0（勾选式上线清单）**

> ⚠️ **三份权威文档口径不一致**：`角色功能所需材料.md` v2.0 与 `待完成.md` 对教辅/辅导员功能清单**定义不同**（例如：所需材料版教辅 5 项〔排课/毕审/考排 + 证明流程/教务知识库〕，待完成版教辅 12 项〔P1 3 + P2 9〕；所需材料版辅导员有 P0 情感预警系列，待完成版辅导员清单未列情感预警）。本表如实列出分歧，最终以用户确认的口径为准。
> 状态：P1 全部落地；P2 存在「后端已就绪、前端缺页面」缺口；数据真实性标注（教辅）已落地

## ⚠️ 结论：未完全对齐角色需求（本审核补充了 P2 缺口）

首批审核只覆盖了 P1（辅导员 11 项、教辅 3 项），**未核对 P2**。权威清单为辅导员 **17 项**（P1 11 + P2 6）、教辅 **12 项**（P1 3 + P2 9）。对齐后现状：

## 一、权威角色需求对齐表

### 辅导员（共 17 项）

| 项 | 后端 | 前端页面/入口 | 状态 |
|----|------|--------------|------|
| P1-1 AI 今日关注 daily_focus | ✅ | ✅ daily_focus_page | 落地 |
| P1-2 数字孪生看板 twin_board | ✅ | ✅ twin_board_page | 落地(硬编码⚠️) |
| P1-3 AI 预测性预警 prediction | ✅ | ✅ prediction_page | 落地(硬编码⚠️) |
| P1-4 班级学情日报 class_report | ✅ | ✅ class_report_page | 落地 |
| P1-5 谈心谈话记录 talk_record | ✅ | ✅ talk_record_page | 落地 |
| P1-6 谈话话术推荐 talk_tips | ✅ | ✅ talk_tips_page | 落地 |
| P1-7 AI 干预方案 intervention | ✅ | ✅ intervention_page | 落地 |
| P1-8 思想档案 ideological | ✅ | ✅ ideological_page | 落地 |
| P1-9 班级性格画像 class_profile | ✅ | ✅ class_profile_page | 落地 |
| P1-10 社区问答管理 community_manage | ✅ | ✅ community_manage_page | 落地(假帖⚠️) |
| P1-11 热点话题感知 hot_topic_sense | ✅ | ✅ hot_topic_sense_page | 落地(假热点⚠️) |
| P2-1 谈话跟进提醒 follow_up_reminders | ✅ 路由已注册 | ❌ 无页面 | **缺前端** |
| P2-2 班级打卡统计 checkin_stats | ✅ 路由已注册 | ❌ 无页面 | **缺前端** |
| P2-3 智能群发 smart_notify | ✅ 路由已注册 | ❌ 无页面 | **缺前端** |
| P2-4 AI 月度简报 monthly_brief | ✅ 路由已注册 | ❌ 无页面 | **缺前端** |
| P2-5 AI 会话洞察 session_insight | ✅ 路由已注册 | ❌ 无页面 | **缺前端** |
| P2-6 流程步骤编辑 process_edit | ✅ | ✅ process_edit_page | 落地 |

> **辅导员缺口 = 5 项 P2 后端已就绪、只缺前端页面+入口**（follow-up-reminders / checkin-stats / smart-notify / monthly-brief / session-insight 路由均在 app.go）。补齐成本低、风险低。

### 教辅（共 12 项）

| 项 | 后端 | 前端页面 | 状态 |
|----|------|---------|------|
| P1-1 AI 排课冲突检测 | ✅ phase3 真实 | ✅ schedule_check_page | 落地 |
| P1-2 AI 毕业资格审核 | ✅ phase3 真实 | ✅ grad_audit_page | 落地 |
| P1-3 AI 考试安排优化 | ✅ phase3 真实 | ✅ exam_arrange_page | 落地 |
| P2-1 AI 通知批量生成 | ❌ 后端无 | ❌ 无 | **缺失** |
| P2-2 教学日历管理 | 部分 | ❌ 无 | **缺** |
| P2-3 AI 材料模板库 | ❌ 模板业务无 | ❌ 无 | **缺失** |
| P2-4 AI 学生信息查询 | ✅ StudentInfoQuery | ❌ 无 | **缺前端** |
| P2-5 AI 文档智能处理 | ✅ | ✅ process_document | 落地(学生/管理侧) |
| P2-6 AI 流程自动化 | ❌ | ❌ 无 | **缺失** |
| P2-7 流程步骤详情管理 | ✅ | ✅ process_edit(共享) | 落地 |
| P2-8 音乐电台 | ✅ | ✅ radio_page | 落地(学生端) |
| P2-9 校园活动报名 | 部分 | ❌ 无 | **缺** |

> **教辅缺口更大**：P2 中 4 项（通知批量/模板库/流程自动化）+ 3 项部分（教学日历/学生信息查询前端/活动报名）未落地，其中后端也缺的需新建。

## 二、数据真实性（核心问题 ⚠️）

### 无问题的部分 ✅
- **assistant** P1 三接口：优先走 `phase3` 真实课表/成绩/考试数据（已导入 10870 条课表+491 学生），空数据诚实兜底为 `reference`。
- **counselor** `GetStudentList`：真实学生 + 脱敏 + 范围锁定。

### 仍有编造/误导风险 ⚠️（缺来源标注）
以下 counselor 接口无真实数据时返回硬编码示例人物且前端无来源标注：
- `GenerateTwinBoard` / `GeneratePredictions` / `GenerateMonthlyBrief` / `GenerateFollowUpReminders` / `GenerateCheckinStats` / `GenerateCommunityManage` / `GenerateHotTopicSense`（硬编码 张明/王芳/李华 + 假分数/假日期）。

> 风险等级：**高**（人+决策数据，辅导员可能对不存在的学生做干预）。

## 三、本增量已落地（教辅数据来源标注，commit 212ff06）

- 新增通用组件 `frontend/lib/widgets/data_src_badge.dart`（真实数据/参考AI，点击弹窗，对齐学生会工作台语义）。
- 应用到教辅 3 页列表头。
- `flutter analyze` 项目级 0 error / 0 warning。

## 四、待办（对齐角色需求的两个有序增量）

**增量 A（P2 前端补齐，低风险）**：辅导员 5 项 P2（follow-up/checkin/smart-notify/monthly/session）+ 教辅 P2-4 学生信息查询 —— **后端已就绪，建前端页 + 注册路由 + 入口**。

**增量 B（不瞎编整改，高风险优先）**：辅导员 7 个接口去硬编码示例人物 + 统一 `DataSrcBadge`；教辅 `GradAudit` 加学生选择器。

**增量 C（后端新建，教辅）**：通知批量生成 / 材料模板库 / 流程自动化（后端也缺，需新建 service+handler+capability）。

## 五、`角色功能所需材料.md` v2.0 额外核对（新增项）

该清单与待完成清单口径不同，额外要求且现状：

| 清单项 | 优先级 | 现状 |
|--------|--------|------|
| 辅导员·情感预警（概览/列表/确认/处理/趋势/分析） | **P0** | ✅ 后端强（emotion 600 处/alert 194/预警 33）+ 前端 `emotion_dashboard_page.dart`（路由 /emotion 已注册）⚠️ profile 菜单未直接出现独立预警入口，需实机验证可达性 |
| 辅导员·学生信息编辑 | P1 | ⚠️ student_list 页面未见明显编辑入口，待核 |
| 辅导员·导入学生（Excel） | P0 | ✅ 已有 admin 导入 /users/import（CounselorImportStudent） |
| 教辅·证明办理流程 | **P1** | ⚠️ 后端有（student_features 11 处）但前端无页面（certificate 前端 0）→ 缺前端 |
| 教辅·教务资料知识库 | **P1** | ❌ 后端/前端均无独立模块（若有走通用知识库） |

> 该清单还列出教师/学院/学校/系统管理等其它角色 P0-P2，不在本次辅导员+教辅范围。
