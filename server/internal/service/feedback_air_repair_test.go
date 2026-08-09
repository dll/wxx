package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// fakeOCRVision 模拟视觉模型（回传固定 OCR 文本）
type fakeOCRVision struct{}

func (f fakeOCRVision) Name() string { return "fake-vision" }
func (f fakeOCRVision) OCR(_ context.Context, _ []llm.OCRImage) (string, error) {
	return "锁屏文字 测试账号 请到教务系统", nil
}

// setupFeedbackAIRepairTest 创建带真实内存库的 FeedbackService
func setupFeedbackAIRepairTest(t *testing.T) (*FeedbackService, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDBFull(t)
	userRepo := repository.NewUserRepo(db)
	screenshotRepo := repository.NewFeedbackScreenshotRepo(db)
	fbRepo := repository.NewFeedbackRepo(db)
	svc := NewFeedbackService(fbRepo, userRepo, screenshotRepo)
	svc.SetDB(db)
	return svc, db
}

// TestAIRepair_LocalFallback LLM/视觉不可用时的本地兜底
func TestAIRepair_LocalFallback(t *testing.T) {
	svc, _ := setupFeedbackAIRepairTest(t)

	fb := &model.Feedback{
		FeedbackID: "fb-local-fallback",
		UserID:     1,
		Username:   "s1",
		Category:   "answer_error",
		Module:     "对话 / 问答",
		Content:    "回答的内容不太准确，希望改进",
		Status:     "pending",
	}
	if _, err := svc.feedbackRepo.Create(fb); err != nil {
		t.Fatalf("creating feedback: %v", err)
	}

	resp, err := svc.AIRepair(context.Background(), fb.FeedbackID, "admin")
	if err != nil {
		t.Fatalf("AIRepair should not error in fallback: %v", err)
	}
	if resp.Module == "" {
		t.Error("expected module non-empty in fallback")
	}
	if resp.Summary == "" {
		t.Error("expected summary non-empty")
	}
	if len(resp.CodeFiles) == 0 {
		t.Error("expected code files matched (fallback)")
	}
}

// TestAIRepair_WithLLM 有 LLM + 视觉时返回 AI 结构化结果
func TestAIRepair_WithLLM(t *testing.T) {
	svc, _ := setupFeedbackAIRepairTest(t)

	mock := llm.NewMockClient("mock")
	mock.ChatFunc = func(_ context.Context, _ *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"module":"对话 / 问答","summary":"回答不准确","code_files":["server/internal/service/chat_service.go"],"root_cause":"检索未命中","repair_hint":"检查FTS"}`,
		}, nil
	}
	svc.SetAIRepairClients(fakeOCRVision{}, mock)

	fb := &model.Feedback{
		FeedbackID:    "fb-ai-llm",
		UserID:        1,
		Username:      "s1",
		Category:      "answer_error",
Module:        "对话 / 问答",
		Content:       "回答内容不准确，希望改进",
		ScreenshotURL: "/uploads/feedback/xxx.png",
		Status:        "pending",
	}
	if _, err := svc.feedbackRepo.Create(fb); err != nil {
		t.Fatalf("creating feedback: %v", err)
	}
	// 保存截图 blob（base64），确保反馈详情 OCR 分支被触发
	if err := svc.screenshotRepo.Save("xxx.png", "image/png", "aW1n", "uploader", 3); err != nil {
		t.Fatalf("saving screenshot: %v", err)
	}

	resp, err := svc.AIRepair(context.Background(), fb.FeedbackID, "admin")
	if err != nil {
		t.Fatalf("AIRepair error: %v", err)
	}
	if resp.Module != "对话 / 问答" {
		t.Errorf("module expected 对话 / 问答, got %q", resp.Module)
	}
	if resp.OCRText == "" {
		t.Error("expected ocr_text from fake vision")
	}
	if len(resp.CodeFiles) != 1 || resp.CodeFiles[0] != "server/internal/service/chat_service.go" {
		t.Errorf("unexpected code_files: %v", resp.CodeFiles)
	}
	if resp.Summary != "回答不准确" {
		t.Errorf("unexpected summary: %v", resp.Summary)
	}
}

// TestParseAIRepairJSON 验证前后噪声容忍解析
func TestParseAIRepairJSON(t *testing.T) {
	raw := "some prefix\n{\"module\":\"语音\",\"summary\":\"m\",\"code_files\":[\"a/x.go\"],\"root_cause\":\"r\",\"repair_hint\":\"h\"}\nsuffix"
	out, err := parseAIRepairJSON(raw)
	if err != nil {
		t.Fatalf("parseAIRepairJSON error: %v", err)
	}
	if out.Module != "语音" || out.Summary != "m" {
		t.Errorf("unexpected parse result: %+v", out)
	}
	if len(out.CodeFiles) != 1 || out.CodeFiles[0] != "a/x.go" {
		t.Errorf("unexpected code_files: %v", out.CodeFiles)
	}
}