# 蔚小芯·多角色粘性增强方案（辅导员/学生会/教辅/教师/管理员）

- 角色：date 2026-08-14
- 视角：从每个角色的日常工作出发，找出"希望用、喜欢用"的粘性增强点
- 原则：新增内容以真实文档/已有数据为准，不瞎编；改动可被 `flutter analyze` 验证

---

## 一、角色全景（系统现有角色与能力）

| 角色 | 定位 | 现有页面/能力 | 粘性缺口 |
|------|------|--------------|---------|
| student 学生 | 全流程服务对象 | 新生引导/开学待办/分年级成长/问答/积分/心理等（已增强） | 已较完善 |
| counselor 辅导员 | 学生教育管理直接责任人 | class_report、daily_focus、talk_record、intervention、prediction、twin_board、hot_topic_sense、ideological、student_list、followup/checkin/session 统计 | **谈心记录缺统计与效果跟进** |
| student_union 学生会 | 活动组织 | event_plan(AI策划)、poster_gen(AI海报) | 活动闭环（报名管理/复盘/社团招新衔接） |
| teacher 教师 | 教学 | daily_overview、class_interact、exam_gen、grading、heatmap、lesson_prep、reflection、style_dist、community_qa | 学情预警联动、AI 评语落地 |
| assistant 教辅 | 事务办理 | exam_arrange、grad_audit、schedule_check | 办理进度透明、学生可查 |
| admin(校学院) | 管理决策 | dashboard、metrics、users、content、audit、data_analysis、twin_screen | 数据大屏下钻、治理闭环 |

---

## 二、辅导员（首选增强，已实现）

> 用户示例："辅导员与学生的交流次数、内容、效果"

### 已实现（TalkRecordPage 增强版）
- **统计面板**：谈话总次数 / 跟进中 / 已解决 / 近30天交流数——直观反映"和多少学生谈了几次、效果如何"
- **状态筛选**：全部 / 跟进中 / 已解决
- **记录完整化**：卡片展示 学生、日期、话题、情绪、摘要、跟进项、状态徽章
- **新增补全字段**：话题、情绪、状态（配合后端 LLM 自动从内容提取 topic/emotion/summary/follow_ups）

### 建议后续（需后端小接口，超出本次前端边界）
1. 记录支持"标记已解决"交互（PUT 更新 status —— 当前 handler 仅有 POST/GET，需加 PUT）
2. "待跟进提醒"聚合：按 follow_ups 过期时间排序，首页红点提示
3. 学生端"我的谈心记录"（学生能看自己被约谈的主题与改进点，形成双向闭环）
4. 月度统计简报（后端已有 MonthlyBrief/SessionInsight/CheckinStats，前端可加看板）

---

## 三、学生会（student_union）

**现状**：AI 活动策划(event_plan)、AI 海报生成(poster_gen)。

### 粘性增强建议（由轻到重）
1. **活动报名管理闭环**：策划→发布→报名名单→活动报名数据（**已实现**：activity_manage_page，含统计/筛选/新建，复用 health/activities 接口）
2. **活动复盘模板**：活动后一键生成复盘（参与人数、满意度、改进点），沉淀到社团
3. **社团招新衔接**：新生指南已有社团介绍，学生会端做"招新数据看板"（报名/咨询数）
4. **AI 议程助手**：把会议/活动安排的碎片信息汇成待办清单（对标学生待办）

---

## 四、教师（teacher）

**现状**：今日教学工作台、课堂互动、试卷生成、作业批改、学情热力图、备课、教学反思、风格分布、社区问答。

**粘性增强建议**（由轻到重）：
1. **AI评语落地**（**已实现**）：grading 生成评语一键复制，可直接粘贴到课业系统/发给学生
2. **学情预警联动**（daily_overview 已有 alerts + heatmap 热力图，教师可提前干预）
3. **答疑闭环**：community_qa 教师回答后学生可"已解决"，教师看采纳率
4. **备课资源复用**：lesson_prep 生成的教案/习题入库，跨班复用

---

## 五、教辅（assistant）

**现状**：排课冲突检测、考场安排、毕业审核。

**粘性增强建议**：
1. **办理进度透明**（**已实现**）：毕业审核页学分进度条 + 已达标/待补项，学生/教辅都能直观看到办理进度
2. **冲突提醒推送**：schedule_check 发现冲突→主动通知相关班级/教务
3. **批量工具**：名单核对、证书材料批量校验，减少重复劳动

---

## 六、管理员（school/college/sys admin）

**现状**：dashboard、metrics、users、content、audit、data_analysis、twin_screen。

**粘性增强建议**：
1. **治理闭环**（**已实现**）：admin_metrics_page 新增"高频兜底问题"区块，展示真实 chat_metrics 里的高频未命中问题及次数，引导补录知识库压降兜底率（后端 /admin/metrics/fallback-questions）；**不硬编码默认值**（metrics 无数据时原为 0.85/0.10 默认值，属展示局限）
2. **指标下钻**：metrics 从学校→学院→年级下钻，定位异常
3. **通知触达**：针对特定人群（如某年级欠费/缺勤）批量站内信

---

## 七、实施优先级建议

| 优先级 | 角色 | 事项 |
|--------|------|------|
| P0 | 辅导员 | 谈心记录统计与效果跟进（**已实现**，前端）+ 待办：标记已解决、学生端入口 |
| P1 | 学生会 | 活动报名管理闭环（**已实现**：activity_manage_page，统计/筛选/新建，复用 health/activities） |
| P1 | 教师 | 学情预警联动（daily_overview alerts+heatmap 已覆盖）、AI评语一键落地（**已实现**：grading_page 复制评语按钮） |
| P2 | 教辅 | 办理进度透明（**已实现**：grad_audit_page 学分进度条 + 已达标/待补项） |
| P2 | 管理员 | 知识兜底率看板 + 治理闭环（**已实现**：admin_metrics_page 高频兜底问题区块 + 后端 /admin/metrics/fallback-questions，基于真实 chat_metrics） |

> 说明：本方案为分角色粘性增强的总纲与盘点；具体实现遵循 project 的 AGENTS.md 纪律（Plan→人审→编码）、逐项文档+git 提交、`flutter analyze` 验证。
