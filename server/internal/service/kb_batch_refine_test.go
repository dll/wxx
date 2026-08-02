package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// ═══ BatchRefine ═══

func TestKBService_BatchRefine_Success(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	mock := llm.NewMockClient("mock")
	mock.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"title":"2026年度国家奖学金评选办法（精修）","summary":"国家奖学金用于奖励特别优秀的学生，每人每年8000元，申请需符合成绩排名与德育要求。","keywords":["国家奖学金","评选","8000元"]}`,
		}, nil
	}
	docSvc := newRefineDocSvc(t, mock)
	svc.SetRefiner(docSvc)

	result := svc.BatchRefine(context.Background(), []string{"policy-scholarship-2026"}, "admin")
	if result.Total != 1 || result.Success != 1 || result.Failed != 0 {
		t.Fatalf("汇总不符: %+v", result)
	}
	item := result.Results[0]
	if !item.OK || !item.Refined {
		t.Fatalf("单条应精修成功: %+v", item)
	}
	if !strings.Contains(item.Title, "精修") {
		t.Errorf("精修标题不符: %q", item.Title)
	}
	if !strings.Contains(item.Tags, "国家奖学金") {
		t.Errorf("精修标签不符: %q", item.Tags)
	}

	// 写库生效：回查标题/摘要/标签已更新
	got, err := svc.Get(context.Background(), "policy-scholarship-2026")
	if err != nil {
		t.Fatalf("回查失败: %v", err)
	}
	if !strings.Contains(got.Title, "精修") {
		t.Errorf("数据库标题未更新: %q", got.Title)
	}
	if !strings.Contains(got.Tags, "国家奖学金") {
		t.Errorf("数据库标签未更新: %q", got.Tags)
	}
}

func TestKBService_BatchRefine_NoRefiner(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	result := svc.BatchRefine(context.Background(), []string{"policy-scholarship-2026"}, "admin")
	if result.Success != 0 || result.Failed != 1 {
		t.Fatalf("未注入精修器时应全部失败: %+v", result)
	}
	if !strings.Contains(result.Results[0].Message, "未启用") {
		t.Errorf("失败原因不符: %q", result.Results[0].Message)
	}
}

func TestKBService_BatchRefine_ResourceNotFound(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)
	svc.SetRefiner(newRefineDocSvc(t, llm.NewMockClient("mock")))

	result := svc.BatchRefine(context.Background(), []string{"no-such-resource"}, "admin")
	if result.Success != 0 || result.Failed != 1 {
		t.Fatalf("不存在的资源应失败: %+v", result)
	}
	if !strings.Contains(result.Results[0].Message, "不存在") {
		t.Errorf("失败原因不符: %q", result.Results[0].Message)
	}
}

func TestKBService_BatchRefine_FallbackKeepsOriginal(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	mock := llm.NewMockClient("mock")
	mock.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return nil, errors.New("模型不可用")
	}
	svc.SetRefiner(newRefineDocSvc(t, mock))

	result := svc.BatchRefine(context.Background(), []string{"policy-scholarship-2026"}, "admin")
	if result.Success != 0 || result.Failed != 1 {
		t.Fatalf("LLM 失败应回退: %+v", result)
	}
	if !result.Results[0].Fallback {
		t.Errorf("应标记 fallback: %+v", result.Results[0])
	}

	// 数据库不得被改动
	got, err := svc.Get(context.Background(), "policy-scholarship-2026")
	if err != nil {
		t.Fatalf("回查失败: %v", err)
	}
	if got.Title != "2026年度国家奖学金评选办法" {
		t.Errorf("数据库标题不应变更: %q", got.Title)
	}
}

func TestKBService_BatchRefine_Partial(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	mock := llm.NewMockClient("mock")
	mock.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"title":"精修标题","summary":"这是一段足够长的摘要内容用于通过校验，描述奖学金评选的整体要求与流程。","keywords":["奖学金"]}`,
		}, nil
	}
	svc.SetRefiner(newRefineDocSvc(t, mock))

	result := svc.BatchRefine(context.Background(),
		[]string{"policy-scholarship-2026", "no-such-resource"}, "admin")
	if result.Total != 2 || result.Success != 1 || result.Failed != 1 {
		t.Fatalf("部分失败汇总不符: %+v", result)
	}
}
