package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/llm"
)

type studentServiceTestClient struct {
	content string
	err     error
}

func (c studentServiceTestClient) Chat(context.Context, *llm.ChatRequest) (*llm.ChatResponse, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &llm.ChatResponse{Content: c.content}, nil
}

func (studentServiceTestClient) Stream(context.Context, *llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	return nil, errors.New("stream not used in student service contract test")
}

func (studentServiceTestClient) Name() string  { return "student-service-test" }
func (studentServiceTestClient) Model() string { return "test-model" }

func TestStudentServiceStudentContract_ParseGreetingAndMotto(t *testing.T) {
	greeting, motto := parseGreetingAndMotto("问候语: 早上好，同学\n激励语：今天也要稳步前进")
	if greeting != "早上好，同学" || motto != "今天也要稳步前进" {
		t.Fatalf("解析结果不符合契约: greeting=%q motto=%q", greeting, motto)
	}

	greeting, motto = parseGreetingAndMotto("无结构化输出")
	if greeting != "" || motto != "" {
		t.Fatalf("无标签输出应返回空值: greeting=%q motto=%q", greeting, motto)
	}
}

func TestStudentService_ParseDiaryJSON_QualityGateAndQuizFallback(t *testing.T) {
	diary, err := parseDiaryJSON("```json\n{\"courses_studied\":[\"数据结构\"],\"key_points\":[\"树遍历\"],\"study_minutes\":60,\"tomorrow_plan\":\"复习\"}\n```")
	if err != nil {
		t.Fatalf("代码块 JSON 应可解析: %v", err)
	}
	if len(diary.Quiz) != 1 || diary.Quiz[0]["correct_index"] != 0 {
		t.Fatalf("缺少 quiz 时应注入可用自测题: %#v", diary.Quiz)
	}

	if _, err := parseDiaryJSON(`{"courses_studied":["数据结构"]}`); err == nil {
		t.Fatal("缺少知识点和 quiz 时应触发质量门槛错误")
	}
}

func TestStudentService_GenerateLearningDiary_FallbackWithoutLLM(t *testing.T) {
	diary, err := (&StudentService{}).GenerateLearningDiary(context.Background(), 42)
	if err != nil {
		t.Fatalf("无依赖时应返回兜底而非错误: %v", err)
	}
	if diary.DataSource != "fallback" || diary.Date == "" || len(diary.KeyPoints) == 0 || len(diary.Quiz) == 0 {
		t.Fatalf("兜底日记字段不完整: %#v", diary)
	}
}

func TestStudentService_GenerateLearningDiary_LLMContract(t *testing.T) {
	svc := &StudentService{llmClient: studentServiceTestClient{content: `{"courses_studied":["编译原理"],"key_points":["词法分析"],"study_minutes":90,"quiz":[{"question":"Q","correct_index":1}],"tomorrow_plan":"复习","encouragement":"继续加油"}`}}
	diary, err := svc.GenerateLearningDiary(context.Background(), 7)
	if err != nil {
		t.Fatalf("LLM 成功路径不应返回错误: %v", err)
	}
	if diary.DataSource != "ai" || diary.CoursesStudied[0] != "编译原理" || diary.StudyMinutes != 90 {
		t.Fatalf("LLM 日记未按契约映射: %#v", diary)
	}
}

func TestStudentService_GenerateLearningDiary_LLMFailureFallsBack(t *testing.T) {
	svc := &StudentService{llmClient: studentServiceTestClient{err: errors.New("upstream unavailable")}}
	diary, err := svc.GenerateLearningDiary(context.Background(), 7)
	if err != nil || diary.DataSource != "fallback" {
		t.Fatalf("LLM 失败应透明兜底: err=%v source=%q", err, diary.DataSource)
	}
	if !strings.Contains(diary.Encouragement, "努力") {
		t.Fatalf("兜底鼓励语缺失: %q", diary.Encouragement)
	}
}
