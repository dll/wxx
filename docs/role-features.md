# 蔚小芯 角色功能聚合表

> 版本：v1.0 · 更新日期：2026-05-16
> 用途：八角色 × 各 AI 能力的对照速查；新增能力时同步登记，避免遗漏。
> 关联文档：`specs/rbac-matrix.md`（端点级权限）/ `docs/phase2-plan.md`（专项规划）/ `server/internal/auth/capabilities.go`（代码源真）。

## 1. 角色总览

| # | 角色 | 代码 | 一句话职能 |
|---|------|------|-----------|
| 1 | 系统管理员 | `sys_admin` | 全局配置、跨学校审计、智能体注册 |
| 2 | 学校 | `school_admin` | 学校级用户/智能体/审计 |
| 3 | 二级学院 | `college_admin` | 本院用户/审计/数字孪生大屏，同时继承辅导员/教师/教辅三条线 |
| 4 | 辅导员 / 班主任 | `counselor` | 思政与日常管理、情感预警、班级看板 |
| 5 | 教师 | `teacher` | 备课、出题、班级互动、教学反思 |
| 6 | 教辅 | `assistant` | 排课冲突、毕业审核、考试安排 |
| 7 | 学生会 / 班团委 | `student_union` | 知识库提交、反馈处理、活动策划 |
| 8 | 学生 | `student` | 智能问答、个人成长画像、办事流程引导 |

继承图：`sys_admin → school_admin → college_admin → {counselor, teacher, assistant} → student_union → student`

## 2. 通用基线（全员可见）

每个登录用户都有，对应 `student` 角色直接 capability：

| 能力 | Capability | 路由（前端） | 接口（后端） |
|------|-----------|--------------|--------------|
| AI 对话 | `self.chat` | `/chat` | `POST /api/v1/chat` |
| 知识大厅 | `self.knowledge.read` | `/browse` | `GET /api/v1/knowledge` |
| 个性推荐 | `self.recommend.read` | （首页卡片） | `GET /api/v1/recommendations` |
| 办事流程 | `self.process.read` | `/enrollment` | `GET /api/v1/student/process-enhanced` |
| 语音 ASR/TTS | `self.voice` | （对话内麦克风） | `POST /api/v1/voice/{asr,tts}` |
| 个人会话 | `self.session.read` / `delete` | `/sessions` | `GET/DELETE /api/v1/sessions` |
| 导出本人 | `self.export.self` | （AnswerCard 按钮） | `POST /api/v1/export` |
| 我的收藏 | — | `/bookmarks` | （客户端） |
| 个人资料 | `self.profile.write` | `/profile` | `GET/PUT /api/v1/user/profile` |
| AI 模型配置 | — | `/profile/model-config` | `GET/PUT /api/v1/user/model-config` |
| 提交反馈 | `self.feedback.submit` | （FAB 菜单） | `POST /api/v1/feedback` |

### 校园文化智能体（全员可见，2026-05-16 新增）

| 能力 | Capability | 路由 | 接口 |
|------|-----------|------|------|
| 校歌曲库 | `self.culture.anthem` | `/culture/anthems` | `GET /api/v1/culture/anthems` |
| 校园广播 | `self.culture.radio` | `/culture/radio` | `GET /api/v1/culture/radio` |
| 学术讲座 | `self.culture.lectures` | `/culture/lectures` | `GET /api/v1/culture/lectures` |
| 校园活动 | `self.culture.events` | `/culture/events` | `GET /api/v1/culture/events` |
| 志愿服务 | `self.culture.volunteer` | `/culture/volunteer` | `GET /api/v1/culture/volunteer` |

> 当前为骨架版，返回种子数据；后续接入「学校广播台音频流」「外链联动（B 站/腾讯会议）」「活动报名流程」「志愿时长认证」。

## 3. 学生（student · 25 项 AI 功能）

| 类别 | 功能 | 路由 | 接口 |
|------|------|------|------|
| 学习生活 | 今日速览 | `/student/daily-briefing` | `/api/v1/student/daily-briefing` |
| 学习生活 | 学习日记 | `/student/learning-diary` | `/api/v1/student/learning-diary` |
| 学习生活 | 每日打卡 | `/student/checkin` | `/api/v1/student/checkin` |
| 学习生活 | 数字孪生 | `/student/digital-twin` | `/api/v1/student/digital-twin` |
| 学习生活 | 性格洞察 | `/student/personality` | `/api/v1/student/personality` |
| 学习生活 | 积分成就 | `/student/achievements` | `/api/v1/student/achievements` |
| 课程学情 | 课程地图 | `/student/course-map` | `/api/v1/student/course-map` |
| 课程学情 | 课程学情 | `/student/course-analytics` | `/api/v1/student/course-analytics` |
| 课程学情 | 学习周报 | `/student/weekly-report` | `/api/v1/student/weekly-report` |
| 思政成长 | 新生计划 | `/student/freshman-plan` | `/api/v1/student/freshman-plan` |
| 思政成长 | 成长路径 | `/student/growth-path` | `/api/v1/student/growth-path` |
| 思政成长 | 政治学习 | `/student/political-study` | `/api/v1/student/political-study` |
| 思政成长 | 思想动态 | `/student/ideological-record` | `/api/v1/student/ideological-record` |
| 思政成长 | 入党进度 | `/student/party-progress` | `/api/v1/student/party-progress` |
| 校园生活 | 生活记录 | `/student/campus-life` | `/api/v1/student/campus-life` |
| 校园生活 | 课表管理 | `/student/schedule` | `/api/v1/student/schedule` |
| 校园生活 | 竞赛赛事 | `/student/competition-match` | `/api/v1/student/competition-match` |
| 校园生活 | 学习搭子 | `/student/study-buddy` | `/api/v1/student/study-buddy` |
| 校园生活 | 心理健康 | `/student/mental-health` | `/api/v1/student/mental-health` |
| 校园生活 | 数字导师 | `/student/digital-mentor` | `/api/v1/student/digital-mentor` |
| 社区互动 | 问答广场 | `/student/qa-plaza` | `/api/v1/student/qa-plaza` |
| 社区互动 | 热点话题 | `/student/hot-topics` | `/api/v1/student/hot-topics` |
| 社区互动 | 问答排行 | `/student/qa-leaderboard` | `/api/v1/student/qa-leaderboard` |
| 社区互动 | 站内私聊 | `/student/private-chat` | `/api/v1/student/private-chat` |
| 流程办理 | 办事增强 | `/student/process-enhanced` | `/api/v1/student/process-enhanced` |

## 4. 学生会（student_union · 4 项专属 + 全部学生功能）

| 功能 | Capability | 路由 | 接口 |
|------|-----------|------|------|
| 知识库提交 | `union.kb.submit` | `/my-submissions` | `POST /api/v1/kb/resources/:id/submit` |
| 反馈管理 | `union.feedback.list` | `/feedback` | `GET/PUT /api/v1/feedback` |
| 活动策划 | `union.event.plan` | `/union/event-plan` | `GET /api/v1/union/event-plan` |
| 海报生成 | `union.poster.gen` | `/union/poster-gen` | `GET /api/v1/union/poster-gen` |

## 5. 辅导员（counselor · 22 项专属 + 学生会全部功能）

| 类别 | 功能 | Capability | 路由 |
|------|------|-----------|------|
| 日常管理 | 今日关注 | `counselor.daily_focus.read` | `/counselor/daily-focus` |
| 日常管理 | 班级日报 | `counselor.class.report` | `/counselor/class-report` |
| 日常管理 | 学生数字孪生 | `counselor.twin.board` | `/counselor/twin-board` |
| 日常管理 | 班级画像 | `counselor.class.profile` | `/counselor/class-profile` |
| 日常管理 | 学生列表 | `counselor.student.list` | `/counselor/student-list` |
| 情感预警 | 预警阅读 | `counselor.alert.read` | `/emotion`（dashboard） |
| 情感预警 | 处理告警 | `counselor.alert.handle` | 同上 |
| 情感预警 | 触发分析 | `counselor.alert.analyze` | `POST /api/v1/emotion/analyze` |
| 情感预警 | 预测预警 | `counselor.prediction.read` | `/counselor/prediction` |
| 情感预警 | 趋势报告 | `counselor.emotion.trends` | `GET /api/v1/emotion/trends` |
| 思政工作 | 思想动态 | `counselor.ideological` | `/counselor/ideological` |
| 思政工作 | 干预方案 | `counselor.intervention.write` | `/counselor/intervention` |
| 思政工作 | 谈心记录 | `counselor.talk.record` | `/counselor/talk-record` |
| 思政工作 | 谈话话术 | `counselor.talk.tips` | `/counselor/talk-tips` |
| 知识与社区 | 知识库 CRUD | `counselor.kb.write` | `/kb/resources` |
| 知识与社区 | 知识审核 | `counselor.kb.review` | `/review` |
| 知识与社区 | 待审核列表 | `counselor.review.pending` | `/review` |
| 知识与社区 | 社区管理 | `counselor.community.manage` | `/counselor/community-manage` |
| 知识与社区 | 热点感知 | `counselor.hot_topic.sense` | `/counselor/hot-topic-sense` |
| 知识与社区 | 流程编辑 | `counselor.process.edit` | `/counselor/process-edit` |
| 校外对接 | 学工/一表通代理 | `counselor.integration.read` | `/api/v1/integration/{xuegong,ybt}` |

## 6. 教师（teacher · 9 项专属 + 学生会基线）

| 功能 | Capability | 路由 |
|------|-----------|------|
| 今日授课概览 | `teacher.daily.overview` | `/teacher/daily-overview` |
| 备课助手 | `teacher.lesson.prep` | `/teacher/lesson-prep` |
| 考试出题 | `teacher.exam.gen` | `/teacher/exam-gen` |
| 课堂互动 | `teacher.class.interact` | `/teacher/class-interact` |
| 作业批改 | `teacher.grading` | `/teacher/grading` |
| 学情热力图 | `teacher.heatmap.read` | `/teacher/heatmap` |
| 教学反思 | `teacher.reflection` | `/teacher/reflection` |
| 学习风格分布 | `teacher.style.dist` | `/teacher/style-distribution` |
| 社区专业答疑 | `teacher.community.qa` | `/teacher/community-qa` |

## 7. 教辅（assistant · 3 项专属 + 学生会基线）

| 功能 | Capability | 路由 |
|------|-----------|------|
| 排课冲突 | `assistant.schedule.check` | `/assistant/schedule-check` |
| 毕业审核 | `assistant.grad.audit` | `/assistant/graduation-audit` |
| 考试安排 | `assistant.exam.arrange` | `/assistant/exam-arrange` |

## 8. 学院管理员（college_admin · 5 项专属 + 同时继承辅导员/教师/教辅）

| 功能 | Capability | 路由 |
|------|-----------|------|
| 本院用户管理 | `college.user.read` | `/admin/users` |
| 本院审计 | `college.audit.read` | `/admin/audit` |
| 本院指标 | `college.metrics.read` | `/admin/metrics` |
| 数字孪生大屏 | `college.twin.screen` | `/college/twin-screen` |
| 数据分析 | `college.data.analysis` | `/college/data-analysis` |

## 9. 学校管理员（school_admin · 3 项专属 + 学院全部）

| 功能 | Capability |
|------|-----------|
| 智能体管理 | `school.agent.write`（路由 `/agents`） |
| 用户管理（学校级） | `school.user.write` |
| 修改用户 | `school.user.update` |

## 10. 系统管理员（sys_admin · 3 项专属 + 学校全部）

| 功能 | Capability |
|------|-----------|
| 全局配置 | `system.settings.write`（路由 `/admin/settings`） |
| 全局审计 | `system.audit.all` |
| 重置任意用户密码 | `system.password.reset` |

## 11. 扩展点

- **第三方应用接入**：详见 `docs/external-apps.md`，校园文化资源/教务/图书馆等可通过统一 manifest 协议挂载
- **新增能力的步骤**：
  1. `server/internal/auth/capabilities.go` 增加 `Capability` 常量
  2. 找到对应 `roleNode.capabilities` 数组追加（高阶角色通过继承自动拥有）
  3. `server/pkg/app/app.go` 注册路由 + `auth.RequireCapability(...)`
  4. 前端 `lib/config/api_config.dart` 增加路径常量
  5. 前端 `lib/config/router.dart` 注册路由
  6. 在本表与 `specs/rbac-matrix.md` 同步登记
