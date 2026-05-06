package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/google/uuid"
)

// KBService 知识库管理业务服务
// 注：ImportResources 不通过 Temporal 调度（活动通过函数字段注入，避免循环依赖），
// 由调用方控制重试策略。
type KBService struct {
	kbRepo *repository.KBRepo
}

// NewKBService 创建知识库服务
func NewKBService(kbRepo *repository.KBRepo) *KBService {
	return &KBService{kbRepo: kbRepo}
}

// List 分页查询知识资源
func (s *KBService) List(ownerScope, ownerID, status, resourceType string, page, pageSize int) ([]*model.KBResource, int, error) {
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
func (s *KBService) Get(resourceID string) (*model.KBResource, error) {
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
func (s *KBService) Browse(ownerScope, ownerID, role, resourceType string, page, pageSize int) (map[string][]*model.KnowledgeCard, int, error) {
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
func (s *KBService) Create(req *model.KBCreateRequest, username string) (*model.KBResource, error) {
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

	id, err := s.kbRepo.Create(kb)
	if err != nil {
		return nil, fmt.Errorf("创建知识资源失败: %w", err)
	}

	log.Printf("知识资源已创建 resource_id=%s id=%d by=%s", resourceID, id, username)

	// 回查完整记录（包含 created_at 等数据库生成字段）
	return s.kbRepo.GetByResourceID(resourceID)
}

// Update 更新知识资源
func (s *KBService) Update(resourceID string, req *model.KBUpdateRequest, username string) (*model.KBResource, error) {
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

	if err := s.kbRepo.Update(existing); err != nil {
		return nil, fmt.Errorf("更新知识资源失败: %w", err)
	}

	log.Printf("知识资源已更新 resource_id=%s by=%s", resourceID, username)

	// 回查最新记录
	return s.kbRepo.GetByResourceID(resourceID)
}

// ImportResources 导入知识资源（NDJSON 格式，逐行 KBResource JSON）
// 幂等键：(resource_id, version, status)；冲突按高版本覆盖、同版本跳过
func (s *KBService) ImportResources(ndjsonData string, username string) (*model.KBImportResponse, error) {
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

// ExportResources 导出知识资源（无分页，用于同步/备份）
func (s *KBService) ExportResources(resourceType, sinceCursor string) ([]*model.KBResource, error) {
	// 增量查询：通过 SQL WHERE 过滤，避免应用层遍历
	return s.kbRepo.ListSince(resourceType, sinceCursor, 5000)
}
