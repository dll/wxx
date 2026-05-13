package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// QAAgent 通用问答子智能体
// 基于 FTS/BM25 检索 + LLM 生成回答，覆盖面最广
type QAAgent struct {
	searchTopK int
}

// NewQAAgent 创建通用问答 Agent
func NewQAAgent() *QAAgent {
	return &QAAgent{searchTopK: 5}
}

func (a *QAAgent) Name() string { return "通用问答" }

func (a *QAAgent) Execute(ctx context.Context, question string, userCtx *model.UserContext, kbRepo *repository.KBRepo) (*AgentResult, error) {
	results, err := kbRepo.Search(question, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, a.searchTopK)
	if err != nil {
		return &AgentResult{
			AgentName:  a.Name(),
			Content:    "",
			Confidence: 0,
		}, nil
	}

	if len(results) == 0 {
		return &AgentResult{
			AgentName:  a.Name(),
			Content:    "",
			Confidence: 0.1,
		}, nil
	}

	// 拼接检索结果供 LLM 回答
	var parts []string
	for i, r := range results {
		parts = append(parts, fmt.Sprintf("资料%d：%s\n%s", i+1, r.Resource.Title, truncate(r.Resource.Content, 800)))
	}
	content := fmt.Sprintf("基于以下资料回答用户问题「%s」：\n\n%s", question, strings.Join(parts, "\n\n"))

	// 构造 sources
	sources := kbResultsToSources(results)

	return &AgentResult{
		AgentName:  a.Name(),
		Content:    content,
		Sources:    sources,
		Confidence: 0.75,
	}, nil
}

// kbResultsToSources 将检索结果转为 Source 列表
func kbResultsToSources(results []*repository.SearchResult) []model.Source {
	sources := make([]model.Source, 0, len(results))
	for _, r := range results {
		sources = append(sources, model.Source{
			ResourceID:     r.Resource.ResourceID,
			Title:          r.Resource.Title,
			Version:        r.Resource.Version,
			SourceLink:     r.Resource.SourceLink,
			RelevanceScore: -r.Score,
		})
	}
	return sources
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
