# 蔚小芯 DPV4P 审核报告 v1

> **审核日期**：2026-07-25  
> **审核范围**：Go/Gin 后端（server/）、Flutter 前端（frontend/）、数据库迁移（migrations/）、设计文档（docs/、specs/）  
> **审核方法**：静态代码审计 + 架构合规检查 + 安全扫描 + 知识管道追踪  
> **严重级别**：阻断（Blocking）> 警告（Warning）> 备注（Note）

---

## 目录

1. [架构合规](#1-架构合规)
2. [安全审计](#2-安全审计)
3. [知识管道（Context Engine）审计](#3-知识管道context-engine审计)
4. [代码质量审计](#4-代码质量审计)
5. [文档完整性审计](#5-文档完整性审计)
6. [综合评分与修复路线图](#6-综合评分与修复路线图)

---

## 1. 架构合规

### 1.1 总体评估

| 标准 | 状态 |
|------|------|
| handler → service → repository 单向分层 | **已违反** — 4 个 handler 含阻断级违规 |
| 禁止 handler 导入 repository | **已违反** — `student_handler`、`kb_handler`、`admin_handler`、`feedback_handler` |
| 禁止 handler 导入 llm | **已违反** — `voice_handler` |
| `service.NewXxx()` 工厂模式 | **通过** — `app.go` 100% 合规 |
| `model.ErrorResponse` 一致使用 | **部分通过** — 7/10 handler 合规 |

### 1.2 阻断级违规

#### B1. student_handler.go — 直接持有并调用 Repository

| 项目 | 位置 |
|------|------|
| 文件 | `server/internal/handler/student_handler.go` |
| 违规导入 | 第 11 行：`"github.com/dll/wxx/server/internal/repository"` |
| 直接持有 Repository 字段 | 第 19 行：`kbRepo *repository.KBRepo` |
| Setter 绕过 Service 层 | 第 28 行：`SetKBRepo(kb *repository.KBRepo)` |
| 直接调用 GetByResourceID | 第 341 行：`h.kbRepo.GetByResourceID(resourceID)` |
| 直接调用 GetProcessSteps | 第 355 行：`h.kbRepo.GetProcessSteps(resourceID)` |
| 所有错误响应使用 `gin.H{}` | 从未使用 `model.ErrorResponse` |

**修复建议**：  
- 移除 `kbRepo` 字段和 `SetKBRepo()`  
- 将 `ProcessEnhanced()` 中的 repository 调用迁移到 `StudentService`  
- 将所有 `gin.H` 响应替换为 `model.ErrorResponse{Code, Message, TraceID}`

#### B2. feedback_handler.go — 直接持有并调用 Repository

| 项目 | 位置 |
|------|------|
| 文件 | `server/internal/handler/feedback_handler.go` |
| 违规导入 | 第 13 行：`"github.com/dll/wxx/server/internal/repository"` |
| 直接持有 Repository | 第 22 行：`screenshotRepo *repository.FeedbackScreenshotRepo` |
| 直接注入 Repository | 第 26 行：构造函数接收具体 Repository 类型 |
| 直接调用 Save | 第 233 行：`h.screenshotRepo.Save(...)` |
| 直接调用 GetByFilename | 第 260 行：`h.screenshotRepo.GetByFilename(...)` |

**修复建议**：  
- 在 `FeedbackService` 中新增 `SaveScreenshot()` / `GetScreenshot()` 方法  
- Handler 改为只依赖 `*service.FeedbackService`

#### B3. voice_handler.go — 直接持有并调用 LLM

| 项目 | 位置 |
|------|------|
| 文件 | `server/internal/handler/voice_handler.go` |
| 违规导入 | 第 12 行：`"github.com/dll/wxx/server/internal/llm"` |
| 直接持有 LLM 客户端 | 第 36 行：`NewVoiceHandler(xfClient *llm.XfyunClient)` |
| 直接调用 ASR | 第 96 行：`h.xfClient.ASR(ctx, pcmBytes)` |
| 直接调用 TTS | 第 146 行：`h.xfClient.TTS(ctx, req.Text, req.Voice)` |

**修复建议**：  
- 引入 `service.VoiceService` 中介层  
- Handler 改为依赖 Service 接口而非具体 LLM 类型

### 1.3 中等违规

#### M1. kb_handler.go — Repository 类型泄漏

| 位置 | 问题 |
|------|------|
| `server/internal/handler/kb_handler.go:11` | 导入 `repository` 包 |
| `kb_handler.go:509` | 在 handler 中构造 `repository.KBQuery{}` |

**修复建议**：将 `KBQuery` 定义为 `model` 包中的 DTO，或通过 Service 层参数传递。

#### M2. admin_handler.go — Repository 类型泄漏

| 位置 | 问题 |
|------|------|
| `server/internal/handler/admin_handler.go:13` | 导入 `repository` 包 |
| `admin_handler.go:216` | 在 handler 中构造 `repository.UserQuery{}` |

**修复建议**：同 M1，将 `UserQuery` 迁移到 `model` 包。

### 1.4 轻微违规

| 文件 | 问题 | 行号 |
|------|------|------|
| `upload_handler.go` | 导入 `auth` 包，内联 RBAC 检查与路由中间件冗余 | 10, 34-35 |
| `auth_handler.go` | 导入 `auth` 包，直接调用 `CapabilitiesOf()` | 7, 336 |
| `student_handler.go` | 所有错误响应使用 `gin.H{}`，无 TraceID | 全部错误路径 |
| `counselor_handler.go` | 错误响应使用 `gin.H{}` | 382 |

---

## 2. 安全审计

### 2.1 阻断级 — 密钥泄露

#### BLOCK-SEC1: `.env` 文件中包含活 API 密钥

| 密钥 | 值（已部分脱敏） |
|------|------------------|
| `ZHIPU_API_KEY` | `b7944aeac28343b6b693bb67da2f84ec.tjp...` |
| `ZHIPU_4V_API_KEY` | `20322a4a95bf4bd68161b1f705aa6603.yHE...` |
| `DEEPSEEK_API_KEY` | `sk-87cc2d9f33fc4393b821a28b9b8703a5` |
| `XFYUN_APP_ID` | `ae4a0e4a` |
| `XFYUN_API_KEY` | `41a81c1e2935994e41cf261748d3a935` |
| `XFYUN_API_SECRET` | `NTI2NzVlOWQ0ZTM5YTgzNGYzZDI5NjQx` |

**风险**：`.env` 虽在 `.gitignore` 中，但文件存在于工作目录。如果曾被提交过，密钥已泄露。  
**修复**：  
1. 立即在对应平台（智谱/DeepSeek/讯飞）轮换所有密钥  
2. 检查 `git log --all -- .env` 确认是否曾被提交  
3. 使用 `git filter-repo` 或 BFG 从历史中清除

#### BLOCK-SEC2: JWT_SECRET 使用占位符

| 位置 | 值 |
|------|----|
| `.env:9` | `JWT_SECRET=change-me-to-a-random-string` |

**风险**：如果此值在生产环境使用，JWT 令牌可被伪造。  
**修复**：生成强随机密钥（`openssl rand -hex 32`）并设置到 Vercel 环境变量。

### 2.2 警告级

#### WARN-SEC1: agent_repo.go 动态列名无白名单

| 位置 | 问题 |
|------|------|
| `server/internal/repository/agent_repo.go:46` | `Update()` 接受 `map[string]interface{}`，所有 key 直接用于 SQL 列名，无白名单验证 |

**修复**：添加列名白名单（参考 `graduation_repo.go:189-203` 的模式）。

#### WARN-SEC2: PII 在日志中以明文记录

| 位置 | 内容 |
|------|------|
| `middleware/jwt.go:115` | `log.Printf("...user=%s role=%s path=%s", claims.Username, ...)` — 学号明文 |
| `middleware/user_upsert.go:27` | `log.Printf("...用户 upsert 失败 user=%s err=%v", userCtx.Username, ...)` — 学号明文 |

**修复**：对 `claims.Username` / `userCtx.Username` 应用 `MaskStudentID()` 后再记录。

#### WARN-SEC3: PII 中间件未检查 GET 请求

| 位置 | 问题 |
|------|------|
| `middleware/pii.go:16` | 仅拦截 `POST/PUT/PATCH`，`GET` 请求参数中的 PII 不检测 |

**修复**：扩展中间件扫描 `c.Request.URL.RawQuery`。

#### WARN-SEC4: 用户模型 API 密钥明文存储

| 位置 | 问题 |
|------|------|
| `repository/model_config_repo.go:44-75` | 用户自定义的 DeepSeek/Zhipu/讯飞密钥以明文存入 SQLite |

**修复**：使用 AES-256-GCM 加密后存储，密钥从环境变量读取。

#### WARN-SEC5: RBAC 范围过滤不级联

| 位置 | 问题 |
|------|------|
| `repository/kb_repo.go:124` | `WHERE (owner_scope = 'school' OR (owner_scope = ? AND owner_id = ?))` 只支持 school 和精确范围，缺 college → class 级联 |

**修复**：改为 `(owner_scope='school' OR (owner_scope='college' AND ...) OR (owner_scope='class' AND ...))`。

#### WARN-SEC6: LIKE 匹配角色名过于宽泛

| 位置 | 问题 |
|------|------|
| `repository/kb_repo.go:128,387,730` | `LIKE '%" + role + "%'` — `"student"` 会匹配 `"student_union"` |

**修复**：使用 `json_array_length` + `json_each` 精确匹配，或 `LIKE '%"student"%'` 加引号避免子串匹配。

#### WARN-SEC7: JWT nbf/iss 未验证

| 位置 | 问题 |
|------|------|
| `middleware/jwt.go:86-92` | `ParseWithClaims` 未验证 `nbf` 和 `iss` 声明 |

**修复**：生成 Token 时设置 `NotBefore: jwt.NewNumericDate(now)`，解析时添加 `jwt.WithIssuer("wxx")`。

### 2.3 备注级

| 位置 | 问题 |
|------|------|
| `middleware/audit.go` | 审计日志全局注册（含 `/health`、`/auth/login`），建议移到 secured 组 |
| `middleware/audit.go:38` | 审计插入在 goroutine 中执行，无优雅关闭处理 |
| `capabilities.go` | `student_union` 继承 `CounselorImportStudent`，需确认是否为预期设计 |
| `rbac-matrix.md` | teacher/assistant 角色在代码中已完整实现，文档仍标注"占位/P1" |
| `auth/*` | 无速率限制，存在暴力破解风险 |

---

## 3. 知识管道（Context Engine）审计

### 3.1 总体评估

| 检查项 | 状态 |
|--------|------|
| 来源附加 `sources[]` | **通过** — 所有 agent 正确附加 |
| 禁止编造（"仅基于上下文回答"） | **通过** — 系统提示词合规 |
| 范围过滤（scope/role/status） | **部分通过** — 详见下方问题 |
| 兜底处理（无信息时） | **通过** |
| 管道顺序（结构化→FTS→拼装→LLM） | **部分通过** — 缺结构化优先步骤 |
| LLM 调用前检查上下文是否充分 | **警告** — 总是调用 LLM，空上下文仍浪费请求 |

### 3.2 高严重性

#### HIGH-KB1: RBAC 范围过滤不级联

| 位置 | 问题 |
|------|------|
| `server/internal/repository/kb_repo.go:124` | scope 过滤仅支持 `school` 和精确范围，`class` 学生看不到 `college` 级资源 |

**影响**：一个 `class=cs2101` 的学生会错过 `college=cs` 的通用资源。  
**修复**：三层级联 `(owner_scope='school' OR (owner_scope='college' AND owner_id=?) OR (owner_scope='class' AND owner_id=?))`

#### HIGH-KB2: SearchFAQ 缺少 OwnerScope 过滤

| 位置 | 问题 |
|------|------|
| `server/internal/repository/kb_repo.go:370-391` | `searchFAQWithQuery()` 不按 `owner_scope`/`owner_id` 过滤 |

**影响**：FAQ 缓存查询可能返回不属当前用户范围的资源。  
**修复**：添加与普通搜索相同的 scope 过滤条件。

### 3.3 中等严重性

| ID | 位置 | 问题 | 修复 |
|----|------|------|------|
| MED-KB1 | `docs/context-engine.md:7` | 文档要求"结构化优先"步骤，代码未实现 | 更新文档或实现结构化查询路径 |
| MED-KB2 | `chat_service.go:183-205` | 搜索+Agent 都返回空时仍调用 LLM，浪费请求 | 添加预检查，空上下文直接返回兜底 |
| MED-KB3 | `kb_repo.go:128` | `LIKE '%role%'` 匹配超集角色 | 改用 JSON 精确匹配 |

### 3.4 低严重性

| ID | 位置 | 问题 | 修复 |
|----|------|------|------|
| LOW-KB1 | `process_agent.go:49` | 无类型匹配结果时返回未过滤来源 | 返回 `Sources: nil` |
| LOW-KB2 | `policy_agent.go:50` | 同上 | 同上 |
| LOW-KB3 | `process_agent.go:78` | 置信度硬编码 0.8 | 从 BM25 分推导 |
| LOW-KB4 | `policy_agent.go:79` | 置信度硬编码 0.85 | 同上 |
| LOW-KB5 | `merger.go:77-81` | 空 agent 的 0 置信度拉低均值 | 仅包含有结果的 agent |
| LOW-KB6 | `chat_service.go:165-168` | 安全过滤在 Agent 执行后运行 | 移到 Agent 执行前 |
| LOW-KB7 | `chat_service.go:108-118` | FAQ 缓存缺少 role 维度 | 缓存 key 包含 role |

---

## 4. 代码质量审计

### 4.1 高严重性

#### HIGH-Q1: KBService 所有方法缺少 context.Context

| 位置 | 问题 |
|------|------|
| `server/internal/service/kb_service.go` | **全部 20 个方法**不接受 `context.Context` 作为首参数 |

**影响**：无请求取消传播、无超时传递、无追踪集成。  
**修复**：为每个方法添加 `ctx context.Context` 首参数。

#### HIGH-Q2: TraceID 未传播到 Service/Repository 层

| 位置 | 问题 |
|------|------|
| `middleware/trace.go` | TraceID 存储在 Gin Context 中，未写入 `context.Context` |

**影响**：Service 层和 Repository 层各自生成独立 trace ID，端到端追踪断裂。  
**修复**：`trace.go` 中同时将 traceID 写入 `c.Request.Context()`。

#### HIGH-Q3: student_handler.go 静默吞错误

| 位置 | 问题 |
|------|------|
| `server/internal/handler/student_handler.go` | 多处 `Generate*()` 调用失败后使用 mock 数据，不记录错误 |

**影响**：生产环境无法追踪为何返回异常数据。  
**修复**：在 `if err != nil` 分支中添加 `log.Printf` 或结构化日志。

### 4.2 中等严重性

| ID | 位置 | 问题 | 修复 |
|----|------|------|------|
| MED-Q1 | `kb_handler.go` 全部错误路径 | 错误响应缺 TraceID | 添加 `TraceID: middleware.GetTraceID(c)` |
| MED-Q2 | `chat_handler.go:61`, `kb_handler.go:111` 等 | 将 `err.Error()` 直接暴露给用户 | 使用通用消息，日志记录完整错误 |
| MED-Q3 | `api_service.dart:39` | 401 回调后错误继续传播给 Provider | 添加 debounce 或标志位防止双重重置 |
| MED-Q4 | `knowledge_provider.dart` | 无 `dispose()` 方法，`listPendingReviews` 静默失败 | 添加 dispose，catch 中存错误状态 |

### 4.3 低严重性

| ID | 位置 | 问题 |
|----|------|------|
| LOW-Q1 | `student_handler.go` | `len(rows) == 0` 时不同路径使用 `nil` 和 `[]` 不一致 |
| LOW-Q2 | `chat_service.go` | `cacheGet`/`cacheSet`/`faqLookup` 不接受 `context.Context` |
| LOW-Q3 | 多处 handler | `Set*Service()` 模式在 init 后不应被调用，建议删除或标记 deprecated |

---

## 5. 文档完整性审计

### 5.1 API 契约

| 检查项 | 状态 |
|--------|------|
| `specs/api-contracts-index.md` 存在 | **通过** |
| 新增 handler 的接口是否在契约中登记 | **抽样发现遗漏**：`/api/v1/forecast/*`、`/api/v1/competition/*`、`/api/v1/plan/*`、`/api/v1/party/*`、`/api/v1/club/*`、`/api/v1/graduation/*` 等 P2 模块端点未在契约中登记 |

**修复**：补充所有 P2 模块端点到 `api-contracts-index.md`。

### 5.2 RBAC 矩阵

| 检查项 | 状态 | 详情 |
|--------|------|------|
| `specs/rbac-matrix.md` 存在 | **通过** |  |
| 代码能力 vs 文档匹配 | **部分不匹配** | `counselor.token.subordinates`、`college.forecast`、`school.agent.write` 在代码中已定义但文档缺失 |
| teacher/assistant 角色状态 | **文档滞后** | 代码已完整实现 teacher(9 caps) + assistant(3 caps)，文档标注为"占位/P1" |

**修复**：同步 RBAC 矩阵，更新 teacher/assistant 状态。

### 5.3 迁移脚本

| 检查项 | 状态 |
|--------|------|
| `server/migrations/` 目录 | **通过** — 33 个迁移文件 |
| 编号连续 | **通过** — 从 001 到 036（含跳过的 002/030/033，有说明） |
| 迁移文件可追溯 | **需补充**：部分迁移未记录对应的 schema 变更文档 |

### 5.4 设计文档对照

| 文档 | 与实际实现对照 |
|------|---------------|
| `docs/蔚小芯智能体.md` | — 架构描述与实际基本一致 |
| `docs/context-engine.md` | **结构化优先步骤未实现**（参见 HIGH-KB1） |
| `docs/ui-answer-card.md` | — 与实际 AnswerCard 结构一致 |
| `docs/deployment.md` | **域名信息过时**：仍提到 `api.pydaydayup.xyz`（已过期） |

---

## 6. 综合评分与修复路线图

### 6.1 评分总览

| 维度 | 阻断 | 警告 | 备注 | 评分 |
|------|------|------|------|------|
| 架构合规 | 3 | 2 | 3 | ⚠️ 4/10 |
| 安全审计 | 2 | 7 | 6 | ⚠️ 5/10 |
| 知识管道 | 2 | 3 | 7 | ⚠️ 6/10 |
| 代码质量 | 3 | 4 | 3 | ⚠️ 5/10 |
| 文档完整性 | 0 | 3 | 1 | ✅ 7/10 |
| **总分** | **10** | **19** | **20** | **⚠️ 5.4/10** |

### 6.2 紧急修复（P0 — 24 小时内）

| 优先级 | ID | 描述 | 预计工时 |
|--------|-----|------|---------|
| P0 | BLOCK-SEC1 | 轮换所有泄露的 API 密钥 | 1h |
| P0 | BLOCK-SEC2 | 设置强 JWT_SECRET | 0.5h |
| P0 | HIGH-KB1 | RBAC scope 过滤级联 | 2h |
| P0 | B1+B2+B3 | 修复 student/feedback/voice handler 架构违规 | 4h |
| P0 | HIGH-Q1 | KBService 注入 context.Context | 3h |

### 6.3 短期修复（P1 — 本周内）

| 优先级 | ID | 描述 | 预计工时 |
|--------|-----|------|---------|
| P1 | WARN-SEC1~7 | 安全警告修复 | 6h |
| P1 | HIGH-Q2 | TraceID 传播到 context.Context | 2h |
| P1 | HIGH-Q3 | student_handler 错误日志 | 1h |
| P1 | MED-KB1~3 | 知识管道中等问题 | 4h |
| P1 | M1+M2 | Repository 类型泄漏 | 2h |

### 6.4 中期优化（P2 — 本月内）

| 优先级 | ID | 描述 | 预计工时 |
|--------|-----|------|---------|
| P2 | MED-Q1~4 | 代码质量中等问题 | 4h |
| P2 | LOW-* | 所有备注级问题 | 8h |
| P2 | 文档更新 | API 契约 + RBAC 矩阵 + teacher/assistant 状态 | 3h |

### 6.5 结论

> **整体评级：⚠️ 需改善（5.4/10）**  
> **建议**：优先修复 5 个 P0 阻断级问题后再进行下一轮功能开发。核心架构和安全存在同比 10 个阻断级问题，知识管道基本正确但有 2 个高严重性漏洞。代码质量和文档相对健康，但 TraceID 断裂和 `context.Context` 缺失影响生产可观测性。

---

*报告生成：2026-07-25 | 审核工具：静态代码审计 + 架构扫描 + 安全模式匹配*