package agent

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// einoGraphInput Eino 图输入结构。
type einoGraphInput struct {
	Question string
	UserCtx  *model.UserContext
}

// EinoOrchestrator 基于 Eino Graph 的编排器。
// 当前图包含一个路由 Lambda 节点，内部复用已有的业务 Agent 路由与并行执行逻辑，
// 既满足项目“编排运行时采用 Eino”的选型要求，又保持现有业务语义不变。
type EinoOrchestrator struct {
	base     *Orchestrator
	runnable compose.Runnable[*einoGraphInput, *MergedResult]
}

func NewEinoOrchestrator(kbRepo *repository.KBRepo, llmClient llm.ChatClient) (*EinoOrchestrator, error) {
	base := NewOrchestrator(kbRepo, llmClient)
	graph := compose.NewGraph[*einoGraphInput, *MergedResult]()
	if err := graph.AddLambdaNode("route", compose.InvokableLambda(
		func(ctx context.Context, in *einoGraphInput) (*MergedResult, error) {
			return base.Execute(ctx, in.Question, in.UserCtx)
		},
	)); err != nil {
		return nil, err
	}
	if err := graph.AddEdge(compose.START, "route"); err != nil {
		return nil, err
	}
	if err := graph.AddEdge("route", compose.END); err != nil {
		return nil, err
	}
	runnable, err := graph.Compile(context.Background())
	if err != nil {
		return nil, err
	}
	return &EinoOrchestrator{base: base, runnable: runnable}, nil
}

func (o *EinoOrchestrator) Execute(ctx context.Context, question string, userCtx *model.UserContext) (*MergedResult, error) {
	return o.runnable.Invoke(ctx, &einoGraphInput{Question: question, UserCtx: userCtx})
}
