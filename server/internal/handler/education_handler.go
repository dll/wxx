package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/gin-gonic/gin"
)

// EducationHandler 学生教育三大模块（就业指导、学业学习、心理健康）HTTP handler
// P4-d 完成：心理健康/就业/健康/学业域 SQL 全部下沉对应 Repo
type EducationHandler struct {
	db           *sql.DB
	mentalRepo   *repository.MentalHealthRepo
	careerRepo   *repository.CareerRepo
	healthRepo   *repository.HealthRepo
	activityRepo *repository.HealthActivityRepo
	studyRepo    *repository.StudyRepo
}

// NewEducationHandler 创建教育模块 handler
func NewEducationHandler(db *sql.DB, mentalRepo *repository.MentalHealthRepo, careerRepo *repository.CareerRepo, healthRepo *repository.HealthRepo, activityRepo *repository.HealthActivityRepo, studyRepo *repository.StudyRepo) *EducationHandler {
	return &EducationHandler{db: db, mentalRepo: mentalRepo, careerRepo: careerRepo, healthRepo: healthRepo, activityRepo: activityRepo, studyRepo: studyRepo}
}

// generateID 生成简短唯一 ID（前缀 + 随机 hex）
func generateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

// ═══════════════════════════════════════════════
// 学业学习模块 /api/v1/study（SQL 已下沉 StudyRepo）
// ═══════════════════════════════════════════════

// ListCourses 课程列表
// GET /api/v1/study/courses?department=&category=&semester=
func (h *EducationHandler) ListCourses(c *gin.Context) {
	list, err := h.studyRepo.ListCourses(c.Query("department"), c.Query("category"), c.Query("semester"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询课程列表失败"})
		return
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

	detail, err := h.studyRepo.GetCourseDetail(id)
	if err == repository.ErrCourseNotFound {
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

	list, err := h.studyRepo.ListMyGrades(userCtx.UserID, c.Query("semester"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询成绩失败"})
		return
	}

	// 按学期分组（保持出现顺序）
	grouped := make(map[string][]*model.GradeItem)
	var semesters []string
	for _, item := range list {
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

	summary, err := h.studyRepo.GetGradeSummary(userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询成绩统计失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    summary,
	})
}

// ListLearningResources 学习资源列表
// GET /api/v1/study/resources?course_id=&resource_type=&page=1&page_size=20
func (h *EducationHandler) ListLearningResources(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	list, total, err := h.studyRepo.ListLearningResources(c.Query("course_id"), c.Query("resource_type"), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习资源失败"})
		return
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
	list, err := h.studyRepo.ListExamSchedules(c.Query("semester"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询考试安排失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
		"total":   len(list),
	})
}
