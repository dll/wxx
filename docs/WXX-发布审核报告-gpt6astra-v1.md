# 蔚小芯发布审核报告（GPT-6 Astra v1）

审核日期：2026-09-08  
审核范围：当前工作区代码、项目规范与部署文档、公开线上入口 `https://wxx-agent.online`。  
审核结论：**不通过发布（Release Blocked）**。

## 1. 结论摘要

项目具备较完整的 Flutter + Go/Gin + SQLite/MySQL + RBAC + Context Engine 代码基础，前端自动化测试通过，角色能力模型也有单元测试覆盖。但当前版本存在编译失败、教师课程数据链路未接通、线上健康检查与文档不一致等发布阻断问题。

在以下问题关闭并重新验证前，不应把当前工作区标记为生产发布版本：

1. Go 全量测试/构建失败：`server/internal/repository/kb_repo_test.go` 缺少 `strings` 导入；`server/internal/service/phase3_service.go` 的 `IndexFunc` 字符常量写法无法编译。
2. 教师课程链路不完整：`TeacherHandler.DailyOverview` 直接返回空 `classes`、`pending_tasks` 和 `alerts`，没有从 JWT 当前用户、`teacher_courses` 或 `course_schedules` 查询；前端教学页因此只能显示“今天暂无课程”。此前的硬编码课程虽然被移除，但真实绑定尚未补上。
3. 教师接口与数据模型存在契约风险：前端 `DailyOverviewPage` 期待 `classes[]`，而 `TeacherService.DailyOverview` 仍是单课程字段模型，服务层也没有实现按教师和日期聚合课表。
4. 线上可观测性异常：`GET https://wxx-agent.online/api/health` 返回 404；`GET https://wxx-agent.online/health` 返回 `text/html` 前端页面，而不是健康 JSON。部署文档同时存在 `/health`、`/api/health`、Cloudflare Pages 和 Caddy 多套口径，需要统一并验证实际发布链路。
5. 无法完成真实教师登录后的端到端验收：本次上下文没有可用于登录的完整凭据/token，因此教师账号 206004 的页面、角色菜单、本人课程白名单和写入权限尚未得到线上实测；这不能被视为通过。

## 2. 已执行检查与结果

| 检查项 | 命令/证据 | 结果 |
|---|---|---|
| Flutter 测试 | `cd frontend; flutter test --no-pub` | 通过，40 项测试全部通过（输出最终为 `All tests passed!`） |
| Flutter 静态分析 | `flutter analyze --no-pub --no-fatal-infos --no-fatal-warnings` | 本次运行未在合理时间内完成，已中止；不能据此宣称分析通过 |
| Go 全量测试 | `go test ./server/... -count=1 -timeout=180s` | 失败；repository 测试缺少 `strings`，service 使用非法 rune literal，handler/service/repository/pkg/app 均受影响 |
| Go 角色测试 | 全量测试输出 | `internal/auth`、`internal/context_engine`、`internal/db` 等部分包通过，但不能抵消全量失败 |
| 线上首页 | `GET https://wxx-agent.online/` | HTTP 200，HTML 可达 |
| 线上 API 健康 | `GET https://wxx-agent.online/api/health` | HTTP 404 |
| 线上根健康 | `GET https://wxx-agent.online/health` | HTTP 200，但 Content-Type 为 `text/html; charset=utf-8`，返回 SPA 页面，不是健康 JSON |
| 线上教师接口（无 token） | `GET https://wxx-agent.online/api/v1/teacher/daily-overview` | HTTP 401，说明路由鉴权存在；未能继续验证账号数据 |
| 发布清单 | `GET https://wxx-agent.online/downloads/release.json` | HTTP 200，清单显示版本 0.0.28；仓库 `frontend/pubspec.yaml` 为 0.0.31+31，线上版本落后于工作区 |

工作区在审核前已有多处未提交修改。报告没有把这些修改误认为已发布，也没有将未相关的脏文件纳入本次审核提交。

## 3. 功能完整性审核

### 3.1 已具备的能力

- Flutter 路由、Provider、Dio API 层和 Material 3 页面结构完整。
- 学生数字画像包含数据可用性标记、空态和规则兜底，前端测试覆盖了 `data_available` 与空画像契约。
- Go 后端包含认证、会话、知识库、情感、语音、学生功能、教师 AI 功能、课程申报/审核、作业与成绩等模块。
- Context Engine 文档明确结构化优先、FTS/BM25 主召回、来源追溯和低置信兜底要求；代码中存在相应检索与来源字段。
- RBAC 角色图和能力测试覆盖 `teacher`、`assistant`、`counselor`、`college_admin` 等继承关系，且测试明确教师不能继承辅导员预警能力。

### 3.2 未达到“功能完善”的部分

- 教师今日教学概览没有真实课程来源，返回空数组是防止错绑的安全降级，但业务功能仍未完成。
- 多个教师 AI 服务在数据不可用时返回 `fallback` 示例内容。页面虽有数据来源字段，但发布前必须确认所有用户可见位置都明确标注示例/兜底，不能把示例成绩、学生数量、覆盖率等当成真实业务数据。
- 部分对接仍是待联调状态。总纲和部署文档把学工/一表通真实对接列为后续工作，因此当前版本不能宣称全量校级系统链路已打通。
- Eino 仍未作为实际编排依赖接入；`docs/context-engine.md` 已明确当前编排是自研路由和并发编排。这属于架构偏差，需要产品/技术负责人书面确认是否接受。

## 4. 数据链路审核

### 4.1 正常链路的代码基础

前端 API 配置使用同域 `/api/v1`，登录后由 Dio 注入 JWT；后端路由按 `/api/v1` 分组并使用能力中间件。知识库仓储、Context Engine、AnswerCard/source 字段和数据库迁移均有实现痕迹。

### 4.2 关键断点

教师教学页链路在后端断开：

```text
JWT 当前用户
   ↓（当前 DailyOverview 未读取）
teacher_id / username
   ↓（当前未查询）
teacher_courses.status=approved 或 course_schedules.owner_username
   ↓（当前未聚合）
classes[] → Flutter 教学页面
```

`DailyOverview` handler 当前返回固定结构的空数组；`GenerateDailyOverview` 只生成问候语和日期，且其模型是单个 `course_name/class_name` 字段，无法直接满足前端的 `classes[]` 列表契约。

线上健康链路也未闭合：正式域名的 `/health` 被 SPA 回退处理，`/api/health` 没有对应可用路由。监控、发布后验收和客户端诊断会得到错误信号。

## 5. 角色与权限审核

- RBAC 能力定义总体清晰，教师与辅导员能力有边界，教师不应读取辅导员个案预警的测试约束存在。
- 教师课程申报、审核和作业课程白名单接口使用 `teacher_id` + `teacher_courses.status=approved` 做强校验，方向正确。
- 但“页面展示课程”与“成绩/作业写入白名单”没有共享同一查询服务。展示接口没有按当前教师过滤，当前通过返回空数组规避了越权，却造成教师功能不可用；恢复课程展示时必须复用 approved 关系和学期/日期过滤，不能重新加入示例数据。
- `college_admin` 多父继承教师、辅导员、教辅能力，权限较宽。发布前需要用真实账号分别验证菜单隐藏、API 403 和数据范围，而不能只依赖角色单测。
- 本次未执行账号 206004 的线上登录验收，角色准确性结论为“代码层部分通过、线上验收未完成”。

## 6. 安全、可靠性与运维风险

- 配置校验要求生产显式 `JWT_SECRET` 且至少 32 字符，并要求至少一个 LLM key；这是正确的安全护栏，但必须确认生产环境实际配置未落回默认值。
- `CORS_ALLOWED_ORIGINS` 默认值仍为 `*`，生产应核对实际环境是否收紧到正式域名。
- 线上发布版本与仓库版本不一致（线上 0.0.28、仓库 0.0.31+31），存在修复未发布、缓存或部署目标错误的风险。
- Go 全量编译失败意味着 CI/发布制品可靠性不足；不能生成可审计的后端发布制品。
- 健康端点返回 SPA HTML 会导致探针误判，且隐藏后端不可用状态。
- 当前工作区包含多个未提交修改和运行状态文件。发布前必须清理、分类或明确保留，并确保发布提交只包含经过审核的源码和文档。

## 7. 发布前必须完成的整改

1. 修复两个 Go 编译错误，执行 `go test ./server/... -count=1` 与 `go build ./server/...`。
2. 新增真实教师课程查询服务：从认证上下文取得用户 ID，按 `teacher_courses` approved 关系和当前学期/日期查询，必要时通过 `owner_username` 解析课表；统一输出前端需要的 `classes[]` 契约。
3. 为教师课程查询增加测试：教师 A 看不到教师 B 的课程；pending/rejected 不展示；approved 才展示；无课时返回诚实空态；课程名称以权威 `courses`/approved 关系为准。
4. 统一健康端点部署：确保正式域名的探针 URL 返回 JSON，状态码、Content-Type、数据库/依赖状态与部署文档一致；修复 SPA 回退对健康路径的覆盖。
5. 统一线上发布版本和仓库版本，重新构建 Web/后端并记录提交 SHA、构建号、部署时间，完成缓存失效验证。
6. 使用测试教师账号完成登录后的 E2E：教学首页、课程列表、备课/作业/成绩接口、越权 403、退出登录和刷新 token。
7. 对政策问答、流程问答和教师 AI 兜底内容抽样核验 `sources`、`data_source`、版本与权限过滤；无来源时不得输出确定政策数字。
8. 完成发布前安全检查：生产密钥、CORS、HTTPS、数据库备份/恢复、限流、审计日志和错误信息脱敏。
9. 重新运行 Flutter analyze、Flutter test、Go test、构建脚本及线上冒烟测试，并把结果附在发布记录中。

## 8. 最终判定

**当前版本未达到发布标准。**

判定依据是发布阻断项而非视觉或轻微 lint：后端全量测试不能编译、教师核心数据链路未实现、线上健康检查失效、线上版本落后且教师真实账号 E2E 未完成。完成第 7 节整改并取得全量测试和线上验收证据后，方可重新申请发布审核。

## 9. 阻断项修复跟踪（2026-09-08）

本次已完成代码侧修复：

- 修复 Go 测试编译错误（缺少 `strings` 导入、非法 rune literal）。
- 教师教学首页改为读取当前 JWT 用户对应的 `teacher_courses.status=approved` 课程；pending/rejected 或其他教师课程不会进入列表；空课程显示诚实提示。
- 修复授课关系查询中 nullable `reviewed_at` 的扫描错误。
- 增加 `/api/health` 后端别名，并在仓库 Caddyfile 中将 `/health` 转发到后端，避免 SPA 回退吞掉探针。
- 新增教师课程白名单单元测试；定向 Go 测试、Go 全量编译门禁和教师页面 Dart 分析通过。

仍需在生产环境完成：部署最新后端/Caddy 配置、确认线上 `/api/health` 和 `/health` 返回 JSON、执行教师 206004 登录后的真实 E2E，并修复全量 Go 测试中现存的检索黄金用例失败与 service 测试数据库并发/关闭问题。完成这些验证前，发布结论保持“待复审”。
