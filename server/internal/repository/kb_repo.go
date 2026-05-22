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
	escapedQuery := escapeQuery(query)

	// BM25 搜索：权重分配 title > summary > content
	// bm25(kb_fts, resource_id权重, title权重, summary权重, content权重)
	rows, err := r.db.Query(
		`SELECT
				kb.id, kb.resource_id, kb.resource_type, kb.owner_scope, kb.owner_id,
				kb.role_scope, kb.version, kb.status, kb.title, kb.summary,
				kb.content, kb.source_link, kb.source_version,
				kb.effective_at, kb.expired_at, kb.tags,
				kb.updated_by, kb.created_at, kb.updated_at,
				rank AS score
			 FROM kb_fts
			 JOIN kb_resources kb ON kb_fts.rowid = kb.id
			 WHERE kb_fts MATCH ?
			   AND kb.status = 'published'
			   AND (kb.owner_scope = 'school' OR (kb.owner_scope = ? AND kb.owner_id = ?))
			   AND kb.role_scope LIKE ?
			 ORDER BY score
			 LIMIT ?`,
		escapedQuery, ownerScope, ownerID, "%"+role+"%", limit,
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

// SearchFAQ 仅在 resource_type='FAQ' 中检索（用于持久化问答缓存命中）
// 返回按 BM25 排序的命中（score 越小越相关）。
// 当前最多返回 limit 条，调用方决定是否使用阈值过滤。
func (r *KBRepo) SearchFAQ(query string, role string, limit int) ([]*SearchResult, error) {
	if limit <= 0 {
		limit = 1
	}
	escapedQuery := escapeQuery(query)

	rows, err := r.db.Query(
		`SELECT
				kb.id, kb.resource_id, kb.resource_type, kb.owner_scope, kb.owner_id,
				kb.role_scope, kb.version, kb.status, kb.title, kb.summary,
				kb.content, kb.source_link, kb.source_version,
				kb.effective_at, kb.expired_at, kb.tags,
				kb.updated_by, kb.created_at, kb.updated_at,
				rank AS score
			 FROM kb_fts
			 JOIN kb_resources kb ON kb_fts.rowid = kb.id
			 WHERE kb_fts MATCH ?
			   AND kb.status = 'published'
			   AND kb.resource_type = 'FAQ'
			   AND (kb.role_scope = '' OR kb.role_scope LIKE ?)
			 ORDER BY score
			 LIMIT ?`,
		escapedQuery, "%"+role+"%", limit,
	)
	if err != nil {
		return nil, fmt.Errorf("FAQ FTS 搜索失败: %w", err)
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

// SetStatus 修改资源状态（用于把过时 FAQ 标为 retired）
func (r *KBRepo) SetStatus(resourceID string, status string) error {
	_, err := r.db.Exec(
		`UPDATE kb_resources SET status = ?, updated_at = datetime('now') WHERE resource_id = ?`,
		status, resourceID,
	)
	return err
}

// Upsert 幂等导入：resource_id 已存在时按版本号决定更新或跳过
// 返回 action: "created" / "updated" / "skipped"
func (r *KBRepo) Upsert(kb *model.KBResource) (int64, string, error) {
	// 查询是否已存在同名资源
	existing, err := r.GetByResourceID(kb.ResourceID)
	if err != nil {
		return 0, "", fmt.Errorf("查询已有资源失败: %w", err)
	}

	if existing != nil {
		// 幂等冲突解决：高版本覆盖低版本；retired 状态必须传播
		if kb.Status == "retired" {
			// retired 无条件覆盖
		} else if compareVersion(kb.Version, existing.Version) < 0 {
			// 导入版本更低，跳过
			return existing.ID, "skipped", nil
		} else if compareVersion(kb.Version, existing.Version) == 0 && existing.Status != "draft" {
			// 相同版本且非草稿，跳过（避免重复导入）
			return existing.ID, "skipped", nil
		}
		// 更新已有记录
		kb.ID = existing.ID
		kb.ResourceID = existing.ResourceID // 保持原 resource_id
		if err := r.Update(kb); err != nil {
			return 0, "", fmt.Errorf("更新资源失败: %w", err)
		}
		return existing.ID, "updated", nil
	}

	// 新建
	id, err := r.Create(kb)
	if err != nil {
		return 0, "", fmt.Errorf("创建资源失败: %w", err)
	}
	return id, "created", nil
}

// compareVersion 比较语义化版本号（兼容 "1" / "1.0" / "1.0.0" 三种形式）
// 返回 -1: v1 < v2, 0: equal, 1: v1 > v2
// 版本格式完全无效时返回 0（视为相同，保守策略）
func compareVersion(v1, v2 string) int {
	var major1, minor1, patch1 int
	var major2, minor2, patch2 int
	// Sscanf 返回成功解析的字段数；部分匹配（如 "2.0"）n>=1 且 err!=nil
	if n, _ := fmt.Sscanf(v1, "%d.%d.%d", &major1, &minor1, &patch1); n == 0 {
		return 0
	}
	if n, _ := fmt.Sscanf(v2, "%d.%d.%d", &major2, &minor2, &patch2); n == 0 {
		return 0
	}

	if major1 != major2 {
		if major1 > major2 {
			return 1
		}
		return -1
	}
	if minor1 != minor2 {
		if minor1 > minor2 {
			return 1
		}
		return -1
	}
	if patch1 != patch2 {
		if patch1 > patch2 {
			return 1
		}
		return -1
	}
	return 0
}

// List 分页查询知识资源（支持 ownerScope/status/resourceType 过滤）
func (r *KBRepo) List(ownerScope, ownerID, status, resourceType string, offset, limit int) ([]*model.KBResource, error) {
	query := `SELECT id, resource_id, resource_type, owner_scope, owner_id,
			role_scope, version, status, title, summary,
			content, source_link, source_version,
			effective_at, expired_at, tags,
			updated_by, created_at, updated_at
		 FROM kb_resources WHERE 1=1`
	args := []interface{}{}

	if ownerScope != "" {
		query += " AND owner_scope = ?"
		args = append(args, ownerScope)
		if ownerID != "" {
			query += " AND owner_id = ?"
			args = append(args, ownerID)
		}
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if resourceType != "" {
		query += " AND resource_type = ?"
		args = append(args, resourceType)
	}

	query += " ORDER BY updated_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询知识列表失败: %w", err)
	}
	defer rows.Close()

	var list []*model.KBResource
	for rows.Next() {
		kb := &model.KBResource{}
		if err := rows.Scan(
			&kb.ID, &kb.ResourceID, &kb.ResourceType, &kb.OwnerScope, &kb.OwnerID,
			&kb.RoleScope, &kb.Version, &kb.Status, &kb.Title, &kb.Summary,
			&kb.Content, &kb.SourceLink, &kb.SourceVersion,
			&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags,
			&kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, kb)
	}
	return list, rows.Err()
}

// ListSince 查询 updated_at >= sinceCursor 的资源（增量导出用）
func (r *KBRepo) ListSince(resourceType, sinceCursor string, limit int) ([]*model.KBResource, error) {
	query := `SELECT id, resource_id, resource_type, owner_scope, owner_id,
			role_scope, version, status, title, summary,
			content, source_link, source_version,
			effective_at, expired_at, tags,
			updated_by, created_at, updated_at
		 FROM kb_resources WHERE status = 'published'`
	args := []interface{}{}

	if resourceType != "" {
		query += " AND resource_type = ?"
		args = append(args, resourceType)
	}
	if sinceCursor != "" {
		query += " AND updated_at >= ?"
		args = append(args, sinceCursor)
	}

	query += " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("增量查询失败: %w", err)
	}
	defer rows.Close()

	var list []*model.KBResource
	for rows.Next() {
		kb := &model.KBResource{}
		if err := rows.Scan(
			&kb.ID, &kb.ResourceID, &kb.ResourceType, &kb.OwnerScope, &kb.OwnerID,
			&kb.RoleScope, &kb.Version, &kb.Status, &kb.Title, &kb.Summary,
			&kb.Content, &kb.SourceLink, &kb.SourceVersion,
			&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags,
			&kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, kb)
	}
	return list, rows.Err()
}

// Count 统计知识资源总数（过滤条件同 List）
func (r *KBRepo) Count(ownerScope, ownerID, status, resourceType string) (int, error) {
	query := "SELECT COUNT(*) FROM kb_resources WHERE 1=1"
	args := []interface{}{}

	if ownerScope != "" {
		query += " AND owner_scope = ?"
		args = append(args, ownerScope)
		if ownerID != "" {
			query += " AND owner_id = ?"
			args = append(args, ownerID)
		}
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if resourceType != "" {
		query += " AND resource_type = ?"
		args = append(args, resourceType)
	}

	var count int
	if err := r.db.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("统计知识资源数量失败: %w", err)
	}
	return count, nil
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

// GetPublishedCards 获取已发布的知识卡片（供知识大厅浏览，面向所有已认证用户）
// ownerScope/ownerID: 归属范围过滤（显示全校 + 当前范围）
// role: 用户角色（过滤可见资源）
// resourceType: 可选类型过滤，空字符串表示全部
// limit/offset: 分页参数，limit<=0 表示不分页
// 返回值: 按 resource_type 分组的卡片映射、全部符合条件的资源总数
func (r *KBRepo) GetPublishedCards(ownerScope, ownerID, role, resourceType string, limit, offset int) (map[string][]*model.KnowledgeCard, int, error) {
	whereClause := ` WHERE status = 'published'
		   AND (owner_scope = 'school' OR (owner_scope = ? AND owner_id = ?))
		   AND role_scope LIKE ?`
	countArgs := []interface{}{ownerScope, ownerID, "%" + role + "%"}
	queryArgs := []interface{}{ownerScope, ownerID, "%" + role + "%"}

	if resourceType != "" {
		whereClause += ` AND resource_type = ?`
		countArgs = append(countArgs, resourceType)
		queryArgs = append(queryArgs, resourceType)
	}

	// 先统计总数（不受分页影响）
	var total int
	countQuery := `SELECT COUNT(*) FROM kb_resources` + whereClause
	if err := r.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计知识大厅数据失败: %w", err)
	}

	// 分页查询
	query := `SELECT resource_id, resource_type, title, summary, tags, source_link
		 FROM kb_resources` + whereClause + `
		 ORDER BY resource_type, updated_at DESC`
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		queryArgs = append(queryArgs, limit, offset)
	}

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询知识大厅数据失败: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]*model.KnowledgeCard)
	for rows.Next() {
		card := &model.KnowledgeCard{}
		if err := rows.Scan(
			&card.ResourceID, &card.ResourceType, &card.Title,
			&card.Summary, &card.Tags, &card.SourceLink,
		); err != nil {
			return nil, 0, err
		}
		result[card.ResourceType] = append(result[card.ResourceType], card)
	}

	return result, total, rows.Err()
}

// escapeQuery 转义 FTS5 查询中的特殊字符
func escapeQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return "\"\""
	}

	// 去除特殊字符
	q = strings.ReplaceAll(q, "\"", "")

	// 对于中文查询，使用 OR 组合每个字的通配符
	// unicode61 按字符分词，"奖学金" 会被分成 "奖" "学" "金"
	// 使用 "奖* OR 学* OR 金*" 可以匹配包含任一字的文档
	runes := []rune(q)
	hasChinese := false
	for _, r := range runes {
		if r >= 0x4E00 && r <= 0x9FFF {
			hasChinese = true
			break
		}
	}

	if hasChinese {
		// 中文查询：每个字后加通配符，用 OR 连接
		var parts []string
		for _, r := range runes {
			if r >= 0x4E00 && r <= 0x9FFF {
				parts = append(parts, string(r)+"*")
			}
		}
		if len(parts) == 0 {
			return "\"\""
		}
		return strings.Join(parts, " OR ")
	}

	// 英文查询：用双引号包裹（精确匹配）
	return "\"" + q + "\""
}
