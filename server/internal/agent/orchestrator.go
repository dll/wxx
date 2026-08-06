package agent

import (
	"context"
	"log"
	"sync"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/util"
)

// Orchestrator 多智能体编排器
// 负责：意图路由 → 并行执行子 Agent → 结果汇聚
type Orchestrator struct {
	router *Router
	merger *ResultMerger
	agents map[string]Agent // 已注册的子 Agent
	kbRepo *repository.KBRepo
}

// NewOrchestrator 创建编排器并注册默认子 Agent
// llmClient 用于需要 LLM 的 Agent（如 EmotionAgent）；可为 nil，对应 Agent 走兜底
func NewOrchestrator(kbRepo *repository.KBRepo, llmClient llm.ChatClient) *Orchestrator {
	o := &Orchestrator{
		router: NewRouter(),
		merger: NewMerger(),
		agents: make(map[string]Agent),
		kbRepo: kbRepo,
	}

	// 注册默认子 Agent（按 router.go intentToAgent 中的名称对齐）
	o.Register(NewQAAgent())
	o.Register(NewPolicyAgent())
	o.Register(NewProcessAgent())
	o.Register(NewMajorAgent())
	o.Register(NewEmotionAgent(llmClient))

	return o
}

// Register 注册子 Agent（以 Key() 作为路由 key）
func (o *Orchestrator) Register(agent Agent) {
	o.agents[agent.Key()] = agent
}

// Execute 执行多智能体协同问答
// 1. 路由 → 2. 并行执行 → 3. 结果汇聚
func (o *Orchestrator) Execute(ctx context.Context, question string, userCtx *model.UserContext) (*MergedResult, error) {
	// 1. 意图路由
	agentNames := o.router.Route(question)
	log.Printf("多智能体路由 [question=%s] agents=%v", util.TruncateString(question, 50), agentNames)

	// 2. 并行执行子 Agent
	results := o.executeParallel(ctx, question, userCtx, agentNames)

	// 3. 结果汇聚
	merged := o.merger.Merge(results)
	log.Printf("多智能体汇聚 agents=%d confidence=%.2f sources=%d",
		merged.AgentCount, merged.Confidence, len(merged.Sources))

	return merged, nil
}

// executeParallel 并行执行多个子 Agent
func (o *Orchestrator) executeParallel(ctx context.Context, question string, userCtx *model.UserContext, agentNames []string) []*AgentResult {
	var wg sync.WaitGroup
	resultCh := make(chan *AgentResult, len(agentNames))

	for _, name := range agentNames {
		agent, ok := o.agents[name]
		if !ok {
			log.Printf("未知 Agent [name=%s]，跳过", name)
			continue
		}

		wg.Add(1)
		go func(a Agent) {
			defer wg.Done()
			result, err := a.Execute(ctx, question, userCtx, o.kbRepo)
			if err != nil {
				log.Printf("Agent 执行失败 [name=%s]: %v", a.Name(), err)
				resultCh <- &AgentResult{
					AgentName:  a.Name(),
					Content:    "",
					Confidence: 0,
				}
				return
			}
			resultCh <- result
		}(agent)
	}

	// 等待全部完成
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var results []*AgentResult
	for r := range resultCh {
		results = append(results, r)
	}
	return results
}
