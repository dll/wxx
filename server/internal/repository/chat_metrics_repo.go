package repository

import (
	"database/sql"
	"time"
)

// ChatMetricsRepo 问答质量指标数据访问
type ChatMetricsRepo struct {
	db *sql.DB
}

// NewChatMetricsRepo 创建问答质量指标 repo
func NewChatMetricsRepo(db *sql.DB) *ChatMetricsRepo {
	return &ChatMetricsRepo{db: db}
}

// ChatMetric 单条指标记录
type ChatMetric struct {
	ID           int64
	SessionID    string
	UserID       int64
	Question     string
	Intent       string
	Confidence   float64
	Fallback     bool
	SourcesCount int
	DurationMs   int64
	TraceID      string
	CreatedAt    string
}

// Insert 写入一条问答质量指标
func (r *ChatMetricsRepo) Insert(m *ChatMetric) error {
	fallbackInt := 0
	if m.Fallback {
		fallbackInt = 1
	}
	_, err := r.db.Exec(
		`INSERT INTO chat_metrics (session_id, user_id, question, intent, confidence, fallback, sources_count, duration_ms, trace_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.SessionID, m.UserID, m.Question, m.Intent, m.Confidence, fallbackInt, m.SourcesCount, m.DurationMs, m.TraceID,
	)
	return err
}

// AggregatedMetrics 聚合指标结果
type AggregatedMetrics struct {
	TotalQuestions int64
	AvgConfidence  float64
	FallbackCount  int64
	FallbackRate   float64
	SourceHitCount int64
	SourceHitRate  float64
	AvgDurationMs  int64
	P95DurationMs  int64
}

// Aggregate 聚合最近 sinceDays 天的质量指标
func (r *ChatMetricsRepo) Aggregate(sinceDays int) (*AggregatedMetrics, error) {
	if sinceDays <= 0 {
		sinceDays = 7
	}
	since := time.Now().AddDate(0, 0, -sinceDays).Format("2006-01-02")

	m := &AggregatedMetrics{}

	// 总数 + 平均置信度 + 兜底数 + 来源命中数 + 平均耗时
	err := r.db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(AVG(confidence), 0),
			COALESCE(SUM(fallback), 0),
			COALESCE(SUM(CASE WHEN sources_count >= 1 THEN 1 ELSE 0 END), 0),
			COALESCE(AVG(duration_ms), 0)
		FROM chat_metrics
		WHERE created_at >= ?
	`, since).Scan(&m.TotalQuestions, &m.AvgConfidence, &m.FallbackCount, &m.SourceHitCount, &m.AvgDurationMs)
	if err != nil {
		return nil, err
	}

	if m.TotalQuestions > 0 {
		m.FallbackRate = float64(m.FallbackCount) / float64(m.TotalQuestions)
		m.SourceHitRate = float64(m.SourceHitCount) / float64(m.TotalQuestions)
	}

	// P95 延迟（近似：取排名第 95% 的值）
	err = r.db.QueryRow(`
		SELECT COALESCE(duration_ms, 0) FROM chat_metrics
		WHERE created_at >= ?
		ORDER BY duration_ms ASC
		LIMIT 1 OFFSET (SELECT CAST(COUNT(*) * 0.95 AS INTEGER) FROM chat_metrics WHERE created_at >= ?)
	`, since, since).Scan(&m.P95DurationMs)
	if err == sql.ErrNoRows {
		m.P95DurationMs = 0
	} else if err != nil {
		// P95 计算失败不致命，用平均值兜底
		m.P95DurationMs = m.AvgDurationMs
	}

	return m, nil
}

// IntentDistribution 按意图分类统计
type IntentCount struct {
	Intent string
	Count  int
}

// TopFallbackQuestion 高频兜底问题
type TopFallbackQuestion struct {
	Question string
	Count    int
}

// TopFallbackQuestions 统计最近 sinceDays 天高频兜底问题（用于知识治理：
// 命中失败率高的问题应补录进知识库）。按出现次数降序，取 topN。
func (r *ChatMetricsRepo) TopFallbackQuestions(sinceDays, topN int) ([]TopFallbackQuestion, error) {
	if sinceDays <= 0 {
		sinceDays = 7
	}
	if topN <= 0 {
		topN = 20
	}
	since := time.Now().AddDate(0, 0, -sinceDays).Format("2006-01-02")

	rows, err := r.db.Query(`
		SELECT TRIM(question) AS q, COUNT(*) AS cnt FROM chat_metrics
		WHERE created_at >= ? AND fallback = 1 AND TRIM(question) != ''
		GROUP BY q ORDER BY cnt DESC, q ASC LIMIT ?
	`, since, topN)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TopFallbackQuestion
	for rows.Next() {
		var tf TopFallbackQuestion
		if err := rows.Scan(&tf.Question, &tf.Count); err != nil {
			return nil, err
		}
		result = append(result, tf)
	}
	return result, rows.Err()
}

// CountByIntent 统计最近 sinceDays 天各意图的调用次数
func (r *ChatMetricsRepo) CountByIntent(sinceDays int) ([]IntentCount, error) {
	if sinceDays <= 0 {
		sinceDays = 7
	}
	since := time.Now().AddDate(0, 0, -sinceDays).Format("2006-01-02")

	rows, err := r.db.Query(`
		SELECT intent, COUNT(*) AS cnt FROM chat_metrics
		WHERE created_at >= ? AND intent != ''
		GROUP BY intent ORDER BY cnt DESC
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []IntentCount
	for rows.Next() {
		var ic IntentCount
		if err := rows.Scan(&ic.Intent, &ic.Count); err != nil {
			return nil, err
		}
		result = append(result, ic)
	}
	return result, rows.Err()
}
