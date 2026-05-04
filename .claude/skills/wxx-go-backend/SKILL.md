---
name: wxx-go-backend
description: 蔚小芯 Go/Gin 后端开发规范。当创建或修改 server/ 下的 Go 源文件、添加新 API 接口、实现 handler/service/repository、编写 SQLite 查询或组织 Gin 路由时触发。也在出现"后端"、"接口"、"API"、"handler"、"service"、"repository"、"Gin"、"路由"等短语时触发，或在为项目生成 Go 代码时触发。确保所有 Go 代码遵循项目严格的分层规则。
---

# 蔚小芯 Go 后端开发

本技能执行蔚小芯 Go/Gin 后端的架构规范。项目使用严格分层来保持业务逻辑可测试，防止 handler 直接操作数据库或调用大模型 API 导致代码混乱。

## 架构分层

```
HTTP 请求
    |
    v
[middleware/]      --> JWT 鉴权、RBAC 检查、限流、审计日志、CORS
    |
    v
[handler/]         --> 解析请求、校验参数、调用 service、组装响应
    |
    v
[service/]         --> 业务逻辑：编排 repository + context_engine + llm + agent
    |
    v
[repository/]      --> 对 SQLite 执行 SQL 查询（仅参数化语句）
[context_engine/]  --> 知识检索管道（FTS + 结构化）
[llm/]             --> 智谱/DeepSeek/讯飞 API 客户端
[agent/]           --> 多智能体编排（通过 Eino）
```

### 分层规则（强制执行）

| 调用方 | 可以调用 | 不可调用 |
|--------|----------|----------|
| handler | service | repository、llm、agent、context_engine |
| service | repository、context_engine、llm、agent | handler |
| repository | （仅 SQLite） | service、handler、llm、context_engine |
| context_engine | repository（用于 FTS/结构化查询） | handler、llm |
| llm | （仅外部 HTTP） | repository、handler |

违反上述规则的代码必须在提交前被阻止。

## 新增接口流程

按以下清单执行：

### 1. 定义路由

在路由配置中（`cmd/server/main.go` 或独立的 `router.go`）：

```go
api := r.Group("/api/v1")
api.Use(middleware.Auth())  // JWT 验证

// 按业务域分组
chat := api.Group("/chat")
chat.Use(middleware.RateLimit(30))  // 每分钟限流
{
    chat.POST("/ask", chatHandler.Ask)
    chat.GET("/sessions", chatHandler.ListSessions)
}
```

### 2. 编写 Handler

Handler 只做三件事：解析、委派、响应。

```go
// internal/handler/chat_handler.go
func (h *ChatHandler) Ask(c *gin.Context) {
    // 1. 解析并校验参数
    var req AskRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse(err))
        return
    }

    // 2. 从中间件获取用户认证上下文
    userCtx := middleware.GetUserContext(c)

    // 3. 委派给 service 处理
    answer, err := h.chatService.Ask(c.Request.Context(), userCtx, req.Question)
    if err != nil {
        c.JSON(500, ErrorResponse(err))
        return
    }

    // 4. 返回响应
    c.JSON(200, answer)
}
```

### 3. 实现 Service

Service 包含业务逻辑，编排多个依赖：

```go
// internal/service/chat_service.go
func (s *ChatService) Ask(ctx context.Context, user UserContext, question string) (*AnswerCard, error) {
    // 1. 通过 Context Engine 查询相关知识
    ctxResult, err := s.contextEngine.Query(ctx, question, user.Role, user.Scope)
    if err != nil {
        return nil, fmt.Errorf("上下文查询失败: %w", err)
    }

    // 2. 用上下文拼装提示词
    prompt := s.assemblePrompt(question, ctxResult)

    // 3. 调用大模型
    rawAnswer, err := s.llmClient.Chat(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("模型调用失败: %w", err)
    }

    // 4. 构建带来源的 AnswerCard
    card := s.buildAnswerCard(rawAnswer, ctxResult.Sources)

    // 5. 保存对话记录
    _ = s.messageRepo.Save(ctx, user.SessionID, "user", question)
    _ = s.messageRepo.Save(ctx, user.SessionID, "assistant", card.Conclusion)

    return card, nil
}
```

### 4. 编写 Repository

Repository 是纯 SQL — 不含业务逻辑、不触及 HTTP、不调用外部 API：

```go
// internal/repository/message_repo.go

// Save 保存一条对话消息到数据库
func (r *MessageRepo) Save(ctx context.Context, sessionID, role, content string) error {
    _, err := r.db.ExecContext(ctx,
        "INSERT INTO messages (session_id, role, content) VALUES (?, ?, ?)",
        sessionID, role, content,
    )
    return err
}

// ListBySession 按会话 ID 查询消息列表，按时间倒序
func (r *MessageRepo) ListBySession(ctx context.Context, sessionID string, limit int) ([]Message, error) {
    rows, err := r.db.QueryContext(ctx,
        "SELECT id, role, content, created_at FROM messages WHERE session_id = ? ORDER BY created_at DESC LIMIT ?",
        sessionID, limit,
    )
    // ... 扫描行数据到 []Message
}
```

## 配置加载

所有配置来自环境变量，在 `internal/config/` 中加载：

```go
// Config 应用配置结构体
type Config struct {
    AppPort      string // APP_PORT，默认 "8080"
    JWTSecret    string // JWT_SECRET
    SQLitePath   string // SQLITE_PATH，默认 "./data/wxx.sqlite"
    ZhipuAPIKey  string // ZHIPU_API_KEY
    DeepSeekKey  string // DEEPSEEK_API_KEY
    // ... 完整列表见 .env.example
}
```

开发环境用 `godotenv` 加载，生产环境直接读取 `os.Getenv`。

## 错误处理

使用 Go 标准的错误包装：

```go
if err != nil {
    return fmt.Errorf("对话服务问答失败: %w", err)
}
```

HTTP 错误响应遵循统一结构：

```go
// ErrorResp 统一错误响应格式
type ErrorResp struct {
    Code    int    `json:"code"`    // 错误码
    Message string `json:"message"` // 错误描述
    TraceID string `json:"trace_id"` // 追踪 ID，用于关联审计日志
}
```

所有错误响应都包含 `trace_id`，用于与 `audit_logs` 进行调试关联。

## 模型定义

`internal/model/` 中的模型映射 `migrations/001_init.sql` 的数据库表：

```go
// User 用户模型，对应 users 表
type User struct {
    ID          int64  `db:"id"`
    Username    string `db:"username"`     // 用户名（唯一）
    DisplayName string `db:"display_name"` // 显示名
    Role        string `db:"role"`         // 角色（六级+扩展）
    OwnerScope  string `db:"owner_scope"`  // 归属范围
    OwnerID     string `db:"owner_id"`     // 归属 ID
    CreatedAt   string `db:"created_at"`   // 创建时间
    UpdatedAt   string `db:"updated_at"`   // 更新时间
}
```

模型结构体必须与 schema 保持一致。如果 schema 变更，同步更新迁移脚本和模型。

## 测试规范

- 单元测试 service 时通过 mock repository 和 llm 接口
- 单元测试 repository 时使用内存 SQLite（`:memory:`）
- Handler 测试使用 `httptest.NewRecorder()` 配合 Gin 测试模式
- 运行所有测试：`make test`
- CI 环境启用竞态检测：`go test -race ./...`

## 依赖管理

已批准的依赖（来自 `go.mod`）：
- `github.com/gin-gonic/gin` — HTTP 框架
- `github.com/golang-jwt/jwt/v5` — JWT 处理
- `github.com/mattn/go-sqlite3` — SQLite 驱动（需要 CGO）
- `github.com/joho/godotenv` — .env 加载

新增依赖需经过讨论 — 项目目标是最小化外部依赖。
