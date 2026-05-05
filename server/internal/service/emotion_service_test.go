package service

import (
	"context"
	"testing"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

func setupEmotionServiceTestDB(t *testing.T) *EmotionService {
	t.Helper()

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	// 重建 emotion_logs 表（001_init.sql 的旧 schema 缺少字段）
	db.Exec("DROP TABLE IF EXISTS emotion_logs")
	db.Exec(`CREATE TABLE emotion_logs (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id         INTEGER NOT NULL REFERENCES users(id),
		session_id      TEXT    NOT NULL,
		message_text    TEXT    NOT NULL DEFAULT '',
		score           REAL    NOT NULL DEFAULT 0,
		risk_level      TEXT    NOT NULL DEFAULT 'low'
		                CHECK(risk_level IN ('low','medium','high','urgent')),
		analysis_json   TEXT    NOT NULL DEFAULT '{}',
		notified        INTEGER NOT NULL DEFAULT 0,
		status          TEXT    NOT NULL DEFAULT 'pending'
		                CHECK(status IN ('pending','acknowledged','resolved')),
		acknowledged_by TEXT    NOT NULL DEFAULT '',
		acknowledged_at TEXT    DEFAULT NULL,
		alert_id        TEXT    NOT NULL DEFAULT '',
		username        TEXT    NOT NULL DEFAULT '',
		created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
	)`)

	// 创建测试用户（ListAlerts 和 GetStats 需要 JOIN users）
	db.Exec(`INSERT INTO users (username, role, display_name)
		VALUES ('testuser', 'student', '测试用户')`)

	return NewEmotionService(repository.NewEmotionRepo(db), llm.NewMockClient("emotion-test"))
}

// ── AnalyzeAndLog 测试 ──

func TestEmotionService_AnalyzeAndLog_LowRisk(t *testing.T) {
	svc := setupEmotionServiceTestDB(t)

	// 设置 mock LLM 返回低风险分析
	mockLLM := svc.llmClient.(*llm.MockClient)
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"score": 0.3, "risk_level": "low", "emotions": ["平静"], "keywords": [], "reasoning": "正常交流", "need_follow_up": false}`,
			FinishReason: "stop",
		}, nil
	}

	log, err := svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-1", "今天天气不错")
	if err != nil {
		t.Fatalf("AnalyzeAndLog 失败: %v", err)
	}
	if log.RiskLevel != "low" {
		t.Errorf("期望 risk_level=low，得到 %s", log.RiskLevel)
	}
	if log.Notified != 0 {
		t.Errorf("低风险不应通知，得到 notified=%d", log.Notified)
	}
	if log.UserID != 1 {
		t.Errorf("期望 UserID=1，得到 %d", log.UserID)
	}
	if log.AlertID == "" {
		t.Error("alert_id 不应为空")
	}
}

func TestEmotionService_AnalyzeAndLog_HighRisk(t *testing.T) {
	svc := setupEmotionServiceTestDB(t)

	mockLLM := svc.llmClient.(*llm.MockClient)
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"score": -0.9, "risk_level": "high", "emotions": ["绝望", "焦虑"], "keywords": ["不想活了"], "reasoning": "学生表达了严重负面情绪", "need_follow_up": true}`,
			FinishReason: "stop",
		}, nil
	}

	log, err := svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-2", "我觉得很绝望")
	if err != nil {
		t.Fatalf("AnalyzeAndLog 失败: %v", err)
	}
	if log.RiskLevel != "high" {
		t.Errorf("期望 risk_level=high，得到 %s", log.RiskLevel)
	}
	if log.Notified != 1 {
		t.Errorf("高风险应通知，得到 notified=%d", log.Notified)
	}
}

func TestEmotionService_AnalyzeAndLog_UrgentRisk(t *testing.T) {
	svc := setupEmotionServiceTestDB(t)

	mockLLM := svc.llmClient.(*llm.MockClient)
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"score": -1.0, "risk_level": "urgent", "emotions": ["绝望"], "keywords": ["自杀"], "reasoning": "紧急情况", "need_follow_up": true}`,
			FinishReason: "stop",
		}, nil
	}

	log, err := svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-3", "我不想活了")
	if err != nil {
		t.Fatalf("AnalyzeAndLog 失败: %v", err)
	}
	if log.RiskLevel != "urgent" {
		t.Errorf("期望 risk_level=urgent，得到 %s", log.RiskLevel)
	}
	if log.Notified != 1 {
		t.Errorf("紧急风险应通知，得到 notified=%d", log.Notified)
	}
}

func TestEmotionService_AnalyzeAndLog_LLMFailureFallback(t *testing.T) {
	svc := setupEmotionServiceTestDB(t)

	mockLLM := svc.llmClient.(*llm.MockClient)
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return nil, context.DeadlineExceeded
	}

	log, err := svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-4", "测试消息")
	if err != nil {
		t.Fatalf("LLM 失败时 AnalyzeAndLog 应走兜底（不应返回错误）: %v", err)
	}
	if log.RiskLevel != "low" {
		t.Errorf("LLM 失败时兜底 risk_level 应为 low，得到 %s", log.RiskLevel)
	}
	if log.Score != 0 {
		t.Errorf("LLM 失败时兜底 score 应为 0，得到 %f", log.Score)
	}
}

// ── ListAlerts / GetStats / GetTrendReport / UpdateAlertStatus ──

func TestEmotionService_ListAlerts(t *testing.T) {
	svc := setupEmotionServiceTestDB(t)

	// 使用 AnalyzeAndLog 创建两条记录
	mockLLM := svc.llmClient.(*llm.MockClient)
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"score": -0.5, "risk_level": "medium", "emotions": ["焦虑"], "keywords": [], "reasoning": "压力", "need_follow_up": false}`,
			FinishReason: "stop",
		}, nil
	}

	svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-a", "消息1")
	svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-b", "消息2")

	alerts, total, err := svc.ListAlerts("", "", "school", "", "sys_admin", 1, 10)
	if err != nil {
		t.Fatalf("ListAlerts 失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望 total=2，得到 %d", total)
	}
	if len(alerts) != 2 {
		t.Errorf("期望 2 条告警，得到 %d", len(alerts))
	}
}

func TestEmotionService_ListAlerts_FilterByRisk(t *testing.T) {
	svc := setupEmotionServiceTestDB(t)

	mockLLM := svc.llmClient.(*llm.MockClient)
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"score": -0.9, "risk_level": "high", "emotions": ["愤怒"], "keywords": [], "reasoning": "高风险", "need_follow_up": true}`,
			FinishReason: "stop",
		}, nil
	}
	svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-h", "高风险消息")

	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"score": 0.5, "risk_level": "low", "emotions": ["开心"], "keywords": [], "reasoning": "积极", "need_follow_up": false}`,
			FinishReason: "stop",
		}, nil
	}
	svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-l", "低风险消息")

	_, total, err := svc.ListAlerts("high", "", "school", "", "sys_admin", 1, 10)
	if err != nil {
		t.Fatalf("ListAlerts 过滤失败: %v", err)
	}
	if total != 1 {
		t.Errorf("过滤 high 后期望 total=1，得到 %d", total)
	}
}

func TestEmotionService_ListAlerts_DefaultPagination(t *testing.T) {
	svc := setupEmotionServiceTestDB(t)

	// page=0, pageSize=0 应被修正为 page=1, pageSize=20
	_, _, err := svc.ListAlerts("", "", "school", "", "sys_admin", 0, 0)
	if err != nil {
		t.Fatalf("ListAlerts 边界分页失败: %v", err)
	}
}

func TestEmotionService_GetStats(t *testing.T) {
	svc := setupEmotionServiceTestDB(t)

	mockLLM := svc.llmClient.(*llm.MockClient)
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"score": -1.0, "risk_level": "urgent", "emotions": ["绝望"], "keywords": [], "reasoning": "紧急", "need_follow_up": true}`,
			FinishReason: "stop",
		}, nil
	}
	svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-urgent", "紧急")

	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"score": -0.7, "risk_level": "high", "emotions": ["焦虑"], "keywords": [], "reasoning": "高", "need_follow_up": true}`,
			FinishReason: "stop",
		}, nil
	}
	svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-high", "高")

	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"score": -0.4, "risk_level": "medium", "emotions": [], "keywords": [], "reasoning": "中", "need_follow_up": false}`,
			FinishReason: "stop",
		}, nil
	}
	svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-med", "中")

	stats, err := svc.GetStats("school", "", "sys_admin")
	if err != nil {
		t.Fatalf("GetStats 失败: %v", err)
	}
	if stats.Urgent != 1 {
		t.Errorf("期望 Urgent=1，得到 %d", stats.Urgent)
	}
	if stats.High != 1 {
		t.Errorf("期望 High=1，得到 %d", stats.High)
	}
	if stats.Medium != 1 {
		t.Errorf("期望 Medium=1，得到 %d", stats.Medium)
	}
	if stats.Pending != 3 {
		t.Errorf("期望 Pending=3，得到 %d", stats.Pending)
	}
}

func TestEmotionService_GetTrendReport(t *testing.T) {
	svc := setupEmotionServiceTestDB(t)

	mockLLM := svc.llmClient.(*llm.MockClient)
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"score": -0.6, "risk_level": "medium", "emotions": ["焦虑"], "keywords": [], "reasoning": "测试", "need_follow_up": false}`,
			FinishReason: "stop",
		}, nil
	}
	svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-tr", "测试趋势")

	report, err := svc.GetTrendReport(7, "school", "", "sys_admin")
	if err != nil {
		t.Fatalf("GetTrendReport 失败: %v", err)
	}
	if report.Days != 7 {
		t.Errorf("期望 Days=7，得到 %d", report.Days)
	}
	if len(report.Points) == 0 {
		t.Error("应有至少 1 个数据点")
	}
	if report.Summary.TotalAnalyses < 1 {
		t.Error("summary 应体现分析次数")
	}
}

func TestEmotionService_GetTrendReport_DefaultDays(t *testing.T) {
	svc := setupEmotionServiceTestDB(t)

	// days=0 应被修正为 7
	report, err := svc.GetTrendReport(0, "school", "", "sys_admin")
	if err != nil {
		t.Fatalf("GetTrendReport 失败: %v", err)
	}
	if report.Days != 7 {
		t.Errorf("days=0 应修正为 7，得到 %d", report.Days)
	}
}

func TestEmotionService_UpdateAlertStatus(t *testing.T) {
	svc := setupEmotionServiceTestDB(t)

	mockLLM := svc.llmClient.(*llm.MockClient)
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"score": -0.5, "risk_level": "medium", "emotions": [], "keywords": [], "reasoning": "测试", "need_follow_up": false}`,
			FinishReason: "stop",
		}, nil
	}
	created, _ := svc.AnalyzeAndLog(context.Background(), 1, "testuser", "sess-upd", "更新测试")

	updated, err := svc.UpdateAlertStatus(created.AlertID, "acknowledged", "counselor1")
	if err != nil {
		t.Fatalf("UpdateAlertStatus 失败: %v", err)
	}
	if updated.Status != "acknowledged" {
		t.Errorf("期望 Status=acknowledged，得到 %s", updated.Status)
	}
	if updated.AcknowledgedBy != "counselor1" {
		t.Errorf("期望 AcknowledgedBy=counselor1，得到 %s", updated.AcknowledgedBy)
	}
}

// ── parseEmotionResponse 纯函数测试 ──

func TestParseEmotionResponse_ValidJSON(t *testing.T) {
	result, err := parseEmotionResponse(`{"score": -0.5, "risk_level": "medium", "emotions": ["焦虑"], "keywords": ["考试"], "reasoning": "学业压力", "need_follow_up": false}`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if result.Score != -0.5 {
		t.Errorf("期望 Score=-0.5，得到 %f", result.Score)
	}
	if result.RiskLevel != "medium" {
		t.Errorf("期望 RiskLevel=medium，得到 %s", result.RiskLevel)
	}
}

func TestParseEmotionResponse_MarkdownWrapped(t *testing.T) {
	result, err := parseEmotionResponse("```json\n{\"score\": 0.8, \"risk_level\": \"low\", \"emotions\": [\"开心\"], \"keywords\": [], \"reasoning\": \"积极\", \"need_follow_up\": false}\n```")
	if err != nil {
		t.Fatalf("解析 markdown 包裹的 JSON 失败: %v", err)
	}
	if result.Score != 0.8 {
		t.Errorf("期望 Score=0.8，得到 %f", result.Score)
	}
}

func TestParseEmotionResponse_InvalidRiskLevel(t *testing.T) {
	result, err := parseEmotionResponse(`{"score": 0, "risk_level": "impossible", "emotions": [], "keywords": [], "reasoning": "", "need_follow_up": false}`)
	if err != nil {
		t.Fatalf("无效 risk_level 不应导致解析失败: %v", err)
	}
	if result.RiskLevel != "low" {
		t.Errorf("无效 risk_level 应修正为 low，得到 %s", result.RiskLevel)
	}
}

func TestParseEmotionResponse_ScoreOutOfRange(t *testing.T) {
	result, err := parseEmotionResponse(`{"score": -5.0, "risk_level": "low", "emotions": [], "keywords": [], "reasoning": "", "need_follow_up": false}`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if result.Score != -1.0 {
		t.Errorf("score 应钳位到 -1.0，得到 %f", result.Score)
	}

	result, err = parseEmotionResponse(`{"score": 5.0, "risk_level": "low", "emotions": [], "keywords": [], "reasoning": "", "need_follow_up": false}`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if result.Score != 1.0 {
		t.Errorf("score 应钳位到 1.0，得到 %f", result.Score)
	}
}

func TestParseEmotionResponse_InvalidJSON(t *testing.T) {
	_, err := parseEmotionResponse("这不是 JSON")
	if err == nil {
		t.Fatal("无效 JSON 应返回错误")
	}
}

// ── buildEmotionPrompt 纯函数测试 ──

func TestBuildEmotionPrompt(t *testing.T) {
	prompt := buildEmotionPrompt("考试压力很大")
	if prompt == "" {
		t.Error("提示词不应为空")
	}
	if !contains(prompt, "考试压力很大") {
		t.Errorf("提示词应包含原始文本，得到: %s", prompt)
	}
}

// ── boolToInt 纯函数测试 ──

func TestBoolToInt(t *testing.T) {
	if boolToInt(true) != 1 {
		t.Error("true 应返回 1")
	}
	if boolToInt(false) != 0 {
		t.Error("false 应返回 0")
	}
}

// ── 辅助函数 ──

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
