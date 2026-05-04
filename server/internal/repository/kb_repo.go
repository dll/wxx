package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/model"
)

// KBRepo 知识库数据访问（含 FTS5/BM25 全文检索）
type KBRepo struct {
	db *sql.DB
}

// NewKBRepo 创建知识库 repo
func NewKBRepo(db *sql.DB) *KBRepo {
	return &KBRepo{db: db}
}

// SearchResult FTS5 搜索结果（含 BM25 相关性分数）
type SearchResult struct {
	Resource model.KBResource
	Score    float64 // BM25 分数（负值，越小越相关）
}

// Search 全文检索，使用 FTS5/BM25 算法
// query: 用户搜索词
// ownerScope/ownerID: 归属范围过滤
// role: 用户角色（过滤可见资源）
// limit: 返回结果数
func (r *KBRepo) Search(query string, ownerScope string, ownerID string, role string, limit int) ([]*SearchResult, error) {
	// BM25 搜索：权重分配 title > summary > content
	// bm25(kb_fts, resource_id权重, title权重, summary权重, content权重)
	rows, err := r.db.Query(
		`SELECT
			kb.id, kb.resource_id, kb.resource_type, kb.owner_scope, kb.owner_id,
			kb.role_scope, kb.version, kb.status, kb.title, kb.summary,
			kb.content, kb.source_link, kb.source_version,
			kb.effective_at, kb.expired_at, kb.tags,
			kb.updated_by, kb.created_at, kb.updated_at,
			bm25(kb_fts, 0, 10, 5, 1) AS score
		 FROM kb_fts
		 JOIN kb_resources kb ON kb_fts.rowid = kb.id
		 WHERE kb_fts MATCH ?
		   AND kb.status = 'published'
		   AND (kb.expired_at IS NULL OR kb.expired_at > datetime('now'))
		   AND (kb.owner_scope = 'school' OR (kb.owner_scope = ? AND kb.owner_id = ?))
		   AND kb.role_scope LIKE ?
		 ORDER BY score
		 LIMIT ?`,
		escapeQuery(query), ownerScope, ownerID, "%"+role+"%", limit,
	)
	if err != nil {
		return nil, fmt.Errorf("FTS 搜索失败: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		sr := &SearchResult{}
		kb := &sr.Resource
		if err := rows.Scan(
			&kb.ID, &kb.ResourceID, &kb.ResourceType, &kb.OwnerScope, &kb.OwnerID,
			&kb.RoleScope, &kb.Version, &kb.Status, &kb.Title, &kb.Summary,
			&kb.Content, &kb.SourceLink, &kb.SourceVersion,
			&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags,
			&kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
			&sr.Score,
		); err != nil {
			return nil, err
		}
		results = append(results, sr)
	}
	return results, rows.Err()
}

// GetByResourceID 根据资源 ID 查询
func (r *KBRepo) GetByResourceID(resourceID string) (*model.KBResource, error) {
	kb := &model.KBResource{}
	err := r.db.QueryRow(
		`SELECT id, resource_id, resource_type, owner_scope, owner_id,
			role_scope, version, status, title, summary,
			content, source_link, source_version,
			effective_at, expired_at, tags,
			updated_by, created_at, updated_at
		 FROM kb_resources WHERE resource_id = ?`, resourceID,
	).Scan(
		&kb.ID, &kb.ResourceID, &kb.ResourceType, &kb.OwnerScope, &kb.OwnerID,
		&kb.RoleScope, &kb.Version, &kb.Status, &kb.Title, &kb.Summary,
		&kb.Content, &kb.SourceLink, &kb.SourceVersion,
		&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags,
		&kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return kb, nil
}

// Create 创建知识资源
func (r *KBRepo) Create(kb *model.KBResource) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO kb_resources
		 (resource_id, resource_type, owner_scope, owner_id, role_scope,
		  version, status, title, summary, content,
		  source_link, source_version, effective_at, expired_at, tags, updated_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		kb.ResourceID, kb.ResourceType, kb.OwnerScope, kb.OwnerID, kb.RoleScope,
		kb.Version, kb.Status, kb.Title, kb.Summary, kb.Content,
		kb.SourceLink, kb.SourceVersion, kb.EffectiveAt, kb.ExpiredAt, kb.Tags, kb.UpdatedBy,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// Update 更新知识资源
func (r *KBRepo) Update(kb *model.KBResource) error {
	_, err := r.db.Exec(
		`UPDATE kb_resources SET
			resource_type = ?, owner_scope = ?, owner_id = ?, role_scope = ?,
			version = ?, status = ?, title = ?, summary = ?, content = ?,
			source_link = ?, source_version = ?, effective_at = ?, expired_at = ?,
			tags = ?, updated_by = ?, updated_at = datetime('now')
		 WHERE resource_id = ?`,
		kb.ResourceType, kb.OwnerScope, kb.OwnerID, kb.RoleScope,
		kb.Version, kb.Status, kb.Title, kb.Summary, kb.Content,
		kb.SourceLink, kb.SourceVersion, kb.EffectiveAt, kb.ExpiredAt,
		kb.Tags, kb.UpdatedBy, kb.ResourceID,
	)
	return err
}

// GetProcessSteps 获取流程步骤（按步骤序号排序）
func (r *KBRepo) GetProcessSteps(resourceID string) ([]*model.ProcessStep, error) {
	rows, err := r.db.Query(
		`SELECT id, resource_id, step_order, title, materials, entry_url, deadline, location, notes
		 FROM process_steps WHERE resource_id = ?
		 ORDER BY step_order ASC`, resourceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*model.ProcessStep
	for rows.Next() {
		s := &model.ProcessStep{}
		if err := rows.Scan(&s.ID, &s.ResourceID, &s.StepOrder, &s.Title,
			&s.Materials, &s.EntryURL, &s.Deadline, &s.Location, &s.Notes); err != nil {
			return nil, err
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

// escapeQuery 转义 FTS5 查询中的特殊字符
func escapeQuery(q string) string {
	// FTS5 特殊字符：* " ( ) OR AND NOT
	// 简单处理：将查询词用双引号包裹
	q = strings.TrimSpace(q)
	if q == "" {
		return "\"\""
	}
	// 去除已有的双引号，再包裹
	q = strings.ReplaceAll(q, "\"", "")
	return "\"" + q + "\""
}
