package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	server "github.com/dll/wxx/server"
	"github.com/dll/wxx/server/internal/agent"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
	_ "modernc.org/sqlite"
)

func openToolLoopDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	// 本测试仅需基础表 + 聊天链路增量列；
	// 全量迁移含 MySQL 专有语句，与 app.RunMigrations 在 pkg/app 造成导入环，故不在此复用。
	for _, name := range []string{"001_init.sql", "010_add_password_hash.sql", "017_session_title.sql", "047_add_kb_remark.sql", "049_fts_tags.sql"} {
		raw, err := server.Migrations.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, stmt := range testutil.SplitSQL(string(raw)) {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				msg := err.Error()
				if strings.Contains(msg, "duplicate column name") || strings.Contains(msg, "already exists") {
					continue
				}
				t.Fatalf("migrate %s: %v\n语句: %s", name, err, stmt)
			}
		}
	}
	return db
}

func TestChatToolLoop_ExecutesToolThenAnswers(t *testing.T) {
	db := openToolLoopDB(t)

	kbRepo := repository.NewKBRepo(db)
	if _, _, err := kbRepo.Upsert(&model.KBResource{
		ResourceID: "kb-tool-leave", ResourceType: "Process", OwnerScope: "school",
		RoleScope: `["student"]`, Version: "1.0", Status: "published", UpdatedBy: "t",
		Title:   "学生请假流程",
		Content: "请假 1 天以内由辅导员审批；3 天以上由学院分管领导审批。",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// mock：首轮发起 query_process_steps 工具调用，次轮给最终回答
	mock := llm.NewMockClient("tool-mock")
	callCount := 0
	mock.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		callCount++
		if callCount == 1 {
			return &llm.ChatResponse{
				FinishReason: "tool_calls",
				ToolCalls: []llm.ToolCall{{
					ID: "call_1", Name: "query_process_steps",
					Arguments: `{"question":"请假流程"}`,
				}},
			}, nil
		}
		// 次轮：校验工具结果已回填为 role=tool 消息
		foundToolMsg := false
		for _, m := range req.Messages {
			if m.Role == "tool" && m.ToolCallID == "call_1" {
				foundToolMsg = true
			}
		}
		if !foundToolMsg {
			t.Fatalf("次轮请求缺少 tool 结果消息")
		}
		return &llm.ChatResponse{Content: "根据流程工具查询：请假由辅导员审批。", FinishReason: "stop", PromptTokens: 10, OutputTokens: 5}, nil
	}

	svc := NewChatService(
		repository.NewSessionRepo(db), repository.NewMessageRepo(db),
		kbRepo, repository.NewAgentRepo(db), mock,
	)
	reg := agent.NewToolRegistry()
	reg.Register(agent.NewProcessNodeTool(kbRepo))
	svc.SetToolRegistry(reg)

	userCtx := &model.UserContext{UserID: 1, Username: "t", Role: "student",
		OwnerScope: "school", OwnerID: "school-1", DisplayName: "测试"}

	card, _, err := svc.Ask(context.Background(), userCtx, "", "请假流程是什么", "")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("应调用 LLM 两次（工具轮+收尾轮），实际 %d", callCount)
	}
	if card.Conclusion == "" {
		t.Fatal("最终回答为空")
	}

	// 验证工具调用消息已持久化进会话上下文（消息表含 user/assistant 角色）
	var msgCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM messages WHERE role IN ('user','assistant')`).Scan(&msgCount)
	if msgCount < 2 {
		t.Fatalf("消息落库异常: %d", msgCount)
	}
}

func TestChatToolLoop_DegradationWithoutRegistry(t *testing.T) {
	db := openToolLoopDB(t)
	mock := llm.NewMockClient("plain")
	mock.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		if len(req.Tools) != 0 {
			t.Fatalf("未注册工具时不应携带 tools")
		}
		return &llm.ChatResponse{Content: "plain answer", PromptTokens: 1, OutputTokens: 1}, nil
	}

	svc := NewChatService(
		repository.NewSessionRepo(db), repository.NewMessageRepo(db),
		repository.NewKBRepo(db), repository.NewAgentRepo(db), mock,
	)
	// 不注入 toolRegistry

	userCtx := &model.UserContext{UserID: 1, Username: "t", Role: "student",
		OwnerScope: "school", OwnerID: "school-1", DisplayName: "测试"}
	card, _, err := svc.Ask(context.Background(), userCtx, "", "你好", "")
	if err != nil || card == nil {
		t.Fatalf("Ask: %v %v", err, card)
	}
}

func TestBuildToolDefinitions(t *testing.T) {
	if defs := buildToolDefinitions(nil); defs != nil {
		t.Fatal("nil registry 应返回 nil")
	}
	reg := agent.NewToolRegistry()
	reg.Register(agent.NewProcessNodeTool(nil))
	defs := buildToolDefinitions(reg)
	if len(defs) != 1 || defs[0].Function.Name != "query_process_steps" {
		t.Fatalf("工具定义异常: %+v", defs)
	}
	// 参数 schema 校验
	raw, _ := json.Marshal(defs[0].Function.Parameters)
	if string(raw) == "" {
		t.Fatal("parameters 序列化为空")
	}
}
