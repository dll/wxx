package service

import (
	"fmt"

	"github.com/dll/wxx/server/internal/repository"
)

// Phase3Service 阶段三数据底座服务（导入 + 教辅真实数据）
type Phase3Service struct {
	repo *repository.DataImportRepo
}

// NewPhase3Service 创建阶段三服务
func NewPhase3Service(repo *repository.DataImportRepo) *Phase3Service {
	return &Phase3Service{repo: repo}
}

// ImportResult 导入结果
type ImportResult struct {
	Total   int      `json:"total"`
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Errors  []string `json:"errors"`
}

// ImportGrades 批量导入成绩（幂等）
func (s *Phase3Service) ImportGrades(grades []*repository.GradeRow) *ImportResult {
	res := &ImportResult{Total: len(grades)}
	for _, g := range grades {
		if g.UserID == "" || g.CourseID == "" {
			res.Errors = append(res.Errors, "学号/课程ID不能为空")
			continue
		}
		created, err := s.repo.UpsertGrade(g)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s/%s: %v", g.UserID, g.CourseID, err))
			continue
		}
		if created {
			res.Created++
		} else {
			res.Updated++
		}
	}
	return res
}

// ImportSchedules 批量导入课表（幂等）
func (s *Phase3Service) ImportSchedules(rows []*repository.ScheduleRow) *ImportResult {
	res := &ImportResult{Total: len(rows)}
	for _, r := range rows {
		if r.CourseID == "" {
			res.Errors = append(res.Errors, "课程ID不能为空")
			continue
		}
		if err := s.repo.UpsertSchedule(r); err != nil {
			res.Errors = append(res.Errors, err.Error())
			continue
		}
		res.Created++
	}
	return res
}

// ── 教辅真实数据 ──

// GetScheduleConflicts 基于真实课表检测排课冲突（同教室/同教师/同班级 同时段）
func (s *Phase3Service) GetScheduleConflicts(semester string) (int, []map[string]interface{}, error) {
	schedules, err := s.repo.ListSchedules(semester)
	if err != nil {
		return 0, nil, err
	}
	var conflicts []map[string]interface{}
	// 按 (weekday, start_period) 分组检测
	slotMap := map[string][]map[string]interface{}{}
	for _, s := range schedules {
		key := fmt.Sprintf("w%v-p%v", s["weekday"], s["start_period"])
		slotMap[key] = append(slotMap[key], s)
	}
	for key, items := range slotMap {
		if len(items) < 2 {
			continue
		}
		// 同教室 / 同教师 冲突
		for i := 0; i < len(items); i++ {
			for j := i + 1; j < len(items); j++ {
				a, b := items[i], items[j]
				if a["location"] == b["location"] && a["location"] != nil && a["location"] != "" {
					conflicts = append(conflicts, map[string]interface{}{
						"type": "教室冲突", "severity": "high",
						"description": fmt.Sprintf("%v 与 %v 在同一时段(%s)使用同一教室 %v",
							a["course_name"], b["course_name"], key, a["location"]),
					})
				}
				if a["teacher"] == b["teacher"] && a["teacher"] != nil && a["teacher"] != "" {
					conflicts = append(conflicts, map[string]interface{}{
						"type": "教师冲突", "severity": "high",
						"description": fmt.Sprintf("教师 %v 在同一时段(%s)需同时上 %v 与 %v",
							a["teacher"], key, a["course_name"], b["course_name"]),
					})
				}
			}
		}
	}
	return len(schedules), conflicts, nil
}

// GetExams 读取真实考试安排
func (s *Phase3Service) GetExams(semester string) ([]map[string]interface{}, error) {
	return s.repo.ListExams(semester)
}

// GetGraduationSummaries 读取成绩聚合（毕业审核数据源）
func (s *Phase3Service) GetGraduationSummaries() ([]*repository.GradeSummary, error) {
	return s.repo.ListGradeSummaries()
}
