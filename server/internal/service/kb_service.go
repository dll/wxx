package service

import (
	"fmt"
	"log"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/google/uuid"
)

// KBService 知识库管理业务服务
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
func (s *KBService) Browse(ownerScope, ownerID, role, resourceType string) (map[string][]*model.KnowledgeCard, error) {
	cards, err := s.kbRepo.GetPublishedCards(ownerScope, ownerID, role, resourceType)
	if err != nil {
		return nil, fmt.Errorf("获取知识大厅数据失败: %w", err)
	}
	return cards, nil
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
