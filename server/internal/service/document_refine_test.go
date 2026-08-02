package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/llm"
)

func newRefineDocSvc(t *testing.T, mock *llm.MockClient) *DocumentService {
	t.Helper()
	svc := NewDocumentService(t.TempDir(), 10)
	svc.SetLLMClient(mock)
	return svc
}

// ═══ parseRefinedMetadata ═══

func TestParseRefinedMetadata_PlainJSON(t *testing.T) {
	raw := `{"title":"转专业办理办法","summary":"符合条件的学生可按学校通知申请转专业，需通过转出学院与转入学院审核。","keywords":["转专业","学籍异动","教务处"]}`
	r, err := parseRefinedMetadata(raw)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if r.Title != "转专业办理办法" {
		t.Errorf("标题不符: %q", r.Title)
	}
	if !strings.Contains(r.Summary, "转专业") {
		t.Errorf("摘要不符: %q", r.Summary)
	}
	if len(r.Keywords) != 3 {
		t.Errorf("关键词数量不符: %v", r.Keywords)
	}
	if !r.Refined {
		t.Error("应标记为精修结果")
	}
}

func TestParseRefinedMetadata_CodeFence(t *testing.T) {
	raw := "```json\n{\"title\":\"奖学金评选办法\",\"summary\":\"用于奖励特别优秀的学生，每人每年8000元。\",\"keywords\":[\"奖学金\",\"评选\"]}\n```"
	r, err := parseRefinedMetadata(raw)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if r.Title != "奖学金评选办法" {
		t.Errorf("标题不符: %q", r.Title)
	}
	if len(r.Keywords) != 2 {
		t.Errorf("关键词数量不符: %v", r.Keywords)
	}
}

func TestParseRefinedMetadata_SurroundingText(t *testing.T) {
	raw := "好的，以下是精修结果：\n{\"title\":\"缓考申请流程\",\"summary\":\"因病或因事无法参加考试的学生可申请缓考，需提前提交申请。\",\"keywords\":[\"缓考\",\"考试\"]}\n希望对你有帮助。"
	r, err := parseRefinedMetadata(raw)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if r.Title != "缓考申请流程" {
		t.Errorf("标题不符: %q", r.Title)
	}
}

func TestParseRefinedMetadata_NoJSON(t *testing.T) {
	if _, err := parseRefinedMetadata("很抱歉，我无法处理这个请求。"); err == nil {
		t.Error("无 JSON 内容应报错")
	}
}

func TestParseRefinedMetadata_KeywordCleanup(t *testing.T) {
	raw := `{"title":"学生证补办","summary":"学生证丢失后可通过线上系统申请补办，需缴纳工本费。","keywords":["学生证,"," 补办","学生证","","材料"]}`
	r, err := parseRefinedMetadata(raw)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(r.Keywords) != 3 {
		t.Errorf("关键词应去重去噪为 3 个，得到: %v", r.Keywords)
	}
}

// ═══ RefineMetadata 兜底逻辑 ═══

func TestRefineMetadata_NoLLM_Fallback(t *testing.T) {
	svc := NewDocumentService(t.TempDir(), 10) // 未注入 LLM
	r := svc.RefineMetadata(context.Background(), "原标题", "原摘要", []string{"a"}, "正文内容")
	if !r.Fallback {
		t.Error("无 LLM 时应回退")
	}
	if r.Refined {
		t.Error("无 LLM 时不应标记为精修")
	}
	if r.Title != "原标题" || len(r.Keywords) == 0 || r.Keywords[0] != "a" {
		t.Errorf("应原样返回兜底值: %+v", r)
	}
}

func TestRefineMetadata_EmptyContent_Fallback(t *testing.T) {
	mock := llm.NewMockClient("mock")
	svc := newRefineDocSvc(t, mock)
	r := svc.RefineMetadata(context.Background(), "原标题", "原摘要", []string{"a"}, "   ")
	if !r.Fallback {
		t.Error("空内容应回退")
	}
}

func TestRefineMetadata_LLMValid(t *testing.T) {
	mock := llm.NewMockClient("mock")
	mock.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"title":"国家奖学金评选办法","summary":"国家奖学金用于奖励特别优秀的学生，每人每年8000元，需成绩排名前10%。","keywords":["国家奖学金","评选","8000元"]}`,
		}, nil
	}
	svc := newRefineDocSvc(t, mock)

	r := svc.RefineMetadata(context.Background(), "旧标题", "旧摘要", []string{"旧"}, "国家奖学金评选办法正文……")
	if r.Fallback {
		t.Error("精修成功不应回退")
	}
	if !r.Refined {
		t.Error("应标记为精修结果")
	}
	if r.Title != "国家奖学金评选办法" {
		t.Errorf("标题不符: %q", r.Title)
	}
	if len(r.Keywords) != 3 {
		t.Errorf("关键词不符: %v", r.Keywords)
	}
}

func TestRefineMetadata_LLMError_Fallback(t *testing.T) {
	mock := llm.NewMockClient("mock")
	mock.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return nil, errors.New("模型不可用")
	}
	svc := newRefineDocSvc(t, mock)

	r := svc.RefineMetadata(context.Background(), "原标题", "原摘要", []string{"a"}, "正文内容……")
	if !r.Fallback {
		t.Error("LLM 报错应回退")
	}
	if r.Title != "原标题" {
		t.Errorf("应保留原标题: %q", r.Title)
	}
}

func TestRefineMetadata_LLMGarbage_Fallback(t *testing.T) {
	mock := llm.NewMockClient("mock")
	mock.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{Content: "这看起来像是一段文本，但并不是 JSON。"}, nil
	}
	svc := newRefineDocSvc(t, mock)

	r := svc.RefineMetadata(context.Background(), "原标题", "原摘要", []string{"a"}, "正文内容……")
	if !r.Fallback {
		t.Error("LLM 输出非法应回退")
	}
}

func TestRefineMetadata_LLMEmptyFields_FillFromOriginal(t *testing.T) {
	mock := llm.NewMockClient("mock")
	mock.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{Content: `{"title":"","summary":"模型给出的摘要内容足够长以通过校验。","keywords":[]}`}, nil
	}
	svc := newRefineDocSvc(t, mock)

	r := svc.RefineMetadata(context.Background(), "原标题", "原摘要", []string{"原词"}, "正文内容……")
	if r.Title != "原标题" {
		t.Errorf("空标题应回填原标题: %q", r.Title)
	}
	if len(r.Keywords) == 0 || r.Keywords[0] != "原词" {
		t.Errorf("空关键词应回填原关键词: %v", r.Keywords)
	}
	if r.Fallback {
		t.Error("补齐后可接受，不应回退")
	}
}

// ═══ truncateDocForRefine ═══

func TestTruncateDocForRefine(t *testing.T) {
	content := strings.Repeat("甲", 10000)
	out := truncateDocForRefine(content, 6000)
	if len([]rune(out)) >= 10000 {
		t.Errorf("应被截断，长度 %d", len([]rune(out)))
	}
	if !strings.Contains(out, "省略") {
		t.Error("应包含省略标记")
	}
	// 头部与尾部内容都应保留
	if !strings.HasPrefix(out, "甲") || !strings.HasSuffix(out, "甲") {
		t.Error("头尾应保留")
	}

	short := strings.Repeat("乙", 100)
	if out := truncateDocForRefine(short, 6000); out != short {
		t.Error("短内容不应截断")
	}
}

func TestValidateRefinedMetadata(t *testing.T) {
	good := &DocumentRefineResult{Title: "转专业办理办法", Summary: "符合条件的学生可按学校通知申请转专业，需通过转出学院与转入学院审核。", Keywords: []string{"转专业"}}
	if !validateRefinedMetadata(good) {
		t.Error("合法结果应通过校验")
	}

	badCases := []*DocumentRefineResult{
		{Title: "", Summary: "足够长的摘要内容。", Keywords: []string{"a"}},
		{Title: "短", Summary: "足够长的摘要内容。", Keywords: []string{"a"}},
		{Title: "标题", Summary: "短摘要", Keywords: []string{"a"}},
		{Title: "标题", Summary: "足够长的摘要内容。", Keywords: []string{}},
	}
	for _, c := range badCases {
		if validateRefinedMetadata(c) {
			t.Errorf("非法结果不应通过校验: %+v", c)
		}
	}
}
