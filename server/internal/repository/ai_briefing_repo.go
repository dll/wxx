package repository

import (
	"database/sql"
	"strings"

	"github.com/dll/wxx/server/internal/model"
)

// AIBriefingRepo AI 简讯数据访问
type AIBriefingRepo struct {
	db *sql.DB
}

// NewAIBriefingRepo 创建 AI 简讯 repo
func NewAIBriefingRepo(db *sql.DB) *AIBriefingRepo {
	return &AIBriefingRepo{db: db}
}

const aiBriefingCols = `id, source, category, topic, summary, content, link, keyword,
	published_at, fetched_at, status, created_by, created_at, updated_at`

// List 按条件分页查询资讯。statusFilter: "" 全部；category: "" 全部；q 关键词（topic/summary/keyword）
func (r *AIBriefingRepo) List(statusFilter, category, q string, page, pageSize int) ([]*model.AIBriefing, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	where := []string{"1=1"}
	args := []any{}
	if statusFilter != "" {
		where = append(where, "status = ?")
		args = append(args, statusFilter)
	}
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	if strings.TrimSpace(q) != "" {
		where = append(where, "(topic LIKE ? OR summary LIKE ? OR keyword LIKE ? OR source LIKE ?)")
		like := "%" + strings.TrimSpace(q) + "%"
		args = append(args, like, like, like, like)
	}
	cond := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM ai_briefings WHERE "+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(
		"SELECT "+aiBriefingCols+" FROM ai_briefings WHERE "+cond+
			" ORDER BY published_at DESC, id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list, err := scanBriefings(rows)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListUserVisible 用户端列表：仅上架(status=1)，按发布时间倒序
func (r *AIBriefingRepo) ListUserVisible(category, q string, limit int) ([]*model.AIBriefing, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	where := []string{"status = 1"}
	args := []any{}
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	if strings.TrimSpace(q) != "" {
		where = append(where, "(topic LIKE ? OR summary LIKE ? OR keyword LIKE ?)")
		like := "%" + strings.TrimSpace(q) + "%"
		args = append(args, like, like, like)
	}
	cond := strings.Join(where, " AND ")

	rows, err := r.db.Query(
		"SELECT "+aiBriefingCols+" FROM ai_briefings WHERE "+cond+
			" ORDER BY published_at DESC, id DESC LIMIT ?",
		append(args, limit)...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBriefings(rows)
}

// Get 单条查询
func (r *AIBriefingRepo) Get(id int64) (*model.AIBriefing, error) {
	var b model.AIBriefing
	var fetchedAt, publishedAt, source, category, summary, content, link, keyword sql.NullString
	var createdBy sql.NullInt64
	var status sql.NullInt64
	err := r.db.QueryRow(
		"SELECT "+aiBriefingCols+" FROM ai_briefings WHERE id = ?", id,
	).Scan(&b.ID, &source, &category, &b.Topic, &summary, &content,
		&link, &keyword, &publishedAt, &fetchedAt, &status,
		&createdBy, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	b.Source = source.String
	b.Category = category.String
	b.Summary = summary.String
	b.Content = content.String
	b.Link = link.String
	b.Keyword = keyword.String
	b.PublishedAt = publishedAt.String
	b.FetchedAt = fetchedAt.String
	b.Status = int(status.Int64)
	b.CreatedBy = createdBy.Int64
	return &b, nil
}

// GetByIDs 批量查询（导出）
func (r *AIBriefingRepo) GetByIDs(ids []int64) ([]*model.AIBriefing, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := r.db.Query(
		"SELECT "+aiBriefingCols+" FROM ai_briefings WHERE id IN ("+strings.Join(placeholders, ",")+
			") ORDER BY published_at DESC, id DESC", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBriefings(rows)
}

// Create 新增资讯（手动录入，created_by 记录录入人）
func (r *AIBriefingRepo) Create(b *model.AIBriefing) (int64, error) {
	res, err := r.db.Exec(`
		INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, fetched_at, status, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Source, b.Category, b.Topic, b.Summary, b.Content, b.Link, b.Keyword,
		b.PublishedAt, b.FetchedAt, b.Status, b.CreatedBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// CreateMany 批量写入（自动抓取）
func (r *AIBriefingRepo) CreateMany(items []*model.AIBriefing) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`
		INSERT INTO ai_briefings (source, category, topic, summary, content, link, keyword, published_at, fetched_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	written := 0
	for _, b := range items {
		if b.Topic == "" {
			continue
		}
		if _, err := stmt.Exec(b.Source, b.Category, b.Topic, b.Summary, b.Content, b.Link, b.Keyword,
			b.PublishedAt, b.FetchedAt); err != nil {
			continue
		}
		written++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return written, nil
}

// Update 更新资讯
func (r *AIBriefingRepo) Update(b *model.AIBriefing) error {
	_, err := r.db.Exec(`
		UPDATE ai_briefings SET source=?, category=?, topic=?, summary=?, content=?, link=?, keyword=?,
			published_at=?, status=?, updated_at=datetime('now') WHERE id=?`,
		b.Source, b.Category, b.Topic, b.Summary, b.Content, b.Link, b.Keyword,
		b.PublishedAt, b.Status, b.ID)
	return err
}

// UpdateStatus 上下架
func (r *AIBriefingRepo) UpdateStatus(id int64, status int) error {
	_, err := r.db.Exec("UPDATE ai_briefings SET status=?, updated_at=datetime('now') WHERE id=?", status, id)
	return err
}

// Delete 删除
func (r *AIBriefingRepo) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM ai_briefings WHERE id = ?", id)
	return err
}

// DeleteMany 批量删除（删除历史记录）
func (r *AIBriefingRepo) DeleteMany(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	res, err := r.db.Exec("DELETE FROM ai_briefings WHERE id IN ("+strings.Join(placeholders, ",")+")", args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ClearAll 清空全部资讯（删除历史记录）
func (r *AIBriefingRepo) ClearAll() (int64, error) {
	res, err := r.db.Exec("DELETE FROM ai_briefings")
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Stats 汇总统计
func (r *AIBriefingRepo) Stats() (*model.AIBriefingStats, error) {
	s := &model.AIBriefingStats{ByCategory: map[string]int{}, BySource: map[string]int{}}
	if err := r.db.QueryRow("SELECT COUNT(*) FROM ai_briefings").Scan(&s.Total); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow("SELECT COUNT(*) FROM ai_briefings WHERE status = 1").Scan(&s.Published); err != nil {
		return nil, err
	}
	s.Draft = s.Total - s.Published
	if err := r.db.QueryRow("SELECT COUNT(*) FROM ai_briefings WHERE fetched_at IS NOT NULL AND fetched_at != ''").Scan(&s.AutoFetched); err != nil {
		return nil, err
	}
	s.Manual = s.Total - s.AutoFetched

	rows, err := r.db.Query("SELECT category, COUNT(*) FROM ai_briefings GROUP BY category")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cat string
			var n int
			if rows.Scan(&cat, &n) == nil {
				s.ByCategory[cat] = n
			}
		}
	}
	rows2, err := r.db.Query("SELECT source, COUNT(*) FROM ai_briefings WHERE source != '' GROUP BY source")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var src string
			var n int
			if rows2.Scan(&src, &n) == nil {
				s.BySource[src] = n
			}
		}
	}
	return s, nil
}

// ── 来源配置 ──

const aiBriefingSourceCols = `id, name, url, category, enabled, fetch_enabled, fetch_time, last_fetch_at, created_at, updated_at`

// ListSources 列出全部来源
func (r *AIBriefingRepo) ListSources() ([]*model.AIBriefingSource, error) {
	rows, err := r.db.Query("SELECT " + aiBriefingSourceCols + " FROM ai_briefing_sources ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.AIBriefingSource
	for rows.Next() {
		src := &model.AIBriefingSource{}
		var lastFetch sql.NullString
		if err := rows.Scan(&src.ID, &src.Name, &src.URL, &src.Category, &src.Enabled,
			&src.FetchEnabled, &src.FetchTime, &lastFetch, &src.CreatedAt, &src.UpdatedAt); err != nil {
			return nil, err
		}
		if lastFetch.Valid {
			src.LastFetchAt = lastFetch.String
		}
		list = append(list, src)
	}
	return list, rows.Err()
}

// ListEnabledFetchSources 启用的可抓取来源
func (r *AIBriefingRepo) ListEnabledFetchSources() ([]*model.AIBriefingSource, error) {
	rows, err := r.db.Query("SELECT " + aiBriefingSourceCols +
		" FROM ai_briefing_sources WHERE enabled = 1 AND fetch_enabled = 1 AND url != '' ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.AIBriefingSource
	for rows.Next() {
		src := &model.AIBriefingSource{}
		var lastFetch sql.NullString
		if err := rows.Scan(&src.ID, &src.Name, &src.URL, &src.Category, &src.Enabled,
			&src.FetchEnabled, &src.FetchTime, &lastFetch, &src.CreatedAt, &src.UpdatedAt); err != nil {
			return nil, err
		}
		if lastFetch.Valid {
			src.LastFetchAt = lastFetch.String
		}
		list = append(list, src)
	}
	return list, rows.Err()
}

// CreateSource 新增来源
func (r *AIBriefingRepo) CreateSource(s *model.AIBriefingSource) (int64, error) {
	res, err := r.db.Exec(`
		INSERT INTO ai_briefing_sources (name, url, category, enabled, fetch_enabled, fetch_time)
		VALUES (?, ?, ?, ?, ?, ?)`,
		s.Name, s.URL, s.Category, s.Enabled, s.FetchEnabled, s.FetchTime)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateSource 更新来源
func (r *AIBriefingRepo) UpdateSource(s *model.AIBriefingSource) error {
	_, err := r.db.Exec(`
		UPDATE ai_briefing_sources SET name=?, url=?, category=?, enabled=?, fetch_enabled=?, fetch_time=?,
			updated_at=datetime('now') WHERE id=?`,
		s.Name, s.URL, s.Category, s.Enabled, s.FetchEnabled, s.FetchTime, s.ID)
	return err
}

// SetSourceLastFetch 记录来源最近抓取时间
func (r *AIBriefingRepo) SetSourceLastFetch(id int64, t string) error {
	_, err := r.db.Exec("UPDATE ai_briefing_sources SET last_fetch_at=? WHERE id=?", t, id)
	return err
}

// DeleteSource 删除来源
func (r *AIBriefingRepo) DeleteSource(id int64) error {
	_, err := r.db.Exec("DELETE FROM ai_briefing_sources WHERE id = ?", id)
	return err
}

func scanBriefings(rows *sql.Rows) ([]*model.AIBriefing, error) {
	var list []*model.AIBriefing
	for rows.Next() {
		b := &model.AIBriefing{}
		var fetchedAt, publishedAt, source, category, summary, content, link, keyword sql.NullString
		var createdBy sql.NullInt64
		var status sql.NullInt64
		if err := rows.Scan(&b.ID, &source, &category, &b.Topic, &summary, &content,
			&link, &keyword, &publishedAt, &fetchedAt, &status,
			&createdBy, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		b.Source = source.String
		b.Category = category.String
		b.Summary = summary.String
		b.Content = content.String
		b.Link = link.String
		b.Keyword = keyword.String
		b.PublishedAt = publishedAt.String
		b.FetchedAt = fetchedAt.String
		b.Status = int(status.Int64)
		b.CreatedBy = createdBy.Int64
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
