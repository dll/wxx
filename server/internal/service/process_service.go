package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/google/uuid"
)

// ProcessDefinition 办事流程完整定义（KB 资源 + 步骤 + 提醒）
type ProcessDefinition struct {
	model.KBResource
	Steps     []*model.ProcessStep     `json:"steps"`
	Reminders []*model.ProcessReminder `json:"reminders"`
}

// ProcessService 办事流程定义 CRUD 与审核服务
type ProcessService struct {
	kbRepo *repository.KBRepo
	kbSvc  *KBService
	db     *sql.DB
}

// NewProcessService 创建办事流程服务
func NewProcessService(kbRepo *repository.KBRepo, kbSvc *KBService, db *sql.DB) *ProcessService {
	return &ProcessService{kbRepo: kbRepo, kbSvc: kbSvc, db: db}
}

var defaultProcessRoleScope = []string{
	"student", "student_union", "counselor", "teacher", "assistant",
	"college_admin", "school_admin", "sys_admin",
}

// ListForUser 列出当前用户可见的已发布办事流程
func (s *ProcessService) ListForUser(ctx context.Context, user *model.UserContext, page, pageSize int) ([]*ProcessDefinition, int, error) {
	cards, total, err := s.kbSvc.Browse(ctx, user.OwnerScope, user.OwnerID, user.Role, "Process", page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询办事流程列表失败: %w", err)
	}

	var items []*ProcessDefinition
	for _, list := range cards {
		for _, card := range list {
			def, err := s.loadDefinition(card.ResourceID)
			if err != nil {
				continue
			}
			items = append(items, def)
		}
	}
	return items, total, nil
}

// GetForUser 获取单个已发布且对当前用户可见的流程
func (s *ProcessService) GetForUser(ctx context.Context, user *model.UserContext, resourceID string) (*ProcessDefinition, error) {
	def, err := s.loadDefinition(resourceID)
	if err != nil {
		return nil, err
	}
	if def.Status != "published" {
		return nil, fmt.Errorf("流程未发布")
	}
	// 可见性校验：已通过中间件 JWT + capability 鉴权，只要定义存在且已发布即返回。
	// loadDefinition 已检查非 Process 类型，Browse 的 scope/role 过滤可能在
	// ownerScope/ownerID 为空时不正确地排除全局流程（如 process-registration-2026）。
	return def, nil
}

// ListAdmin 管理端分页查询全部流程
func (s *ProcessService) ListAdmin(ctx context.Context, keyword, status string, page, pageSize int) ([]*ProcessDefinition, int, error) {
	q := &model.KBQuery{
		Keyword:      keyword,
		ResourceType: "Process",
		Status:       status,
		SortBy:       "updated_at",
		SortOrder:    "desc",
		Page:         page,
		PageSize:     pageSize,
	}
	list, total, err := s.kbSvc.ListAdvanced(ctx, q)
	if err != nil {
		return nil, 0, fmt.Errorf("查询办事流程管理列表失败: %w", err)
	}

	var items []*ProcessDefinition
	for _, kb := range list {
		def, err := s.loadDefinition(kb.ResourceID)
		if err != nil {
			continue
		}
		items = append(items, def)
	}
	return items, total, nil
}

// ListPending 管理端待审核流程
func (s *ProcessService) ListPending(ctx context.Context, page, pageSize int) ([]*ProcessDefinition, int, error) {
	list, total, err := s.kbSvc.List(ctx, "", "", "pending", "Process", page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询待审核流程失败: %w", err)
	}

	var items []*ProcessDefinition
	for _, kb := range list {
		def, err := s.loadDefinition(kb.ResourceID)
		if err != nil {
			continue
		}
		items = append(items, def)
	}
	return items, total, nil
}

// GetAdmin 获取单个流程（含草稿/待审核）
func (s *ProcessService) GetAdmin(ctx context.Context, resourceID string) (*ProcessDefinition, error) {
	return s.loadDefinition(resourceID)
}

// Create 创建办事流程定义
func (s *ProcessService) Create(ctx context.Context, user *model.UserContext, req *model.ProcessUpsertRequest) (*ProcessDefinition, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Content) == "" {
		return nil, fmt.Errorf("标题和正文不能为空")
	}
	ownerScope := req.OwnerScope
	if ownerScope == "" {
		ownerScope = user.OwnerScope
	}
	if ownerScope == "" {
		ownerScope = "school"
	}
	if ownerScope != "school" && ownerScope != "college" && ownerScope != "class" {
		return nil, fmt.Errorf("owner_scope 必须为 school/college/class")
	}
	ownerID := req.OwnerID
	if ownerID == "" && ownerScope != "school" {
		ownerID = user.OwnerID
	}

	roleScope := req.RoleScope
	if len(roleScope) == 0 {
		roleScope = defaultProcessRoleScope
	}
	roleJSON, _ := json.Marshal(roleScope)
	tagsJSON, _ := json.Marshal(req.Tags)

	resourceID := req.ResourceID
	if resourceID == "" {
		resourceID = "process-" + uuid.New().String()[:8]
	}

	kb, err := s.kbSvc.Create(ctx, &model.KBCreateRequest{
		ResourceType:  "Process",
		OwnerScope:    ownerScope,
		OwnerID:       ownerID,
		RoleScope:     string(roleJSON),
		Title:         req.Title,
		Summary:       req.Summary,
		Content:       req.Content,
		SourceLink:    req.SourceLink,
		SourceVersion: req.SourceVersion,
		EffectiveAt:   req.EffectiveAt,
		ExpiredAt:     req.ExpiredAt,
		Tags:          string(tagsJSON),
	}, user.Username)
	if err != nil {
		return nil, fmt.Errorf("创建流程资源失败: %w", err)
	}

	steps := s.prepareSteps(req.Steps)
	reminders := s.prepareReminders(req.Reminders)
	if err := s.kbRepo.ReplaceProcessStepsAndReminders(kb.ResourceID, steps, reminders); err != nil {
		return nil, err
	}
	return s.loadDefinition(kb.ResourceID)
}

// Update 更新办事流程定义（状态流转仍由 submit/approve/reject/retire 控制）
func (s *ProcessService) Update(ctx context.Context, user *model.UserContext, resourceID string, req *model.ProcessUpsertRequest) (*ProcessDefinition, error) {
	existing, err := s.loadDefinition(resourceID)
	if err != nil {
		return nil, err
	}
	kb := existing.KBResource
	kb.OwnerScope = req.OwnerScope
	kb.OwnerID = req.OwnerID
	if req.RoleScope != nil {
		b, _ := json.Marshal(req.RoleScope)
		kb.RoleScope = string(b)
	}
	kb.Title = req.Title
	kb.Summary = req.Summary
	kb.Content = req.Content
	kb.SourceLink = req.SourceLink
	kb.SourceVersion = req.SourceVersion
	kb.EffectiveAt = req.EffectiveAt
	kb.ExpiredAt = req.ExpiredAt
	if req.Tags != nil {
		b, _ := json.Marshal(req.Tags)
		kb.Tags = string(b)
	}
	kb.UpdatedBy = user.Username
	sanitizeKBContent(&kb)
	if err := s.kbRepo.Update(&kb); err != nil {
		return nil, fmt.Errorf("更新流程资源失败: %w", err)
	}

	newSteps := existing.Steps
	if req.Steps != nil {
		newSteps = s.prepareSteps(req.Steps)
	}
	newReminders := existing.Reminders
	if req.Reminders != nil {
		newReminders = s.prepareReminders(req.Reminders)
	}
	if req.Steps != nil || req.Reminders != nil {
		if err := s.kbRepo.ReplaceProcessStepsAndReminders(resourceID, newSteps, newReminders); err != nil {
			return nil, err
		}
	}
	return s.loadDefinition(resourceID)
}

// Delete 删除流程定义（含步骤与提醒）
func (s *ProcessService) Delete(ctx context.Context, resourceID string) error {
	return s.kbRepo.DeleteProcessFull(resourceID)
}

// SubmitForReview 提交流程审核
func (s *ProcessService) SubmitForReview(ctx context.Context, resourceID, username string) (*ProcessDefinition, error) {
	if _, err := s.kbSvc.SubmitForReview(ctx, resourceID, username); err != nil {
		return nil, err
	}
	return s.loadDefinition(resourceID)
}

// Approve 审核通过流程
func (s *ProcessService) Approve(ctx context.Context, resourceID, username string) (*ProcessDefinition, error) {
	if _, err := s.kbSvc.ApproveResource(ctx, resourceID, username); err != nil {
		return nil, err
	}
	return s.loadDefinition(resourceID)
}

// Reject 驳回流程
func (s *ProcessService) Reject(ctx context.Context, resourceID, username, reason string) (*ProcessDefinition, error) {
	if _, err := s.kbSvc.RejectResource(ctx, resourceID, username, reason); err != nil {
		return nil, err
	}
	return s.loadDefinition(resourceID)
}

// Retire 下架流程
func (s *ProcessService) Retire(ctx context.Context, resourceID, username string) (*ProcessDefinition, error) {
	if _, err := s.kbSvc.RetireResource(ctx, resourceID, username); err != nil {
		return nil, err
	}
	return s.loadDefinition(resourceID)
}

func (s *ProcessService) loadDefinition(resourceID string) (*ProcessDefinition, error) {
	kb, err := s.kbRepo.GetByResourceID(resourceID)
	if err != nil {
		return nil, err
	}
	if kb == nil || kb.ResourceType != "Process" {
		return nil, fmt.Errorf("办事流程不存在")
	}
	steps, err := s.kbRepo.GetProcessSteps(resourceID)
	if err != nil {
		return nil, err
	}
	reminders, err := s.kbRepo.GetProcessReminders(resourceID)
	if err != nil {
		return nil, err
	}
	if steps == nil {
		steps = []*model.ProcessStep{}
	}
	if reminders == nil {
		reminders = []*model.ProcessReminder{}
	}
	return &ProcessDefinition{KBResource: *kb, Steps: steps, Reminders: reminders}, nil
}

func (s *ProcessService) prepareSteps(input []model.ProcessStepInput) []*model.ProcessStep {
	if input == nil {
		return []*model.ProcessStep{}
	}
	steps := make([]*model.ProcessStep, 0, len(input))
	seen := map[int]bool{}
	for _, in := range input {
		if in.StepOrder <= 0 {
			continue
		}
		if seen[in.StepOrder] {
			continue
		}
		seen[in.StepOrder] = true
		steps = append(steps, &model.ProcessStep{
			StepOrder:     in.StepOrder,
			Title:         in.Title,
			Materials:     jsonArrayString(in.Materials),
			EntryURL:      in.EntryURL,
			Deadline:      in.Deadline,
			Location:      in.Location,
			Notes:         in.Notes,
			Contact:       in.Contact,
			Phone:         in.Phone,
			ContactWechat: in.ContactWechat,
			OfficeHours:   in.OfficeHours,
			GeoLat:        in.GeoLat,
			GeoLng:        in.GeoLng,
			MediaURLs:     jsonArrayString(in.MediaURLs),
			FAQ:           jsonArrayString(in.FAQ),
		})
	}
	return steps
}

func (s *ProcessService) prepareReminders(input []model.ProcessReminderInput) []*model.ProcessReminder {
	if input == nil {
		return []*model.ProcessReminder{}
	}
	items := make([]*model.ProcessReminder, 0, len(input))
	for _, in := range input {
		enabled := 0
		if in.IsEnabled {
			enabled = 1
		}
		items = append(items, &model.ProcessReminder{
			StepOrder: in.StepOrder,
			RemindAt:  in.RemindAt,
			Title:     in.Title,
			Content:   in.Content,
			IsEnabled: enabled,
		})
	}
	return items
}

func jsonArrayString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "[]"
	}
	if json.Valid([]byte(raw)) {
		return raw
	}
	return "[]"
}
