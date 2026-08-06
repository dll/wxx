package service

import (
	"fmt"
	"log"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// GraduationService 毕设选题业务服务
type GraduationService struct {
	graduationRepo *repository.GraduationRepo
}

// NewGraduationService 创建毕设选题服务
func NewGraduationService(graduationRepo *repository.GraduationRepo) *GraduationService {
	return &GraduationService{graduationRepo: graduationRepo}
}

// ListAdvisors 获取导师列表
func (s *GraduationService) ListAdvisors(college string, page, pageSize int) ([]*model.Advisor, int, error) {
	return s.graduationRepo.ListAdvisors(college, page, pageSize)
}

// ListTopics 获取选题列表
func (s *GraduationService) ListTopics(college, major, difficulty, status string, batch, page, pageSize int) ([]*model.ThesisTopic, int, error) {
	return s.graduationRepo.ListTopics(college, major, difficulty, status, batch, page, pageSize)
}

// GetTopic 获取选题详情
func (s *GraduationService) GetTopic(id int64) (*model.ThesisTopic, error) {
	return s.graduationRepo.GetTopic(id)
}

// GetMySelection 获取我的选题
func (s *GraduationService) GetMySelection(userID int64) (*model.StudentTopicSelection, error) {
	selection, err := s.graduationRepo.GetUserSelection(userID)
	if err != nil {
		return nil, err
	}
	return selection, nil
}

// SelectTopic 学生选题（含业务校验）
func (s *GraduationService) SelectTopic(userID int64, username, displayName, ownerScope string, topicID int64, reason string) (int64, error) {
	// 检查是否已有选题记录
	existing, err := s.graduationRepo.GetUserSelection(userID)
	if err != nil {
		return 0, fmt.Errorf("查询已有选题失败: %w", err)
	}
	if existing != nil && existing.Status != "changed" {
		return 0, fmt.Errorf("已有选题记录，如需改题请联系导师")
	}

	// 获取选题信息
	topic, err := s.graduationRepo.GetTopic(topicID)
	if err != nil {
		return 0, fmt.Errorf("获取选题信息失败: %w", err)
	}

	// 检查选题是否已满
	if topic.SelectedCount >= topic.MaxStudents {
		return 0, fmt.Errorf("该选题已满，请选择其他题目")
	}

	selection := &model.StudentTopicSelection{
		UserID:          userID,
		StudentID:       username,
		StudentName:     displayName,
		College:         ownerScope,
		Batch:           2026,
		TopicID:         topicID,
		AdvisorID:       topic.AdvisorID,
		Status:          "pending",
		PreferenceOrder: 1,
		Reason:          reason,
	}

	id, err := s.graduationRepo.CreateSelection(selection)
	if err != nil {
		log.Printf("选题失败: %v", err)
		return 0, fmt.Errorf("选题操作失败: %w", err)
	}
	return id, nil
}

// ListMilestones 获取里程碑列表
func (s *GraduationService) ListMilestones(batch int) ([]*model.GraduationMilestone, error) {
	return s.graduationRepo.ListMilestones(batch)
}

// GetStats 获取选题统计
func (s *GraduationService) GetStats(batch int) (map[string]interface{}, error) {
	return s.graduationRepo.GetTopicStats(batch)
}

// ListSelections 获取选题记录列表（管理员）
func (s *GraduationService) ListSelections(topicID int64, batch, page, pageSize int) ([]*model.StudentTopicSelection, int, error) {
	return s.graduationRepo.ListSelections(topicID, batch, page, pageSize)
}

// ConfirmSelection 确认选题
func (s *GraduationService) ConfirmSelection(id int64) error {
	return s.graduationRepo.UpdateSelectionStatus(id, "confirmed")
}

// CreateTopic 新增毕设选题（管理端）
func (s *GraduationService) CreateTopic(t *model.ThesisTopic) (int64, error) {
	return s.graduationRepo.CreateTopic(t)
}

// UpdateTopic 更新毕设选题（管理端）
func (s *GraduationService) UpdateTopic(id int64, fields map[string]interface{}) error {
	return s.graduationRepo.UpdateTopic(id, fields)
}

// DeleteTopic 删除毕设选题（管理端）
func (s *GraduationService) DeleteTopic(id int64) error {
	return s.graduationRepo.DeleteTopic(id)
}

