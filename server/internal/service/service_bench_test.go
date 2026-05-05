package service

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// openBenchDB 在内存中创建 SQLite 并执行迁移
func openBenchDB(b *testing.B) *sql.DB {
	b.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		b.Fatalf("打开内存数据库失败: %v", err)
	}
	b.Cleanup(func() { db.Close() })

	migrationPath := "../../migrations/001_init.sql"
	sqlContent, err := os.ReadFile(migrationPath)
	if err != nil {
		migrationPath = "migrations/001_init.sql"
		sqlContent, err = os.ReadFile(migrationPath)
		if err != nil {
			b.Fatalf("读取迁移文件失败: %v", err)
		}
	}
	for _, stmt := range testutil.SplitSQL(string(sqlContent)) {
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			b.Fatalf("迁移失败: %v", err)
		}
	}

	return db
}

// ── ChatService.Ask 全链路性能基准 ──

func setupChatBenchService(b *testing.B) *ChatService {
	b.Helper()

	db := openBenchDB(b)

	sessionRepo := repository.NewSessionRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	kbRepo := repository.NewKBRepo(db)
	agentRepo := repository.NewAgentRepo(db)

	// 插入知识库数据
	for i := 0; i < 50; i++ {
		kbRepo.Create(&model.KBResource{
			ResourceID:   "chat-bench-kb-" + string(rune('0'+i%10)),
			ResourceType: "Policy",
			OwnerScope:   "school",
			RoleScope:    "student",
			Version:      "1.0",
			Status:       "published",
			Title:        "政策文档",
			Summary:      "包含奖学金和入学政策信息",
			Content:      "详细政策内容涵盖申请条件、所需材料、审核流程、办理时间等信息。",
			UpdatedBy:    "bench",
		})
	}

	mockLLM := llm.NewMockClient("bench-llm")
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content:      "根据知识库内容，回答你的问题。",
			FinishReason: "stop",
			PromptTokens: 200,
			OutputTokens: 100,
		}, nil
	}

	return NewChatService(sessionRepo, messageRepo, kbRepo, agentRepo, mockLLM)
}

func BenchmarkChatAsk_NewSession(b *testing.B) {
	svc := setupChatBenchService(b)
	userCtx := &model.UserContext{
		UserID: 1, Username: "bench", Role: "student",
		OwnerScope: "school", OwnerID: "school-1", DisplayName: "基准用户",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := svc.Ask(context.Background(), userCtx, "", "奖学金如何申请？", "")
		if err != nil {
			b.Fatalf("Ask 失败: %v", err)
		}
	}
}

func BenchmarkChatAsk_ExistingSession(b *testing.B) {
	svc := setupChatBenchService(b)
	userCtx := &model.UserContext{
		UserID: 1, Username: "bench", Role: "student",
		OwnerScope: "school", OwnerID: "school-1", DisplayName: "基准用户",
	}

	// 预先创建会话
	_, sessionID, err := svc.Ask(context.Background(), userCtx, "", "你好", "")
	if err != nil {
		b.Fatalf("创建基准会话失败: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := svc.Ask(context.Background(), userCtx, sessionID, "继续聊天", "")
		if err != nil {
			b.Fatalf("Ask 失败: %v", err)
		}
	}
}

// ── AnswerCard Marshal 性能基准 ──

func BenchmarkAnswerCardMarshal(b *testing.B) {
	card := &model.AnswerCard{
		Conclusion: "根据学校政策规定，奖学金申请需要满足以下条件：1. 成绩排名前30%；2. 无违纪记录；3. 家庭经济困难证明。具体申请流程为：登录学工系统 → 填写申请表 → 提交材料 → 等待审核。",
		TraceID:    "trace-bench-12345",
		Confidence: 0.85,
		Fallback:   false,
		Sources: []model.Source{
			{ResourceID: "src-1", Title: "奖学金管理办法", Version: "2.0", RelevanceScore: 0.95},
			{ResourceID: "src-2", Title: "学生资助实施细则", Version: "1.5", RelevanceScore: 0.82},
			{ResourceID: "src-3", Title: "学工系统操作指南", Version: "3.0", RelevanceScore: 0.71},
		},
		FollowUps: []string{"申请需要哪些材料？", "办理地点在哪里？", "错过截止日期怎么办？"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MarshalAnswerCard(card)
	}
}

// ── AuthService Login 性能基准 ──

func BenchmarkLoginByUsername(b *testing.B) {
	db := openBenchDB(b)

	cfg := &config.Config{
		JWTSecret:      "bench-jwt-secret-32chars-minimum",
		JWTExpireHours: 24,
	}
	userRepo := repository.NewUserRepo(db)
	authSvc := NewAuthService(cfg, userRepo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := authSvc.LoginByUsername("benchuser-" + string(rune('0'+i%10)))
		if err != nil {
			b.Fatalf("LoginByUsername 失败: %v", err)
		}
	}
}
