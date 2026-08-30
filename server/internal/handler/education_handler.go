package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/gin-gonic/gin"
)

// EducationHandler 学生教育三大模块（就业指导、学业学习、心理健康）HTTP handler
// P4-d 进行中：心理健康域 SQL 已下沉 MentalHealthRepo，其余域后续按同范式迁移
type EducationHandler struct {
	db         *sql.DB
	mentalRepo *repository.MentalHealthRepo
}

// NewEducationHandler 创建教育模块 handler
func NewEducationHandler(db *sql.DB, mentalRepo *repository.MentalHealthRepo) *EducationHandler {
	return &EducationHandler{db: db, mentalRepo: mentalRepo}
}

// generateID 生成简短唯一 ID（前缀 + 随机 hex）
func generateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

// ═══════════════════════════════════════════════
// 一、就业指导模块 /api/v1/career
// ═══════════════════════════════════════════════

// ListCareerPolicies 就业政策列表
// GET /api/v1/career/policies?category=&level=&page=1&page_size=20

// ═══════════════════════════════════════════════
// 二、学业学习模块 /api/v1/study
// ═══════════════════════════════════════════════
// (就业指导→education_career_handler.go  心理健康→education_mental_health_handler.go)

func (h *EducationHandler) ListCourses(c *gin.Context) {
	department := c.Query("department")
	category := c.Query("category")
	semester := c.Query("semester")

	var where []string
	var args []interface{}
	where = append(where, "status = 'active'")
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
	whereSQL := strings.Join(where, " AND ")

	rows, err := h.db.Query(
		"SELECT id, course_id, course_code, course_name, credit, hours, category, department, "+
			"teacher, description, semester, created_at "+
			"FROM courses WHERE "+whereSQL+" ORDER BY department ASC, course_code ASC, id ASC",
		args...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询课程列表失败"})
		return
	}
	defer rows.Close()

	type CourseItem struct {
		ID          int64   `json:"id"`
		CourseID    string  `json:"course_id"`
		CourseCode  string  `json:"course_code"`
		CourseName  string  `json:"course_name"`
		Credit      float64 `json:"credit"`
		Hours       int     `json:"hours"`
		Category    string  `json:"category"`
		Department  string  `json:"department"`
		Teacher     string  `json:"teacher"`
		Description string  `json:"description"`
		Semester    string  `json:"semester"`
		CreatedAt   string  `json:"created_at"`
	}

	var list []*CourseItem
	for rows.Next() {
		item := &CourseItem{}
		if err := rows.Scan(&item.ID, &item.CourseID, &item.CourseCode, &item.CourseName,
			&item.Credit, &item.Hours, &item.Category, &item.Department, &item.Teacher,
			&item.Description, &item.Semester, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
		"total":   len(list),
	})
}

// GetCourse 课程详情
// GET /api/v1/study/courses/:id
func (h *EducationHandler) GetCourse(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "课程ID不能为空"})
		return
	}

	type CourseDetail struct {
		ID            int64   `json:"id"`
		CourseID      string  `json:"course_id"`
		CourseCode    string  `json:"course_code"`
		CourseName    string  `json:"course_name"`
		Credit        float64 `json:"credit"`
		Hours         int     `json:"hours"`
		Category      string  `json:"category"`
		Department    string  `json:"department"`
		Teacher       string  `json:"teacher"`
		Description   string  `json:"description"`
		Syllabus      string  `json:"syllabus"`
		Prerequisites string  `json:"prerequisites"`
		Textbook      string  `json:"textbook"`
		References    string  `json:"references"`
		Semester      string  `json:"semester"`
		Status        string  `json:"status"`
		CreatedAt     string  `json:"created_at"`
		UpdatedAt     string  `json:"updated_at"`
	}

	detail := &CourseDetail{}
	err := h.db.QueryRow(
		"SELECT id, course_id, course_code, course_name, credit, hours, category, department, "+
			"teacher, description, syllabus, prerequisites, textbook, references, semester, status, "+
			"created_at, updated_at FROM courses WHERE course_id = ? AND status = 'active'", id,
	).Scan(&detail.ID, &detail.CourseID, &detail.CourseCode, &detail.CourseName,
		&detail.Credit, &detail.Hours, &detail.Category, &detail.Department, &detail.Teacher,
		&detail.Description, &detail.Syllabus, &detail.Prerequisites, &detail.Textbook,
		&detail.References, &detail.Semester, &detail.Status, &detail.CreatedAt, &detail.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "课程不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询课程详情失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    detail,
	})
}

// ListMyGrades 我的成绩列表（按学期分组）
// GET /api/v1/study/grades?semester=
func (h *EducationHandler) ListMyGrades(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	semester := c.Query("semester")

	var where []string
	var args []interface{}
	where = append(where, "user_id = ?")
	args = append(args, userCtx.UserID)
	if semester != "" {
		where = append(where, "semester = ?")
		args = append(args, semester)
	}
	whereSQL := strings.Join(where, " AND ")

	rows, err := h.db.Query(
		"SELECT id, course_id, course_name, semester, grade_type, score, gpa, rank, grade_level, "+
			"passed, credits_earned, created_at "+
			"FROM student_grades WHERE "+whereSQL+" ORDER BY semester DESC, id DESC",
		args...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询成绩失败"})
		return
	}
	defer rows.Close()

	type GradeItem struct {
		ID            int64   `json:"id"`
		CourseID      string  `json:"course_id"`
		CourseName    string  `json:"course_name"`
		Semester      string  `json:"semester"`
		GradeType     string  `json:"grade_type"`
		Score         float64 `json:"score"`
		GPA           float64 `json:"gpa"`
		Rank          int     `json:"rank"`
		GradeLevel    string  `json:"grade_level"`
		Passed        int     `json:"passed"`
		CreditsEarned float64 `json:"credits_earned"`
		CreatedAt     string  `json:"created_at"`
	}

	grouped := make(map[string][]*GradeItem)
	var semesters []string
	for rows.Next() {
		item := &GradeItem{}
		if err := rows.Scan(&item.ID, &item.CourseID, &item.CourseName, &item.Semester,
			&item.GradeType, &item.Score, &item.GPA, &item.Rank, &item.GradeLevel,
			&item.Passed, &item.CreditsEarned, &item.CreatedAt); err != nil {
			continue
		}
		if _, ok := grouped[item.Semester]; !ok {
			semesters = append(semesters, item.Semester)
		}
		grouped[item.Semester] = append(grouped[item.Semester], item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":      0,
		"message":   "success",
		"data":      grouped,
		"semesters": semesters,
	})
}

// GetGradeSummary 成绩统计（GPA、学分、排名）
// GET /api/v1/study/grades/summary
func (h *EducationHandler) GetGradeSummary(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	type GradeSummary struct {
		TotalGPA        float64 `json:"total_gpa"`
		TotalCredits    float64 `json:"total_credits"`
		EarnedCredits   float64 `json:"earned_credits"`
		TotalCourses    int     `json:"total_courses"`
		PassedCourses   int     `json:"passed_courses"`
		FailedCourses   int     `json:"failed_courses"`
		AverageScore    float64 `json:"average_score"`
		ClassRank       int     `json:"class_rank"`
		ClassTotal      int     `json:"class_total"`
		CurrentSemester string  `json:"current_semester"`
		SemesterGPA     float64 `json:"semester_gpa"`
		SemesterCredits float64 `json:"semester_credits"`
	}

	summary := &GradeSummary{}

	row := h.db.QueryRow(
		"SELECT COUNT(*) as total, SUM(CASE WHEN passed = 1 THEN 1 ELSE 0 END) as passed, "+
			"COALESCE(SUM(CASE WHEN passed = 1 THEN credits_earned ELSE 0 END), 0) as earned_credits, "+
			"COALESCE(AVG(score), 0) as avg_score, "+
			"COALESCE(AVG(gpa), 0) as avg_gpa, "+
			"COALESCE(SUM(credits_earned), 0) as total_credits "+
			"FROM student_grades WHERE user_id = ? AND grade_type = 'final'",
		userCtx.UserID,
	)
	var totalCourses, passedCourses int
	var earnedCredits, avgScore, avgGPA, totalCredits float64
	err := row.Scan(&totalCourses, &passedCourses, &earnedCredits, &avgScore, &avgGPA, &totalCredits)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询成绩统计失败"})
		return
	}

	summary.TotalCourses = totalCourses
	summary.PassedCourses = passedCourses
	summary.FailedCourses = totalCourses - passedCourses
	summary.EarnedCredits = earnedCredits
	summary.AverageScore = math.Round(avgScore*100) / 100
	summary.TotalGPA = math.Round(avgGPA*100) / 100
	summary.TotalCredits = totalCredits

	semRow := h.db.QueryRow(
		"SELECT semester, COALESCE(AVG(gpa), 0), COALESCE(SUM(credits_earned), 0) "+
			"FROM student_grades WHERE user_id = ? AND grade_type = 'final' "+
			"GROUP BY semester ORDER BY semester DESC LIMIT 1",
		userCtx.UserID,
	)
	var currentSemester string
	var semGPA, semCredits float64
	_ = semRow.Scan(&currentSemester, &semGPA, &semCredits)
	summary.CurrentSemester = currentSemester
	summary.SemesterGPA = math.Round(semGPA*100) / 100
	summary.SemesterCredits = semCredits

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    summary,
	})
}

// ListLearningResources 学习资源列表
// GET /api/v1/study/resources?course_id=&resource_type=&page=1&page_size=20
func (h *EducationHandler) ListLearningResources(c *gin.Context) {
	courseID := c.Query("course_id")
	resourceType := c.Query("resource_type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var where []string
	var args []interface{}
	where = append(where, "status = 'active'")
	if courseID != "" {
		where = append(where, "course_id = ?")
		args = append(args, courseID)
	}
	if resourceType != "" {
		where = append(where, "resource_type = ?")
		args = append(args, resourceType)
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM learning_resources WHERE "+whereSQL, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习资源失败"})
		return
	}

	rows, err := h.db.Query(
		"SELECT id, resource_id, course_id, course_name, title, resource_type, chapter, "+
			"file_url, author, download_count, view_count, tags, created_at "+
			"FROM learning_resources WHERE "+whereSQL+" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习资源失败"})
		return
	}
	defer rows.Close()

	type ResourceItem struct {
		ID            int64  `json:"id"`
		ResourceID    string `json:"resource_id"`
		CourseID      string `json:"course_id"`
		CourseName    string `json:"course_name"`
		Title         string `json:"title"`
		ResourceType  string `json:"resource_type"`
		Chapter       string `json:"chapter"`
		FileURL       string `json:"file_url"`
		Author        string `json:"author"`
		DownloadCount int    `json:"download_count"`
		ViewCount     int    `json:"view_count"`
		Tags          string `json:"tags"`
		CreatedAt     string `json:"created_at"`
	}

	var list []*ResourceItem
	for rows.Next() {
		item := &ResourceItem{}
		if err := rows.Scan(&item.ID, &item.ResourceID, &item.CourseID, &item.CourseName,
			&item.Title, &item.ResourceType, &item.Chapter, &item.FileURL, &item.Author,
			&item.DownloadCount, &item.ViewCount, &item.Tags, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":      0,
		"message":   "success",
		"data":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ListExamSchedules 考试安排
// GET /api/v1/study/exams?semester=
func (h *EducationHandler) ListExamSchedules(c *gin.Context) {
	semester := c.Query("semester")

	var where []string
	var args []interface{}
	if semester != "" {
		where = append(where, "semester = ?")
		args = append(args, semester)
	}
	whereSQL := "1=1"
	if len(where) > 0 {
		whereSQL = strings.Join(where, " AND ")
	}

	rows, err := h.db.Query(
		"SELECT id, exam_id, course_id, course_name, exam_type, date, time_start, time_end, "+
			"location, seat, semester, created_at "+
			"FROM exam_schedules WHERE "+whereSQL+" ORDER BY date ASC, time_start ASC, id ASC",
		args...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询考试安排失败"})
		return
	}
	defer rows.Close()

	type ExamItem struct {
		ID         int64  `json:"id"`
		ExamID     string `json:"exam_id"`
		CourseID   string `json:"course_id"`
		CourseName string `json:"course_name"`
		ExamType   string `json:"exam_type"`
		Date       string `json:"date"`
		TimeStart  string `json:"time_start"`
		TimeEnd    string `json:"time_end"`
		Location   string `json:"location"`
		Seat       string `json:"seat"`
		Semester   string `json:"semester"`
		CreatedAt  string `json:"created_at"`
	}

	var list []*ExamItem
	for rows.Next() {
		item := &ExamItem{}
		if err := rows.Scan(&item.ID, &item.ExamID, &item.CourseID, &item.CourseName,
			&item.ExamType, &item.Date, &item.TimeStart, &item.TimeEnd, &item.Location,
			&item.Seat, &item.Semester, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
		"total":   len(list),
	})
}

// ═══════════════════════════════════════════════
// 三、心理健康模块 /api/v1/mental
// ═══════════════════════════════════════════════

// ListPsychScales 心理测评量表列表
// GET /api/v1/mental/scales?category=
