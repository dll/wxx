package service

import (
	"fmt"
	"sort"
	"strings"

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

// MatchCompetitions 基于学生专业/学院对真实竞赛做个性化匹配排序。
// 数据来源为 competitions 表（真实数据），按类别与专业关键词打分，仅返回可报名/进行中的赛事。
// 无匹配或库为空时返回空列表，由 handler 决定是否回落。
func (s *StudentFeaturesService) MatchCompetitions(major, college string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 10
	}
	// 拉取较大范围候选（up to 100），在内存里按相关度排序
	items, _, err := s.repo.ListCompetitions("", "", "", 1, 100)
	if err != nil {
		return nil, fmt.Errorf("MatchCompetitions: %w", err)
	}

	// 专业/学院文本 → 类别偏好关键词
	profile := strings.ToLower(major + " " + college)
	categoryHints := map[string][]string{
		"programming": {"计算机", "软件", "网络", "信息", "数据", "人工智能", "网络空间", "cs", "coding", "程序"},
		"electronics": {"电子", "通信", "自动化", "物联网", "电气"},
		"math":        {"数学", "统计", "应用数学"},
		"english":     {"英语", "外语", "翻译"},
		"innovation":  {"创新", "创业", "管理", "经济"},
	}

	type scored struct {
		item  map[string]interface{}
		score int
	}
	var ranked []scored
	for _, it := range items {
		// 仅推荐尚可参与的赛事
		st, _ := it["status"].(string)
		if st == "finished" {
			continue
		}
		score := 0
		cat, _ := it["category"].(string)
		// 类别与专业画像匹配
		if hints, ok := categoryHints[cat]; ok {
			for _, kw := range hints {
				if strings.Contains(profile, kw) {
					score += 10
					break
				}
			}
		}
		// 报名中优先
		switch st {
		case "registration", "open":
			score += 5
		case "upcoming":
			score += 2
		}
		// 级别加权：国家级 > 省级 > 市级 > 校级
		switch lev, _ := it["level"].(string); lev {
		case "national":
			score += 3
		case "provincial":
			score += 2
		case "municipal":
			score += 1
		}
		ranked = append(ranked, scored{item: it, score: score})
	}

	// 稳定排序：分数降序，保持原有 competition_date DESC 次序
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	out := make([]map[string]interface{}, 0, limit)
	for i, r := range ranked {
		if i >= limit {
			break
		}
		item := r.item
		item["match_score"] = r.score
		out = append(out, item)
	}
	return out, nil
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

// ── 竞赛管理（管理端）──

// AdminListCompetitions 管理端竞赛列表
func (s *StudentFeaturesService) AdminListCompetitions(level, category, status string, page, pageSize int) ([]map[string]interface{}, int, error) {
	return s.repo.AdminListCompetitions(level, category, status, page, pageSize)
}

// AdminCreateCompetition 新增竞赛
func (s *StudentFeaturesService) AdminCreateCompetition(fields map[string]interface{}) (int64, error) {
	return s.repo.AdminCreateCompetition(fields)
}

// AdminUpdateCompetition 更新竞赛
func (s *StudentFeaturesService) AdminUpdateCompetition(id int64, fields map[string]interface{}) error {
	return s.repo.AdminUpdateCompetition(id, fields)
}

// AdminDeleteCompetition 删除竞赛
func (s *StudentFeaturesService) AdminDeleteCompetition(id int64) error {
	return s.repo.AdminDeleteCompetition(id)
}
