// Package repository 学业学习模块仓库（P4-d：从 education_handler 下沉的 8 处裸 SQL）。
package repository

import (
	"database/sql"
	"errors"

	"github.com/dll/wxx/server/internal/model"
)

// ErrCourseNotFound 课程不存在。
var ErrCourseNotFound = errors.New("课程不存在")

// StudyRepo 学业学习数据访问层。
type StudyRepo struct {
	db *sql.DB
}

// NewStudyRepo 创建学业学习仓库。
func NewStudyRepo(db *sql.DB) *StudyRepo {
	return &StudyRepo{db: db}
}

// ListCourses 课程列表。
func (r *StudyRepo) ListCourses(department, category, semester string) ([]*model.Course, error) {
	where := []string{"status = 'active'"}
	args := []interface{}{}
	if department != "" {
		where = append(where, "department = ?")
		args = append(args, department)
	}
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	if semester != "" {
		where = append(where, "semester = ?")
		args = append(args, semester)
	}

	rows, err := r.db.Query(
		"SELECT id, course_id, course_code, course_name, credit, hours, category, department, "+
			"teacher, description, semester, created_at "+
			"FROM courses "+buildWhereClause(where)+" ORDER BY department ASC, course_code ASC, id ASC",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.Course, 0)
	for rows.Next() {
		item := &model.Course{}
		if err := rows.Scan(&item.ID, &item.CourseID, &item.CourseCode, &item.CourseName,
			&item.Credit, &item.Hours, &item.Category, &item.Department, &item.Teacher,
			&item.Description, &item.Semester, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// GetCourseDetail 活动状态课程详情（无记录返回 ErrCourseNotFound）。
func (r *StudyRepo) GetCourseDetail(courseID string) (*model.CourseDetail, error) {
	detail := &model.CourseDetail{}
	err := r.db.QueryRow(
		"SELECT id, course_id, course_code, course_name, credit, hours, category, department, "+
			"teacher, description, syllabus, prerequisites, textbook, references, semester, status, "+
			"created_at, updated_at FROM courses WHERE course_id = ? AND status = 'active'", courseID,
	).Scan(&detail.ID, &detail.CourseID, &detail.CourseCode, &detail.CourseName,
		&detail.Credit, &detail.Hours, &detail.Category, &detail.Department, &detail.Teacher,
		&detail.Description, &detail.Syllabus, &detail.Prerequisites, &detail.Textbook,
		&detail.References, &detail.Semester, &detail.Status, &detail.CreatedAt, &detail.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrCourseNotFound
	}
	if err != nil {
		return nil, err
	}
	return detail, nil
}

// ListMyGrades 我的成绩（按学期倒序）。
func (r *StudyRepo) ListMyGrades(userID int64, semester string) ([]*model.GradeItem, error) {
	where := []string{"user_id = ?"}
	args := []interface{}{userID}
	if semester != "" {
		where = append(where, "semester = ?")
		args = append(args, semester)
	}

	rows, err := r.db.Query(
		"SELECT id, course_id, course_name, semester, grade_type, score, gpa, rank, grade_level, "+
			"passed, credits_earned, created_at "+
			"FROM student_grades "+buildWhereClause(where)+" ORDER BY semester DESC, id DESC",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.GradeItem, 0)
	for rows.Next() {
		item := &model.GradeItem{}
		if err := rows.Scan(&item.ID, &item.CourseID, &item.CourseName, &item.Semester,
			&item.GradeType, &item.Score, &item.GPA, &item.Rank, &item.GradeLevel,
			&item.Passed, &item.CreditsEarned, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// GetGradeSummary 成绩统计。
func (r *StudyRepo) GetGradeSummary(userID int64) (*model.GradeSummary, error) {
	s := &model.GradeSummary{}
	row := r.db.QueryRow(
		"SELECT COUNT(*) as total, SUM(CASE WHEN passed = 1 THEN 1 ELSE 0 END) as passed, "+
			"COALESCE(SUM(CASE WHEN passed = 1 THEN credits_earned ELSE 0 END), 0) as earned_credits, "+
			"COALESCE(AVG(score), 0) as avg_score, "+
			"COALESCE(AVG(gpa), 0) as avg_gpa, "+
			"COALESCE(SUM(credits_earned), 0) as total_credits "+
			"FROM student_grades WHERE user_id = ? AND grade_type = 'final'",
		userID,
	)
	var totalCourses, passedCourses int
	var earnedCredits, avgScore, avgGPA, totalCredits float64
	if err := row.Scan(&totalCourses, &passedCourses, &earnedCredits, &avgScore, &avgGPA, &totalCredits); err != nil {
		return nil, err
	}

	s.TotalCourses = totalCourses
	s.PassedCourses = passedCourses
	s.FailedCourses = totalCourses - passedCourses
	s.EarnedCredits = earnedCredits
	s.AverageScore = round2(avgScore)
	s.TotalGPA = round2(avgGPA)
	s.TotalCredits = totalCredits

	// 最近有成绩的学期作为"当前学期"（无记录时静默为零值）
	semRow := r.db.QueryRow(
		"SELECT semester, COALESCE(AVG(gpa), 0), COALESCE(SUM(credits_earned), 0) "+
			"FROM student_grades WHERE user_id = ? AND grade_type = 'final' "+
			"GROUP BY semester ORDER BY semester DESC LIMIT 1",
		userID,
	)
	var currentSemester string
	var semGPA, semCredits float64
	if err := semRow.Scan(&currentSemester, &semGPA, &semCredits); err == nil {
		s.CurrentSemester = currentSemester
		s.SemesterGPA = round2(semGPA)
		s.SemesterCredits = semCredits
	}
	return s, nil
}

// round2 保留两位小数。
func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// ListLearningResources 分页学习资源。
func (r *StudyRepo) ListLearningResources(courseID, resourceType string, page, pageSize int) ([]*model.LearningResource, int, error) {
	where := []string{"status = 'active'"}
	args := []interface{}{}
	if courseID != "" {
		where = append(where, "course_id = ?")
		args = append(args, courseID)
	}
	if resourceType != "" {
		where = append(where, "resource_type = ?")
		args = append(args, resourceType)
	}
	whereSQL := buildWhereClause(where)

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM learning_resources "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(
		"SELECT id, resource_id, course_id, course_name, title, resource_type, chapter, "+
			"file_url, author, download_count, view_count, tags, created_at "+
			"FROM learning_resources "+whereSQL+" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, pageOffset(page, pageSize))...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*model.LearningResource, 0)
	for rows.Next() {
		item := &model.LearningResource{}
		if err := rows.Scan(&item.ID, &item.ResourceID, &item.CourseID, &item.CourseName,
			&item.Title, &item.ResourceType, &item.Chapter, &item.FileURL, &item.Author,
			&item.DownloadCount, &item.ViewCount, &item.Tags, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}

// ListExamSchedules 考试安排。
func (r *StudyRepo) ListExamSchedules(semester string) ([]*model.ExamSchedule, error) {
	where := []string{}
	args := []interface{}{}
	if semester != "" {
		where = append(where, "semester = ?")
		args = append(args, semester)
	}

	rows, err := r.db.Query(
		"SELECT id, exam_id, course_id, course_name, exam_type, date, time_start, time_end, "+
			"location, seat, semester, created_at "+
			"FROM exam_schedules "+buildWhereClause(where)+" ORDER BY date ASC, time_start ASC, id ASC",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.ExamSchedule, 0)
	for rows.Next() {
		item := &model.ExamSchedule{}
		if err := rows.Scan(&item.ID, &item.ExamID, &item.CourseID, &item.CourseName,
			&item.ExamType, &item.Date, &item.TimeStart, &item.TimeEnd, &item.Location,
			&item.Seat, &item.Semester, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, rows.Err()
}
