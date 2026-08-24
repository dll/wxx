package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/repository"
)

// HomeworkService 教师作业信息发布 + 课程成绩统计（只读）服务（P2 轻量版）
// 蔚小芯侧重教育非教学：作业仅做信息发布+成绩统计，不做学生提交/批改/内容/附件/流转。
// 归属强约束：发布/编辑前校验该教师对该 (course_id, semester) 已有 approved 授课关系（teacher_courses），
// 对称 R3 成绩强校验语义；绝不造数据、0 行诚实空。
type HomeworkService struct {
	homeworkRepo     *repository.HomeworkRepo
	teacherCourseSvc *TeacherCourseService
}

// NewHomeworkService 创建作业服务。
// teacherCourseSvc 用于归属强校验（GetTeacherCourseStatus / ListTeacherCourses approved 白名单）。
func NewHomeworkService(homeworkRepo *repository.HomeworkRepo, teacherCourseSvc *TeacherCourseService) *HomeworkService {
	return &HomeworkService{
		homeworkRepo:     homeworkRepo,
		teacherCourseSvc: teacherCourseSvc,
	}
}

// PublishHomework 发布作业信息（写库前强校验 approved 授课关系）。
// teacherID 强制取当前登录教师（杜绝代发，与 ImportTeacherGrades created_by 铁律一致）。
// 入参：course_id 复用 CourseExists+normalizeCourseID 校验，title/semester 必填。
func (s *HomeworkService) PublishHomework(ctx context.Context, teacherID int64, courseID, courseName, semester, title, description, publishAt, dueAt string) (int64, bool, error) {
	if teacherID <= 0 {
		return 0, false, fmt.Errorf("教师不能为空")
	}
	courseID = strings.TrimSpace(courseID)
	semester = strings.TrimSpace(semester)
	title = strings.TrimSpace(title)
	if courseID == "" || semester == "" || title == "" {
		return 0, false, fmt.Errorf("课程、学期、标题必填")
	}
	// M：课程必须存在于主数据表（防虚构课程号）
	exists, err := s.teacherCourseSvc.repo.CourseExists(courseID)
	if err != nil {
		return 0, false, fmt.Errorf("课程存在性校验失败: %v", err)
	}
	if !exists {
		return 0, false, fmt.Errorf("课程不存在，请核对课程ID")
	}
	// 归属强校验：仅 approved 授课关系可发布作业（pending→待审核；rejected→联系教辅；无→先申报）
	ok, status, err := s.teacherCourseSvc.GetTeacherCourseStatus(ctx, teacherID, courseID, semester)
	if err != nil {
		return 0, false, fmt.Errorf("授课归属校验失败: %v", err)
	}
	if !ok {
		return 0, false, fmt.Errorf("您尚未申报该课程授课关系，请先申报并经教辅/教务审核确认后发布")
	}
	if status != repository.CourseStatusApproved {
		switch status {
		case repository.CourseStatusPending:
			return 0, false, fmt.Errorf("该课程授课申报待审核中，请待教辅/教务确认(approved)后发布")
		case repository.CourseStatusRejected:
			return 0, false, fmt.Errorf("该课程授课申报已被驳回，请与教辅确认后重新申报再发布")
		default:
			return 0, false, fmt.Errorf("授课关系状态异常(%s)，无法发布作业", status)
		}
	}

	return s.homeworkRepo.Publish(&repository.Homework{
		TeacherID:   teacherID,
		CourseID:    courseID,
		CourseName:  strings.TrimSpace(courseName),
		Semester:    semester,
		Title:       title,
		Description: strings.TrimSpace(description),
		PublishAt:   strings.TrimSpace(publishAt),
		DueAt:       strings.TrimSpace(dueAt),
		Status:      repository.HomeworkStatusActive,
	})
}

// UpdateHomework 编辑本人作业（仅本人，course_id/semester 归属不随编辑改变）。
func (s *HomeworkService) UpdateHomework(ctx context.Context, id, teacherID int64, title, description, publishAt, dueAt string) error {
	if id <= 0 {
		return fmt.Errorf("作业ID无效")
	}
	hw, err := s.homeworkRepo.GetOwnHomework(id, teacherID)
	if err != nil {
		return fmt.Errorf("查询作业失败: %v", err)
	}
	if hw == nil {
		return fmt.Errorf("作业不存在或无权编辑")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("标题必填")
	}
	if hw.Status == repository.HomeworkStatusArchived {
		return fmt.Errorf("作业已下架，不可编辑，请重新发布")
	}
	return s.homeworkRepo.Update(id, &repository.Homework{
		CourseName:  hw.CourseName,
		Title:       title,
		Description: strings.TrimSpace(description),
		PublishAt:   strings.TrimSpace(publishAt),
		DueAt:       strings.TrimSpace(dueAt),
		Status:      hw.Status,
	})
}

// ArchiveHomework 下架作业（软删：置 archived，审计可溯）。仅本人作业。
func (s *HomeworkService) ArchiveHomework(ctx context.Context, id, teacherID int64) error {
	hw, err := s.homeworkRepo.GetOwnHomework(id, teacherID)
	if err != nil {
		return fmt.Errorf("查询作业失败: %v", err)
	}
	if hw == nil {
		return fmt.Errorf("作业不存在或无权下架")
	}
	if hw.Status == repository.HomeworkStatusArchived {
		return fmt.Errorf("作业已下架")
	}
	return s.homeworkRepo.Archive(id)
}

// ListMyHomework 教师本人的作业清单。
func (s *HomeworkService) ListMyHomework(ctx context.Context, teacherID int64) ([]repository.Homework, error) {
	list, err := s.homeworkRepo.ListByTeacher(teacherID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []repository.Homework{}
	}
	return list, nil
}

// ListApprovedCourses 教师本人 approved 授课课程白名单（前端课程下拉数据源，仅真实 approved）。
func (s *HomeworkService) ListApprovedCourses(ctx context.Context, teacherID int64) ([]repository.TeacherCourse, error) {
	return s.teacherCourseSvc.repo.ListTeacherCourses(teacherID, repository.CourseStatusApproved, 500)
}

// GradeStatsByCourse 课程成绩只读统计（按课程维度）。
// 仅 approved 授课课程可查（白名单校验）；非 approved 返回空 + not_available（诚实口径）。
// 0 行返回 total=0 + not_available=true，不补造分布。
func (s *HomeworkService) GradeStatsByCourse(ctx context.Context, teacherID int64, courseID, semester string) (*repository.CourseGradeStats, error) {
	if courseID == "" || semester == "" {
		return nil, fmt.Errorf("课程和学期必填")
	}
	// 白名单：仅本人 approved 授课课程可统计
	courseID = strings.TrimSpace(courseID)
	ok, status, err := s.teacherCourseSvc.GetTeacherCourseStatus(ctx, teacherID, courseID, semester)
	if err != nil {
		return nil, fmt.Errorf("授课归属校验失败: %v", err)
	}
	if !ok || status != repository.CourseStatusApproved {
		return &repository.CourseGradeStats{
			CourseID:     courseID,
			Semester:     semester,
			Total:        0,
			NotAvailable: true,
			Levels:       map[string]int{"优秀": 0, "良好": 0, "及格": 0, "不及格": 0},
		}, nil
	}
	return s.homeworkRepo.GradeStatsByCourse(courseID, semester)
}
