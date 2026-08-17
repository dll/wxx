package service

import (
	"context"
	"fmt"

	"github.com/dll/wxx/server/internal/repository"
)

// TeacherCourseService 教师授课关系申报+教辅审核服务（R3，越权边界升级）
// 覆盖：教师申报授课关系 + 本人申报查询 + 教辅/教务待审列表 + 审核 + 成绩强校验判据暴露。
// approved 唯一来源为教辅真实审核操作，绝不脚本批量置位。
type TeacherCourseService struct {
	repo *repository.TeacherCourseRepo
}

func NewTeacherCourseService(repo *repository.TeacherCourseRepo) *TeacherCourseService {
	return &TeacherCourseService{repo: repo}
}

// SubmitTeacherCourse 教师申报授课关系（幂等，门控 TeacherGradeWrite=调用方已校验教师身份）
// teacherID 强制取当前登录教师 user_id（杜绝代他人申报）。
// R3 补强 M：申报入参校验——course_id 必须存在于 courses 主数据表，拒绝虚构课程号审批后写假成绩。
func (s *TeacherCourseService) SubmitTeacherCourse(ctx context.Context, teacherID int64, courseID, courseName, semester string) (int64, string, error) {
	if teacherID <= 0 {
		return 0, "", fmt.Errorf("教师不能为空")
	}
	if courseID == "" || semester == "" {
		return 0, "", fmt.Errorf("课程和学期不能为空")
	}
	// M：申报前校验课程存在。courses.course_id 为权威口径；不存在→拒绝，避免虚构课程号被 approve 后写虚构成绩。
	exists, err := s.repo.CourseExists(courseID)
	if err != nil {
		return 0, "", fmt.Errorf("课程存在性校验失败: %v", err)
	}
	if !exists {
		return 0, "", fmt.Errorf("课程不存在，请核对课程ID")
	}
	tc := &repository.TeacherCourse{
		TeacherID:  teacherID,
		CourseID:   courseID,
		CourseName: courseName,
		Semester:   semester,
		CreatedBy:  teacherID,
	}
	return s.repo.SubmitTeacherCourse(tc)
}

// ListMyTeacherCourses 教师查询本人申报列表
func (s *TeacherCourseService) ListMyTeacherCourses(ctx context.Context, teacherID int64) ([]repository.TeacherCourse, error) {
	return s.repo.ListTeacherCourses(teacherID, "", 500)
}

// ListPendingTeacherCourses 教辅/教务查询待审核申报列表
func (s *TeacherCourseService) ListPendingTeacherCourses(ctx context.Context, limit int) ([]repository.TeacherCourse, error) {
	return s.repo.ListPendingTeacherCourses(limit)
}

// ReviewTeacherCourse 教辅/教务审核（门控 TeacherCourseReview=调用方已校验教辅身份）
func (s *TeacherCourseService) ReviewTeacherCourse(ctx context.Context, id, reviewerID int64, reviewerName, status, note string) error {
	if reviewerID <= 0 {
		return fmt.Errorf("审核人不能为空")
	}
	return s.repo.ReviewTeacherCourse(id, reviewerID, reviewerName, status, note)
}

// CountPending 待审核申报条数（角标）
func (s *TeacherCourseService) CountPending(ctx context.Context) (int, error) {
	return s.repo.CountPendingTeacherCourses()
}

// GetTeacherCourseStatus 成绩强校验判据（P1）：是否存在 + 状态（pending/approved/rejected）
func (s *TeacherCourseService) GetTeacherCourseStatus(ctx context.Context, teacherID int64, courseID, semester string) (bool, string, error) {
	return s.repo.GetTeacherCourseStatus(teacherID, courseID, semester)
}
