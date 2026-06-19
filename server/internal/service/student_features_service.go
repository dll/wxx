package service

import (
	"fmt"

	"github.com/dll/wxx/server/internal/repository"
)

// StudentFeaturesService 学生功能业务服务（竞赛+规划+入党+社团）
type StudentFeaturesService struct {
	repo *repository.StudentFeaturesRepo
}

// NewStudentFeaturesService 创建学生功能服务
func NewStudentFeaturesService(repo *repository.StudentFeaturesRepo) *StudentFeaturesService {
	return &StudentFeaturesService{repo: repo}
}

// ══════════════════════════════════════════════════════════════
// 学科竞赛
// ══════════════════════════════════════════════════════════════

func (s *StudentFeaturesService) ListCompetitions(level, category, status string, page, pageSize int) ([]map[string]interface{}, int, error) {
	items, total, err := s.repo.ListCompetitions(level, category, status, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("ListCompetitions: %w", err)
	}
	return items, total, nil
}

func (s *StudentFeaturesService) GetCompetition(id int64) (map[string]interface{}, error) {
	item, err := s.repo.GetCompetition(id)
	if err != nil {
		return nil, fmt.Errorf("GetCompetition: %w", err)
	}
	return item, nil
}

func (s *StudentFeaturesService) RegisterCompetition(competitionID, userID int64, studentID, studentName, college, major, className, teamName, teamMembers, advisorName string) (int64, error) {
	id, err := s.repo.RegisterCompetition(competitionID, userID, studentID, studentName, college, major, className, teamName, teamMembers, advisorName)
	if err != nil {
		return 0, fmt.Errorf("RegisterCompetition: %w", err)
	}
	return id, nil
}

func (s *StudentFeaturesService) GetMyCompetitionRegistrations(userID int64) ([]map[string]interface{}, error) {
	items, err := s.repo.GetMyCompetitionRegistrations(userID)
	if err != nil {
		return nil, fmt.Errorf("GetMyCompetitionRegistrations: %w", err)
	}
	return items, nil
}

func (s *StudentFeaturesService) SubmitWork(regID int64, workTitle, workDesc, workFileURL string) error {
	if err := s.repo.SubmitWork(regID, workTitle, workDesc, workFileURL); err != nil {
		return fmt.Errorf("SubmitWork: %w", err)
	}
	return nil
}

func (s *StudentFeaturesService) GetCompetitionStats() (map[string]interface{}, error) {
	stats, err := s.repo.GetCompetitionStats()
	if err != nil {
		return nil, fmt.Errorf("GetCompetitionStats: %w", err)
	}
	return stats, nil
}

// ══════════════════════════════════════════════════════════════
// 大学规划
// ══════════════════════════════════════════════════════════════

func (s *StudentFeaturesService) ListPlanTemplates(category string) ([]map[string]interface{}, error) {
	items, err := s.repo.ListPlanTemplates(category)
	if err != nil {
		return nil, fmt.Errorf("ListPlanTemplates: %w", err)
	}
	return items, nil
}

func (s *StudentFeaturesService) ListMyPlans(userID int64) ([]map[string]interface{}, error) {
	items, err := s.repo.ListMyPlans(userID)
	if err != nil {
		return nil, fmt.Errorf("ListMyPlans: %w", err)
	}
	return items, nil
}

func (s *StudentFeaturesService) CreatePlan(userID int64, templateID int, title, category string, academicYear, semester int, goals string) (int64, error) {
	if goals == "" {
		goals = "[]"
	}
	id, err := s.repo.CreatePlan(userID, templateID, title, category, academicYear, semester, goals)
	if err != nil {
		return 0, fmt.Errorf("CreatePlan: %w", err)
	}
	return id, nil
}

func (s *StudentFeaturesService) SubmitPlan(planID int64) error {
	if err := s.repo.UpdatePlanStatus(planID, "submitted", ""); err != nil {
		return fmt.Errorf("SubmitPlan: %w", err)
	}
	return nil
}

func (s *StudentFeaturesService) ReviewPlan(planID int64, status, comment string) error {
	allowedStatuses := map[string]bool{"approved": true, "rejected": true, "in_progress": true, "completed": true}
	if !allowedStatuses[status] {
		return fmt.Errorf("无效的审核状态: %s", status)
	}
	if err := s.repo.UpdatePlanStatus(planID, status, comment); err != nil {
		return fmt.Errorf("ReviewPlan: %w", err)
	}
	return nil
}

// ══════════════════════════════════════════════════════════════
// 入党教育
// ══════════════════════════════════════════════════════════════

func (s *StudentFeaturesService) ListPartyStages() ([]map[string]interface{}, error) {
	items, err := s.repo.ListPartyStages()
	if err != nil {
		return nil, fmt.Errorf("ListPartyStages: %w", err)
	}
	return items, nil
}

func (s *StudentFeaturesService) GetMyPartyProgress(userID int64) (map[string]interface{}, error) {
	item, err := s.repo.GetMyPartyProgress(userID)
	if err != nil {
		return nil, fmt.Errorf("GetMyPartyProgress: %w", err)
	}
	return item, nil
}

func (s *StudentFeaturesService) UpdatePartyProgress(userID int64, stage, notes string) error {
	allowedStages := map[string]bool{"applicant": true, "activist": true, "development": true, "probation": true, "member": true}
	if !allowedStages[stage] {
		return fmt.Errorf("无效的入党阶段: %s", stage)
	}
	if err := s.repo.UpdatePartyProgress(userID, stage, notes); err != nil {
		return fmt.Errorf("UpdatePartyProgress: %w", err)
	}
	return nil
}

func (s *StudentFeaturesService) ListMyStudyRecords(userID int64) ([]map[string]interface{}, error) {
	items, err := s.repo.ListMyStudyRecords(userID)
	if err != nil {
		return nil, fmt.Errorf("ListMyStudyRecords: %w", err)
	}
	return items, nil
}

func (s *StudentFeaturesService) AddStudyRecord(userID int64, studyType, title, content string, duration int, studyDate, certificate string) (int64, error) {
	if studyType == "" {
		studyType = "theory"
	}
	id, err := s.repo.AddStudyRecord(userID, studyType, title, content, duration, studyDate, certificate)
	if err != nil {
		return 0, fmt.Errorf("AddStudyRecord: %w", err)
	}
	return id, nil
}

func (s *StudentFeaturesService) GetPartyStats() (map[string]interface{}, error) {
	stats, err := s.repo.GetPartyStats()
	if err != nil {
		return nil, fmt.Errorf("GetPartyStats: %w", err)
	}
	return stats, nil
}

// ══════════════════════════════════════════════════════════════
// 社团生活
// ══════════════════════════════════════════════════════════════

func (s *StudentFeaturesService) ListClubs(category string, page, pageSize int) ([]map[string]interface{}, int, error) {
	items, total, err := s.repo.ListClubs(category, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("ListClubs: %w", err)
	}
	return items, total, nil
}

func (s *StudentFeaturesService) GetClub(id int64) (map[string]interface{}, error) {
	item, err := s.repo.GetClub(id)
	if err != nil {
		return nil, fmt.Errorf("GetClub: %w", err)
	}
	return item, nil
}

func (s *StudentFeaturesService) JoinClub(clubID, userID int64, studentID, studentName, role string) (int64, error) {
	id, err := s.repo.JoinClub(clubID, userID, studentID, studentName, role)
	if err != nil {
		return 0, fmt.Errorf("JoinClub: %w", err)
	}
	return id, nil
}

func (s *StudentFeaturesService) GetMyClubs(userID int64) ([]map[string]interface{}, error) {
	items, err := s.repo.GetMyClubs(userID)
	if err != nil {
		return nil, fmt.Errorf("GetMyClubs: %w", err)
	}
	return items, nil
}

func (s *StudentFeaturesService) ListClubActivities(clubID int64, status string, page, pageSize int) ([]map[string]interface{}, int, error) {
	items, total, err := s.repo.ListClubActivities(clubID, status, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("ListClubActivities: %w", err)
	}
	return items, total, nil
}

func (s *StudentFeaturesService) RegisterClubActivity(activityID, userID int64, studentName string) (int64, error) {
	id, err := s.repo.RegisterClubActivity(activityID, userID, studentName)
	if err != nil {
		return 0, fmt.Errorf("RegisterClubActivity: %w", err)
	}
	return id, nil
}
