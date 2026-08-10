package model

// AIBriefing AI 简讯（首页资讯）
type AIBriefing struct {
	ID          int64  `json:"id" db:"id"`
	Source      string `json:"source" db:"source"`
	Category    string `json:"category" db:"category"`
	Topic       string `json:"topic" db:"topic"`
	Summary     string `json:"summary" db:"summary"`
	Content     string `json:"content" db:"content"`
	Link        string `json:"link" db:"link"`
	Keyword     string `json:"keyword" db:"keyword"`
	PublishedAt string `json:"published_at" db:"published_at"`
	FetchedAt   string `json:"fetched_at" db:"fetched_at"`
	Status      int    `json:"status" db:"status"`
	CreatedBy   int64  `json:"created_by" db:"created_by"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}

// AIBriefingSource AI 简讯来源配置
type AIBriefingSource struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	URL          string `json:"url" db:"url"`
	Category     string `json:"category" db:"category"`
	Enabled      int    `json:"enabled" db:"enabled"`
	FetchEnabled int    `json:"fetch_enabled" db:"fetch_enabled"`
	FetchTime    string `json:"fetch_time" db:"fetch_time"`
	LastFetchAt  string `json:"last_fetch_at" db:"last_fetch_at"`
	CreatedAt    string `json:"created_at" db:"created_at"`
	UpdatedAt    string `json:"updated_at" db:"updated_at"`
}

// AIBriefingStats AI 简讯汇总统计
type AIBriefingStats struct {
	Total       int            `json:"total"`
	Published   int            `json:"published"`
	Draft       int            `json:"draft"`
	AutoFetched int            `json:"auto_fetched"`
	Manual      int            `json:"manual"`
	ByCategory  map[string]int `json:"by_category"`
	BySource    map[string]int `json:"by_source"`
}
