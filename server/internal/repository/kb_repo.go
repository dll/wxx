package repository

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/util"
)

// KBRepo 知识库数据访问（含 FTS5/BM25 全文检索；MySQL 下退化为结构化检索）
type KBRepo struct {
	db    *sql.DB
	mysql bool // MySQL 方言标志：FTS5/json_each 等 SQLite 特性不可用
}

// NewKBRepo 创建知识库 repo
func NewKBRepo(db *sql.DB) *KBRepo {
	return &KBRepo{db: db, mysql: dbutil.IsMySQL(db)}
}

// roleScopeCond 生成角色可见性过滤 SQL 片段（按方言）
// 参数 ? 为用户角色。SQLite 用 JSON1 扩展函数，MySQL 用 JSON_* 函数。
func (r *KBRepo) roleScopeCond(alias string) string {
	col := "role_scope"
	if alias != "" {
		col = alias + ".role_scope"
	}
	if r.mysql {
		return fmt.Sprintf(`(? IN ('sys_admin','school_admin') OR %s = '' OR (JSON_VALID(%s) AND (JSON_LENGTH(%s) = 0 OR JSON_CONTAINS(%s, JSON_QUOTE(?)))))`, col, col, col, col)
	}
	return fmt.Sprintf(`(? IN ('sys_admin','school_admin') OR %s = '' OR (json_valid(%s) AND (json_array_length(%s) = 0 OR EXISTS (SELECT 1 FROM json_each(%s) WHERE value = ?))))`, col, col, col, col)
}

// KBStats 知识资源统计
type KBStats struct {
	Total     int            `json:"total"`
	Draft     int            `json:"draft"`
	Pending   int            `json:"pending"`
	Published int            `json:"published"`
	Retired   int            `json:"retired"`
	ByType    map[string]int `json:"by_type"`
}

// SearchResult FTS5 搜索结果（含 BM25 相关性分数）
type SearchResult struct {
	Resource model.KBResource
	Score    float64 // BM25 分数（负值，越小越相关）
	// LowConfidence 瞬态标记（不落库）：检索相关性全部低于阈值、仅为避免空结果而强留的最佳结果。
	// 下游据此走兜底，避免基于弱相关资料编造政策关键数字（CE-02）。
	LowConfidence bool
	// IsStructured 标记：来自结构化检索（title/category/tags 匹配），具有最高优先级
	IsStructured bool
}

// Search 全文检索，使用 FTS5/BM25 算法（增强精准度版）
// query: 用户搜索词
// ownerScope/ownerID: 归属范围过滤
// role: 用户角色（过滤可见资源）
// limit: 返回结果数
// 返回经过相关性校验的结果，过滤掉不相关的内容
func (r *KBRepo) Search(query string, ownerScope string, ownerID string, role string, limit int) ([]*SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	// MySQL 无 FTS5 虚拟表：退化为结构化检索（title/tags/category LIKE 匹配）
	if r.mysql {
		return r.SearchStructured(query, ownerScope, ownerID, role, limit)
	}

	// 阶段一：短语精确匹配优先（最高优先级）
	phraseQuery := buildPhraseQuery(query)
	phraseResults, err := r.searchWithQuery(phraseQuery, ownerScope, ownerID, role, limit)
	if err == nil && len(phraseResults) > 0 {
		// 短语匹配结果加分，标记为高置信度
		for _, r := range phraseResults {
			r.Score -= 8.0 // BM25分数越小越相关，减8表示最高优先级
		}
	}

	// 阶段二：NEAR 邻近匹配（字之间距离≤3）
	nearQuery := buildNearQuery(query)
	nearResults, err := r.searchWithQuery(nearQuery, ownerScope, ownerID, role, limit)
	if err == nil && len(nearResults) > 0 {
		for _, r := range nearResults {
			r.Score -= 3.0 // NEAR匹配次高优先级
		}
	}

	// 阶段三：宽松 OR 匹配（兜底召回）
	looseQuery := buildLooseQuery(query)
	looseResults, err := r.searchWithQuery(looseQuery, ownerScope, ownerID, role, limit*2)
	if err != nil && len(phraseResults) == 0 && len(nearResults) == 0 {
		return nil, err
	}

	// 合并去重，按分数排序
	merged := mergeResults(phraseResults, nearResults, looseResults)

	// 相关性二次过滤：标题/摘要必须至少包含一个核心词
	filtered := filterByRelevance(merged, query)

	// 按分数排序（越小越相关）
	sortResultsByScore(filtered)

	// 限制返回数量
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

// SearchStructured 结构化优先检索：按 title/category/tags 匹配
// 不依赖 FTS5 索引，直接对 kb_resources 的结构化字段做精确/模糊匹配。
// MySQL 下不能把整个问题当整句 LIKE（中文自然语言不分词必然空结果），
// 因此拆成中文二元词组对 title/tags/summary/content 做 OR 召回，再按命中词组加权排序。
func (r *KBRepo) SearchStructured(query string, ownerScope string, ownerID string, role string, limit int) ([]*SearchResult, error) {
	return r.searchStructured(query, ownerScope, ownerID, role, limit, "")
}

// searchStructured 结构化检索实现；typeFilter 非空时限定资源类型（如 FAQ）
func (r *KBRepo) searchStructured(query string, ownerScope string, ownerID string, role string, limit int, typeFilter string) ([]*SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}

	// 中文分词：提取二元词组作为检索词（MySQL 无 FTS5 分词器，手工 bigram 召回）
	words := extractChineseBigrams(query)
	if len(words) == 0 {
		words = []string{query}
	}
	// 限制词数，避免 SQL 过长（取前 8 个高频词组即可覆盖常见查询）
	if len(words) > 8 {
		words = words[:8]
	}

	// WHERE：整句精确 + 每个词对 title/tags/summary/content 的 LIKE OR 组合
	orParts := []string{"kb.title = ?"}
	args := []any{query}
	for _, w := range words {
		like := "%" + w + "%"
		orParts = append(orParts, "(kb.title LIKE ? OR kb.tags LIKE ? OR kb.summary LIKE ? OR kb.content LIKE ?)")
		args = append(args, like, like, like, like)
	}
	whereLike := strings.Join(orParts, " OR ")

	// 追加 owner_scope / role_scope / limit 参数（与 SQL 占位符顺序一致）
	args = append(args, ownerScope, ownerID, ownerScope, ownerID, role, role, limit)

	sql := `SELECT
				kb.id, kb.resource_id, kb.resource_type, kb.owner_scope, kb.owner_id,
				kb.role_scope, kb.version, kb.status, kb.title, kb.summary,
				kb.content, kb.source_link, kb.source_version,
				kb.effective_at, kb.expired_at, kb.tags, kb.remark,
				kb.updated_by, kb.created_at, kb.updated_at,
				0 AS priority
			 FROM kb_resources kb
			 WHERE kb.status = 'published'
			   AND (` + whereLike + `)`
	if typeFilter != "" {
		sql += ` AND kb.resource_type = '` + typeFilter + `'`
	}
	sql += ` AND (kb.owner_scope = 'school' OR (kb.owner_scope = 'college' AND (? = '' OR kb.owner_id = ?)) OR (kb.owner_scope = 'class' AND (? = '' OR kb.owner_id = ?)))
			   AND ` + r.roleScopeCond("kb") + `
			 LIMIT ?`

	rows, err := r.db.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		sr := &SearchResult{IsStructured: true}
		kb := &sr.Resource
		var priority float64
		if err := rows.Scan(
			&kb.ID, &kb.ResourceID, &kb.ResourceType, &kb.OwnerScope, &kb.OwnerID,
			&kb.RoleScope, &kb.Version, &kb.Status, &kb.Title, &kb.Summary,
			&kb.Content, &kb.SourceLink, &kb.SourceVersion,
			&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags, &kb.Remark,
			&kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
			&priority,
		); err != nil {
			continue
		}
		// 按命中词组加权打分（title 权重最高，其次 tags/summary/content），
		// 负分确保结构化结果在 FTS 结果之前（原逻辑 -100 基准）
		sr.Score = -100.0 - structuredScore(kb, words)
		results = append(results, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 按得分升序（越小越相关，与 FTS 排序方向一致），取 limit
	sort.SliceStable(results, func(i, j int) bool { return results[i].Score < results[j].Score })
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// structuredScore 计算资源与检索词组的命中得分：title 命中权重最高
func structuredScore(kb *model.KBResource, words []string) float64 {
	score := 0.0
	for _, w := range words {
		if strings.Contains(kb.Title, w) {
			score += 4
		}
		if strings.Contains(kb.Tags, w) {
			score += 3
		}
		if strings.Contains(kb.Summary, w) {
			score += 2
		}
		if strings.Contains(kb.Content, w) {
			score += 2
		}
	}
	return score
}

// searchWithQuery 使用指定 FTS 查询语句执行搜索
func (r *KBRepo) searchWithQuery(ftsQuery string, ownerScope string, ownerID string, role string, limit int) ([]*SearchResult, error) {
	if ftsQuery == "" {
		return nil, nil
	}
	rows, err := r.db.Query(
		`SELECT
				kb.id, kb.resource_id, kb.resource_type, kb.owner_scope, kb.owner_id,
				kb.role_scope, kb.version, kb.status, kb.title, kb.summary,
				kb.content, kb.source_link, kb.source_version,
				kb.effective_at, kb.expired_at, kb.tags, kb.remark,
				kb.updated_by, kb.created_at, kb.updated_at,
				bm25(kb_fts, 0.0, 10.0, 3.0, 1.0, 4.0) AS score
			 FROM kb_fts
			 JOIN kb_resources kb ON kb_fts.rowid = kb.id
			 WHERE kb_fts MATCH ?
			   AND kb.status = 'published'
			   AND (kb.owner_scope = 'school' OR (kb.owner_scope = 'college' AND (? = '' OR kb.owner_id = ?)) OR (kb.owner_scope = 'class' AND (? = '' OR kb.owner_id = ?)))
			   AND `+r.roleScopeCond("kb")+`
			 ORDER BY score
			 LIMIT ?`,
		ftsQuery, ownerScope, ownerID, ownerScope, ownerID, role, role, limit,
	)
	if err != nil {
		return nil, err
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
			&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags, &kb.Remark,
			&kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
			&sr.Score,
		); err != nil {
			continue
		}
		results = append(results, sr)
	}
	return results, rows.Err()
}

// buildPhraseQuery 构建短语精确匹配查询（中文按字符连续排列即短语）
func buildPhraseQuery(query string) string {
	runes := []rune(query)
	var parts []string
	for _, r := range runes {
		if r >= 0x4E00 && r <= 0x9FFF {
			parts = append(parts, string(r))
		} else if !isSpaceRune(r) {
			parts = append(parts, string(r))
		}
	}
	if len(parts) < 2 {
		return ""
	}
	return "\"" + strings.Join(parts, " ") + "\""
}

// buildNearQuery 构建 NEAR 邻近匹配查询（字之间最大距离3）
// 注意：FTS5 的 NEAR() 语法不支持内部使用前缀通配符 *
// 使用精确单字匹配，利用 unicode61 单字分词的特性，字与字邻近=词组邻近
func buildNearQuery(query string) string {
	runes := []rune(query)
	var parts []string
	for _, r := range runes {
		if r >= 0x4E00 && r <= 0x9FFF {
			parts = append(parts, string(r))
		} else if !isSpaceRune(r) {
			parts = append(parts, string(r))
		}
	}
	if len(parts) < 2 {
		return ""
	}
	return "NEAR(" + strings.Join(parts, " ") + ", 3)"
}

// escapeFTS 转义 FTS5 保留运算符，防止语法错误或注入
func escapeFTS(s string) string {
	s = strings.ToUpper(s)
	s = strings.ReplaceAll(s, "AND", "")
	s = strings.ReplaceAll(s, "OR", "")
	s = strings.ReplaceAll(s, "NOT", "")
	s = strings.ReplaceAll(s, "NEAR", "")
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	return s
}

// buildLooseQuery 构建宽松 OR 匹配查询（兜底召回）
func buildLooseQuery(query string) string {
	query = escapeFTS(query)
	runes := []rune(query)
	var parts []string
	for _, r := range runes {
		if r >= 0x4E00 && r <= 0x9FFF {
			parts = append(parts, string(r)+"*")
		} else if !isSpaceRune(r) {
			parts = append(parts, string(r)+"*")
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " OR ")
}

func isSpaceRune(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '，' || r == '。' || r == '？' || r == '！' || r == '、'
}

// mergeResults 合并多轮搜索结果并去重（保留分数最低=最相关的）
func mergeResults(groups ...[]*SearchResult) []*SearchResult {
	best := make(map[int64]*SearchResult)
	for _, group := range groups {
		for _, r := range group {
			existing, ok := best[r.Resource.ID]
			if !ok || r.Score < existing.Score {
				best[r.Resource.ID] = r
			}
		}
	}
	var results []*SearchResult
	for _, r := range best {
		results = append(results, r)
	}
	return results
}

// filterByRelevance 相关性二次过滤
// 规则：标题或摘要中必须包含用户问题中的至少一个核心词（中文两字词及以上）
// 目的：过滤掉只在正文中零散出现几个字的完全不相关文档
func filterByRelevance(results []*SearchResult, query string) []*SearchResult {
	if len(results) == 0 {
		return results
	}

	// 提取查询中的中文二元词组（核心词）
	bigrams := extractChineseBigrams(query)
	if len(bigrams) == 0 {
		// 查询太短，直接返回
		return results
	}

	var filtered []*SearchResult
	for _, r := range results {
		title := r.Resource.Title
		summary := r.Resource.Summary
		text := title + summary

		// 检查标题/摘要中是否包含至少一个二元词组
		matched := false
		for _, bg := range bigrams {
			if strings.Contains(text, bg) {
				matched = true
				break
			}
		}

		// 如果标题中有任何一个查询词，也算通过（单字在标题中也算强信号）
		if !matched {
			for _, ch := range query {
				if ch >= 0x4E00 && ch <= 0x9FFF && strings.Contains(title, string(ch)) {
					matched = true
					break
				}
			}
		}

		if matched {
			filtered = append(filtered, r)
		}
	}

	// 如果过滤后为空，放宽条件：检查全文是否包含至少3个查询字符
	if len(filtered) == 0 && len(results) > 0 {
		for _, r := range results {
			text := r.Resource.Title + r.Resource.Summary + r.Resource.Content
			count := 0
			for _, ch := range query {
				if ch >= 0x4E00 && ch <= 0x9FFF && strings.Contains(text, string(ch)) {
					count++
				}
			}
			if count >= 3 || float64(count)/float64(len([]rune(query))) >= 0.5 {
				filtered = append(filtered, r)
			}
		}
	}

	// 还是为空？返回原始结果（避免完全无结果）
	if len(filtered) == 0 {
		return results
	}

	return filtered
}

// extractChineseBigrams 提取中文二元词组
func extractChineseBigrams(s string) []string {
	runes := []rune(s)
	var bigrams []string
	seen := make(map[string]bool)
	for i := 0; i < len(runes)-1; i++ {
		if runes[i] >= 0x4E00 && runes[i] <= 0x9FFF &&
			runes[i+1] >= 0x4E00 && runes[i+1] <= 0x9FFF {
			bg := string(runes[i : i+2])
			if !seen[bg] {
				seen[bg] = true
				bigrams = append(bigrams, bg)
			}
		}
	}
	return bigrams
}

// sortResultsByScore 按 BM25 分数排序（越小越相关）
func sortResultsByScore(results []*SearchResult) {
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score < results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// SearchFAQ 仅在 resource_type='FAQ' 中检索（用于持久化问答缓存命中）
// 返回按 BM25 排序的命中（score 越小越相关）。
// 使用与 Search 相同的三阶段检索策略，确保 FAQ 匹配精准
func (r *KBRepo) SearchFAQ(query string, ownerScope string, ownerID string, role string, limit int) ([]*SearchResult, error) {
	if limit <= 0 {
		limit = 1
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	// MySQL 无 FTS5 虚拟表：退化为结构化检索，仅限 FAQ 类型
	if r.mysql {
		return r.searchStructured(query, ownerScope, ownerID, role, limit, "FAQ")
	}

	// 阶段一：短语精确匹配
	phraseQ := buildPhraseQuery(query)
	phraseR, _ := r.searchFAQWithQuery(phraseQ, ownerScope, ownerID, role, limit)
	for _, r := range phraseR {
		r.Score -= 8.0 // 短语精确匹配，最高优先级
	}

	// 阶段二：NEAR 邻近匹配
	nearQ := buildNearQuery(query)
	nearR, _ := r.searchFAQWithQuery(nearQ, ownerScope, ownerID, role, limit)
	for _, r := range nearR {
		r.Score -= 3.0 // NEAR邻近匹配，次高优先级
	}

	// 阶段三：宽松 OR 匹配
	looseQ := buildLooseQuery(query)
	looseR, err := r.searchFAQWithQuery(looseQ, ownerScope, ownerID, role, limit*2)
	if err != nil && len(phraseR) == 0 && len(nearR) == 0 {
		return nil, fmt.Errorf("FAQ FTS 搜索失败: %w", err)
	}

	// 合并去重 + 相关性过滤
	merged := mergeResults(phraseR, nearR, looseR)
	filtered := filterByRelevance(merged, query)
	sortResultsByScore(filtered)

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

// searchFAQWithQuery 在 FAQ 资源中执行指定 FTS 查询
func (r *KBRepo) searchFAQWithQuery(ftsQuery string, ownerScope string, ownerID string, role string, limit int) ([]*SearchResult, error) {
	if ftsQuery == "" {
		return nil, nil
	}
	rows, err := r.db.Query(
		`SELECT
				kb.id, kb.resource_id, kb.resource_type, kb.owner_scope, kb.owner_id,
				kb.role_scope, kb.version, kb.status, kb.title, kb.summary,
				kb.content, kb.source_link, kb.source_version,
				kb.effective_at, kb.expired_at, kb.tags, kb.remark,
				kb.updated_by, kb.created_at, kb.updated_at,
				bm25(kb_fts, 0.0, 10.0, 3.0, 1.0, 4.0) AS score
			 FROM kb_fts
			 JOIN kb_resources kb ON kb_fts.rowid = kb.id
			 WHERE kb_fts MATCH ?
			   AND kb.status = 'published'
			   AND kb.resource_type = 'FAQ'
			   AND (kb.owner_scope = 'school' OR (kb.owner_scope = 'college' AND (? = '' OR kb.owner_id = ?)) OR (kb.owner_scope = 'class' AND (? = '' OR kb.owner_id = ?)))
			   AND `+r.roleScopeCond("kb")+`
			 ORDER BY score
			 LIMIT ?`,
		ftsQuery, ownerScope, ownerID, ownerScope, ownerID, role, role, limit,
	)
	if err != nil {
		return nil, err
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
			&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags, &kb.Remark,
			&kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
			&sr.Score,
		); err != nil {
			continue
		}
		results = append(results, sr)
	}
	return results, rows.Err()
}

// SetStatus 修改资源状态（用于把过时 FAQ 标为 retired）
func (r *KBRepo) SetStatus(resourceID string, status string) error {
	_, err := r.db.Exec(
		`UPDATE kb_resources SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE resource_id = ?`,
		status, resourceID,
	)
	return err
}

// Upsert 幂等导入：resource_id 已存在时按版本号决定更新或跳过
// 返回 action: "created" / "updated" / "skipped"
//
// 覆盖 NDJSON 导入与 ChatService 的 FAQ 缓存写入两条路径，
// 在写库前统一清洗内容（FTS5 触发器会在写入瞬间索引 title/summary/content）。
func (r *KBRepo) Upsert(kb *model.KBResource) (int64, string, error) {
	id, action, _, err := r.UpsertDetailed(kb)
	return id, action, err
}

// UpsertDetailed 返回 (id, action, reason, error)。
// reason 用于区分 skipped 是否因版本冲突：version_low / duplicate / other。
func (r *KBRepo) UpsertDetailed(kb *model.KBResource) (int64, string, string, error) {
	existing, err := r.GetByResourceID(kb.ResourceID)
	if err != nil {
		return 0, "", "", fmt.Errorf("查询已有资源失败: %w", err)
	}
	return r.upsertDetailedExisting(kb, existing)
}

// UpsertDetailedExisting 幂等写入（调用方已持有 existing 记录时使用），避免重复查询。
// 语义与 UpsertDetailed 一致；existing 为 nil 表示新建。
func (r *KBRepo) UpsertDetailedExisting(kb *model.KBResource, existing *model.KBResource) (int64, string, string, error) {
	return r.upsertDetailedExisting(kb, existing)
}

// upsertDetailedExisting 幂等写入核心：复用调用方已查到的 existing，避免重复查询。
// 语义与 UpsertDetailed 完全一致；existing 为 nil 表示新建。
func (r *KBRepo) upsertDetailedExisting(kb *model.KBResource, existing *model.KBResource) (int64, string, string, error) {
	kb.Title = util.SanitizeKnowledgeContent(kb.Title)
	kb.Summary = util.SanitizeKnowledgeContent(kb.Summary)
	kb.Content = util.SanitizeKnowledgeContentByType(kb.Content, kb.ResourceType)

	if existing != nil {
		// 幂等冲突解决：高版本覆盖低版本；retired 状态必须传播
		if kb.Status == "retired" {
			// retired 无条件覆盖
		} else if compareVersion(kb.Version, existing.Version) < 0 {
			// 导入版本更低，跳过
			return existing.ID, "skipped", "version_low", nil
		} else if compareVersion(kb.Version, existing.Version) == 0 && existing.Status != "draft" {
			// 相同版本且非草稿，跳过（避免重复导入）
			return existing.ID, "skipped", "duplicate", nil
		}
		// 更新已有记录
		kb.ID = existing.ID
		kb.ResourceID = existing.ResourceID // 保持原 resource_id
		if err := r.Update(kb); err != nil {
			return 0, "", "", fmt.Errorf("更新资源失败: %w", err)
		}
		return existing.ID, "updated", "", nil
	}

	// 新建
	id, err := r.Create(kb)
	if err != nil {
		return 0, "", "", fmt.Errorf("创建资源失败: %w", err)
	}
	return id, "created", "", nil
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
			effective_at, expired_at, tags, remark,
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
			&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags, &kb.Remark,
			&kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, kb)
	}
	return list, rows.Err()
}

// ListSince 查询 updated_at >= sinceCursor 的资源（增量导出用）
// 安全修复 RB-01：新增 callerScope/callerOwnerID 强制按调用者数据范围过滤（纵深防御，与路由能力门叠加）
//   - callerScope == "school"（或空，兼容旧调用）：可导出全部已发布资源
//   - callerScope == "college"：仅本院资源 + 全校公共（school 级）资源
//   - callerScope == "class"：仅本班资源 + 上级公共（school/college）资源
func (r *KBRepo) ListSince(resourceType, sinceCursor, callerScope, callerOwnerID string, limit int) ([]*model.KBResource, error) {
	query := `SELECT id, resource_id, resource_type, owner_scope, owner_id,
			role_scope, version, status, title, summary,
			content, source_link, source_version,
			effective_at, expired_at, tags, remark,
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

	// scope 过滤：school 级（或空）不限；college/class 级仅本范围 + 上级公共资源
	switch callerScope {
	case "college":
		query += " AND (owner_scope = 'school' OR (owner_scope = 'college' AND owner_id = ?))"
		args = append(args, callerOwnerID)
	case "class":
		query += " AND (owner_scope = 'school' OR owner_scope = 'college' OR (owner_scope = 'class' AND owner_id = ?))"
		args = append(args, callerOwnerID)
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
			&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags, &kb.Remark,
			&kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, kb)
	}
	return list, rows.Err()
}

// ListSincePage 分页增量导出：使用 (updated_at, resource_id) 复合游标，
// 保证同一秒内多条数据也能无重复、无遗漏地遍历。
func (r *KBRepo) ListSincePage(resourceType, sinceCursor, callerScope, callerOwnerID string, limit int) ([]*model.KBResource, string, error) {
	if limit <= 0 {
		limit = 1000
	}
	if limit > 5000 {
		limit = 5000
	}

	query := `SELECT id, resource_id, resource_type, owner_scope, owner_id,
			role_scope, version, status, title, summary,
			content, source_link, source_version,
			effective_at, expired_at, tags, remark,
			updated_by, created_at, updated_at
		 FROM kb_resources WHERE status = 'published'`
	args := []interface{}{}

	if resourceType != "" {
		query += " AND resource_type = ?"
		args = append(args, resourceType)
	}
	cursorUpdated, cursorID, _ := strings.Cut(sinceCursor, "|")
	if cursorUpdated != "" {
		query += " AND (updated_at > ? OR (updated_at = ? AND resource_id > ?))"
		args = append(args, cursorUpdated, cursorUpdated, cursorID)
	}

	switch callerScope {
	case "college":
		query += " AND (owner_scope = 'school' OR (owner_scope = 'college' AND owner_id = ?))"
		args = append(args, callerOwnerID)
	case "class":
		query += " AND (owner_scope = 'school' OR owner_scope = 'college' OR (owner_scope = 'class' AND owner_id = ?))"
		args = append(args, callerOwnerID)
	}

	query += " ORDER BY updated_at ASC, resource_id ASC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("分页增量查询失败: %w", err)
	}
	defer rows.Close()

	var list []*model.KBResource
	for rows.Next() {
		kb := &model.KBResource{}
		if err := rows.Scan(
			&kb.ID, &kb.ResourceID, &kb.ResourceType, &kb.OwnerScope, &kb.OwnerID,
			&kb.RoleScope, &kb.Version, &kb.Status, &kb.Title, &kb.Summary,
			&kb.Content, &kb.SourceLink, &kb.SourceVersion,
			&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags, &kb.Remark,
			&kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
		); err != nil {
			return nil, "", err
		}
		list = append(list, kb)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(list) == limit {
		last := list[len(list)-1]
		nextCursor = last.UpdatedAt + "|" + last.ResourceID
	}
	return list, nextCursor, nil
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
				effective_at, expired_at, tags, remark,
				updated_by, created_at, updated_at
			 FROM kb_resources WHERE resource_id = ?`, resourceID,
	).Scan(
		&kb.ID, &kb.ResourceID, &kb.ResourceType, &kb.OwnerScope, &kb.OwnerID,
		&kb.RoleScope, &kb.Version, &kb.Status, &kb.Title, &kb.Summary,
		&kb.Content, &kb.SourceLink, &kb.SourceVersion,
		&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags, &kb.Remark,
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

// GetByResourceIDs 批量按 resource_id 查询，返回以 resource_id 为键的映射。
// 仅返回存在的资源；不存在的 ID 不出现在映射中。用于批量操作的权限复核，避免逐条 N+1 查询。
func (r *KBRepo) GetByResourceIDs(resourceIDs []string) (map[string]*model.KBResource, error) {
	if len(resourceIDs) == 0 {
		return map[string]*model.KBResource{}, nil
	}
	placeholders := make([]string, len(resourceIDs))
	args := make([]interface{}, len(resourceIDs))
	for i, id := range resourceIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := r.db.Query(
		fmt.Sprintf(
			`SELECT id, resource_id, resource_type, owner_scope, owner_id,
					role_scope, version, status, title, summary,
					content, source_link, source_version,
					effective_at, expired_at, tags, remark,
					updated_by, created_at, updated_at
			 FROM kb_resources WHERE resource_id IN (%s)`,
			strings.Join(placeholders, ","),
		), args...,
	)
	if err != nil {
		return nil, fmt.Errorf("批量查询知识资源失败: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*model.KBResource, len(resourceIDs))
	for rows.Next() {
		kb := &model.KBResource{}
		if err := rows.Scan(
			&kb.ID, &kb.ResourceID, &kb.ResourceType, &kb.OwnerScope, &kb.OwnerID,
			&kb.RoleScope, &kb.Version, &kb.Status, &kb.Title, &kb.Summary,
			&kb.Content, &kb.SourceLink, &kb.SourceVersion,
			&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags, &kb.Remark,
			&kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("批量查询知识资源失败: %w", err)
		}
		result[kb.ResourceID] = kb
	}
	return result, rows.Err()
}

// Create 创建知识资源
func (r *KBRepo) Create(kb *model.KBResource) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO kb_resources
			 (resource_id, resource_type, owner_scope, owner_id, role_scope,
			  version, status, title, summary, content,
			  source_link, source_version, effective_at, expired_at, tags, remark, updated_by)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		kb.ResourceID, kb.ResourceType, kb.OwnerScope, kb.OwnerID, kb.RoleScope,
		kb.Version, kb.Status, kb.Title, kb.Summary, kb.Content,
		kb.SourceLink, kb.SourceVersion, kb.EffectiveAt, kb.ExpiredAt, kb.Tags, kb.Remark, kb.UpdatedBy,
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
				tags = ?, remark = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP
			 WHERE resource_id = ?`,
		kb.ResourceType, kb.OwnerScope, kb.OwnerID, kb.RoleScope,
		kb.Version, kb.Status, kb.Title, kb.Summary, kb.Content,
		kb.SourceLink, kb.SourceVersion, kb.EffectiveAt, kb.ExpiredAt,
		kb.Tags, kb.Remark, kb.UpdatedBy, kb.ResourceID,
	)
	return err
}

// CountProcessSteps 查询某流程资源的实际步骤数（用于校准 totalSteps）
func (r *KBRepo) CountProcessSteps(resourceID string) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM process_steps WHERE resource_id = ?`, resourceID,
	).Scan(&count)
	return count, err
}

// GetProcessSteps 获取流程步骤（按步骤序号排序）
func (r *KBRepo) GetProcessSteps(resourceID string) ([]*model.ProcessStep, error) {
	rows, err := r.db.Query(
		`SELECT id, resource_id, step_order, title, materials, entry_url, deadline, location, notes,
		        contact, phone, contact_wechat, office_hours, geo_lat, geo_lng, media_urls, faq
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
			&s.Materials, &s.EntryURL, &s.Deadline, &s.Location, &s.Notes,
			&s.Contact, &s.Phone, &s.ContactWechat, &s.OfficeHours,
			&s.GeoLat, &s.GeoLng, &s.MediaURLs, &s.FAQ); err != nil {
			return nil, err
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

// GetProcessReminders 获取流程提醒（按提醒时间排序）
func (r *KBRepo) GetProcessReminders(resourceID string) ([]*model.ProcessReminder, error) {
	rows, err := r.db.Query(
		`SELECT id, process_id, step_order, remind_at, title, content, is_enabled, created_at, updated_at
		 FROM process_reminders WHERE process_id = ? ORDER BY remind_at ASC, id ASC`, resourceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.ProcessReminder
	for rows.Next() {
		r := &model.ProcessReminder{}
		if err := rows.Scan(&r.ID, &r.ProcessID, &r.StepOrder, &r.RemindAt,
			&r.Title, &r.Content, &r.IsEnabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// ReplaceProcessStepsAndReminders 原子替换流程步骤与提醒
func (r *KBRepo) ReplaceProcessStepsAndReminders(resourceID string, steps []*model.ProcessStep, reminders []*model.ProcessReminder) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("开启流程更新事务失败: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM process_steps WHERE resource_id = ?`, resourceID); err != nil {
		return fmt.Errorf("清空流程步骤失败: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM process_reminders WHERE process_id = ?`, resourceID); err != nil {
		return fmt.Errorf("清空流程提醒失败: %w", err)
	}

	for _, s := range steps {
		if _, err := tx.Exec(
			`INSERT INTO process_steps
			 (resource_id, step_order, title, materials, entry_url, deadline, location, notes,
			  contact, phone, contact_wechat, office_hours, geo_lat, geo_lng, media_urls, faq)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			resourceID, s.StepOrder, s.Title, s.Materials, s.EntryURL, s.Deadline, s.Location, s.Notes,
			s.Contact, s.Phone, s.ContactWechat, s.OfficeHours, s.GeoLat, s.GeoLng, s.MediaURLs, s.FAQ,
		); err != nil {
			return fmt.Errorf("写入流程步骤失败: %w", err)
		}
	}

	for _, reminder := range reminders {
		if _, err := tx.Exec(
			`INSERT INTO process_reminders (process_id, step_order, remind_at, title, content, is_enabled)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			resourceID, reminder.StepOrder, reminder.RemindAt, reminder.Title, reminder.Content, reminder.IsEnabled,
		); err != nil {
			return fmt.Errorf("写入流程提醒失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交流程更新事务失败: %w", err)
	}
	return nil
}

// DeleteProcessFull 删除流程定义及其步骤和提醒
func (r *KBRepo) DeleteProcessFull(resourceID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("开启流程删除事务失败: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM process_reminders WHERE process_id = ?`, resourceID); err != nil {
		return fmt.Errorf("删除流程提醒失败: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM process_steps WHERE resource_id = ?`, resourceID); err != nil {
		return fmt.Errorf("删除流程步骤失败: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM kb_resources WHERE resource_id = ? AND resource_type = 'Process'`, resourceID); err != nil {
		return fmt.Errorf("删除流程资源失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交流程删除事务失败: %w", err)
	}
	return nil
}

// GetPublishedCards 获取已发布的知识卡片（供知识大厅浏览，面向所有已认证用户）
// ownerScope/ownerID: 归属范围过滤（显示全校 + 当前范围）
// role: 用户角色（过滤可见资源）
// resourceType: 可选类型过滤，空字符串表示全部
// limit/offset: 分页参数，limit<=0 表示不分页
// 返回值: 按 resource_type 分组的卡片映射、全部符合条件的资源总数
func (r *KBRepo) GetPublishedCards(ownerScope, ownerID, role, resourceType string, limit, offset int) (map[string][]*model.KnowledgeCard, int, error) {
	whereClause := ` WHERE status = 'published'
		   AND (owner_scope = 'school' OR (owner_scope = 'college' AND (? = '' OR owner_id = ?)) OR (owner_scope = 'class' AND (? = '' OR owner_id = ?)))
		   AND ` + r.roleScopeCond("")
	countArgs := []interface{}{ownerScope, ownerID, ownerScope, ownerID, role, role}
	queryArgs := []interface{}{ownerScope, ownerID, ownerScope, ownerID, role, role}

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
	query := `SELECT resource_id, resource_type, title, summary, tags, remark, source_link
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
			&card.Summary, &card.Tags, &card.Remark, &card.SourceLink,
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

	// 去除 FTS5 保留运算符与特殊字符
	q = escapeFTS(q)
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

// ════════ 高级查询与批量操作 ════════

// ListAdvanced 高级知识资源查询（搜索+多条件筛选+排序+分页）
func (r *KBRepo) ListAdvanced(q *model.KBQuery) ([]*model.KBResource, int, error) {
	baseQuery := ` FROM kb_resources WHERE 1=1`
	countQuery := `SELECT COUNT(*)` + baseQuery
	listQuery := `SELECT id, resource_id, resource_type, owner_scope, owner_id,
			role_scope, version, status, title, summary,
			content, source_link, source_version,
			effective_at, expired_at, tags, remark,
			updated_by, created_at, updated_at` + baseQuery
	var args []interface{}

	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		listQuery += " AND (title LIKE ? OR summary LIKE ? OR content LIKE ? OR tags LIKE ?)"
		countQuery += " AND (title LIKE ? OR summary LIKE ? OR content LIKE ? OR tags LIKE ?)"
		args = append(args, kw, kw, kw, kw)
	}
	if q.ResourceType != "" {
		listQuery += " AND resource_type = ?"
		countQuery += " AND resource_type = ?"
		args = append(args, q.ResourceType)
	}
	if q.Status != "" {
		listQuery += " AND status = ?"
		countQuery += " AND status = ?"
		args = append(args, q.Status)
	}
	if q.OwnerScope != "" {
		listQuery += " AND owner_scope = ?"
		countQuery += " AND owner_scope = ?"
		args = append(args, q.OwnerScope)
		if q.OwnerID != "" {
			listQuery += " AND owner_id = ?"
			countQuery += " AND owner_id = ?"
			args = append(args, q.OwnerID)
		}
	}
	if q.UpdatedBy != "" {
		listQuery += " AND updated_by = ?"
		countQuery += " AND updated_by = ?"
		args = append(args, q.UpdatedBy)
	}
	if q.Tag != "" {
		listQuery += " AND tags LIKE ?"
		countQuery += " AND tags LIKE ?"
		args = append(args, "%"+q.Tag+"%")
	}

	// 排序
	sortBy := q.SortBy
	if sortBy == "" {
		sortBy = "updated_at"
	}
	sortOrder := strings.ToUpper(q.SortOrder)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}
	validSortCols := map[string]bool{
		"updated_at": true, "created_at": true, "title": true,
		"resource_type": true, "status": true, "version": true,
	}
	if !validSortCols[sortBy] {
		sortBy = "updated_at"
	}
	listQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// 分页
	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	listQuery += " LIMIT ? OFFSET ?"
	listArgs := append([]interface{}{}, args...)
	listArgs = append(listArgs, pageSize, offset)

	// 查询总数
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计知识资源数量失败: %w", err)
	}

	// 查询列表
	rows, err := r.db.Query(listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询知识资源列表失败: %w", err)
	}
	defer rows.Close()

	var list []*model.KBResource
	for rows.Next() {
		kb := &model.KBResource{}
		if err := rows.Scan(
			&kb.ID, &kb.ResourceID, &kb.ResourceType, &kb.OwnerScope, &kb.OwnerID,
			&kb.RoleScope, &kb.Version, &kb.Status, &kb.Title, &kb.Summary,
			&kb.Content, &kb.SourceLink, &kb.SourceVersion,
			&kb.EffectiveAt, &kb.ExpiredAt, &kb.Tags, &kb.Remark,
			&kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, kb)
	}
	return list, total, rows.Err()
}

// GetDistinctValues 获取某列的去重值（用于筛选下拉）
func (r *KBRepo) GetDistinctValues(column string) ([]string, error) {
	validCols := map[string]bool{
		"resource_type": true, "status": true, "owner_scope": true,
		"updated_by": true, "version": true,
	}
	if !validCols[column] {
		return nil, fmt.Errorf("不支持的列: %s", column)
	}

	rows, err := r.db.Query(
		fmt.Sprintf("SELECT DISTINCT %s FROM kb_resources WHERE %s != '' ORDER BY %s", column, column, column),
	)
	if err != nil {
		return nil, fmt.Errorf("查询去重值失败: %w", err)
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			continue
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

// BatchUpdateStatus 批量更新知识资源状态
func (r *KBRepo) BatchUpdateStatus(resourceIDs []string, status string, operator string) (int64, error) {
	if len(resourceIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(resourceIDs))
	args := make([]interface{}, 0, len(resourceIDs)+2)
	args = append(args, status, operator)
	for i, id := range resourceIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(
		"UPDATE kb_resources SET status = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP WHERE resource_id IN (%s)",
		strings.Join(placeholders, ","),
	)
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("批量更新状态失败: %w", err)
	}
	return result.RowsAffected()
}

// BatchDelete 批量删除知识资源
func (r *KBRepo) BatchDelete(resourceIDs []string) (int64, error) {
	if len(resourceIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(resourceIDs))
	args := make([]interface{}, 0, len(resourceIDs))
	for i, id := range resourceIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(
		"DELETE FROM kb_resources WHERE resource_id IN (%s)",
		strings.Join(placeholders, ","),
	)
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("批量删除失败: %w", err)
	}
	return result.RowsAffected()
}

// GetStats 获取知识资源统计
func (r *KBRepo) GetStats() (*KBStats, error) {
	stats := &KBStats{
		ByType: make(map[string]int),
	}

	// 按状态统计
	rows, err := r.db.Query(`SELECT status, COUNT(*) FROM kb_resources GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("统计状态失败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		stats.Total += count
		switch status {
		case "draft":
			stats.Draft = count
		case "pending":
			stats.Pending = count
		case "published":
			stats.Published = count
		case "retired":
			stats.Retired = count
		}
	}
	defer rows.Close()

	// 按类型统计
	typeRows, err := r.db.Query(`SELECT resource_type, COUNT(*) FROM kb_resources GROUP BY resource_type`)
	if err != nil {
		return stats, nil
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var t string
		var count int
		if err := typeRows.Scan(&t, &count); err != nil {
			continue
		}
		stats.ByType[t] = count
	}
	return stats, nil
}
