package service

import (
	"context"
	"strconv"

	"github.com/dll/wxx/server/internal/context_engine"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// contextEngineSearcher 将生产知识库端口适配为 Context Engine 的检索端口。
// 适配层只负责类型转换；权限过滤仍由 KBRepo 在 SQL 层执行。
type contextEngineSearcher struct {
	repo interface {
		Search(string, string, string, string, int) ([]*repository.SearchResult, error)
		SearchStructured(string, string, string, string, int) ([]*repository.SearchResult, error)
	}
}

func (a contextEngineSearcher) Search(question, ownerScope, ownerID, role string, limit int) ([]context_engine.KBSearchItem, error) {
	items, err := a.repo.Search(question, ownerScope, ownerID, role, limit)
	return toContextEngineItems(items), err
}

func (a contextEngineSearcher) SearchStructured(question, ownerScope, ownerID, role string, limit int) ([]context_engine.KBSearchItem, error) {
	items, err := a.repo.SearchStructured(question, ownerScope, ownerID, role, limit)
	return toContextEngineItems(items), err
}

func toContextEngineItems(items []*repository.SearchResult) []context_engine.KBSearchItem {
	result := make([]context_engine.KBSearchItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, context_engine.KBSearchItem{
			ResourceID: item.Resource.ResourceID, Title: item.Resource.Title,
			Summary: item.Resource.Summary, Content: item.Resource.Content,
			ResourceType: item.Resource.ResourceType, OwnerScope: item.Resource.OwnerScope,
			Score: item.Score, EffectiveAt: valueOrEmpty(item.Resource.EffectiveAt),
			ExpiredAt: valueOrEmpty(item.Resource.ExpiredAt), IsStructured: item.IsStructured,
		})
	}
	return result
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type contextEngineHistory struct {
	repo interface {
		GetRecentContext(string, int) ([]*model.Message, error)
	}
}

func (h contextEngineHistory) GetRecentMessages(sessionID string, limit int) ([]context_engine.HistoryMessage, error) {
	items, err := h.repo.GetRecentContext(sessionID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]context_engine.HistoryMessage, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, context_engine.HistoryMessage{Role: item.Role, Content: item.Content})
		}
	}
	return result, nil
}

func newProductionContextEngine(kbRepo interface {
	Search(string, string, string, string, int) ([]*repository.SearchResult, error)
	SearchStructured(string, string, string, string, int) ([]*repository.SearchResult, error)
}, messageRepo interface {
	GetRecentContext(string, int) ([]*model.Message, error)
}) *context_engine.Engine {
	return context_engine.New(contextEngineSearcher{repo: kbRepo}, contextEngineHistory{repo: messageRepo})
}

func (s *ChatService) retrieveWithContextEngine(ctx context.Context, userCtx *model.UserContext, question string) []*repository.SearchResult {
	if s.contextEngine == nil {
		return s.retrieveWithRelevance(userCtx, question, "")
	}
	result, err := s.contextEngine.Query(ctx, &context_engine.QueryRequest{
		Question: question, UserID: strconv.FormatInt(userCtx.UserID, 10), Role: userCtx.Role,
		OwnerScope: userCtx.OwnerScope, OwnerID: userCtx.OwnerID, TopK: 5,
	})
	if err != nil {
		return s.retrieveWithRelevance(userCtx, question, "")
	}
	items := make([]*repository.SearchResult, 0, len(result.Results))
	for _, item := range result.Results {
		if item == nil {
			continue
		}
		items = append(items, &repository.SearchResult{Resource: model.KBResource{
			ResourceID: item.ResourceID, Title: item.Title, Summary: item.Summary,
			Content: item.Content, ResourceType: item.ResourceType, OwnerScope: item.OwnerScope,
		}, Score: item.Score, IsStructured: item.IsStructured})
	}
	return items
}
