package service

import (
	"context"
	"errors"
	"testing"

	"github.com/dll/wxx/server/internal/llm"
)

type teacherServiceTestClient struct{ content string }

func (c teacherServiceTestClient) Chat(context.Context, *llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: c.content}, nil
}
func (teacherServiceTestClient) Stream(context.Context, *llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	return nil, errors.New("stream not used")
}
func (teacherServiceTestClient) Name() string  { return "teacher-test" }
func (teacherServiceTestClient) Model() string { return "teacher-test" }

func TestTeacherServiceFallbackContracts(t *testing.T) {
	svc := NewTeacherService(nil)
	plan, err := svc.GenerateLessonPlan(context.Background(), "", "")
	if err != nil || plan.DataSource != "fallback" || plan.Topic == "" || len(plan.KeyPoints) == 0 {
		t.Fatalf("教案兜底契约失败: err=%v plan=%#v", err, plan)
	}
	paper, err := svc.GenerateExam(context.Background(), "")
	if err != nil || paper.DataSource != "fallback" || paper.TotalScore != 100 || len(paper.SampleQuestions) == 0 {
		t.Fatalf("试卷兜底契约失败: err=%v paper=%#v", err, paper)
	}
	grading, err := svc.GradeAssignments(context.Background(), "数据结构")
	if err != nil || grading.DataSource != "fallback" || grading.Graded == 0 || len(grading.CommonIssues) == 0 {
		t.Fatalf("批改兜底契约失败: err=%v grading=%#v", err, grading)
	}
}

func TestTeacherServiceLLMContracts(t *testing.T) {
	client := teacherServiceTestClient{content: `{"outline":"自定义大纲","key_points":["重点"],"difficulties":["难点"],"strategies":["策略"],"interactions":["互动"],"homework":["作业"]}`}
	plan, err := NewTeacherService(client).GenerateLessonPlan(context.Background(), "编译原理", "CS101")
	if err != nil || plan.DataSource != "ai" || plan.Outline != "自定义大纲" {
		t.Fatalf("LLM 教案契约失败: err=%v plan=%#v", err, plan)
	}

	examClient := teacherServiceTestClient{content: `{"title":"测试卷","total_score":100,"duration":90,"sections":[{"type":"选择题"}],"sample_questions":[{"question":"Q","answer":"A"}]}`}
	paper, err := NewTeacherService(examClient).GenerateExam(context.Background(), "操作系统")
	if err != nil || paper.DataSource != "ai" || paper.Title != "测试卷" || paper.Duration != 90 {
		t.Fatalf("LLM 试卷契约失败: err=%v paper=%#v", err, paper)
	}
}
