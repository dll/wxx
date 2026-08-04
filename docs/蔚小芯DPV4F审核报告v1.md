# 蔚小芯 DPV4F 审核报告 v1（2026-08-04 复核更新）

> 审核日期：2026-08-03；复核更新：2026-08-04 <br>
> 审核对象：`docs/蔚小芯智能体.md`（v1.5）与当前工作区已实现代码 <br>
> 审核基线：原报告 HEAD `cd52d07`；修复复核基线 `1d20434`；当前 HEAD `a6b2be0` 并纳入当前工作区改动 <br>
> 审核方法：文档契约逐项对照 + 后端路由/服务/存储源码审读 + 前端路由/页面/导出实现审读 + `go build` / `go test` / `flutter analyze` 验证 <br>
> 严重级别：阻断（Blocking）> 警告（Warning）> 备注（Note）

---

## 0. 修复状态更新（2026-08-04）

第一优先级阻断项已按以下状态修复或部分修复。本文档第 1-12 节已按 2026-08-04 复核后的当前状态更新，不再将已解决或外部联调前置项重复列为当前阻断问题：

| 原 ID | 修复内容 | 状态 |
|---|---|---|
| DPV4F-B1 | 后端新增 `POST /chat/stream` SSE 流式问答；LLM 客户端新增 `Stream` 流式接口；Web 前端 `chat_stream_web.dart` 已消费增量事件 | 已实现 |
| DPV4F-B2 | 引入 `github.com/cloudwego/eino v0.9.13`，新增 `server/internal/agent/eino_orchestrator.go`，以 Eino Graph 作为多智能体运行时入口 | 已实现 |
| DPV4F-B3 | 新增 `POST /auth/sso/callback` 与 `AuthService.LoginBySSOTicket`，支持 OAuth2 code/ticket 交换、用户信息映射、`SSO_MOCK=true` 演示模式 | 已实现代码链路，真实校方 SSO 仍需联调参数 |
| DPV4F-B4 | 学工/一表通代理增加超时重试与状态返回；真实凭证和校方联调环境仍需提供 | 部分，外部联调依赖 |
| DPV4F-B5 | 新增 `RetentionService` 与启动清理任务，默认审计/导出日志 180 天、会话/情感记录 365 天，可通过环境变量配置 | 已实现 |
| DPV4F-B6 | 新增标准知识包 `GET /kb/export/package`、`POST /kb/import/package`，实现 `manifest.json + resources.ndjson + attachments/`、复合 cursor 分页、sha256 与 HMAC 校验；新增 `uploadId/chunkIndex/totalChunks` 初始化、上传、状态、完成接口 | 已实现 |
| DPV4F-B7 | 后端多格式导出补齐 docx/xlsx/ics/png/json/md/pdf；PDF/PNG 使用 CJK 字体渲染；`export_logs` 已写入导出审计 | 后端已实现，前端导出入口仍以 PDF/PNG/MD 为主 |

### 外部依赖暂缓项（明确不在本轮处理）

以下事项依赖校方/第三方提供参数与联调环境，当前项目侧无法独立闭环，已明确暂缓并写入本报告，后续审核不再将其重复列为“未实现/阻断”：

| 事项 | 暂缓原因 | 恢复条件 |
|---|---|---|
| 真实校方 SSO 参数与联调环境 | 需要门户/信息中心提供 SSO 客户端、回调地址、票据与用户接口 | 校方提供联调参数后，按已实现的 `POST /auth/sso/callback` 链路联调 |
| 学工系统真实凭证与联调环境 | 需要 `XUEGONG_BASE_URL`、`XUEGONG_TOKEN` 及校方接口授权 | 校方提供凭证与接口后，按已实现的代理链路联调 |
| 一表通真实凭证与联调环境 | 需要 `YBT_BASE_URL`、`YBT_TOKEN` 及校方接口授权 | 校方提供凭证与接口后，按已实现的代理链路联调 |

修复验证：

| 命令 | 结果 |
|---|---|
| `go build ./server/...` | 通过 |
| `go test ./server/internal/service -run 'TestKnowledge\|TestExport\|TestRetention'` | 通过 |
| `TestKnowledgeImportChunkResume` | 通过（乱序上传、状态查询、整包校验、完成后导入） |
| `go test ./server/internal/agent ./server/internal/handler -run 'TestAgent\|TestChat\|TestExport\|TestAuth'` | 通过 |
| 全量 `go test ./server/internal/...` | 仍有 2 个真实 DOCX 解析用例失败（非本次阻断项，见 W8） |

---

## 目录

1. [结论摘要](#1-结论摘要)
2. [需求满足度总览](#2-需求满足度总览)
3. [架构与技术选型审计](#3-架构与技术选型审计)
4. [核心功能审计](#4-核心功能审计)
5. [Context Engine 审计](#5-context-engine-审计)
6. [知识库同步与多格式导出审计](#6-知识库同步与多格式导出审计)
7. [安全与合规审计](#7-安全与合规审计)
8. [工程质量与测试验证](#8-工程质量与测试验证)
9. [问题清单](#9-问题清单)
10. [综合评分](#10-综合评分)
11. [修复路线图](#11-修复路线图)
12. [最终结论](#12-最终结论)

---

## 1. 结论摘要

项目已经形成一套可编译、可演示的 Flutter + Go/Gin + SQLite 学工智能体骨架，**核心 P0/P1 功能多数已有真实后端实现**，包括六/八级角色 RBAC、知识库 CRUD、FTS5/BM25 检索、多智能体路由与汇聚、语音 ASR/TTS、情感预警、会话、反馈、推荐、审计日志、PII 脱敏与内容安全过滤。

截至 2026-08-04 复核，原 7 个阻断项已全部处理或转入外部联调阶段：

1. **智能问答已支持 SSE 流式**，并已接入 Eino Graph；`/chat/stream` 增量输出，Eino 初始化失败时回退自研编排。
2. **知识同步已按标准包协议实现**：`manifest.json + resources.ndjson + attachments`、复合 cursor 分页、断点续传、sha256 与 HMAC 校验。
3. **多格式导出后端已补齐** docx/xlsx/ics/png/json/md/pdf，PDF/PNG 使用 CJK 字体渲染，导出审计已写入 `export_logs`；前端导出入口仍以 PDF/PNG/MD 为主。
4. **P0 真实数据链路仍依赖校方联调**：SSO callback 与学工/一表通代理链路已实现，真实参数、凭证和环境未提供。
5. **保留策略已落地**：审计/导出日志默认 180 天，会话/情感记录默认 365 天；情感分析独立授权仍属未处理警告项。

综合评分建议：**7.4/10**。功能与阻断项已明显收敛，但正式验收仍需校方外部联调、200 条评测 KPI 报告，以及 W1-W9 等收尾项。

---

## 2. 需求满足度总览

| 分期 | 需求项 | 状态 | 说明 |
|---|---|---|---|
| P0 | Gin + JWT + RBAC | 满足 | 能力授权已覆盖，角色数量超出文档 6 级要求 |
| P0 | Context Engine 主链路 | 部分 | 行为上“结构化优先 + FTS/BM25 + sources”成立；但 `context_engine` 包在生产代码中未实例化，实际链路在 `ChatService` |
| P0 | 智能问答 Eino 编排 + SSE | 满足 | `POST /chat/stream` SSE 增量输出；Eino Graph 已接入，初始化失败回退自研编排 |
| P0 | SQLite 审计日志 | 满足 | `AuditLog` 中间件写入 `audit_logs` |
| P0 | 知识库 CRUD + 导入导出 | 满足（后端）/部分（前端入口） | 标准包、分页、断点续传、hash/HMAC 已实现；前端导出仍以 PDF/PNG/MD 为主 |
| P0 | Flutter 多端前端 | 部分 | Flutter/Dio/Provider 存在；本地存储用 `shared_preferences`，未用文档规定的 Hive |
| P0 | 入学/离校教育知识域 | 部分 | 知识大厅与流程数据存在；种子数据约 45 条迁移资源 + 50 行未接线的 NDJSON，未验证覆盖完整入学/离校场景 |
| P0 | 至少一条学工/一表通只读代理 | 部分（外部联调前置） | 代理与超时重试已实现，真实凭证/联调环境待校方提供 |
| P1 | 语音 ASR/TTS | 满足 | 讯飞 WebSocket 客户端 + 后端代理 + 前端录音/播放 |
| P1 | 情感预警 | 部分 | 分析/告警/趋势/处理均有实现；缺少情感单独授权 |
| P1 | 多智能体管理端 | 满足 | CRUD/启停/提示词存在；Eino Graph 已接入，失败回退自研编排 |
| P1 | 会话管理、消息重试、前端打磨 | 满足 | 有对应页面、provider 与错误状态 |
| P2 | 推荐 API | 满足 | 基于历史提问、角色权重、季节、多样性兜底 |
| P2 | Temporal 关键链路 | 部分 | 依赖与工作流代码存在，默认环境未启用，Vercel 明确禁用 |
| P2 | 个性化推荐增强/趋势报表 | 部分 | 基础推荐存在；文档中的 P2 深度形态未完整落地 |

---

## 3. 架构与技术选型审计

### 3.1 技术选型对照

| 文档选型 | 实际实现 | 结论 |
|---|---|---|
| Flutter + Dart | `frontend/`，Flutter 项目 | 满足 |
| Dio | `frontend/lib/services/api_service.dart` | 满足 |
| Provider | `frontend/lib/providers/*.dart` | 满足 |
| Hive | 未在 `pubspec.yaml` 中找到；实际使用 `shared_preferences` | 未满足 |
| Golang + Gin | `server/` + `github.com/gin-gonic/gin` | 满足 |
| JWT | `server/internal/jwtutil`、`middleware/jwt.go` | 满足 |
| RBAC 六级角色 | `server/internal/auth/capabilities.go`，实际含 guest/teacher/assistant 等 9 角色 | 满足并扩展 |
| SQLite（含 FTS） | `modernc.org/sqlite`、`kb_fts` | 满足 |
| Eino 编排 | `github.com/cloudwego/eino v0.9.13` + `server/internal/agent/eino_orchestrator.go`；失败时回退自研编排 | 满足 |
| 智谱/DeepSeek/讯飞 | `server/internal/llm/*` | 满足 |
| 讯飞 ASR/TTS WebSocket | `server/internal/llm/xfyun.go` | 满足 |
| Temporal 可选 | `go.temporal.io/sdk` + `server/internal/temporal` | 满足可选要求 |
| 不使用 Docker/Coze/本地大模型 | 未见 Docker/Coze 依赖 | 满足 |

### 3.2 架构问题与处理状态

| ID | 级别 | 问题 | 证据 |
|---|---|---|---|
| DPV4F-B1 | 已解决 | 原：智能问答缺少 SSE 流式；现：`POST /chat/stream` 提供 SSE 增量输出 | `server/pkg/app/app.go`、`server/internal/handler/chat_handler.go`、前端 `chat_stream_web.dart` |
| DPV4F-B2 | 已解决 | 原：Eino 未接入；现：Eino Graph 编排已接入，初始化失败回退自研编排 | `go.mod`、`server/internal/agent/eino_orchestrator.go` |
| DPV4F-W1 | 警告 | `context_engine` 包存在但生产代码未调用，文档所称的 Context Engine 实际由 `ChatService` + `KBRepo` 直接实现 | `server/internal/context_engine/engine.go:87` 定义 `New`，生产代码无引用 |
| DPV4F-W2 | 警告 | 前端本地存储偏离技术栈要求 | `frontend/pubspec.yaml:42` 使用 `shared_preferences`，未用 Hive |

---

## 4. 核心功能审计

### 4.1 智能问答

| 检查项 | 结果 | 证据 |
|---|---|---|
| 会话归属校验 | 满足 | `chat_service.go:148-155` 校验会话属于当前用户 |
| 多智能体路由 | 满足 | `agent/router.go:247` 关键词路由；Eino Graph 已接入，失败回退自研编排 |
| 知识检索 | 满足 | `chat_service.go:179-202` 结构化优先 + FTS 合并 + 相关性过滤 |
| LLM 前脱敏 | 满足 | `chat_service.go:213`、`util/pii_mask.go` |
| 内容过滤 | 满足 | `chat_service.go:168,242` |
| 低置信兜底 | 满足 | `chat_service.go:206-209,449-461` |
| sources 必填 | 满足 | `chat_service.go:405-433` 从检索结果/多智能体结果构造 |
| AnswerCard 结构化 | 部分 | 只填充 `conclusion/sources/followUps/confidence/fallback`；`steps/risks/actions` 基本为空，未从 LLM 输出解析为结构化卡片 | `chat_service.go:396-466` |
| SSE 流式 | 满足 | `POST /chat/stream` 已提供 SSE 增量输出，前端 `chat_stream_web.dart` 消费 |

### 4.2 情感分析预警

| 检查项 | 结果 | 证据 |
|---|---|---|
| LLM 情感分析 | 满足 | `emotion_service.go:137-153` |
| 连续高风险升级 | 满足 | `emotion_service.go:122-135` |
| 告警列表/处理/趋势 | 满足 | `emotion_handler.go:70-205` |
| 异步聊天情感分析 | 满足 | `chat_handler.go:85-100` |
| 情感独立授权 | 未满足 | 文档 `docs/privacy-policy.md:35` 与前端 `consent_page.dart` 提及情感数据，但 `middleware/consent.go` 只有统一授权，未发现独立“情感分析”授权开关或弹窗 |

### 4.3 语音、推荐、会话、反馈

| 功能 | 结果 | 证据 |
|---|---|---|
| ASR/TTS | 满足 | `voice_handler.go:39-157`、`xfyun.go:56-315` |
| 个性化推荐 | 满足 | `recommendation_service.go:79-166` |
| 会话列表/消息/删除 | 满足 | `session_handler.go`、`session_repo.go` |
| 反馈闭环 | 满足 | `feedback_handler.go`、`feedback_service.go` |
| 教师/教辅/学院高级功能 | 警告 | 大量 `data_source: "fallback"` 与硬编码“张明/示例学生”数据，说明是演示兜底而非真实数据闭环 | `teacher_service.go:92`、`assistant_service.go:189`、`counselor_handler.go:38` |

---

## 5. Context Engine 审计

### 5.1 行为链路

实际生产链路可以归纳为：

1. 结构化检索：`kb_repo.go:102` `SearchStructured`，按 title/tags/type 直接匹配。
2. FTS/BM25：`kb_repo.go:49` `Search`，使用 `kb_fts` 与 `bm25()`。
3. 范围过滤：`kb_repo.go:133-134,185-186` 按 `owner_scope/owner_id/role_scope/status=published` 过滤。
4. 上下文拼装：`chat_service.go:284-355` 将知识、多智能体结果、历史消息拼入 system prompt。
5. 来源附加与兜底：`chat_service.go:396-466`。

### 5.2 结论

行为层面满足“结构化优先 + FTS/BM25 + 上下文拼装 + sources + 兜底”的主链路。但存在以下问题：

| ID | 级别 | 问题 | 证据 |
|---|---|---|---|
| DPV4F-W1 | 警告 | `context_engine` 包未接入生产链路，形成文档/目录结构与实际实现不一致 | 生产代码无 `context_engine.New` 或 `engine.Query` |
| DPV4F-W3 | 警告 | 缺少文档 3.3.5 的最小质量看板：没有引用覆盖率、命中率、过期引用率、权限拦截率的统计实现；`chat_metrics` 只统计置信度/兜底/sources 数/P95 时延 | `chat_metrics_repo.go:34-112` |
| DPV4F-N1 | 备注 | 向量检索、Agentic RAG、Long Context 均为可选 P1/P2 能力，当前未实现，与“可插拔/按需”定位不冲突，但文档宣传需注明未启用 | `go.mod`、`context_engine` |

---

## 6. 知识库同步与多格式导出审计

### 6.1 知识资源模型

`KBResource` 已覆盖 `resourceId/resourceType/ownerScope/roleScope/version/status/title/summary/content/sourceLink/sourceVersion/effectiveAt/expiredAt/tags/updatedBy/updatedAt`，与文档 6.8.6 基本一致。`ProcessStep` 也覆盖材料、入口、时限、地点、联系方式、坐标等。

### 6.2 导出

| 文档要求 | 当前实现（2026-08-04） | 结论 |
|---|---|---|
| `manifest.json + resources.ndjson + attachments/` | `GET /kb/export/package` 输出标准知识包 | 满足 |
| 全量/增量导出 | 支持全量与增量参数 | 满足 |
| 单调递增 cursor | 使用复合 cursor 分页，避免单时间戳漏改或排序不稳定 | 满足 |
| 分页 `limit/nextCursor/hasMore/exportBatchId` | 已实现复合 cursor 分页与 `hasMore` | 满足 |
| 资源 hash 与包签名 | 已实现 sha256 与 HMAC 校验 | 满足 |
| 审计记录导出人/角色/格式/是否敏感字段 | `export_logs` 已由导出审计服务写入 | 满足 |

证据：

- `server/internal/service/knowledge_package_service.go`
- `server/internal/service/export_log_service.go`
- `server/internal/repository/export_log_repo.go`
- `server/internal/handler/export_handler.go`

### 6.3 导入

| 文档要求 | 当前实现 | 结论 |
|---|---|---|
| 标准包或 NDJSON | `POST /kb/import/package` 支持标准知识包；原 NDJSON/JSON wrapper 保留 | 满足 |
| zip/manifest/hash/签名校验 | 已实现 sha256 与 HMAC 校验 | 满足 |
| `uploadId/chunkIndex/totalChunks` 断点续传 | 已实现初始化、分片上传、状态查询、完成导入 | 满足 |
| 幂等与冲突策略 | `(resourceId, version)` 比较存在，但版本比较仅支持 `%d.%d.%d`，文档版本 `YYYYMMDD-vN` 会解析失败并被保守视为相同 | 部分 |
| retired 传播 | 有，`status=retired` 无条件覆盖 | 满足 |

证据：

- `server/internal/service/knowledge_import_resume.go`
- `server/internal/service/knowledge_package_service.go`
- `server/internal/handler/kb_handler.go`
- `server/internal/service/kb_service.go`

### 6.4 多格式导出

| 文档要求 | 当前实现 |
|---|---|
| PDF | 后端已支持，使用 CJK 字体渲染；前端 PDF 仍以“打开新窗口打印”为主 |
| Word(.docx) | 后端已支持 |
| Markdown | 后端与前端均有，前端是复制到剪贴板 |
| Excel(.xlsx) | 后端已支持 |
| PNG 长图 | 后端已支持（CJK 渲染），前端仍以 HTML 长图为主 |
| ICS | 后端已支持 |
| JSON | 后端支持 |

证据：

- `server/internal/service/export_render.go`
- `server/internal/service/export_service.go`
- `server/internal/handler/export_handler.go`
- `frontend/lib/widgets/export_dialog.dart:29-58` 提供 PDF/PNG/MD，但均为前端 HTML/剪贴板方案

---

## 7. 安全与合规审计

### 7.1 已满足项

| 项 | 结果 | 证据 |
|---|---|---|
| JWT 2 小时过期 | 满足 | `config.go` 默认 `JWT_EXPIRE_HOURS=2` |
| Token 吊销版本 | 满足 | `user_upsert.go:48-55` |
| 能力级 RBAC | 满足 | `app.go` 大量 `RequireCapability` |
| 限流 | 满足 | 全局限流、登录限流、聊天用户限流 |
| 审计日志 | 满足 | `middleware/audit.go:36` |
| PII 检测/脱敏 | 满足 | `middleware/pii.go`、`util/pii_mask.go` |
| 内容安全过滤 | 满足 | `util/content_filter.go` |
| 统一隐私协议/用户协议授权 | 满足 | `middleware/consent.go`、`frontend/lib/pages/consent` |
| 数据保留与自动清理 | 满足 | `retention_service.go`：审计/导出日志 180 天，会话/情感记录 365 天，可通过环境变量配置 |
| bcrypt 密码 | 满足 | `auth_service.go` |
| 生产密钥校验 | 满足 | `config.Validate` 对 `JWT_SECRET` 强校验 |

### 7.2 未满足/风险项与外部联调前置项

| ID | 级别 | 问题 | 证据 |
|---|---|---|---|
| DPV4F-B3 | 外部联调 | SSO 代码链路已实现（`POST /auth/sso/callback`、`SSO_MOCK=true`），真实校方票据/回调参数待提供 | `server/internal/service/auth_service.go`、`server/internal/handler/auth_handler.go` |
| DPV4F-B4 | 外部联调 | 学工/一表通代理已实现超时重试与状态返回，真实凭证与接口环境待校方提供 | `server/internal/service/integration_service.go` |
| DPV4F-W4 | 警告 | 情感分析独立授权缺失 | `middleware/consent.go:13-50` 只有统一授权；`chat_handler.go:85` 异步情感分析直接执行 |
| DPV4F-W5 | 警告 | 限频配额与文档 9.4 不符：文档建议学生 20 次/日、辅导员 50 次/日，默认配置为 200 次/日、3000 次/月 | `config.go` `DAILY_CHAT_QUOTA_PER_USER=200` |
| DPV4F-W6 | 警告 | 未实现“双模型路由/超时切换”，只按 DeepSeek 优先选择单一模型，失败即兜底 | `app.go:121-131`、`chat_service.go:227-236` |
| DPV4F-W7 | 警告 | API 契约偏差：文档 A.5 错误码为 `0/4xxx/5xxx`，实现为 `10001/20001/30001...`；部分路径也不一致（如 `/users/me` vs `/user/profile`、`/chat/ask` vs `/chat`） | `model/errcode.go`、`app.go:656,815` |

---

## 8. 工程质量与测试验证

### 8.1 命令验证

| 命令 | 结果 |
|---|---|
| `go build ./server/...` | 通过 |
| `go test ./server/internal/service -run 'TestKnowledge\|TestExport\|TestRetention'` | 通过 |
| `go test ./server/internal/agent ./server/internal/handler -run 'TestAgent\|TestChat\|TestExport\|TestAuth'` | 通过 |
| 全量 `go test ./server/internal/...` | 失败：`TestParseRealHandbookEndToEnd` 与 `TestReadDocxRealFile` 对 `data/滁州学院2026级普通专升本新生入学须知.docx` 解析为空 |
| `flutter analyze --no-fatal-infos --no-fatal-warnings` | 退出码 0，但输出 181 条问题，其中含 `chat_page.dart` 多条 warning |

### 8.2 测试问题

| ID | 级别 | 问题 | 证据 |
|---|---|---|---|
| DPV4F-W8 | 警告 | 真实 DOCX 端到端用例失败，说明文档解析对新版“入学须知”仍不稳定，知识导入“Excel/Word/PDF/Markdown/网页链接”中 Word 链路存在风险 | `document_service_test.go:354`、`docparse_e2e_test.go:74-94` |
| DPV4F-W9 | 警告 | 前端静态分析未达到文档“零错误零警告”，且 `chat_page.dart` 有 `dead_null_aware_expression` 等 warning | `flutter analyze` 输出 |

### 8.3 评测与种子数据

| 项 | 结果 |
|---|---|
| 评测基线 | `server/testdata/eval/` 下 8 个业务域各 25 条，共 200 条；`server/cmd/eval/main.go` 可跑联调评测 |
| 评测通过率 | 未提供当前 HEAD 的通过率报告，不能据此认定命中率 >=85% |
| 迁移种子 | 迁移 SQL 中 `kb_resources` 插入约 45 条 |
| 种子 NDJSON | `server/data/seed/resources.ndjson` 有 50 行，但未发现迁移/启动代码加载该文件，运行时是否为 50 条未验证 |

---

## 9. 问题清单

截至 2026-08-04 复核，原 `DPV4F-B1`、`DPV4F-B2`、`DPV4F-B5`、`DPV4F-B6`、`DPV4F-B7` 已解决，`DPV4F-W10` 已随 B7 的导出审计落地解决，详见第 0 节。本节仅保留当前仍需处理或等待外部环境的事项。

| # | ID | 级别 | 模块 | 问题 |
|---|---|---|---|---|
| 1 | DPV4F-B3 | 外部联调 | 对接 | SSO 代码链路已实现，需校方参数与联调环境 |
| 2 | DPV4F-B4 | 外部联调 | 对接 | 学工/一表通代理已实现，需真实凭证与接口联调 |
| 3 | DPV4F-W1 | 警告 | Context Engine | `context_engine` 包未接入生产链路 |
| 4 | DPV4F-W2 | 警告 | 前端 | Hive 未落地，使用 shared_preferences |
| 5 | DPV4F-W3 | 警告 | 质量看板 | 文档 3.3.5 指标缺多数实现 |
| 6 | DPV4F-W4 | 警告 | 合规 | 情感分析缺少独立授权 |
| 7 | DPV4F-W5 | 警告 | 限流 | 配额默认值与文档不符 |
| 8 | DPV4F-W6 | 警告 | LLM 容灾 | 无双模型超时切换 |
| 9 | DPV4F-W7 | 警告 | API 契约 | 路径与错误码与附录 A 不一致 |
| 10 | DPV4F-W8 | 警告 | 文档解析 | 真实 DOCX 用例失败 |
| 11 | DPV4F-W9 | 警告 | 前端质量 | analyze 181 条问题 |
| 12 | DPV4F-N1 | 备注 | 可选能力 | 向量/Agentic RAG/Long Context 未启用 |
| 13 | DPV4F-N2 | 备注 | 演示数据 | 大量教师/教辅/学生扩展功能走 `fallback` 硬编码 |

---

## 10. 综合评分

| 维度 | 分数 | 说明 |
|---|---:|---|
| 架构与技术选型 | 8.5/10 | Gin/JWT/RBAC/SQLite/Flutter 扎实；Eino/SSE 已接入；Hive 仍缺 |
| Context Engine | 8.0/10 | 行为链路成立；包未接线、质量看板缺项 |
| 核心功能实现度 | 8.0/10 | P0/P1 主体完成；真实 SSO/数据源依赖外部联调 |
| 知识同步与导出 | 8.0/10 | 标准包、分页、签名、断点续传、多格式后端已实现；前端导出入口仍以 PDF/PNG/MD 为主 |
| 安全与合规 | 7.0/10 | 保留策略与基础安全已落地；真实 SSO 联调、情感授权、配额不一致仍待处理 |
| 工程质量 | 6.5/10 | 构建与目标测试通过，两个真实 DOCX 用例仍失败，前端 analyze 有 warning |
| 上线就绪度 | 5.0/10 | 真实认证/数据源/生产凭据/验收 KPI 报告缺失 |
| **综合** | **7.4/10** | 可演示，待外部联调与验收证据闭环 |

---

## 11. 修复路线图

### 11.1 外部联调与验收前置项

1. 校方提供真实 SSO 参数后，按已实现的 `POST /auth/sso/callback` 链路完成联调并验证登录。
2. 校方提供学工/一表通凭证后，完成真实只读链路联调，并输出数据源验收结果。
3. 基于 `server/testdata/eval/` 200 条评测集输出命中率、引用覆盖率、兜底率等 KPI 报告。

### 11.2 第二优先级（警告项）

- 让 `context_engine` 包真正承载检索管道，或删除死代码并同步 README/架构说明。
- 补齐文档 3.3.5 质量看板指标。
- 增加情感分析独立授权开关/弹窗。
- 修复 DOCX 解析失败用例并保持 `go test ./...` 全绿。
- 清理 `flutter analyze` warning/info，至少达到文档声明的零错误零警告口径。
- 调整配额默认值或同步文档；实现双模型超时切换。
- 统一附录 A 的路径与错误码，或升级文档为当前实际契约。
- 将 50 行种子 NDJSON 接入初始化/迁移，保证冷启动知识量可验证。
- 落地 Hive，或同步技术栈文档后保持现有本地存储方案。
- 前端导出入口补齐 Word/Excel/ICS 等格式，或将后端格式能力接入前端。

---

## 12. 最终结论

**整体评级：可演示，待外部联调后正式验收。**

截至 2026-08-04，原 7 个阻断项已实现或转入外部联调，正文不再将它们列为当前代码缺口。当前主要风险集中在真实校方 SSO/学工/一表通联调、200 条评测 KPI 报告，以及 W1-W9 等收尾项；完成这些后可按附录 A 重新组织正式验收。

---

*报告生成：2026-08-03 | 复核更新：2026-08-04 | 审核工具：源码静态审读 + 文档对照 + `go build` + `go test` + `flutter analyze`*
