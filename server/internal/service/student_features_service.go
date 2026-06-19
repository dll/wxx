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
	return s.repo.ListCompetitions(level, category, status, page, pageSize)
}

func (s *StudentFeaturesService) GetCompetition(id int64) (map[string]interface{}, error) {
	return s.repo.GetCompetition(id)
}

func (s *StudentFeaturesService) RegisterCompetition(competitionID, userID int64, studentID, studentName, college, major, className, teamName, teamMembers, advisorName string) (int64, error) {
	return s.repo.RegisterCompetition(competitionID, userID, studentID, studentName, college, major, className, teamName, teamMembers, advisorName)
}

func (s *StudentFeaturesService) GetMyCompetitionRegistrations(userID int64) ([]map[string]interface{}, error) {
	return s.repo.GetMyCompetitionRegistrations(userID)
}

func (s *StudentFeaturesService) SubmitWork(regID int64, workTitle, workDesc, workFileURL string) error {
	return s.repo.SubmitWork(regID, workTitle, workDesc, workFileURL)
}

func (s *StudentFeaturesService) GetCompetitionStats() (map[string]interface{}, error) {
	return s.repo.GetCompetitionStats()
}

// ══════════════════════════════════════════════════════════════
// 大学规划
// ══════════════════════════════════════════════════════════════

func (s *StudentFeaturesService) ListPlanTemplates(category string) ([]map[string]interface{}, error) {
	return s.repo.ListPlanTemplates(category)
}

func (s *StudentFeaturesService) ListMyPlans(userID int64) ([]map[string]interface{}, error) {
	return s.repo.ListMyPlans(userID)
}

func (s *StudentFeaturesService) CreatePlan(userID int64, templateID int, title, category string, academicYear, semester int, goals string) (int64, error) {
	if goals == "" {
		goals = "[]"
	}
	return s.repo.CreatePlan(userID, templateID, title, category, academicYear, semester, goals)
}

func (s *StudentFeaturesService) SubmitPlan(planID int64) error {
	return s.repo.UpdatePlanStatus(planID, "submitted", "")
}

func (s *StudentFeaturesService) ReviewPlan(planID int64, status, comment string) error {
	allowedStatuses := map[string]bool{"approved": true, "rejected": true, "in_progress": true, "completed": true}
	if !allowedStatuses[status] {
		return fmt.Errorf("无效的审核状态: %s", status)
	}
	return s.repo.UpdatePlanStatus(planID, status, comment)
}

// ══════════════════════════════════════════════════════════════
// 入党教育
// ══════════════════════════════════════════════════════════════

func (s *StudentFeaturesService) ListPartyStages() ([]map[string]interface{}, error) {
	return s.repo.ListPartyStages()
}

func (s *StudentFeaturesService) GetMyPartyProgress(userID int64) (map[string]interface{}, error) {
	return s.repo.GetMyPartyProgress(userID)
}

func (s *StudentFeaturesService) UpdatePartyProgress(userID int64, stage, notes string) error {
	allowedStages := map[string]bool{"applicant": true, "activist": true, "development": true, "probation": true, "member": true}
	if !allowedStages[stage] {
		return fmt.Errorf("无效的入党阶段: %s", stage)
	}
	return s.repo.UpdatePartyProgress(userID, stage, notes)
}

func (s *StudentFeaturesService) ListMyStudyRecords(userID int64) ([]map[string]interface{}, error) {
	return s.repo.ListMyStudyRecords(userID)
}

func (s *StudentFeaturesService) AddStudyRecord(userID int64, studyType, title, content string, duration int, studyDate, certificate string) (int64, error) {
	if studyType == "" {
		studyType = "theory"
	}
	return s.repo.AddStudyRecord(userID, studyType, title, content, duration, studyDate, certificate)
}

func (s *StudentFeaturesService) GetPartyStats() (map[string]interface{}, error) {
	return s.repo.GetPartyStats()
}

// ══════════════════════════════════════════════════════════════
// 社团生活
// ══════════════════════════════════════════════════════════════

func (s *StudentFeaturesService) ListClubs(category string, page, pageSize int) ([]map[string]interface{}, int, error) {
	return s.repo.ListClubs(category, page, pageSize)
}

func (s *StudentFeaturesService) GetClub(id int64) (map[string]interface{}, error) {
	return s.repo.GetClub(id)
}

func (s *StudentFeaturesService) JoinClub(clubID, userID int64, studentID, studentName, role string) (int64, error) {
	return s.repo.JoinClub(clubID, userID, studentID, studentName, role)
}

func (s *StudentFeaturesService) GetMyClubs(userID int64) ([]map[string]interface{}, error) {
	return s.repo.GetMyClubs(userID)
}

func (s *StudentFeaturesService) ListClubActivities(clubID int64, status string, page, pageSize int) ([]map[string]interface{}, int, error) {
	return s.repo.ListClubActivities(clubID, status, page, pageSize)
}

func (s *StudentFeaturesService) RegisterClubActivity(activityID, userID int64, studentName string) (int64, error) {
	return s.repo.RegisterClubActivity(activityID, userID, studentName)
}
