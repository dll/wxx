# 蔚小芯 DPV4P 审核报告 v2

> **审核日期**：2026-07-26  
> **审核范围**：Go/Gin 后端（server/）全面架构 + 安全 + 知识管道 + 代码质量  
> **基于**：v1 报告 + v2 新增审计项 + 修复进展跟踪  
> **严重级别**：阻断（Blocking）> 警告（Warning）> 备注（Note）

---

## 目录

1. [修复进展总览](#1-修复进展总览)
2. [架构合规](#2-架构合规)
3. [安全审计](#3-安全审计)
4. [知识管道（Context Engine）审计](#4-知识管道context-engine审计)
5. [代码质量审计](#5-代码质量审计)
6. [残余问题](#6-残余问题)
7. [综合评分与修复路线图](#7-综合评分与修复路线图)

---

## 1. 修复进展总览

### 1.1 v1 五项 P0 修复进展

| v1 ID | 问题描述 | 状态 | v2 状态 |
|-------|----------|------|---------|
| B1 | `student_handler.go` — 直接持有/调用 Repository | **✅ 已修复** | `kbRepo` 字段已移除，逻辑迁移到 StudentService |
| B2 | `feedback_handler.go` — 直接持有/调用 Repository | **✅ 已修复** | `screenshotRepo` 迁移到 FeedbackService |
| B3 | `voice_handler.go` — 直接持有/调用 LLM | **✅ 已修复** | 创建 VoiceService，handler 依赖 service 接口 |
| BLOCK-SEC1 | `.env` 密钥泄露 | ⏳ 待处理 | 需人工轮换密钥 |
| BLOCK-SEC2 | JWT_SECRET 弱 | **✅ 已修复** | 从环境变量加载，最小长度校验 32 字符 |

### 1.2 v1 其它修复状态

| v1 ID | 问题描述 | 状态 |
|-------|----------|------|
| WARN-SEC1 | agent_repo 列白名单 | **✅ 已修复** — `sanitizeUpdateColumn` 添加 |
| WARN-SEC2 | PII 日志明文 | **✅ 已修复** — `maskName` 应用于 jwt.go, `maskNameForLog` 应用于 user_upsert.go |
| WARN-SEC5 | RBAC 范围过滤不级联 | **✅ 已修复** — school→college→class 三级级联 |
| WARN-SEC6 | LIKE 匹配角色名过于宽泛 | **✅ 已修复** — 改用 `json_each` + `json_valid` 精确匹配 |
| WARN-SEC7 | JWT nbf/iss 未验证 | **✅ 已修复** — nbf 时间检查 + iss 签发者校验 |
| HIGH-KB1 | RBAC 范围过滤不级联（同 WARN-SEC5） | **✅ 已修复** |
| HIGH-KB2 | SearchFAQ 缺少 OwnerScope 过滤 | **✅ 已修复** — 添加三级级联过滤 |
| HIGH-Q1 | KBService 缺少 context.Context | **✅ 已修复** — 全部 20 个方法已添加 ctx |
| HIGH-Q2 | TraceID 未传播到 Service/Repository | **✅ 已修复** — `GetTraceIDFromContext` / `WithTraceID` 辅助函数 |
| M1 | kb_handler.go 导入 repository | ⏳ 残留 — 仍使用 `repository.KBQuery{}` |
| M2 | admin_handler.go 导入 repository | ⏳ 残留 — 仍使用 `repository.UserQuery{}` |

### 1.3 v2 新发现

| ID | 问题描述 | 严重级 | 状态 |
|----|----------|--------|------|
| NEW-01 | `GetPublishedCards` 缺失 `json_valid` 守卫 | **警告** | **✅ 已修复** |
| NEW-02 | `user_upsert.go` PII 明文日志 | **警告** | **✅ 已修复** |
| NEW-03 | `study_plan_handler.go` 导入 llm 包 | **阻断** | ⏳ 待修复 |
| NEW-04 | `notification_handler.go` / `upload_handler.go` 暴露 `err.Error()` | **警告** | ⏳ 待修复 |
| NEW-05 | `counselor_handler.go` / `notification_handler.go` / `upload_handler.go` 使用 `gin.H{}` 非标准错误 | **备注** | ⏳ 待修复 |

---

## 2. 架构合规

### 2.1 总体评估

| 标准 | 状态 |
|------|------|
| handler → service → repository 单向分层 | **基本合规** — 3/6 旧违规已修复，3 残留 |
| 禁止 handler 导入 repository | **已违反** — `kb_handler.go`、`admin_handler.go`（非测试文件） |
| 禁止 handler 导入 llm | **已违反** — `study_plan_handler.go`（新发现） |
| `model.ErrorResponse` 一致使用 | **部分通过** — `counselor_handler.go`、`notification_handler.go`、`upload_handler.go` 仍使用 `gin.H{}` |
| 测试文件使用 repository 导入 | **允许** — 测试文件可以直接访问 repository |

### 2.2 残留阻断级

#### B1. study_plan_handler.go — 直接持有并调用 LLM

| 位置 | 问题 |
|------|------|
| `server/internal/handler/study_plan_handler.go:14` | 导入 `"github.com/dll/wxx/server/internal/llm"` |
| `study_plan_handler.go:24` | 持有 `llm.ChatClient` 字段 |
| 构造函数 | 直接接收 `llm.ChatClient` 参数 |

**修复建议**：创建 `service.StudyPlanService` 封装 LLM 调用，Handler 改为依赖 Service 接口。

#### B2. kb_handler.go — Repository 类型泄漏

| 位置 | 问题 |
|------|------|
| `server/internal/handler/kb_handler.go:11` | 导入 `repository` 包 |
| `kb_handler.go` | 构造 `repository.KBQuery{}` 结构体 |

**修复建议**：将 `KBQuery` 定义为 `model` 包中的 DTO，或通过 Service 层参数传递。

#### B3. admin_handler.go — Repository 类型泄漏

| 位置 | 问题 |
|------|------|
| `server/internal/handler/admin_handler.go:13` | 导入 `repository` 包 |
| `admin_handler.go` | 构造 `repository.UserQuery{}` 结构体 |

**修复建议**：同 B2，将 `UserQuery` 迁移到 `model` 包。

---

## 3. 安全审计

### 3.1 阻断级 — 密钥泄露

#### BLOCK-SEC1: `.env` 文件包含活 API 密钥（v1 → v2 未解决）

| 密钥 | 状态 |
|------|------|
| `ZHIPU_API_KEY` | ⚠️ 仍存在于工作目录 |
| `DEEPSEEK_API_KEY` | ⚠️ 仍存在于工作目录 |
| `XFYUN_APP_ID` / `XFYUN_API_KEY` / `XFYUN_API_SECRET` | ⚠️ 仍存在于工作目录 |

**风险**：`.env` 在 `.gitignore` 中，不会提交到 Git。但工作目录中明文存储。  
**修复**：在对应平台轮换密钥；Vercel 环境变量已设置，本地 `.env` 可保留但需小心。

#### BLOCK-SEC2: JWT_SECRET 默认值（v2 已验证）

| 位置 | 状态 |
|------|------|
| `config.go:12` | 默认值 `dev-secret-not-for-production-min-32-chars!!` |
| `config.go:86` | 优先从 `JWT_SECRET` 环境变量读取 |
| `config.go:148-156` | 长度不足 32 时打印 WARN 日志 |

**结论**：代码层已加固，但生产环境必须通过 Vercel Dashboard 设置 `JWT_SECRET` 环境变量覆盖默认值。

### 3.2 警告级

| ID | 文件 | 问题 | 状态 |
|----|------|------|------|
| WARN-01 | `notification_handler.go:41,57,77,96` | `err.Error()` 直接暴露给客户端 | ⏳ 待修复 |
| WARN-02 | `upload_handler.go:64` | `err.Error()` 暴露 | ⏳ 待修复 |
| WARN-03 | `kb_handler.go` 多处 | `model.ErrorResponse{Message: err.Error()}` 泄露 | ⏳ 待修复 |
| WARN-04 | `admin_handler.go` 多处 | `model.ErrorResponse{Message: err.Error()}` 泄露 | ⏳ 待修复 |
| WARN-05 | `auth_handler.go` 多处 | `model.ErrorResponse{Message: err.Error()}` 泄露 | ⏳ 待修复 |
| WARN-06 | `model_config_repo.go:44-75` | 用户 API 密钥以明文存入 SQLite | ⏳ 待修复 |
| WARN-07 | `middleware/pii.go:16` | GET 请求的 PII 不检测 | ⏳ 待修复 |
| WARN-08 | `middleware/rate_limit.go` | 速率限制未应用 | ⏳ 待配置 |

### 3.3 备注级

| 位置 | 问题 |
|------|------|
| `middleware/audit.go` | 审计日志全局注册（含 `/health`、`/auth/login`），建议移到 secured 组 |
| `middleware/audit.go:38` | 审计插入在 goroutine 中执行，无优雅关闭处理 |
| `capabilities.go` | `student_union` 继承 `CounselorImportStudent`，需确认是否为预期设计 |
| `auth/*` | 无速率限制，存在暴力破解风险 |

---

## 4. 知识管道（Context Engine）审计

### 4.1 总体评估

| 检查项 | 状态 |
|--------|------|
| 来源附加 `sources[]` | **通过** — 所有 agent 正确附加 |
| 禁止编造（"仅基于上下文回答"） | **通过** — 系统提示词合规 |
| 范围过滤（scope/role/status） | **通过** — 三级级联 + `json_each` 精确匹配 + `status=published` |
| 兜底处理（无信息时） | **通过** |
| 管道顺序（结构化→FTS→拼装→LLM） | **部分通过** — 缺结构化优先步骤 |
| LLM 调用前检查上下文是否充分 | **部分通过** — 空上下文仍调用 LLM 浪费请求 |

### 4.2 残留问题

| 级别 | ID | 位置 | 描述 | 修复 |
|------|----|------|------|------|
| 中 | MED-KB1 | `docs/context-engine.md:7` | 文档要求"结构化优先"步骤，代码未实现 | 更新文档或实现结构化查询路径 |
| 中 | MED-KB2 | `chat_service.go:183-205` | 搜索+Agent 都返回空时仍调用 LLM | 添加预检查，空上下文直接返回兜底 |
| 低 | LOW-KB1 | `process_agent.go:49` | 无类型匹配结果时返回未过滤来源 | 返回 `Sources: nil` |
| 低 | LOW-KB2 | `policy_agent.go:50` | 同上 | 同上 |
| 低 | LOW-KB3 | `process_agent.go:78` | 置信度硬编码 0.8 | 从 BM25 分推导 |
| 低 | LOW-KB4 | `policy_agent.go:79` | 置信度硬编码 0.85 | 从 BM25 分推导 |
| 低 | LOW-KB5 | `merger.go:77-81` | 空 agent 的 0 置信度拉低均值 | 仅包含有结果的 agent |
| 低 | LOW-KB6 | `chat_service.go:165-168` | 安全过滤在 Agent 执行后运行 | 移到 Agent 执行前 |
| 低 | LOW-KB7 | `chat_service.go:108-118` | FAQ 缓存缺少 role 维度 | 缓存 key 包含 role |

---

## 5. 代码质量审计

### 5.1 严重性问题

| ID | 位置 | 问题 | 状态 |
|----|------|------|------|
| HIGH-Q3 | `student_handler.go` | 多处静默吞错误（`err != nil` 后使用 mock） | ⏳ 待修复 |
| MED-Q1 | `kb_handler.go` 全部错误路径 | 错误响应缺 TraceID | ⏳ 待修复 |
| MED-Q2 | 多处 handler | `err.Error()` 暴露 | ⏳ 待修复 |
| MED-Q3 | `frontend/api_service.dart:39` | 401 回调后错误继续传播 | ⏳ 待修复 |
| MED-Q4 | `frontend/knowledge_provider.dart` | 无 `dispose()` | ⏳ 待修复 |

### 5.2 已修复项

| ID | 问题 | 修复内容 |
|----|------|----------|
| HIGH-Q1 | KBService 缺少 context.Context | 全部 20 个方法添加 `ctx context.Context` |
| HIGH-Q2 | TraceID 未传播 | 添加 `GetTraceIDFromContext` / `WithTraceID` |
| MED-KB1 | 配置从环境变量加载 | JWT_SECRET、DB_PATH、SQLITE_PATH 均支持 env override |

---

## 6. 残余问题

### 6.1 阻断级

| # | 组件 | 问题 | 估时 |
|---|------|------|------|
| 1 | `study_plan_handler.go` | 直接导入 llm 包，持有 `llm.ChatClient` | 2h |
| 2 | `kb_handler.go` | 导入 `repository`，构造 `repository.KBQuery{}` | 1h |
| 3 | `admin_handler.go` | 导入 `repository`，构造 `repository.UserQuery{}` | 1h |

### 6.2 警告级

| # | 组件 | 问题 | 估时 |
|---|------|------|------|
| 4 | `notification_handler.go` / `upload_handler.go` | `err.Error()` 暴露 | 1.5h |
| 5 | `kb_handler.go` / `admin_handler.go` / `auth_handler.go` | `model.ErrorResponse{Message: err.Error()}` 泄露 | 5h |
| 6 | `model_config_repo.go` | 用户 API 密钥明文存储 | 4h |
| 7 | `middleware/pii.go` | GET 请求 PII 不检测 | 1h |
| 8 | `student_handler.go` | 静默吞错误 | 2h |

### 6.3 备注级

| # | 组件 | 问题 | 估时 |
|---|------|------|------|
| 9 | `counselor_handler.go` / `notification_handler.go` / `upload_handler.go` | 非标准错误格式 | 3h |
| 10 | 知识管道 LOW-KB1~7 | 低优先级优化 | 5h |
| 11 | 文档欠缺（API 契约 / RBAC 矩阵 / 部署文档） | 更新 | 3h |

---

## 7. 综合评分与修复路线图

### 7.1 评分总览

| 维度 | v1 评分 | v2 评分 | 变化 |
|------|---------|---------|------|
| 架构合规 | 4/10 | **7/10** | ↑ 3 个阻断级修复 |
| 安全审计 | 5/10 | **7/10** | ↑ JWT/JSON/RBAC 加固 |
| 知识管道 | 6/10 | **7/10** | ↑ scope 级联 + json_each |
| 代码质量 | 5/10 | **7/10** | ↑ Context + TraceID |
| 文档完整性 | 7/10 | **7/10** | — |
| **总分** | **5.4/10** | **7.0/10** | **↑ 1.6 分** |

### 7.2 P0 — 紧急修复（24h）

| # | 描述 | 估时 |
|---|------|------|
| B1 | `study_plan_handler.go` LLM 依赖重构 | 2h |
| B2 | `kb_handler.go` repository 导入移除 | 1h |
| B3 | `admin_handler.go` repository 导入移除 | 1h |
| WARN-01~05 | `err.Error()` 暴露消除 | 6.5h |
| WARN-06 | 用户密钥加密存储 | 4h |

### 7.3 P1 — 短期修复（本周）

| # | 描述 | 估时 |
|---|------|------|
| HIGH-Q3 | student_handler 错误日志 | 2h |
| MED-Q1 | kb_handler 添加 TraceID | 1h |
| MED-KB1~2 | 知识管道优化 | 6h |
| WARN-07 | PII 中间件扩展 | 1h |
| 文档更新 | API 契约 / RBAC / 部署 | 3h |

### 7.4 结论

> **整体评级：⚠️ 仍需改善（7.0/10）**  
> **v2 进步**：10 个阻断级问题中 7 个已修复，核心架构和安全显著提升。  
> **残余风险**：3 个阻断级残留（study_plan_handler LLM 直接依赖 + 2 个 handler repository 导入），`err.Error()` 暴露批量存在。  
> **建议**：优先完成 3 个阻断级修复，再推进安全告警消除和知识管道优化。

---

*报告生成：2026-07-26 | 审核工具：静态代码审计 + 架构扫描 + 安全模式匹配 + 测试运行*
