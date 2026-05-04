---
name: wxx-go-backend
description: Go/Gin backend development conventions for WeiXiaoXin (蔚小芯). Triggers when creating or modifying Go source files in server/, adding new API endpoints, implementing handlers/services/repositories, working with SQLite queries, or structuring the Gin router. Also triggers on phrases like "后端", "接口", "API", "handler", "service", "repository", "Gin", "路由", or when generating Go code for the project. Use this skill to ensure all Go code follows the project's strict layering rules.
---

# 蔚小芯 Go Backend Development

This skill enforces the Go/Gin backend architecture for 蔚小芯. The project uses strict layering to keep business logic testable and prevent the spaghetti that comes from handlers directly touching databases or LLM APIs.

## Architecture Layers

```
HTTP Request
    |
    v
[middleware/]  --> JWT auth, RBAC check, rate limit, audit log, CORS
    |
    v
[handler/]     --> Parse request, validate params, call service, format response
    |
    v
[service/]     --> Business logic: orchestrate repository + context_engine + llm + agent
    |
    v
[repository/]  --> SQL queries against SQLite (parameterized only)
[context_engine/] --> Knowledge retrieval pipeline (FTS + structured)
[llm/]         --> 智谱/DeepSeek/讯飞 API clients
[agent/]       --> Multi-agent orchestration via Eino
```

### Layer Rules (ENFORCED)

| From | Can call | Cannot call |
|------|----------|-------------|
| handler | service | repository, llm, agent, context_engine |
| service | repository, context_engine, llm, agent | handler |
| repository | (SQLite only) | service, handler, llm, context_engine |
| context_engine | repository (for FTS/structured queries) | handler, llm |
| llm | (external HTTP only) | repository, handler |

Violations of these rules should be flagged and blocked before commit.

## Adding a New Endpoint

Follow this checklist:

### 1. Define the Route

In the router setup (to be implemented in `cmd/server/main.go` or a dedicated `router.go`):

```go
api := r.Group("/api/v1")
api.Use(middleware.Auth())  // JWT validation

// Group by domain
chat := api.Group("/chat")
chat.Use(middleware.RateLimit(30))  // per-minute limit
{
    chat.POST("/ask", chatHandler.Ask)
    chat.GET("/sessions", chatHandler.ListSessions)
}
```

### 2. Write the Handler

Handlers do THREE things only: parse, delegate, respond.

```go
// internal/handler/chat_handler.go
func (h *ChatHandler) Ask(c *gin.Context) {
    // 1. Parse & validate
    var req AskRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse(err))
        return
    }

    // 2. Extract auth context from middleware
    userCtx := middleware.GetUserContext(c)

    // 3. Delegate to service
    answer, err := h.chatService.Ask(c.Request.Context(), userCtx, req.Question)
    if err != nil {
        c.JSON(500, ErrorResponse(err))
        return
    }

    // 4. Respond
    c.JSON(200, answer)
}
```

### 3. Implement the Service

Services contain business logic and orchestrate multiple dependencies:

```go
// internal/service/chat_service.go
func (s *ChatService) Ask(ctx context.Context, user UserContext, question string) (*AnswerCard, error) {
    // 1. Query Context Engine for relevant knowledge
    ctxResult, err := s.contextEngine.Query(ctx, question, user.Role, user.Scope)
    if err != nil {
        return nil, fmt.Errorf("context query: %w", err)
    }

    // 2. Assemble prompt with context
    prompt := s.assemblePrompt(question, ctxResult)

    // 3. Call LLM
    rawAnswer, err := s.llmClient.Chat(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("llm call: %w", err)
    }

    // 4. Build AnswerCard with sources
    card := s.buildAnswerCard(rawAnswer, ctxResult.Sources)

    // 5. Save message to history
    _ = s.messageRepo.Save(ctx, user.SessionID, "user", question)
    _ = s.messageRepo.Save(ctx, user.SessionID, "assistant", card.Conclusion)

    return card, nil
}
```

### 4. Write the Repository

Repositories are pure SQL — no business logic, no HTTP, no external APIs:

```go
// internal/repository/message_repo.go
func (r *MessageRepo) Save(ctx context.Context, sessionID, role, content string) error {
    _, err := r.db.ExecContext(ctx,
        "INSERT INTO messages (session_id, role, content) VALUES (?, ?, ?)",
        sessionID, role, content,
    )
    return err
}

func (r *MessageRepo) ListBySession(ctx context.Context, sessionID string, limit int) ([]Message, error) {
    rows, err := r.db.QueryContext(ctx,
        "SELECT id, role, content, created_at FROM messages WHERE session_id = ? ORDER BY created_at DESC LIMIT ?",
        sessionID, limit,
    )
    // ... scan rows into []Message
}
```

## Config Loading

All configuration comes from environment variables, loaded in `internal/config/`:

```go
type Config struct {
    AppPort      string // APP_PORT, default "8080"
    JWTSecret    string // JWT_SECRET
    SQLitePath   string // SQLITE_PATH, default "./data/wxx.sqlite"
    ZhipuAPIKey  string // ZHIPU_API_KEY
    DeepSeekKey  string // DEEPSEEK_API_KEY
    // ... see .env.example for full list
}
```

Load with `godotenv` for dev, raw `os.Getenv` for production.

## Error Handling

Use Go's standard error wrapping:

```go
if err != nil {
    return fmt.Errorf("chat service ask: %w", err)
}
```

HTTP error responses follow a consistent structure:

```go
type ErrorResp struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    TraceID string `json:"trace_id"`
}
```

Include `trace_id` in all error responses for debugging correlation with `audit_logs`.

## Model Definitions

Models in `internal/model/` map to SQLite tables from `migrations/001_init.sql`:

```go
type User struct {
    ID          int64  `db:"id"`
    Username    string `db:"username"`
    DisplayName string `db:"display_name"`
    Role        string `db:"role"`
    OwnerScope  string `db:"owner_scope"`
    OwnerID     string `db:"owner_id"`
    CreatedAt   string `db:"created_at"`
    UpdatedAt   string `db:"updated_at"`
}
```

Keep model structs aligned with the schema. If schema changes, update both the migration and the model.

## Testing

- Unit test services by mocking repository and llm interfaces
- Unit test repositories with an in-memory SQLite (`:memory:`)
- Handler tests use `httptest.NewRecorder()` with the Gin test mode
- Run all tests: `make test`
- Run with race detector during CI: `go test -race ./...`

## Package Dependencies

Approved dependencies (from `go.mod`):
- `github.com/gin-gonic/gin` — HTTP framework
- `github.com/golang-jwt/jwt/v5` — JWT handling
- `github.com/mattn/go-sqlite3` — SQLite driver (CGO required)
- `github.com/joho/godotenv` — .env loading

Adding new dependencies requires discussion — the project aims for minimal external deps.
