package agent

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/util"
)

// agentTimeout 单个子 Agent 的执行超时：防止慢 Agent 挂死拖垮整个对话链路。
const agentTimeout = 30 * time.Second

// Orchestrator 多智能体编排器
// 负责：意图路由 → 并行执行子 Agent（带超时与 panic 恢复）→ 结果汇聚
type Orchestrator struct {
	router *Router
	merger *ResultMerger
	agents map[string]Agent // 已注册的子 Agent
	tools  *ToolRegistry    // A3：校园场景工具注册表
	kbRepo *repository.KBRepo
}

// NewOrchestrator 创建编排器并注册默认子 Agent
// llmClient 用于需要 LLM 的 Agent（如 EmotionAgent）；可为 nil，对应 Agent 走兜底
func NewOrchestrator(kbRepo *repository.KBRepo, llmClient llm.ChatClient) *Orchestrator {
	o := &Orchestrator{
		router: NewRouter(),
		merger: NewMerger(),
		agents: make(map[string]Agent),
		tools:  NewToolRegistry(),
		kbRepo: kbRepo,
	}

	// 注册默认子 Agent（按 router.go intentToAgent 中的名称对齐）
	o.Register(NewQAAgent())
	o.Register(NewPolicyAgent())
	o.Register(NewProcessAgent())
	o.Register(NewMajorAgent())
	o.Register(NewEmotionAgent(llmClient))

	// 注册校园场景工具（A3）：确定性数据查询
	o.tools.Register(NewProcessNodeTool(kbRepo))

	return o
}

// Tools 暴露工具注册表（供 Agent 与未来 function calling 使用）
func (o *Orchestrator) Tools() *ToolRegistry { return o.tools }

// Register 注册子 Agent（以 Key() 作为路由 key）
func (o *Orchestrator) Register(agent Agent) {
	o.agents[agent.Key()] = agent
}

// Execute 执行多智能体协同问答
// 1. 路由 → 2. 并行执行（单 Agent 超时 + panic 恢复）→ 3. 结果汇聚
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
// 稳定化（A3）：单个 Agent 失败/超时/panic 只降级为空结果，绝不影响其它 Agent 与主链路。
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
			// panic 恢复：Agent 是独立 goroutine，裸 panic 会击穿 recover 中间件直接崩掉整个服务
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[ERROR] Agent panic 已恢复 [name=%s]: %v", a.Name(), r)
					resultCh <- &AgentResult{AgentName: a.Name(), Content: "", Confidence: 0}
				}
			}()

			start := time.Now()
			// 单 Agent 超时：与主链路 ctx 解耦，慢 Agent 只损失自身结果
			agentCtx, cancel := context.WithTimeout(ctx, agentTimeout)
			defer cancel()

			result, err := a.Execute(agentCtx, question, userCtx, o.kbRepo)
			if err != nil {
				log.Printf("Agent 执行失败 [name=%s] 耗时=%s: %v", a.Name(), time.Since(start).Round(time.Millisecond), err)
				resultCh <- &AgentResult{
					AgentName:  a.Name(),
					Content:    "",
					Confidence: 0,
				}
				return
			}
			log.Printf("Agent 完成 [name=%s] 耗时=%s confidence=%.2f", a.Name(), time.Since(start).Round(time.Millisecond), result.Confidence)
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
