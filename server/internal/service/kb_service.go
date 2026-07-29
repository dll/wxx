package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/util"
	"github.com/google/uuid"
)

// validStatuses 合法的知识资源状态
var validStatuses = map[string]bool{
	"draft": true, "pending": true, "published": true, "retired": true,
}

// KBService 知识库管理业务服务
// 注：ImportResources 不通过 Temporal 调度（活动通过函数字段注入，避免循环依赖），
// 由调用方控制重试策略。
type KBService struct {
	kbRepo *repository.KBRepo
	db     *sql.DB
}

// NewKBService 创建知识库服务
func NewKBService(kbRepo *repository.KBRepo, db *sql.DB) *KBService {
	return &KBService{kbRepo: kbRepo, db: db}
}

// List 分页查询知识资源
func (s *KBService) List(ctx context.Context, ownerScope, ownerID, status, resourceType string, page, pageSize int) ([]*model.KBResource, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	list, err := s.kbRepo.List(ownerScope, ownerID, status, resourceType, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询知识列表失败: %w", err)
	}

	total, err := s.kbRepo.Count(ownerScope, ownerID, status, resourceType)
	if err != nil {
		return nil, 0, fmt.Errorf("统计知识总数失败: %w", err)
	}

	return list, total, nil
}

// Get 获取知识资源详情
func (s *KBService) Get(ctx context.Context, resourceID string) (*model.KBResource, error) {
	kb, err := s.kbRepo.GetByResourceID(resourceID)
	if err != nil {
		return nil, fmt.Errorf("查询知识详情失败: %w", err)
	}
	if kb == nil {
		return nil, fmt.Errorf("知识资源不存在: %s", resourceID)
	}
	return kb, nil
}

// Browse 知识大厅浏览（按类型分组，面向所有已认证用户）
// page/pageSize 控制分页；pageSize<=0 时返回全部（不分页）
func (s *KBService) Browse(ctx context.Context, ownerScope, ownerID, role, resourceType string, page, pageSize int) (map[string][]*model.KnowledgeCard, int, error) {
	var limit, offset int
	if pageSize > 0 {
		if page < 1 {
			page = 1
		}
		limit = pageSize
		offset = (page - 1) * pageSize
	}

	cards, total, err := s.kbRepo.GetPublishedCards(ownerScope, ownerID, role, resourceType, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("获取知识大厅数据失败: %w", err)
	}
	return cards, total, nil
}

// Create 创建知识资源
func (s *KBService) Create(ctx context.Context, req *model.KBCreateRequest, username string) (*model.KBResource, error) {
	resourceID := uuid.New().String()

	kb := &model.KBResource{
		ResourceID:    resourceID,
		ResourceType:  req.ResourceType,
		OwnerScope:    req.OwnerScope,
		OwnerID:       req.OwnerID,
		RoleScope:     req.RoleScope,
		Version:       "1.0",
		Status:        "draft",
		Title:         req.Title,
		Summary:       req.Summary,
		Content:       req.Content,
		SourceLink:    req.SourceLink,
		SourceVersion: req.SourceVersion,
		EffectiveAt:   req.EffectiveAt,
		ExpiredAt:     req.ExpiredAt,
		Tags:          req.Tags,
		UpdatedBy:     username,
	}

	sanitizeKBContent(kb)

	id, err := s.kbRepo.Create(kb)
	if err != nil {
		return nil, fmt.Errorf("创建知识资源失败: %w", err)
	}

	log.Printf("知识资源已创建 resource_id=%s id=%d by=%s", resourceID, id, username)

	// 回查完整记录（包含 created_at 等数据库生成字段）
	return s.kbRepo.GetByResourceID(resourceID)
}

// sanitizeKBContent 入库前清洗标题、摘要与正文。
//
// 必须在写库前完成：migrations/001_init.sql 的 FTS5 触发器会在 INSERT/UPDATE
// 时自动索引 title/summary/content（draft 状态也索引），一旦写入即无法阻止污染
// 内容进入 BM25 排序、各 agent 的系统提示词、sources[] 摘要与导出。
//
// FAQ 类型的 content 存放序列化 AnswerCard JSON，按类型跳过标签剥离。
func sanitizeKBContent(kb *model.KBResource) {
	if kb == nil {
		return
	}
	kb.Title = util.SanitizeKnowledgeContent(kb.Title)
	kb.Summary = util.SanitizeKnowledgeContent(kb.Summary)
	kb.Content = util.SanitizeKnowledgeContentByType(kb.Content, kb.ResourceType)
}

// Update 更新知识资源
func (s *KBService) Update(ctx context.Context, resourceID string, req *model.KBUpdateRequest, username string) (*model.KBResource, error) {
	// 查询现有资源
	existing, err := s.kbRepo.GetByResourceID(resourceID)
	if err != nil {
		return nil, fmt.Errorf("查询知识资源失败: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("知识资源不存在: %s", resourceID)
	}

	// 合并更新字段（仅覆盖非空值）
	if req.ResourceType != "" {
		existing.ResourceType = req.ResourceType
	}
	if req.OwnerScope != "" {
		existing.OwnerScope = req.OwnerScope
	}
	if req.OwnerID != "" {
		existing.OwnerID = req.OwnerID
	}
	if req.RoleScope != "" {
		existing.RoleScope = req.RoleScope
	}
	if req.Status != "" {
		existing.Status = req.Status
	}
	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Summary != "" {
		existing.Summary = req.Summary
	}
	if req.Content != "" {
		existing.Content = req.Content
	}
	if req.SourceLink != "" {
		existing.SourceLink = req.SourceLink
	}
	if req.SourceVersion != "" {
		existing.SourceVersion = req.SourceVersion
	}
	if req.EffectiveAt != nil {
		existing.EffectiveAt = req.EffectiveAt
	}
	if req.ExpiredAt != nil {
		existing.ExpiredAt = req.ExpiredAt
	}
	if req.Tags != "" {
		existing.Tags = req.Tags
	}
	existing.UpdatedBy = username

	sanitizeKBContent(existing)

	if err := s.kbRepo.Update(existing); err != nil {
		return nil, fmt.Errorf("更新知识资源失败: %w", err)
	}

	log.Printf("知识资源已更新 resource_id=%s by=%s", resourceID, username)

	// 回查最新记录
	return s.kbRepo.GetByResourceID(resourceID)
}

// SubmitForReview 提交知识资源进入审核流程（draft → pending）
// 仅限 student_union 及以上角色调用
func (s *KBService) SubmitForReview(ctx context.Context, resourceID, username string) (*model.KBResource, error) {
	existing, err := s.kbRepo.GetByResourceID(resourceID)
	if err != nil {
		return nil, fmt.Errorf("查询知识资源失败: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("知识资源不存在: %s", resourceID)
	}
	if existing.Status != "draft" {
		return nil, fmt.Errorf("仅草稿状态可提交审核，当前状态: %s", existing.Status)
	}

	existing.Status = "pending"
	existing.UpdatedBy = username
	if err := s.kbRepo.Update(existing); err != nil {
		return nil, fmt.Errorf("更新状态失败: %w", err)
	}
	log.Printf("知识资源已提交审核 resource_id=%s by=%s", resourceID, username)

	// 发送审核通知给学院管理员
	go s.sendReviewNotification(existing, username)

	return s.kbRepo.GetByResourceID(resourceID)
}

// sendReviewNotification 发送知识审核通知给学院管理员
func (s *KBService) sendReviewNotification(kb *model.KBResource, submitter string) {
	if s.db == nil {
		return
	}
	title := "新知识待审核"
	content := fmt.Sprintf("用户「%s」提交了知识资源「%s」（%s），请及时审核。", submitter, kb.Title, kb.ResourceID)

	// 查询所有学院管理员和学校管理员角色的用户
	rows, err := s.db.Query(
		`SELECT id FROM users WHERE role IN ('college_admin', 'school_admin', 'sys_admin') AND status = 'active'`,
	)
	if err != nil {
		log.Printf("查询审核管理员失败: %v", err)
		return
	}
	defer rows.Close()

	var adminIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		adminIDs = append(adminIDs, id)
	}

	for _, adminID := range adminIDs {
		_, err := s.db.Exec(
			`INSERT INTO user_notifications (user_id, title, content, type, related_type, related_id, is_read)
			 VALUES (?, ?, ?, 'kb_review', 'kb_resource', ?, 0)`,
			adminID, title, content, kb.ID,
		)
		if err != nil {
			log.Printf("发送审核通知失败: admin_id=%d resource_id=%s err=%v", adminID, kb.ResourceID, err)
		}
	}
	log.Printf("审核通知已发送: resource_id=%s admin_count=%d", kb.ResourceID, len(adminIDs))
}

// sendReviewResultNotification 发送审核结果通知给上传者
func (s *KBService) sendReviewResultNotification(kb *model.KBResource, title string, content string) {
	if s.db == nil || kb.UpdatedBy == "" {
		return
	}
	// 根据用户名查找用户ID
	var userID int64
	err := s.db.QueryRow(
		`SELECT id FROM users WHERE username = ?`,
		kb.UpdatedBy,
	).Scan(&userID)
	if err != nil || userID == 0 {
		log.Printf("查找上传者用户失败: username=%s err=%v", kb.UpdatedBy, err)
		return
	}

	_, err = s.db.Exec(
		`INSERT INTO user_notifications (user_id, title, content, type, related_type, related_id, is_read)
		 VALUES (?, ?, ?, 'kb_result', 'kb_resource', ?, 0)`,
		userID, title, content, kb.ID,
	)
	if err != nil {
		log.Printf("发送审核结果通知失败: user_id=%d resource_id=%s err=%v", userID, kb.ResourceID, err)
	}
}

// ApproveResource 审核通过知识资源（pending → published）
// 仅限 counselor 及以上角色调用
func (s *KBService) ApproveResource(ctx context.Context, resourceID, username string) (*model.KBResource, error) {
	existing, err := s.kbRepo.GetByResourceID(resourceID)
	if err != nil {
		return nil, fmt.Errorf("查询知识资源失败: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("知识资源不存在: %s", resourceID)
	}
	if existing.Status != "pending" {
		return nil, fmt.Errorf("仅待审核状态可批准，当前状态: %s", existing.Status)
	}

	existing.Status = "published"
	existing.UpdatedBy = username
	if err := s.kbRepo.Update(existing); err != nil {
		return nil, fmt.Errorf("更新状态失败: %w", err)
	}
	log.Printf("知识资源审核通过 resource_id=%s by=%s", resourceID, username)

	// 通知上传者审核通过
	go s.sendReviewResultNotification(existing, "审核通过", fmt.Sprintf("您提交的知识资源「%s」已通过审核。", existing.Title))

	return s.kbRepo.GetByResourceID(resourceID)
}

// RejectResource 驳回知识资源（pending → draft），附带驳回理由
// 仅限 counselor 及以上角色调用
func (s *KBService) RejectResource(ctx context.Context, resourceID, username, reason string) (*model.KBResource, error) {
	existing, err := s.kbRepo.GetByResourceID(resourceID)
	if err != nil {
		return nil, fmt.Errorf("查询知识资源失败: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("知识资源不存在: %s", resourceID)
	}
	if existing.Status != "pending" {
		return nil, fmt.Errorf("仅待审核状态可驳回，当前状态: %s", existing.Status)
	}

	existing.Status = "draft"
	existing.UpdatedBy = username
	if err := s.kbRepo.Update(existing); err != nil {
		return nil, fmt.Errorf("更新状态失败: %w", err)
	}
	log.Printf("知识资源已驳回 resource_id=%s by=%s reason=%s", resourceID, username, reason)

	// 通知上传者审核驳回
	go s.sendReviewResultNotification(existing, "审核驳回", fmt.Sprintf("您提交的知识资源「%s」被驳回，原因：%s。", existing.Title, reason))

	return s.kbRepo.GetByResourceID(resourceID)
}

// RetireResource 下架知识资源（published → retired）
// 仅限 counselor 及以上角色调用
func (s *KBService) RetireResource(ctx context.Context, resourceID, username string) (*model.KBResource, error) {
	existing, err := s.kbRepo.GetByResourceID(resourceID)
	if err != nil {
		return nil, fmt.Errorf("查询知识资源失败: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("知识资源不存在: %s", resourceID)
	}
	if existing.Status != "published" {
		return nil, fmt.Errorf("仅已发布状态可下架，当前状态: %s", existing.Status)
	}

	existing.Status = "retired"
	existing.UpdatedBy = username
	if err := s.kbRepo.Update(existing); err != nil {
		return nil, fmt.Errorf("更新状态失败: %w", err)
	}
	log.Printf("知识资源已下架 resource_id=%s by=%s", resourceID, username)
	return s.kbRepo.GetByResourceID(resourceID)
}

// ImportResources 导入知识资源（NDJSON 格式，逐行 KBResource JSON）
// 幂等键：(resource_id, version, status)；冲突按高版本覆盖、同版本跳过
func (s *KBService) ImportResources(ctx context.Context, ndjsonData string, username string) (*model.KBImportResponse, error) {
	lines := strings.Split(strings.TrimSpace(ndjsonData), "\n")
	results := make([]*model.KBImportResult, 0, len(lines))
	var created, updated, skipped int

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var kb model.KBResource
		if err := json.Unmarshal([]byte(line), &kb); err != nil {
			results = append(results, &model.KBImportResult{
				ResourceID: fmt.Sprintf("line-%d", i+1),
				Title:      "",
				Action:     "skipped",
				Message:    fmt.Sprintf("JSON 解析失败: %v", err),
			})
			skipped++
			continue
		}

		// 校验必填字段
		if kb.ResourceID == "" || kb.Title == "" || kb.Content == "" || kb.ResourceType == "" {
			results = append(results, &model.KBImportResult{
				ResourceID: kb.ResourceID,
				Title:      kb.Title,
				Action:     "skipped",
				Message:    "缺少必填字段 (resource_id / title / content / resource_type)",
			})
			skipped++
			continue
		}

		// 校验资源类型
		if kb.ResourceType != "Policy" && kb.ResourceType != "Process" && kb.ResourceType != "FAQ" && kb.ResourceType != "Activity" {
			results = append(results, &model.KBImportResult{
				ResourceID: kb.ResourceID,
				Title:      kb.Title,
				Action:     "skipped",
				Message:    fmt.Sprintf("无效资源类型: %s", kb.ResourceType),
			})
			skipped++
			continue
		}

		// 设置导入者
		kb.UpdatedBy = username

		// 幂等导入
		_, action, err := s.kbRepo.Upsert(&kb)
		if err != nil {
			results = append(results, &model.KBImportResult{
				ResourceID: kb.ResourceID,
				Title:      kb.Title,
				Action:     "skipped",
				Message:    fmt.Sprintf("写入失败: %v", err),
			})
			skipped++
			continue
		}

		results = append(results, &model.KBImportResult{
			ResourceID: kb.ResourceID,
			Title:      kb.Title,
			Action:     action,
			Message:    fmt.Sprintf("操作成功: %s", action),
		})

		switch action {
		case "created":
			created++
		case "updated":
			updated++
		default:
			skipped++
		}
	}

	log.Printf("知识导入完成 total=%d created=%d updated=%d skipped=%d by=%s", len(results), created, updated, skipped, username)

	return &model.KBImportResponse{
		Code:    0,
		Message: "导入完成",
		Data:    results,
		Total:   len(results),
		Created: created,
		Updated: updated,
		Skipped: skipped,
	}, nil
}

// ListPending 查询所有待审核知识资源
func (s *KBService) ListPending(ctx context.Context, page, pageSize int) ([]*model.KBResource, int, error) {
	return s.List(ctx, "", "", "pending", "", page, pageSize)
}

// ExportResources 导出知识资源（无分页，用于同步/备份）
// 安全修复 RB-01：callerScope/callerOwnerID 由 handler 从 JWT 上下文注入，服务层按调用者数据范围过滤
func (s *KBService) ExportResources(ctx context.Context, resourceType, sinceCursor, callerScope, callerOwnerID string) ([]*model.KBResource, error) {
	// 增量查询：通过 SQL WHERE 过滤，避免应用层遍历
	return s.kbRepo.ListSince(resourceType, sinceCursor, callerScope, callerOwnerID, 5000)
}

// ════════ 高级查询与批量操作 ════════

// ListAdvanced 高级知识资源查询（搜索+多条件筛选+排序+分页）
func (s *KBService) ListAdvanced(ctx context.Context, q *model.KBQuery) ([]*model.KBResource, int, error) {
	if q == nil {
		q = &model.KBQuery{}
	}
	list, total, err := s.kbRepo.ListAdvanced(q)
	if err != nil {
		return nil, 0, fmt.Errorf("高级查询知识资源失败: %w", err)
	}
	return list, total, nil
}

// GetDictValues 获取字典值（用于筛选下拉）
func (s *KBService) GetDictValues(ctx context.Context, column string) ([]string, error) {
	values, err := s.kbRepo.GetDistinctValues(column)
	if err != nil {
		return nil, fmt.Errorf("获取字典值失败: %w", err)
	}
	return values, nil
}

// BatchUpdateStatus 批量更新知识资源状态
func (s *KBService) BatchUpdateStatus(ctx context.Context, resourceIDs []string, status string, operator string) (int64, error) {
	if len(resourceIDs) == 0 {
		return 0, fmt.Errorf("资源ID列表不能为空")
	}
	if !validStatuses[status] {
		return 0, fmt.Errorf("无效的状态值: %s", status)
	}
	count, err := s.kbRepo.BatchUpdateStatus(resourceIDs, status, operator)
	if err != nil {
		return 0, fmt.Errorf("批量更新状态失败: %w", err)
	}
	log.Printf("批量更新知识资源状态 count=%d status=%s by=%s", count, status, operator)
	return count, nil
}

// BatchDelete 批量删除知识资源
func (s *KBService) BatchDelete(ctx context.Context, resourceIDs []string, operator string) (int64, error) {
	if len(resourceIDs) == 0 {
		return 0, fmt.Errorf("资源ID列表不能为空")
	}
	count, err := s.kbRepo.BatchDelete(resourceIDs)
	if err != nil {
		return 0, fmt.Errorf("批量删除失败: %w", err)
	}
	log.Printf("批量删除知识资源 count=%d by=%s", count, operator)
	return count, nil
}

// BatchApprove 批量审核通过
func (s *KBService) BatchApprove(ctx context.Context, resourceIDs []string, operator string) (int64, error) {
	return s.BatchUpdateStatus(ctx, resourceIDs, "published", operator)
}

// BatchReject 批量驳回
func (s *KBService) BatchReject(ctx context.Context, resourceIDs []string, operator string) (int64, error) {
	return s.BatchUpdateStatus(ctx, resourceIDs, "draft", operator)
}

// BatchRetire 批量下架
func (s *KBService) BatchRetire(ctx context.Context, resourceIDs []string, operator string) (int64, error) {
	return s.BatchUpdateStatus(ctx, resourceIDs, "retired", operator)
}

// GetStats 获取知识资源统计
func (s *KBService) GetStats(ctx context.Context) (*repository.KBStats, error) {
	stats, err := s.kbRepo.GetStats()
	if err != nil {
		return nil, fmt.Errorf("获取统计数据失败: %w", err)
	}
	return stats, nil
}
