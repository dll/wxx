# 蔚小芯 DPV4F 审核报告 v1

> 审核日期：2026-08-03  
> 审核对象：`docs/蔚小芯智能体.md`（v1.5）与当前工作区已实现代码  
> 审核基线：HEAD `cd52d07`，并纳入当前未提交工作区改动  
> 审核方法：文档契约逐项对照 + 后端路由/服务/存储源码审读 + 前端路由/页面/导出实现审读 + `go build` / `go test` / `flutter analyze` 验证  
> 严重级别：阻断（Blocking）> 警告（Warning）> 备注（Note）

---

## 0. 修复状态更新（2026-08-04）

第一优先级阻断项已按以下状态修复或部分修复：

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

但对照《蔚小芯智能体.md》v1.5 的验收要求，当前状态**尚未达到正式验收门槛**。主要阻断点集中在：

1. **智能问答不是 SSE 流式**，且未引入文档固定选型 Eino。
2. **知识同步导出/导入未按标准包协议实现**：无 `manifest.json + resources.ndjson + attachments` 标准包、无分页/游标/断点续传、导入不校验 hash/签名。
3. **多格式导出仅支持 PDF/JSON/Markdown**，且 PDF 用标准 Helvetica 生成，中文会被替换成 `?`，Word/Excel/PNG/ICS 均未实现。
4. **P0 真实数据链路未闭环**：SSO 未接入，学工/一表通仅有未配置即不可用的 HTTP 代理，没有已验证的真实只读数据。
5. **合规缺口**：情感分析没有独立授权弹窗；会话/情感/审计/导出日志没有 180 天或一学年保留与自动清理机制；导出日志表存在但没有代码写入。

综合评分建议：**6.3/10**。工程骨架合格，但按文档“上线门槛”和附录 A 联调验收用例衡量，仍有较多阻断级缺口。

---

## 2. 需求满足度总览

| 分期 | 需求项 | 状态 | 说明 |
|---|---|---|---|
| P0 | Gin + JWT + RBAC | 满足 | 能力授权已覆盖，角色数量超出文档 6 级要求 |
| P0 | Context Engine 主链路 | 部分 | 行为上“结构化优先 + FTS/BM25 + sources”成立；但 `context_engine` 包在生产代码中未实例化，实际链路在 `ChatService` |
| P0 | 智能问答 Eino 编排 + SSE | 未满足 | 无 Eino 依赖；`/chat` 为普通 JSON 响应，无 SSE 流式 |
| P0 | SQLite 审计日志 | 满足 | `AuditLog` 中间件写入 `audit_logs` |
| P0 | 知识库 CRUD + 导入导出 | 部分 | CRUD 与 NDJSON 导入存在；导出为 JSON 包裹，未达标准包协议 |
| P0 | Flutter 多端前端 | 部分 | Flutter/Dio/Provider 存在；本地存储用 `shared_preferences`，未用文档规定的 Hive |
| P0 | 入学/离校教育知识域 | 部分 | 知识大厅与流程数据存在；种子数据约 45 条迁移资源 + 50 行未接线的 NDJSON，未验证覆盖完整入学/离校场景 |
| P0 | 至少一条学工/一表通只读代理 | 未满足 | 代理代码存在，但依赖空环境变量，无真实联调证据 |
| P1 | 语音 ASR/TTS | 满足 | 讯飞 WebSocket 客户端 + 后端代理 + 前端录音/播放 |
| P1 | 情感预警 | 部分 | 分析/告警/趋势/处理均有实现；缺少情感单独授权 |
| P1 | 多智能体管理端 | 部分 | CRUD/启停/提示词存在；运行器是自研 `Orchestrator`，不是 Eino |
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
| Eino 编排 | `go.mod` 中无 `github.com/cloudwego/eino`；自研 `agent.Orchestrator` | 未满足 |
| 智谱/DeepSeek/讯飞 | `server/internal/llm/*` | 满足 |
| 讯飞 ASR/TTS WebSocket | `server/internal/llm/xfyun.go` | 满足 |
| Temporal 可选 | `go.temporal.io/sdk` + `server/internal/temporal` | 满足可选要求 |
| 不使用 Docker/Coze/本地大模型 | 未见 Docker/Coze 依赖 | 满足 |

### 3.2 架构问题

| ID | 级别 | 问题 | 证据 |
|---|---|---|---|
| DPV4F-B1 | 阻断 | 智能问答缺少 SSE 流式；文档 P0 明确要求 SSE，附录 A.3 示例与 7.0.1 进度也声明“Eino 编排 + SSE” | `server/pkg/app/app.go:656` 注册普通 `POST /chat`；`server/internal/handler/chat_handler.go:40` 一次性返回 JSON；全库未找到 `text/event-stream` |
| DPV4F-B2 | 阻断 | Eino 未接入，多智能体运行器为自研并发协程 | `go.mod` 无 Eino；`server/internal/agent/orchestrator.go:25` |
| DPV4F-W1 | 警告 | `context_engine` 包存在但生产代码未调用，文档所称的 Context Engine 实际由 `ChatService` + `KBRepo` 直接实现 | `server/internal/context_engine/engine.go:87` 定义 `New`，生产代码无引用 |
| DPV4F-W2 | 警告 | 前端本地存储偏离技术栈要求 | `frontend/pubspec.yaml:42` 使用 `shared_preferences`，未用 Hive |

---

## 4. 核心功能审计

### 4.1 智能问答

| 检查项 | 结果 | 证据 |
|---|---|---|
| 会话归属校验 | 满足 | `chat_service.go:148-155` 校验会话属于当前用户 |
| 多智能体路由 | 满足 | `agent/router.go:247` 关键词路由；`orchestrator.go:49` 并行执行 |
| 知识检索 | 满足 | `chat_service.go:179-202` 结构化优先 + FTS 合并 + 相关性过滤 |
| LLM 前脱敏 | 满足 | `chat_service.go:213`、`util/pii_mask.go` |
| 内容过滤 | 满足 | `chat_service.go:168,242` |
| 低置信兜底 | 满足 | `chat_service.go:206-209,449-461` |
| sources 必填 | 满足 | `chat_service.go:405-433` 从检索结果/多智能体结果构造 |
| AnswerCard 结构化 | 部分 | 只填充 `conclusion/sources/followUps/confidence/fallback`；`steps/risks/actions` 基本为空，未从 LLM 输出解析为结构化卡片 | `chat_service.go:396-466` |
| SSE 流式 | 未满足 | 见 DPV4F-B1 |

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

| 文档要求 | 当前实现 | 结论 |
|---|---|---|
| `manifest.json + resources.ndjson + attachments/` | 返回 JSON：`{code, message, manifest, data:[]}` | 未满足 |
| 全量/增量导出 | 支持 `resource_type` 与 `since` 参数 | 部分 |
| 单调递增 cursor | `since` 为时间戳，`ListSince` 用 `updated_at >= ?` 过滤 | 部分，可能漏改同一秒或排序不稳定 |
| 分页 `limit/nextCursor/hasMore/exportBatchId` | 无，单次最多 5000 条 | 未满足 |
| 资源 hash 与包签名 | 仅对整段 JSON 响应做可选 HMAC；无 `resourcesSha256/attachmentsSha256` | 未满足 |
| 审计记录导出人/角色/格式/是否敏感字段 | `export_logs` 表存在，但未发现写入代码；前端导出走本地 HTML，不调用后端 | 未满足 |

证据：

- `server/internal/handler/export_handler.go:96-155`
- `server/internal/repository/kb_repo.go:639-691`
- `server/internal/service/kb_service.go:494-501`
- `server/internal/model/dto.go:72-86`

### 6.3 导入

| 文档要求 | 当前实现 | 结论 |
|---|---|---|
| 标准包或 NDJSON | 支持 NDJSON 与 JSON wrapper | 部分 |
| zip/manifest/hash/签名校验 | 未实现 | 未满足 |
| `uploadId/chunkIndex/totalChunks` 断点续传 | 未实现 | 未满足 |
| 幂等与冲突策略 | `(resourceId, version)` 比较存在，但版本比较仅支持 `%d.%d.%d`，文档版本 `YYYYMMDD-vN` 会解析失败并被保守视为相同 | 部分 |
| retired 传播 | 有，`status=retired` 无条件覆盖 | 满足 |

证据：

- `server/internal/handler/kb_handler.go:197-264`
- `server/internal/service/kb_service.go:394-485`
- `server/internal/repository/kb_repo.go:507-578`

### 6.4 多格式导出

| 文档要求 | 当前实现 |
|---|---|
| PDF | 后端“手写最小 PDF”，但 `pdfEscape` 将非 ASCII 替换为 `?`，中文内容不可读；前端 PDF 是“打开新窗口打印” |
| Word(.docx) | 未实现 |
| Markdown | 后端与前端均有，前端是复制到剪贴板 |
| Excel(.xlsx) | 未实现 |
| PNG 长图 | 前端“新窗口 HTML 长图”，非真实导出文件；后端不支持 |
| ICS | 未实现 |
| JSON | 后端支持 |

证据：

- `server/internal/service/export_service.go:12-24` 只有 `pdf/json/md`
- `server/internal/service/export_service.go:214-230` 非 ASCII 转 `?`
- `server/internal/handler/export_handler.go:54-65` 仅接受 `json/md/pdf`
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
| bcrypt 密码 | 满足 | `auth_service.go` |
| 生产密钥校验 | 满足 | `config.Validate` 对 `JWT_SECRET` 强校验 |

### 7.2 未满足/风险项

| ID | 级别 | 问题 | 证据 |
|---|---|---|---|
| DPV4F-B3 | 阻断 | 无真实 SSO；当前只有账号密码、二维码、游客注册，文档 P0 要求统一认证或“Mock 到真实切换”，代码仍为开发登录 | `auth_service.go:246-283`；全库未发现 SSO callback/票据换取实现 |
| DPV4F-B4 | 阻断 | 学工/一表通未接入真实数据，仅代理框架；文档 P0 要求至少一条真实只读数据 | `integration_service.go:34-56`；`.env` 与代码未见有效联调凭证 |
| DPV4F-B5 | 阻断 | 缺少 180 天审计日志、一学年会话/情感数据自动清理或匿名化；保留要求没有落地 | `audit_repo.go`、`session_repo.go`、`emotion_repo.go` 均无清理任务；全库未找到 retention/purge 调度 |
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
| `go test ./server/internal/...` | 失败：`TestParseRealHandbookEndToEnd` 与 `TestReadDocxRealFile` 对 `data/滁州学院2026级普通专升本新生入学须知.docx` 解析为空 |
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

| # | ID | 级别 | 模块 | 问题 |
|---|---|---|---|---|
| 1 | DPV4F-B1 | 阻断 | 智能问答 | 无 SSE 流式 |
| 2 | DPV4F-B2 | 阻断 | 多智能体 | 未接入 Eino，与最终选型不符 |
| 3 | DPV4F-B3 | 阻断 | 对接 | 无真实 SSO 统一认证 |
| 4 | DPV4F-B4 | 阻断 | 对接 | 学工/一表通无真实数据链路 |
| 5 | DPV4F-B5 | 阻断 | 合规 | 无数据保留期与自动清理 |
| 6 | DPV4F-B6 | 阻断 | 知识同步 | 导出/导入未按标准包协议，无 hash/签名/分页/断点续传 |
| 7 | DPV4F-B7 | 阻断 | 多格式导出 | 仅 PDF/JSON/MD；PDF 中文不可读；Word/Excel/PNG/ICS 缺失 |
| 8 | DPV4F-W1 | 警告 | Context Engine | `context_engine` 包未接入生产链路 |
| 9 | DPV4F-W2 | 警告 | 前端 | Hive 未落地，使用 shared_preferences |
| 10 | DPV4F-W3 | 警告 | 质量看板 | 文档 3.3.5 指标缺多数实现 |
| 11 | DPV4F-W4 | 警告 | 合规 | 情感分析缺少独立授权 |
| 12 | DPV4F-W5 | 警告 | 限流 | 配额默认值与文档不符 |
| 13 | DPV4F-W6 | 警告 | LLM 容灾 | 无双模型超时切换 |
| 14 | DPV4F-W7 | 警告 | API 契约 | 路径与错误码与附录 A 不一致 |
| 15 | DPV4F-W8 | 警告 | 文档解析 | 真实 DOCX 用例失败 |
| 16 | DPV4F-W9 | 警告 | 前端质量 | analyze 181 条问题 |
| 17 | DPV4F-W10 | 警告 | 导出审计 | `export_logs` 无写入 |
| 18 | DPV4F-N1 | 备注 | 可选能力 | 向量/Agentic RAG/Long Context 未启用 |
| 19 | DPV4F-N2 | 备注 | 演示数据 | 大量教师/教辅/学生扩展功能走 `fallback` 硬编码 |

---

## 10. 综合评分

| 维度 | 分数 | 说明 |
|---|---:|---|
| 架构与技术选型 | 7.5/10 | Gin/JWT/RBAC/SQLite/Flutter 扎实；缺 Eino/Hive/SSE |
| Context Engine | 8.0/10 | 行为链路成立；包未接线、质量看板缺项 |
| 核心功能实现度 | 7.0/10 | P0/P1 主体在；AnswerCard 结构化字段、真实对接未闭环 |
| 知识同步与导出 | 4.0/10 | 标准包、分页、签名、断点续传、多格式均未达标 |
| 安全与合规 | 6.5/10 | 基础安全到位；SSO、数据保留、情感授权、配额不一致 |
| 工程质量 | 6.5/10 | 构建通过，单测有两个真实 DOCX 失败，前端 analyze 有 warning |
| 上线就绪度 | 4.0/10 | 真实认证/数据源/生产凭据/验收 KPI 报告缺失 |
| **综合** | **6.3/10** | 可演示、不可按文档验收上线 |

---

## 11. 修复路线图

### 11.1 第一优先级（阻断项）

1. 将 `/chat` 改为 SSE 流式响应，并在文档或代码中明确 Eino 接入计划；若继续自研编排，必须修改需求文档并说明理由。
2. 按文档 6.8.7/6.8.8 实现标准知识导出包：`manifest.json + resources.ndjson + attachments`、单调 cursor、分页、资源 hash、HMAC 签名、导入校验与断点续传。
3. 补齐多格式导出：PDF 使用支持 CJK 的字体库或成熟库，并实现 Word、Excel、PNG、ICS、JSON；所有导出写入 `export_logs`。
4. 接入真实 SSO 或至少完成可切换的 SSO 回调；接入并验证一条真实学工/一表通只读数据。
5. 落地数据保留策略：审计日志 180 天、会话/情感记录一学年自动清理或匿名化；增加情感分析独立授权。

### 11.2 第二优先级（警告项）

- 让 `context_engine` 包真正承载检索管道，或删除死代码并同步 README/架构说明。
- 补齐文档 3.3.5 质量看板指标。
- 修复 DOCX 解析失败用例并保持 `go test ./...` 全绿。
- 清理 `flutter analyze` warning/info，至少达到文档声明的零错误零警告口径。
- 调整配额默认值或同步文档；实现双模型超时切换。
- 统一附录 A 的路径与错误码，或升级文档为当前实际契约。
- 将 50 行种子 NDJSON 接入初始化/迁移，保证冷启动知识量可验证。

---

## 12. 最终结论

**整体评级：可演示，未达正式验收。**

当前代码已具备“智能问答 + 知识库 + 语音 + 情感 + 会话 + 反馈 + 管理端”的可运行原型，工程基础好于纯 mock 项目。但文档固定了 Eino/SSE、标准知识同步包、多格式导出、真实 SSO 与真实只读对接、隐私保留策略等硬验收点，这些在 v1 报告所核验的工作区中仍处于缺口状态。建议先把 7 个阻断项和文档/代码一致性修复后再进入正式验收；KPI 命中率、引用覆盖率、兜底率需基于 200 条评测集输出可复核报告。

---

*报告生成：2026-08-03 | 审核工具：源码静态审读 + 文档对照 + `go build` + `go test` + `flutter analyze`*
