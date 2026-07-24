package service

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/google/uuid"
)

// ProcessRecordService 办事流程办理记录业务服务
type ProcessRecordService struct {
	repo   *repository.ProcessRecordRepo
	kbRepo *repository.KBRepo // 可选：用于校准 totalSteps
}

// NewProcessRecordService 创建服务
func NewProcessRecordService(repo *repository.ProcessRecordRepo, kbRepo *repository.KBRepo) *ProcessRecordService {
	return &ProcessRecordService{repo: repo, kbRepo: kbRepo}
}

// flowLabelOf 流程类型显示名映射，前端没传时使用
var flowLabels = map[string]string{
	"enrollment":   "新生入学",
	"graduation":   "毕业离校",
	"leave":        "请假办理",
	"major_change": "转专业",
	"student_loan": "助学贷款",
	"student-loan": "助学贷款",
	"scholarship":  "奖学金申请",
}

func flowLabelOf(flowType string) string {
	if l, ok := flowLabels[flowType]; ok {
		return l
	}
	return flowType
}

// StartOrResume 开始/恢复某流程的办理（同一用户同一流程类型仅一条进行中记录）
func (s *ProcessRecordService) StartOrResume(userID int64, flowType, flowLabel string, totalSteps int) (*model.ProcessRecord, error) {
	if flowType == "" {
		return nil, fmt.Errorf("flow_type 不能为空")
	}

	// 以后端 process_steps 实际行数为准，校准 totalSteps（避免前端传 0 或错误值）
	if s.kbRepo != nil {
		if resourceID := flowTypeToResourceID(flowType); resourceID != "" {
			if realCount, err := s.kbRepo.CountProcessSteps(resourceID); err == nil && realCount > 0 {
				totalSteps = realCount
			}
		}
	}

	existing, err := s.repo.GetActiveByUserFlow(userID, flowType)
	if err != nil {
		return nil, fmt.Errorf("查询流程记录失败: %w", err)
	}
	if existing != nil {
		// 已有进行中记录，更新 totalSteps 并返回
		if totalSteps > 0 && existing.TotalSteps != totalSteps {
			existing.TotalSteps = totalSteps
		}
		if flowLabel != "" && existing.FlowLabel != flowLabel {
			existing.FlowLabel = flowLabel
		}
		_ = s.repo.Update(existing)
		return existing, nil
	}

	// 创建新记录
	if flowLabel == "" {
		flowLabel = flowLabelOf(flowType)
	}
	rec := &model.ProcessRecord{
		RecordID:       "proc-" + uuid.New().String()[:8],
		UserID:         userID,
		FlowType:       flowType,
		FlowLabel:      flowLabel,
		CurrentStep:    0,
		CompletedSteps: "[]",
		TotalSteps:     totalSteps,
		Status:         "in_progress",
	}
	id, err := s.repo.Create(rec)
	if err != nil {
		return nil, fmt.Errorf("创建办事记录失败: %w", err)
	}
	rec.ID = id
	return rec, nil
}

// UpdateProgress 更新某流程的当前步骤与已完成集合
func (s *ProcessRecordService) UpdateProgress(userID int64, flowType string, currentStep int, completedSteps []int, notes string) (*model.ProcessRecord, error) {
	rec, err := s.repo.GetActiveByUserFlow(userID, flowType)
	if err != nil {
		return nil, fmt.Errorf("查询流程记录失败: %w", err)
	}
	if rec == nil {
		return nil, fmt.Errorf("流程记录不存在，请先开始办理")
	}

	// 已完成步骤排序去重
	if completedSteps == nil {
		completedSteps = []int{}
	}
	uniq := make(map[int]struct{})
	for _, v := range completedSteps {
		uniq[v] = struct{}{}
	}
	sorted := make([]int, 0, len(uniq))
	for v := range uniq {
		sorted = append(sorted, v)
	}
	sort.Ints(sorted)
	bytes, _ := json.Marshal(sorted)

	rec.CurrentStep = currentStep
	rec.CompletedSteps = string(bytes)
	if notes != "" {
		rec.Notes = notes
	}

	// 全部完成则置为 completed
	if rec.TotalSteps > 0 && len(sorted) >= rec.TotalSteps {
		rec.Status = "completed"
	} else {
		rec.Status = "in_progress"
	}

	if err := s.repo.Update(rec); err != nil {
		return nil, fmt.Errorf("更新流程记录失败: %w", err)
	}
	return rec, nil
}

// ListMine 查询当前用户的全部办事记录
func (s *ProcessRecordService) ListMine(userID int64, limit int) ([]*model.ProcessRecord, error) {
	return s.repo.ListByUser(userID, limit)
}

// flowTypeToResourceID 将前端流程类型映射到 KB resource_id（与 student_handler.mapFlowToResource 保持一致）
func flowTypeToResourceID(flowType string) string {
	switch flowType {
	case "graduation":
		return "process-graduation-2026"
	case "major-transfer", "major_transfer", "major_change":
		return "process-major-change-2026"
	case "student-loan", "student_loan":
		return "process-student-loan-2026"
	case "leave":
		return "process-leave-2026"
	case "scholarship":
		return "process-scholarship-2026"
	case "enrollment":
		return "process-registration-2026"
	default:
		return ""
	}
}
