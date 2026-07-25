// Package context_engine 知识检索管道（Context Engine）。
// 核心流程：意图分类 → 结构化查询 → FTS/BM25 检索 → 来源加权重排 → 上下文拼装 → 来源附加。
// 暴露统一的 Query 接口供 service 层调用，handler 禁止直接使用。
package context_engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/logger"
)

// SearchResult 检索结果（与 repository.SearchResult 对齐）
type SearchResult struct {
	ResourceID   string  `json:"resource_id"`
	Title        string  `json:"title"`
	Summary      string  `json:"summary"`
	Content      string  `json:"content"`
	ResourceType string  `json:"resource_type"` // Policy/Process/FAQ/Activity
	OwnerScope   string  `json:"owner_scope"`
	Score        float64 `json:"score"`       // BM25 原始分
	TrustScore   float64 `json:"trust_score"` // 加权后置信度（CE-09）
	Snippet      string  `json:"snippet"`     // 命中片段（CE-07）
	EffectiveAt  string  `json:"effective_at"`
	ExpiredAt    string  `json:"expired_at"`
}

// QueryRequest 检索请求
type QueryRequest struct {
	Question   string
	UserID     string
	Role       string
	OwnerScope string
	OwnerID    string
	SessionID  string
	TopK       int
}

// QueryResult 检索结果集
type QueryResult struct {
	Results    []*SearchResult
	Intent     Intent
	ContextStr string // 拼装好的上下文文本，直接喂给 LLM
}

// KBSearcher 知识库检索接口（由 repository 实现）
type KBSearcher interface {
	Search(question, ownerScope, ownerID, role string, limit int) ([]KBSearchItem, error)
}

// KBSearchItem 知识库搜索条目（适配 repository 返回）
type KBSearchItem struct {
	ResourceID   string
	Title        string
	Summary      string
	Content      string
	ResourceType string
	OwnerScope   string
	Score        float64
	EffectiveAt  string
	ExpiredAt    string
}

// HistoryProvider 历史消息提供接口（CE-10）
type HistoryProvider interface {
	GetRecentMessages(sessionID string, limit int) ([]HistoryMessage, error)
}

// HistoryMessage 历史消息
type HistoryMessage struct {
	Role    string
	Content string
}

// Engine 检索管道核心
type Engine struct {
	searcher KBSearcher
	history  HistoryProvider
}

// New 创建 Context Engine 实例
func New(searcher KBSearcher, history HistoryProvider) *Engine {
	return &Engine{
		searcher: searcher,
		history:  history,
	}
}

// Query 执行完整检索管道：意图分类 → 检索 → 加权重排 → 过滤过期 → 拼装上下文
func (e *Engine) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	if req.TopK <= 0 {
		req.TopK = 5
	}

	// ── 1. 意图分类（CE-06：固定优先级 + 置信度） ──
	intent := ClassifyIntent(req.Question)
	log := logger.WithContext(ctx)
	log.Info("检索管道启动", "intent", intent.Category, "confidence", intent.Confidence, "question", req.Question)

	// ── 2. FTS/BM25 检索 ──
	if e.searcher == nil {
		return &QueryResult{Intent: intent}, nil
	}

	items, err := e.searcher.Search(req.Question, req.OwnerScope, req.OwnerID, req.Role, req.TopK*2)
	if err != nil {
		return nil, fmt.Errorf("知识检索失败: %w", err)
	}

	// ── 3. 来源可信度加权（CE-09） ──
	results := make([]*SearchResult, 0, len(items))
	now := time.Now()
	for _, item := range items {
		// 过滤过期资源（CE-09）
		if item.ExpiredAt != "" {
			if t, err := time.Parse("2006-01-02", item.ExpiredAt); err == nil && t.Before(now) {
				continue
			}
		}

		sr := &SearchResult{
			ResourceID:   item.ResourceID,
			Title:        item.Title,
			Summary:      item.Summary,
			Content:      item.Content,
			ResourceType: item.ResourceType,
			OwnerScope:   item.OwnerScope,
			Score:        item.Score,
			EffectiveAt:  item.EffectiveAt,
			ExpiredAt:    item.ExpiredAt,
			Snippet:      extractSnippet(item.Content, req.Question),
		}
		sr.TrustScore = computeTrustScore(sr)
		results = append(results, sr)
	}

	// 按 TrustScore 排序（降序）
	sortByTrust(results)

	// 截取 TopK
	if len(results) > req.TopK {
		results = results[:req.TopK]
	}

	// ── 4. 上下文拼装（CE-10：智能历史选取） ──
	contextStr := e.buildContext(req, results, intent)

	return &QueryResult{
		Results:    results,
		Intent:     intent,
		ContextStr: contextStr,
	}, nil
}

// buildContext 拼装 LLM 上下文（知识 + 相关历史）
func (e *Engine) buildContext(req *QueryRequest, results []*SearchResult, intent Intent) string {
	var sb strings.Builder

	// 知识上下文
	if len(results) > 0 {
		sb.WriteString("【参考资料】\n")
		for i, r := range results {
			sb.WriteString(fmt.Sprintf("[%d] %s（%s", i+1, r.Title, r.ResourceType))
			if r.EffectiveAt != "" {
				sb.WriteString(fmt.Sprintf("，生效日期: %s", r.EffectiveAt))
			}
			sb.WriteString("）\n")
			if r.Snippet != "" {
				sb.WriteString(r.Snippet + "\n")
			} else if r.Summary != "" {
				sb.WriteString(r.Summary + "\n")
			}
			sb.WriteString("\n")
		}
	}

	// CE-10: 历史消息选取（相关性优先，不固定 6 条）
	if e.history != nil && req.SessionID != "" {
		msgs, err := e.history.GetRecentMessages(req.SessionID, 10)
		if err == nil && len(msgs) > 0 {
			relevant := selectRelevantHistory(msgs, req.Question, 4)
			if len(relevant) > 0 {
				sb.WriteString("【对话历史（相关片段）】\n")
				for _, m := range relevant {
					sb.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
				}
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}
