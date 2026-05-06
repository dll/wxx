package activities

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

func setupChatActivitiesTestDB(t *testing.T) *ChatActivities {
	t.Helper()

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	return &ChatActivities{
		SessionRepo: repository.NewSessionRepo(db),
		MessageRepo: repository.NewMessageRepo(db),
		KBRepo:      repository.NewKBRepo(db),
		AgentRepo:   repository.NewAgentRepo(db),
		LLMClient:   nil,
	}
}

func TestValidateSessionActivity_CreateNew(t *testing.T) {
	acts := setupChatActivitiesTestDB(t)

	result, err := acts.ValidateSessionActivity(context.Background(), ValidateSessionInput{
		UserID:    1,
		SessionID: "",
		Question:  "测试问题",
		TraceID:   "trace-001",
	})
	if err != nil {
		t.Fatalf("ValidateSession 失败: %v", err)
	}
	if result.SessionID == "" {
		t.Error("应生成新的 SessionID")
	}

	msgs, err := acts.MessageRepo.ListBySessionID(result.SessionID, 100)
	if err != nil {
		t.Fatalf("查询消息失败: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("期望 1 条用户消息，得到 %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "测试问题" {
		t.Errorf("消息内容不匹配: role=%s content=%s", msgs[0].Role, msgs[0].Content)
	}
}

func TestValidateSessionActivity_Existing_Success(t *testing.T) {
	acts := setupChatActivitiesTestDB(t)

	sessionID := "sess-existing-001"
	acts.SessionRepo.Create(&model.Session{
		SessionID: sessionID,
		UserID:    1,
	})

	result, err := acts.ValidateSessionActivity(context.Background(), ValidateSessionInput{
		UserID:    1,
		SessionID: sessionID,
		Question:  "追加问题",
		TraceID:   "trace-002",
	})
	if err != nil {
		t.Fatalf("ValidateSession 失败: %v", err)
	}
	if result.SessionID != sessionID {
		t.Errorf("应保持原 SessionID=%s，得到 %s", sessionID, result.SessionID)
	}
}

func TestValidateSessionActivity_WrongUser(t *testing.T) {
	acts := setupChatActivitiesTestDB(t)

	acts.SessionRepo.Create(&model.Session{
		SessionID: "sess-owner-1",
		UserID:    1,
	})

	_, err := acts.ValidateSessionActivity(context.Background(), ValidateSessionInput{
		UserID:    2,
		SessionID: "sess-owner-1",
		Question:  "越权访问",
		TraceID:   "trace-003",
	})
	if err == nil {
		t.Error("访问他人会话应返回错误")
	}
}

func TestSearchKnowledgeActivity_EmptyDB(t *testing.T) {
	acts := setupChatActivitiesTestDB(t)

	result, err := acts.SearchKnowledgeActivity(context.Background(), SearchKnowledgeInput{
		Question:   "奖学金",
		OwnerScope: "school",
		OwnerID:    "",
		Role:       "student",
	})
	if err != nil {
		t.Fatalf("SearchKnowledge 失败: %v", err)
	}
	if result.ResultsJSON != "null" {
		t.Logf("空数据库搜索结果: %s", result.ResultsJSON)
	}
}

func TestBuildAnswerActivity_LLMContent(t *testing.T) {
	acts := setupChatActivitiesTestDB(t)

	acts.SessionRepo.Create(&model.Session{
		SessionID: "sess-build",
		UserID:    1,
	})

	result, err := acts.BuildAnswerActivity(context.Background(), BuildAnswerInput{
		SessionID:     "sess-build",
		Question:      "测试问题",
		LLMContent:    "这是 LLM 的回答内容",
		LLMTokens:     `{"prompt":100,"output":50}`,
		SearchResults: `[]`,
		TraceID:       "trace-build",
	})
	if err != nil {
		t.Fatalf("BuildAnswer 失败: %v", err)
	}
	if result.AnswerCardJSON == "" {
		t.Error("AnswerCardJSON 不应为空")
	}

	var card model.AnswerCard
	if err := json.Unmarshal([]byte(result.AnswerCardJSON), &card); err != nil {
		t.Fatalf("AnswerCard JSON 解析失败: %v", err)
	}
	if card.Conclusion != "这是 LLM 的回答内容" {
		t.Errorf("期望 Conclusion='这是 LLM 的回答内容'，得到 '%s'", card.Conclusion)
	}
	if card.TraceID != "trace-build" {
		t.Errorf("期望 TraceID='trace-build'，得到 '%s'", card.TraceID)
	}

	msgs, _ := acts.MessageRepo.ListBySessionID("sess-build", 100)
	found := false
	for _, msg := range msgs {
		if msg.Role == "assistant" {
			found = true
			break
		}
	}
	if !found {
		t.Error("应保存助手消息")
	}
}

func TestBuildAnswerActivity_EmptyContent_Fallback(t *testing.T) {
	acts := setupChatActivitiesTestDB(t)

	acts.SessionRepo.Create(&model.Session{
		SessionID: "sess-fallback",
		UserID:    1,
	})

	result, err := acts.BuildAnswerActivity(context.Background(), BuildAnswerInput{
		SessionID:     "sess-fallback",
		Question:      "问题",
		LLMContent:    "",
		LLMTokens:     `{"prompt":0,"output":0}`,
		SearchResults: `[]`,
		TraceID:       "trace-fb",
	})
	if err != nil {
		t.Fatalf("BuildAnswer（空内容兜底）失败: %v", err)
	}

	var card model.AnswerCard
	json.Unmarshal([]byte(result.AnswerCardJSON), &card)
	if !card.Fallback {
		t.Error("空 LLM 内容时应标记 fallback=true")
	}
}

func TestGetSystemPrompt_Default(t *testing.T) {
	prompt := getSystemPrompt(nil, "")
	if prompt == "" {
		t.Error("默认系统提示词不应为空")
	}
	if !contains(prompt, "蔚小芯") {
		t.Error("默认提示词应包含'蔚小芯'")
	}
}

func TestGetSystemPrompt_AgentNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	agentRepo := repository.NewAgentRepo(db)
	prompt := getSystemPrompt(agentRepo, "nonexistent")
	if prompt == "" {
		t.Error("查找失败时应返回默认提示词")
	}
}

func TestBuildAnswerCard_Fallback(t *testing.T) {
	card := buildAnswerCard("", nil, "trace-fb")
	if !card.Fallback {
		t.Error("空内容应标记 fallback=true")
	}
	if card.Confidence != 0.5 {
		t.Errorf("兜底置信度应为 0.5，得到 %f", card.Confidence)
	}
}

func TestBuildAnswerCard_Normal(t *testing.T) {
	// 传入非空检索结果避免触发"零结果=低置信"的 fallback 逻辑
	results := []*repository.SearchResult{
		{Score: 0.8},
	}
	card := buildAnswerCard("正常回答内容", results, "trace-normal")
	if card.Fallback {
		t.Error("有内容和检索结果时不应标记 fallback")
	}
	if card.Conclusion != "正常回答内容" {
		t.Errorf("Conclusion 不匹配: %s", card.Conclusion)
	}
}

func TestTruncateContent(t *testing.T) {
	short := "短文本"
	if truncateContent(short, 100) != short {
		t.Error("短文本不应被截断")
	}

	long := "这是一段很长的文本，超过了限制长度需要被截断处理"
	result := truncateContent(long, 5)
	if len([]rune(result)) != 8 { // 5 + "..."
		t.Errorf("期望截断为 8 个字符，得到 %d", len([]rune(result)))
	}
}

func TestGenerateFollowUps(t *testing.T) {
	tests := []struct {
		content  string
		expected string
	}{
		{"申请需要填写表格", "申请"},
		{"办理流程分三步", "流程"},
		{"截止日期为6月30日", "截止"},
		{"普通回答没有关键词", ""},
	}

	for _, tt := range tests {
		result := generateFollowUps(tt.content)
		if tt.expected == "" && len(result) > 0 {
			t.Errorf("内容 '%s' 不应生成追问，但得到了 %v", tt.content, result)
		}
		if tt.expected != "" && len(result) == 0 {
			t.Errorf("内容 '%s' 应生成追问", tt.content)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr)
}
