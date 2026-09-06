# 蔚小芯重构基线与批次计划

> 建立日期：2026-09-04。本文记录重构前可复现基线，避免结构调整改变既有业务契约。

## 一、当前基线

| 检查项 | 命令 | 结果 |
|---|---|---|
| Go 单元测试 | `go test ./server/...` | 通过 |
| Flutter 单元/Widget 测试 | `cd frontend && flutter test` | 通过（33 项） |
| Flutter 静态分析 | `flutter analyze --no-pub --no-fatal-infos --no-fatal-warnings` | 通过，50 条 info（已自动修复 260 条确定性 lint） |
| Go 文件规模 | `server/` | 453 个 Go 文件，144 个测试文件 |
| Flutter 文件规模 | `frontend/lib/` | 269 个 Dart 文件 |
| 数据库迁移 | `server/migrations/` | 114 个迁移 |

Go 测试包含 agent、auth、context_engine、handler、repository、service、Temporal 和 app 等关键包。Flutter 分析结果目前以 info 为主，不阻断构建，但应按风险分批收敛。

## 二、已知重构热点

### Flutter

- `pages/vopc/vopc_page.dart`：约 126 KB
- `pages/home/home_page.dart`：约 95 KB
- `pages/admin/my_submissions_page.dart`：约 76 KB
- `pages/chat/chat_page.dart`：约 71 KB
- `pages/profile/profile_page.dart`：约 66 KB
- `pages/campus/campus_map_page.dart`：约 66 KB

### Go

- `internal/service/student_service.go`：约 78 KB
- `pkg/app/routes.go`：约 74 KB
- `internal/service/document_service.go`：约 52 KB
- `internal/model/entity.go`：约 49 KB
- `internal/repository/kb_repo.go`：约 48 KB
- `internal/handler/vopc_handler.go`：约 40 KB

这些文件优先按职责拆分，保持公开 API、路由、数据库字段和 AnswerCard 契约不变。

## 三、重构批次

### 批次一：治理与基线固化

- README、AGENTS 与真实目录结构对齐。
- CI 同时覆盖 Go 和 Flutter 基础检查。
- 保留未跟踪运行状态文件，未经确认不删除；临时文件后续单独归档。

### 批次二：后端路由与依赖组装（阶段完成）

- 已完成：将根路由、健康检查、Flutter 静态资源和 SPA 回退提取至 `server/pkg/app/routes_static.go`。
- 已完成：将公开 API（认证、版本、校园报到、公开知识）提取至 `server/pkg/app/routes_public.go`。
- 已完成：将 vOPC 项目协作与风险治理路由提取至 `server/pkg/app/routes_vopc.go`。
- 已完成：将问答、会话、知识浏览和情感路由提取至 `server/pkg/app/routes_self.go`。
- 已完成：将问题预案和毕设选题路由提取至 `routes_forecast.go`、`routes_graduation.go`。
- 已完成：将竞赛、大学规划、入党教育和社团生活路由提取至 `routes_student_features.go`。
- 已完成：将知识库、知识包同步、智能体、语音、集成和词元统计路由提取至 `routes_platform.go`。
- 已完成：将管理端统计、用户、审计、设置和版本路由提取至 `routes_admin_core.go`。
- 已完成：将当前用户资料、凭证、偏好和安全设置提取至 `routes_user.go`，并增加拆分注册函数接入断言。
- 回归测试已改为读取拆分后的路由文件集合，路由路径与中间件行为保持不变。
- 保持中间件顺序、鉴权和路由路径不变。

### 批次三：后端领域服务（准备中）

- 优先拆分学生、文档、辅导员和知识库服务/仓储。
- repository 只负责数据访问，service 负责业务规则，handler 只负责协议适配。

当前已完成路由层领域边界，尚未大规模迁移 service/repository 业务实现。已为 `student_service.go`、`teacher_service.go` 建立兜底、LLM 解析与质量门槛接口测试；`kb_repo.go` 已有搜索、权限过滤、CRUD 与分页相关测试；问题预案服务已补齐对话热点关键词聚合并加入排序测试。下一步在保持这些契约的前提下，再进行方法迁移。

- 已完成首个方法迁移增量：教师教案解析与兜底从 `teacher_service.go` 提取至 `teacher_lesson_plan.go`，公开构造函数和调用契约不变。
- 已完成第二个方法迁移增量：教师试卷解析与兜底从 `teacher_service.go` 提取至 `teacher_exam.go`，公开构造函数和调用契约不变。
- 已完成第三个方法迁移增量：教师作业批改解析与兜底从 `teacher_service.go` 提取至 `teacher_grading.go`，公开构造函数和调用契约不变。
- 已完成第四个方法迁移增量：教师课堂互动生成与兜底从 `teacher_service.go` 提取至 `teacher_interaction.go`，公开构造函数和调用契约不变。
- 已完成第五个方法迁移增量：教师今日授课概览模型与生成逻辑从 `teacher_service.go` 提取至 `teacher_daily_overview.go`，公开构造函数和调用契约不变。
- 已完成第六个方法迁移增量：教师知识点覆盖检查与课程思政建议从 `teacher_service.go` 提取至 `teacher_coverage.go`，公开构造函数和调用契约不变。
- 已完成第七个方法迁移增量：教师学生数字孪生教学视图与课程 FAQ 从 `teacher_service.go` 提取至 `teacher_twin_faq.go`，公开构造函数和调用契约不变。
- 已完成第八个方法迁移增量：教师个性化教学建议从 `teacher_service.go` 提取至 `teacher_personalized.go`，公开构造函数和调用契约不变。
- 已完成第九个方法迁移增量：教师班级热力图、学习风格分布和社区答疑从 `teacher_service.go` 提取至 `teacher_learning_insights.go`，公开构造函数和调用契约不变。
- 已完成学生服务首个方法迁移增量：专业知识图谱与笔记助手从 `student_service.go` 提取至 `student_learning_tools.go`，公开构造函数和调用契约不变。
- 已完成学生服务第二个方法迁移增量：简历、职业模拟和校友匹配从 `student_service.go` 提取至 `student_career_tools.go`，公开构造函数和调用契约不变。
- 已完成学生服务第三个边界增量：知识库标签解析从 `student_service.go` 提取至 `student_kb_helpers.go`，保留 `parseTags` 兼容别名并增加契约测试。
- 已完成学生学习周报契约测试，覆盖无数据时的稳定周次、行动建议、时间分布和来源标记；下一步继续迁移周报实现。
- 已完成学生学习日记兜底构造提取至 `student_diary_fallback.go`，公开服务方法和 fallback 字段契约保持不变。
- 已完成学生问答广场兜底数据构造提取至 `student_qa_fallback.go`，真实 FAQ 查询、标签和来源标记保持不变。
- 已完成学生校园热点兜底数据构造提取至 `student_hot_topics.go`，真实 Activity 查询和来源标记保持不变。
- 已完成学生问答排行榜参考榜单与无数据兜底构造提取至 `student_qa_leaderboard.go`，真实提问聚合逻辑保持不变。
- 已完成学生站内私聊参考数据提取至 `student_private_chat.go`，公开方法契约保持不变。
- 已完成学生动态导师生成逻辑提取至 `student_dynamic_mentor.go`，保留风格映射、LLM 覆盖和来源标记。
- 已完成学生增强职业模拟逻辑提取至 `student_enhanced_career.go`，保留阶段、技能差距、薪资投影和 AI 建议契约。
- 已完成学生课程学情看板逻辑提取至 `student_course_analytics.go`，保留真实成绩、班级基准、薄弱课程和 LLM 建议契约。
- 已完成学生成长路径聚合入口提取至 `student_growth_path.go`，保留数字孪生五维快照、学业阶段、里程碑和 LLM 总结契约。
- 已完成学生课程学情与成长路径迁移后的遗留实现清理，`student_service.go` 不再保留对应重复方法体。
- 已完成学生学业预警入口提取至 `student_academic_warning.go`，保留风险等级、因素、建议、资源和 AI 追加建议契约。
- 已完成学生心理健康报告入口提取至 `student_mental_health.go`，保留情感记录聚合、风险引导和安全兜底契约。
- 已完成学生学习周报入口提取至 `student_weekly_report.go`，保留周次、时间分布、真实交互统计、行动建议和来源标记契约。
- 已完成学生学习伙伴匹配入口提取至 `student_study_buddy.go`，保留院系筛选、匹配评分、姓名脱敏和兜底契约。
- 已完成学生模拟面试生成入口提取至 `student_mock_interview.go`，保留岗位默认值、题目提示、评分和 AI 补充契约。
- 已完成学生问答广场入口提取至 `student_qa_plaza.go`，保留 FAQ 检索、标签解析、来源链接和兜底契约。
- 已完成学生校园热点入口提取至 `student_hot_topics_service.go`，保留 Activity 检索、热度趋势、来源链接和兜底契约。
- 已完成学生问答排行榜入口提取至 `student_qa_leaderboard_service.go`，保留真实热门提问聚合、截断和参考榜单契约。
- 学生服务连续拆分后的后端全量编译门禁已通过：`go test ./server/... -run '^$'`，各 server 包均可正常编译。
- 已完成文档服务统一解析出口提取至 `document_parse.go`，保留内容清洗、元数据提取、质量门槛和解析警告契约。
- 已完成文档服务 LLM 元数据精修入口提取至 `document_refine.go`，保留超时保护、字段补齐、校验失败回退和人工确认契约。
- 已完成辅导员谈心记录与 LLM 摘要入口提取至 `counselor_talk_record.go`，保留结构化字段、无模型兜底和解析契约。
- 已完成辅导员谈心话术推荐入口提取至 `counselor_talk_tips.go`，保留画像提示词、字段解析和无模型兜底契约。
- 已完成辅导员风险干预方案入口提取至 `counselor_intervention.go`，保留风险等级、紧急措施、长期方案和 LLM 回退契约。
- 已完成辅导员会话洞察入口提取至 `counselor_session_insight.go`，保留话题、情绪趋势、诉求、建议和 AI 追加分析契约。
- 已完成辅导员谈心跟进提醒入口提取至 `counselor_followup.go`，保留逾期判定、优先级统计、真实记录和 AI 建议契约。
- 已完成辅导员班级打卡统计迁移后的遗留实现清理，主服务仅保留真实入口。
- 已完成辅导员通知群发与班级打卡统计入口提取至 `counselor_notifications.go`，保留受众版本、AI 改写和无数据诚实返回契约。

### 批次四：Flutter 页面组件化（进行中）

- 优先处理 VOPC、首页、问芯、校园地图。
- 将页面容器、状态、请求、区块 Widget、表单和对话框分离。
- 已完成：通知关联类型映射从 `notification_page.dart` 抽出至独立路由工具，并增加 Dart 单元测试；后续继续按页面优先级拆分大型容器。
- 已完成：学生首页加载骨架从 `home_page.dart` 抽出至 `student_home_skeleton.dart`，保持首页状态与视觉契约不变，并通过 Flutter 定向测试。
- 已完成：首页欢迎横幅从 `home_page.dart` 抽出至 `home_welcome_banner.dart`，保留年级主题、问芯入口和时间问候展示契约。
- 已完成：首页加载错误提示卡从 `home_page.dart` 抽出至 `home_error_card.dart`，通过重试回调保留加载失败与恢复契约。
- 已完成：首页校历信息条从 `home_page.dart` 抽出至 `home_calendar_bar.dart`，保留周次、星期和学期信息展示契约。
- 已完成：首页今日概览统计卡从 `home_page.dart` 抽出至 `home_overview_item.dart`，保留课程、任务、通知和计划计数展示契约。
- 已完成：首页课程、任务和活动的通用空状态卡从 `home_page.dart` 抽出至 `home_empty_card.dart`，保持无数据提示与主题色契约。
- 已完成：文档解析与精修迁移后的遗留实现清理，`document_service.go` 不再保留重复入口。
- 已完成：问芯空会话欢迎头部从 `chat_page.dart` 抽出至 `chat_empty_intro.dart`，保留主题渐变、引导文案和推荐区块布局契约。
- 已完成：问芯加载状态气泡从 `chat_page.dart` 抽出至 `chat_loading_bubble.dart`，保持发送中状态展示与消息列表布局契约。
- 已完成：校园地图报到步骤卡从 `campus_map_page.dart` 抽出至 `campus_step_card.dart`，通过回调保留步骤选择、完成状态和地图容器交互契约。
- 已完成：知识治理页面统计状态徽标从 `my_submissions_page.dart` 抽出至 `resource_stat_chip.dart`，保持资源统计颜色、数量和标签展示契约。
- 已完成：知识治理资源类型分布从 `my_submissions_page.dart` 抽出至 `resource_type_stats.dart`，保持类型映射、颜色和计数展示契约。
- 已完成：vOPC 页面错误卡从 `vopc_page.dart` 抽出至 `vopc_error_card.dart`，保持 HTTP 状态提示和重试回调契约。
- 已完成：vOPC 页面元信息标签从 `vopc_page.dart` 抽出至 `vopc_meta_chip.dart`，保持项目阶段、风险、状态和治理标签展示契约。
- 已完成：vOPC 任务卡从 `vopc_page.dart` 抽出至 `vopc_task_card.dart`，通过状态回调保留任务状态流转和权限禁用契约。

### 批次五：静态质量收敛（阶段完成）

已完成：使用 `dart fix` 修复 244 条确定性问题（const、花括号、废弃 API、无效转换等）。
剩余 50 条主要是异步 `BuildContext`、Web 平台专用库提示和少量未使用私有方法，需逐项人工确认后处理；模型 `part-of` 提示已统一为文件 URI。

## 四、每批验收

- Go：`gofmt -l .` 为空，`go vet -tags fts5 ./...` 通过，`go test -race -tags fts5 ./...` 通过。
- Flutter：`flutter test` 通过，`flutter analyze` 不新增 error/warning。
- API 路由、RBAC、sources、fallback、数据库迁移行为保持兼容。
- 每个可演示增量同步更新文档并单独 Git 提交。
