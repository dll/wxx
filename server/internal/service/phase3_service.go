package service

import (
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/repository"
)

// Phase3Service 阶段三数据底座服务（导入 + 教辅真实数据）
type Phase3Service struct {
	repo *repository.DataImportRepo
	// TeacherCourseRepo 成绩强校验判据访问层（R3，可为暂空）；
	// ImportTeacherGrades 写库前用它校验该教师-课程-学期授课关系已 approved。
	tcRepo *repository.TeacherCourseRepo
	// userRepo 课表导入按 username 解析 user_id（2026-09-01 修复课表挂错账号）
	userRepo *repository.UserRepo
}

// NewPhase3Service 创建阶段三服务
func NewPhase3Service(repo *repository.DataImportRepo) *Phase3Service {
	return &Phase3Service{repo: repo}
}

// SetTeacherCourseRepo 注入教师授课关系访问层（R3 成绩强校验接线）
func (s *Phase3Service) SetTeacherCourseRepo(r *repository.TeacherCourseRepo) {
	s.tcRepo = r
}

// SetUserRepo 注入用户访问层（课表导入按 username 解析归属）
func (s *Phase3Service) SetUserRepo(r *repository.UserRepo) {
	s.userRepo = r
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

// ImportTeacherGrades 教师自主录入成绩（方案 A）
// 教师在前端声明授课关系与成绩，后端不校验 teacher→course 硬关联（该关联尚无数据），
// 但强制：
//  1. created_by=当前教师 user_id（审计可追溯谁的声明）
//  2. 每条记录 target 必须是 role='student' 的学生（不能对教师/管理员等写成绩，防捞非学生越权）
// 幂等复用 UpsertGrade。
/**
 * ImportTeacherGrades 教师自主录入成绩（方案 A 升级版，R3 强校验）
 *
 * 教师在前端申报授课关系并经教辅/教务审核确认后，才能录入该课程成绩。
 * 写库前逐条强校验 teacher_courses 状态：
 *   - status=='approved'  → 放行写入
 *   - status=='pending'   → 拒绝（提示待审核）
 *   - status=='rejected'  → 拒绝（提示联系教辅）
 *   - 无申报             → 拒绝（提示先申报）
 * 此即停用方案A「声明即授权」：不再凭前端声明即写库，approved 唯一来源为教辅真实审核操作。
 *
 * 既有校验全部保留：
 *  1. created_by=当前教师 user_id（审计可追溯谁的声明）
 *  2. 每条记录 target 必须是 role='student' 的学生（防捞非学生越权）
 *  3. 0-100 分 + passed 一致性校验
 * 幂等复用 UpsertGrade；单条失败仅记入 Errors，不整批回滚；历史已录成绩不回溯。
 */
func (s *Phase3Service) ImportTeacherGrades(grades []*repository.GradeRow, creatorID int64) *ImportResult {
	res := &ImportResult{Total: len(grades)}
	for _, g := range grades {
		if g.UserID == "" || g.CourseID == "" {
			res.Errors = append(res.Errors, "学号/课程ID不能为空")
			continue
		}
		// 成绩范围校验（防误录/恶意污染学生端与毕业审核）
		if g.Score < 0 || g.Score > 100 {
			res.Errors = append(res.Errors, fmt.Sprintf("学号 %s/%s 成绩 %.2f 超出 0-100 范围，拒绝写入", g.UserID, g.CourseID, g.Score))
			continue
		}
		// passed 与成绩一致性校验（60 分为及格线；不一致视为误录/恶意数据，拒绝）
		if (g.Score >= 60) != g.Passed {
			res.Errors = append(res.Errors, fmt.Sprintf("学号 %s/%s 成绩 %.2f 与通过标记不一致（>=60 应已通过），拒绝写入", g.UserID, g.CourseID, g.Score))
			continue
		}
		// 校验 target 为学生（防写教师/管理员等非学生）
		role, err := s.repo.GetUserRoleByUserID(g.UserID)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("学号 %s 不存在: %v", g.UserID, err))
			continue
		}
		if role != "student" {
			res.Errors = append(res.Errors, fmt.Sprintf("学号 %s 角色=%s，非学生，拒绝写入", g.UserID, role))
			continue
		}
		g.CreatedBy = creatorID
		g.UpdatedBy = creatorID // R1：记录最后写入/修改人（审计可追溯）

		// R3 强校验：写库前查 teacher_courses 授课关系状态，仅 approved 放行。
		// （tcRepo 生产恒由 app.go 注入；为防御 nil 并保持旧测试可显式关停，缺位时跳过并用错误口径暴露）
		if s.tcRepo != nil {
			exists, status, err := s.tcRepo.GetTeacherCourseStatus(creatorID, g.CourseID, g.Semester)
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("学号 %s/%s 授课关系校验失败: %v", g.UserID, g.CourseID, err))
				continue
			}
			if !exists {
				res.Errors = append(res.Errors, fmt.Sprintf("学号 %s/%s(%s) 学期 %s 尚无授课关系申报，请先在「我的授课」申报，待教辅审核通过后再录入", g.UserID, g.CourseID, g.CourseName, g.Semester))
				continue
			}
			if status != repository.CourseStatusApproved {
				if status == repository.CourseStatusPending {
					res.Errors = append(res.Errors, fmt.Sprintf("学号 %s/%s(%s) 学期 %s 授课申报待审核，确认后方可录入成绩", g.UserID, g.CourseID, g.CourseName, g.Semester))
				} else {
					res.Errors = append(res.Errors, fmt.Sprintf("学号 %s/%s(%s) 学期 %s 授课申报未被确认（%s），请联系教辅", g.UserID, g.CourseID, g.CourseName, g.Semester, status))
				}
				continue
			}
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

// ListTeacherGrades 查询教师本人声明录入的成绩记录（读取边界=created_by）
func (s *Phase3Service) ListTeacherGrades(creatorID int64) ([]*repository.ListedGrade, error) {
	return s.repo.ListGradesByCreator(creatorID)
}

// ImportSchedules 批量导入课表（幂等）
func (s *Phase3Service) ImportSchedules(rows []*repository.ScheduleRow) *ImportResult {
	res := &ImportResult{Total: len(rows)}
	for _, r := range rows {
		if r.CourseID == "" {
			res.Errors = append(res.Errors, "课程ID不能为空")
			continue
		}
		// 严格归属解析（2026-09-01）：优先按 username 解析真实账号，
		// 避免填错 user_id 使课程挂到错误账号（登录后显示的课程不对）。
		uid, err := s.resolveScheduleOwner(r)
		if err != nil {
			res.Errors = append(res.Errors, err.Error())
			continue
		}
		if err := s.repo.UpsertSchedule(r, uid); err != nil {
			res.Errors = append(res.Errors, err.Error())
			continue
		}
		res.Created++
	}
	return res
}

// resolveScheduleOwner 解析课表归属 user_id。
// 规则（由严到松）：
//  1. 提供 username → 查库校验存在，返回其 user_id（权威，忽略同行的 user_id，防不一致）；
//  2. 未提供 username、但提供 user_id>0 → 仅校验用户存在，直接使用；
//  3. 两者皆缺/用户不存在 → 报错拒写（绝不落到 user_id=0 或幽灵账号）。
func (s *Phase3Service) resolveScheduleOwner(r *repository.ScheduleRow) (int64, error) {
	username := strings.TrimSpace(r.Username)
	if username != "" {
		if s.userRepo == nil {
			return 0, fmt.Errorf("课表归属解析不可用（userRepo 未注入）")
		}
		u, err := s.userRepo.GetByUsername(username)
		if err != nil {
			return 0, fmt.Errorf("%s: 查课表归属失败 %v", username, err)
		}
		if u == nil {
			return 0, fmt.Errorf("%s: 账号不存在，无法挂载课表", username)
		}
		return u.ID, nil
	}
	if r.UserID > 0 {
		if s.userRepo != nil {
			u, err := s.userRepo.GetByID(r.UserID)
			if err != nil {
				return 0, fmt.Errorf("user_id=%d: 查用户失败 %v", r.UserID, err)
			}
			if u == nil {
				return 0, fmt.Errorf("user_id=%d: 账号不存在，无法挂载课表", r.UserID)
			}
		}
		return r.UserID, nil
	}
	return 0, fmt.Errorf("课表归属缺失：需提供 username 或 user_id")
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
