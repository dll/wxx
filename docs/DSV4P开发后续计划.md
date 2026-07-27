# 蔚小芯 DSV4P 开发后续计划

> 基于全面审计（v1+v2）的结果，本计划列出按优先级排列的后续工作。  
> 涉及领域：功能完整度补全、架构整改、安全加固、测试覆盖。  
> 评审编号：B=阻断级, W=警告级, N=备注级

---

## P0 — 紧急修复（本轮完成）

| 编号 | 组件 | 问题描述 | 状态 |
|------|------|----------|------|
| B-01 | `kb_handler.go` | 导入 `repository` 包，违反分层 | ✅ 已修复 (v1 B1-B3, v2 残余) |
| B-02 | `admin_handler.go` | 导入 `repository` 包，违反分层 | ✅ 已修复 |
| B-03 | `study_plan_handler.go` | 导入 `llm` 包，直接持有 `llm.ChatClient` | ✅ 已修复 |
| B-04 | `GetPublishedCards` | 缺失 `json_valid` 守卫（第 732 行） | ✅ 已修复 |
| B-05 | `user_upsert.go` | PII 明文记录到日志 | ✅ 已修复 |

## P1 — 功能完整度补全

### 1.1 架构合规（Handler → Service 分层）

| 编号 | 文件 | 问题 | 修复方案 | 估时 |
|------|------|------|----------|------|
| B-02 | `admin_handler.go:13` | 导入 `repository`，构造 `repository.UserQuery{}` | 将 `UserQuery` 迁移到 `model` 包，或通过 Service 层包装 | 2h |
| B-03 | `study_plan_handler.go:14` | 导入 `llm`，持有 `llm.ChatClient` | 创建 `service.StudyPlanService` 封装 LLM 调用 | 2h |
| W-01 | `kb_handler.go:509` | `repository.KBQuery{}` 在 handler 中构造 | 同 B-02，KBQuery → model 包 | 1h |
| W-02 | `feedback_handler.go` / `voice_handler.go` | 导入 `uuid` 工具包 | 允许（非业务包），关闭 | — |
| W-03 | `counselor_handler.go` / `notification_handler.go` / `upload_handler.go` | 使用 `gin.H{}` 而非 `model.ErrorResponse` | 全体统一为标准响应格式 | 3h |

### 1.2 安全加固

| 编号 | 文件 | 问题 | 修复方案 | 估时 |
|------|------|------|----------|------|
| B-06 | `.env` 文件 | 密钥明文存在于工作目录 | 轮换所有 API 密钥；git 历史清理 | 1h |
| B-07 | `JWT_SECRET` | 默认值 `dev-secret-not-for-production...` | 生产部署时通过环境变量设置强密钥 | 即时 |
| W-04 | `notification_handler.go:41,57,77,96` | `err.Error()` 暴露给客户端 | 使用通用错误消息，完整错误记录到日志 | 1h |
| W-05 | `upload_handler.go:64` | `err.Error()` 暴露给客户端 | 同上 | 0.5h |
| W-06 | `kb_handler.go` 多处 | `err.Error()` 通过 `model.ErrorResponse` 泄露 | 同上 | 2h |
| W-07 | `admin_handler.go` 多处 | `err.Error()` 泄露 | 同上 | 2h |
| W-08 | `auth_handler.go` 多处 | `err.Error()` 泄露 | 同上 | 1h |
| W-09 | `model_config_repo.go` | 用户 API 密钥明文存储 | AES-256-GCM 加密后存储 | 4h |
| W-10 | `middleware/pii.go:16` | 仅拦截 POST/PUT/PATCH，GET 请求的 PII 不检测 | 扩展扫描 `RawQuery` | 1h |
| W-11 | `middleware/rate_limit.go` | 观察新文件是否存在 | 添加速率限制配置 | 2h |

### 1.3 知识管道（Context Engine）

| 编号 | 组件 | 问题 | 修复方案 | 估时 |
|------|------|------|----------|------|
| MED-01 | `context-engine.md:7` | 文档要求"结构化优先"，代码未实现 | 实现结构化查询优先路径，或更新文档 | 4h |
| MED-02 | `chat_service.go:183-205` | 搜索+Agent 都返回空时仍调用 LLM | 添加预检查，空上下文直接返回兜底 | 2h |
| LOW-01 | `process_agent.go:49` | 无类型匹配时返回未过滤来源 | 返回 `Sources: nil` | 0.5h |
| LOW-02 | `policy_agent.go:50` | 同上 | 同上 | 0.5h |
| LOW-03 | `process_agent.go:78` / `policy_agent.go:79` | 置信度硬编码 0.8/0.85 | 从 BM25 分推导 | 1h |
| LOW-04 | `merger.go:77-81` | 空 agent 的 0 置信度拉低均值 | 仅包含有结果的 agent | 0.5h |
| LOW-05 | `chat_service.go:165-168` | 安全过滤在 Agent 执行后 | 移到 Agent 执行前 | 1h |
| LOW-06 | `chat_service.go:108-118` | FAQ 缓存缺少 role 维度 | 缓存 key 包含 role | 0.5h |
| N-01 | `docs/context-engine.md` | 结构化优先步骤未实现 | 更新文档描述与实际一致 | 1h |

### 1.4 代码质量

| 编号 | 组件 | 问题 | 修复方案 | 估时 |
|------|------|------|----------|------|
| MED-03 | `student_handler.go` | 多处静默吞错误（`err != nil` 后使用 mock） | 添加 `log.Printf` 记录错误 | 2h |
| MED-04 | `kb_handler.go` 全部错误路径 | 错误响应缺 TraceID | 添加 `TraceID: middleware.GetTraceID(c)` | 1h |
| MED-05 | `chat_handler.go:61`, `kb_handler.go:111` | `err.Error()` 暴露 | 使用通用消息 + 日志 | 3h |
| MED-06 | `chat_service.go` | `cacheGet`/`cacheSet`/`faqLookup` 不接受 Context | 添加 `ctx context.Context` 参数 | 2h |
| LOW-07 | `student_handler.go` | `len(rows) == 0` 时 nil vs `[]` 不一致 | 统一为非 nil 空切片 | 1h |
| LOW-08 | 各处 `Set*Service()` 方法 | Init 后不应可调用 | 删除或标记 deprecated | 2h |

### 1.5 测试覆盖

| 编号 | 组件 | 当前覆盖 | 目标 |
|------|------|----------|------|
| T-01 | `service/chat_service.go` | 0% | 核心问答路径 + 知识检索 + 兜底 |
| T-02 | `handler/student_handler.go` | ~10% | 学生端 30+ 个端点的 200/400/500 响应 |
| T-03 | `handler/counselor_handler.go` | 0% | 辅导员 20+ 个端点 |
| T-04 | `handler/admin_handler.go` | ~30% | 用户管理 + 导入 + 设置 |
| T-05 | `llm/deepseek.go` | 0% | 超时、重试、错误响应 |
| T-06 | `llm/zhipu.go` | 0% | 同上 |
| T-07 | `middleware/rate_limit.go` | 0% | 限流基本测试 |
| T-08 | `repository/kb_repo.go` | ~70% | FTS、搜索边界条件 |

### 1.6 文档

| 编号 | 组件 | 问题 | 修复 |
|------|------|------|------|
| D-01 | `specs/api-contracts-index.md` | 遗漏 P2 模块端点 | 补充 `/forecast/*`、`/competition/*`、`/plan/*`、`/party/*`、`/club/*`、`/graduation/*` |
| D-02 | `specs/rbac-matrix.md` | teacher/assistant 标注占位/P1 但代码已完整 | 同步为已实现 |
| D-03 | `specs/rbac-matrix.md` | 缺失 `counselor.token.subordinates`、`college.forecast`、`school.agent.write` | 补充 |
| D-04 | `docs/deployment.md` | 仍引用 `api.pydaydayup.xyz`（已过期） | 更新为当前域名 |

---

## 内置测试用户账号

所有种子用户的初始密码统一为 **`wxx@2025`**（由 `029_student_user_import.sql` 中的 bcrypt hash 设定）。

| 用户名 | 角色 | 归属域 | 归属 ID | 说明 |
|--------|------|--------|---------|------|
| `sysadmin` | `sys_admin` | `school` | `all` | 系统管理员，最高权限 |
| `schooladmin` | `school_admin` | `school` | `all` | 学校管理员 |
| `collegeadmin` | `college_admin` | `college` | `cs` | 计算机学院管理员 |
| `counselor_cs` | `counselor` | `college` | `cs` | 计算机学院辅导员 |
| `counselor_math` | `counselor` | `college` | `math` | 数理学院辅导员 |
| `stunion` | `student_union` | `college` | `cs` | 学生会主席 |
| `student_cs` | `student` | `college` | `cs` | 计算机学院学生 |
| `student_math` | `student` | `college` | `math` | 数理学院学生 |
| `teacher1` | `teacher` | `college` | `cs` | 教师 |
| `assistant1` | `assistant` | `college` | `cs` | 教辅 |
| `admin` | `sys_admin` | `school` | `all` | 系统管理员（旧版） |
| `student1` | `student` | `college` | `cs` | 通用学生 |
| `counselor1` | `counselor` | `college` | `cs` | 通用辅导员 |
| `counselor2` | `counselor` | `college` | `math` | 通用辅导员 |

> ⚠️ 若 `wxx@2025` 不生效，说明数据库未执行 `029_student_user_import.sql` 迁移。  
> 此时可尝试密码 = 用户名（如 `sysadmin` / `sysadmin`），或由系统管理员在后台重置：  
> `POST /api/v1/admin/users/{username}/password` 设置新密码。

---

## 执行路线图

```
P0（本轮） → 全部 5 个阻断级已修复 ✅
   │
   ▼
P1（本周） → 架构合规 + 安全加固 + 知识管道中高问题
   │
   ▼
P2（本月） → 代码质量 + 测试覆盖 + 文档补全
```

## 预计总工时

| 阶段 | 工时 | 交付物 |
|------|------|--------|
| P0 架构修复 | 4h | 3 个 handler 重构完成 |
| P1 安全加固 | 11.5h | 所有 err.Error() 消除 + API 密钥管理 + PII 脱敏 |
| P1 知识管道 | 9h | Context Engine 对齐文档 |
| P2 代码质量 | 11h | TraceID + Context + 错误统一 |
| P2 测试覆盖 | 2-3 天 | 核心模块测试用例 |
| P2 文档 | 3h | 契约 + RBAC 矩阵 + 部署文档 |
