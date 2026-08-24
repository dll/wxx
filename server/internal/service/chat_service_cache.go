package service

import (
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"time"

	"github.com/dll/wxx/server/internal/model"
)

// answerCache 问答结果缓存（用于入学/离校等固定流程问题，避免重复调用 LLM）
// 缓存 24 小时，后台每 30 分钟清理过期条目
var (
	answerCache   = make(map[string]*answerCacheEntry)
	answerCacheMu sync.RWMutex
)

const answerCacheTTL = 24 * time.Hour

func init() {
	go func() {
		for {
			time.Sleep(30 * time.Minute)
			answerCacheMu.Lock()
			now := time.Now()
			for k, v := range answerCache {
				if now.Sub(v.CachedAt) > answerCacheTTL {
					delete(answerCache, k)
				}
			}
			answerCacheMu.Unlock()
		}
	}()
}

type answerCacheEntry struct {
	Card      *model.AnswerCard
	SessionID string
	CachedAt  time.Time
}

// cacheKeyForQuestion 为问题生成缓存键（去空格 + 小写后取 FNV-1a 64-bit 哈希）
func cacheKeyForQuestion(q string) string {
	normalized := strings.ToLower(strings.TrimSpace(q))
	h := fnv.New64a()
	h.Write([]byte(normalized))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func isProcessCacheableQuestion(question string) bool {
	q := strings.TrimSpace(question)
	if q == "" {
		return false
	}
	keywords := []string{"入学", "迎新", "报到", "离校", "毕业", "转专业", "助学贷款", "流程"}
	for _, keyword := range keywords {
		if strings.Contains(q, keyword) {
			return true
		}
	}
	return false
}

// cacheGet 从缓存读取（仅限无 agentID 且无 sessionID 的固定流程问题）
func (s *ChatService) cacheGet(question string) *model.AnswerCard {
	cacheKey := cacheKeyForQuestion(question)
	answerCacheMu.RLock()
	entry, ok := answerCache[cacheKey]
	answerCacheMu.RUnlock()
	if !ok {
		return nil
	}
	if time.Since(entry.CachedAt) > answerCacheTTL {
		return nil
	}
	return entry.Card
}

// cacheSet 写入缓存（仅限无 agentID 的固定流程问题）
func (s *ChatService) cacheSet(question, sessionID string, card *model.AnswerCard) {
	if sessionID != "" {
		return
	}
	cacheKey := cacheKeyForQuestion(question)
	answerCacheMu.Lock()
	if _, exists := answerCache[cacheKey]; !exists {
		answerCache[cacheKey] = &answerCacheEntry{
			Card:      card,
			SessionID: sessionID,
			CachedAt:  time.Now(),
		}
	}
	answerCacheMu.Unlock()
}
