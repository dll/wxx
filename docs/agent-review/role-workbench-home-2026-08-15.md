# 各角色登录首页「角色工作台」设计（2026-08-15）

> 目标：各角色登录后首页场景差异贴合其工作职能，且愿意用蔚小芯完成日常工作。
> 原则：**只接线不重建** —— 所有目标工作台页面已存在，本任务仅在首页加「角色工作台」卡片区，按能力门控分发。

## 一、现状诊断（已实证）

| 角色 | 当前首页 | 问题 |
|---|---|---|
| 学生 | 个性化首页最完整（课表/任务/成长/专区） | ✅ 已好，不动 |
| 教师 | 登录直进 DailyOverviewPage（今日授课概览） | ✅ 已差异化 |
| 辅导员 | 仅「预警概览」 | 🟡 缺专属工作台入口 |
| 教辅(assistant) | **几乎空白** | ❌ 粘性最高角色却无工作台 |
| 学生会(student_union) | 无专属区 | 🟡 有 union/workbench 未接进首页 |
| 学院/学校书记 | 仅「管理专区」 | 🟡 缺教育成果大屏/孪生大屏入口 |
| 系统管理 | 「管理专区」 | 🟡 够用 |

## 二、方案：首页加「角色工作台」卡片区

位置：登录后欢迎横幅之下、学生专属区之前（`home_page.dart` 的 `ListView`）。

按**能力门控**（`CapabilityUtils.has/hasAny`，能力驱动而非硬编码角色，与 profile 页一致）显示各角色专属工作台。所有目标页均已有路由，仅接入口。

### 每角色工作台卡片 → 直达页面

| 角色 | 工作台标题 | 卡片（能力门控） | 直达路由（已存在） |
|---|---|---|---|
| 辅导员 | 辅导员工作台 | 情感预警 / 谈心记录 / 班级画像 / 思想动态 | `/emotion`、`/counselor/talk`、班级看板、`/counselor/ideological` |
| 教辅 | 教辅工作台 | 后勤服务台(待办) / 毕业去向登记(待审核) / 排课 / 考试 / 毕业审核 | `/assistant/facility-workbench`、`/secretary/outcome-manage`、排课/考试/毕业审核 |
| 学生会 | 学生会工作台 | 成员活跃 / 活动分析 / 活动策划 / 反馈处理 | `/union/workbench`、`/union/activity-manage` |
| 学院书记 | 书记工作台 | 教育成果大屏 / 数字孪生大屏 / 数据分析 / 情感预警 | `/secretary/education-outcome`、`/college/twin-screen`、`/college/data-analysis`、`/emotion` |
| 学校书记 | 书记工作台 | 教育成果大屏(全校) / 孪生 / 数据分析 / 情感预警 | 同上 |
| 系统管理 | 管理工作台 | 系统配置 / 用户管理 / 审计 / AI简讯管理 | `/admin/settings`、`/admin/users`、`/admin/audit`、`/admin/ai-briefings` |

### 数据增强（教辅首页 badge）
- 教辅工作台「毕业去向待审核」卡片显示 `pendingCount` badge（`SecretaryProvider.fetchPendingCount()` 已有）
- 辅导员工作台预警数来自 `EmotionProvider.stats`（已加载）

### 门控能力对照
- 辅导员：`counselorAlertRead`（预警）/ `counselorTalkRecord`（谈心）/ `counselorTwinBoard`（班级）/ `counselorIdeological`（思想）
- 教辅：`assistantScheduleCheck`（排课）/ `assistantGradAudit`（毕业审核）/ `assistantExamArrange`（考试）/ `outcomeReview`或`outcomeRecordWrite`（毕业登记）/ facility 能力（后勤）
- 学生会：`unionFeedbackList`（反馈）/ `unionEventPlan`（活动策划）/ union工作台
- 书记：`outcomeDashboard`（教育成果）/ `collegeTwinScreen`（孪生）/ `collegeDataAnalysis`（数据）
- 系统管理：`systemSettingsWrite`（系统配置）/ `systemAuditAll`（审计）

## 三、技术实现
1. `home_page.dart` 新增 `_buildRoleWorkbench(ThemeData)` 方法，在 welcome banner 后插入。
2. 按能力构造卡片列表（各自颜色/图标/路由），复用现有 `_buildKnowledgeCard` 组件风格。
3. 教辅卡片带待审核 badge；辅导员卡片带预警数。
4. 学生、教师的 `build()` 早返回保持不变，不进入该区。

## 四、诚实边界
- 全部接已有真实接口/已有页面，不造伪数据；无权限角色不显示对应卡片。
- 教辅待审核数来自 `fetchPendingCount()` 真实接口。

## 五、改动文件
- `frontend/lib/pages/home/home_page.dart`（新增 _buildRoleWorkbench + _WorkbenchEntry + 首页插入）
- 设计文档 + Git 提交(feat: 角色工作台首页区分)
- flutter analyze 0 error ✅

## 六、落地状态（2026-08-15 编码完成）
- ## ✓ 已实现：home_page.dart 插入 `if (loggedIn) _buildRoleWorkbench(theme)`（学生/教师区之前）
- ✓ `_buildRoleWorkbench` 按能力门控收集各角色工作卡片，复用 `_buildKnowledgeCard`
- ✓ `_WorkbenchEntry` 数据类；`_workbenchTitle` 角色标题映射
- ✓ 正确路由：`/counselor/talk-record`、`/counselor/class-profile`、`/assistant/facility-workbench`(role=='assistant' 门控，facility 无独立能力)等
- ✓ flutter analyze 0 error（新增 info 级单行 if lint，与既有风格一致）
