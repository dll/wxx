package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// HomeworkHandler 教师作业信息发布 + 课程成绩统计接口（P2 轻量版）
// 端点为新增附加，不触碰存量 outcome / gov_ticket / teacher_course 逻辑。
// 门控 TeacherGradeWrite（上层 RequireCapability）；本层再做登录/教师身份校验（双保险）。
type HomeworkHandler struct {
	svc *service.HomeworkService
}

func NewHomeworkHandler(svc *service.HomeworkService) *HomeworkHandler {
	return &HomeworkHandler{svc: svc}
}

func homeworkTeacherID(c *gin.Context) (int64, string) {
	if u := middleware.GetUserContext(c); u != nil {
		return u.UserID, u.Role
	}
	return 0, ""
}

// PublishHomework 发布作业信息
// POST /api/v1/teacher/homework  (teacher.grade.write)
func (h *HomeworkHandler) PublishHomework(c *gin.Context) {
	var req struct {
		CourseID    string `json:"course_id"`
		CourseName  string `json:"course_name"`
		Semester    string `json:"semester"`
		Title       string `json:"title"`
		Description string `json:"description"`
		PublishAt   string `json:"publish_at"`
		DueAt       string `json:"due_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, opRole := homeworkTeacherID(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	if opRole != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅教师本人可发布作业"})
		return
	}
	id, existed, err := h.svc.PublishHomework(c.Request.Context(), opID, req.CourseID, req.CourseName,
		req.Semester, req.Title, req.Description, req.PublishAt, req.DueAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	msg := "作业发布成功"
	if existed {
		msg = "该课程作业已发布（同课程同学期同标题已存在），未重复发布"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": msg, "id": id, "existed": existed, "data_source": "real"})
}

// UpdateHomework 编辑本人作业
// PUT /api/v1/teacher/homework/:id  (teacher.grade.write)
func (h *HomeworkHandler) UpdateHomework(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		PublishAt   string `json:"publish_at"`
		DueAt       string `json:"due_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, _ := homeworkTeacherID(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	if err := h.svc.UpdateHomework(c.Request.Context(), id, opID, req.Title, req.Description, req.PublishAt, req.DueAt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "作业已更新"})
}

// ArchiveHomework 下架作业（软删）
// DELETE /api/v1/teacher/homework/:id  (teacher.grade.write)
func (h *HomeworkHandler) ArchiveHomework(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, _ := homeworkTeacherID(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	if err := h.svc.ArchiveHomework(c.Request.Context(), id, opID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "作业已下架（可复核，不物理删除）"})
}

// ListMyHomework 教师本人的作业清单
// GET /api/v1/teacher/homework/mine  (teacher.grade.write)
func (h *HomeworkHandler) ListMyHomework(c *gin.Context) {
	opID, _ := homeworkTeacherID(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	list, err := h.svc.ListMyHomework(c.Request.Context(), opID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if list == nil {
		list = []repository.Homework{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "list": list, "data_source": "real"})
}

// ListApprovedCourses 教师本人 approved 授课课程白名单（前端课程下拉数据源，仅真实 approved）
// GET /api/v1/teacher/homework/courses  (teacher.grade.write)
func (h *HomeworkHandler) ListApprovedCourses(c *gin.Context) {
	opID, _ := homeworkTeacherID(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	list, err := h.svc.ListApprovedCourses(c.Request.Context(), opID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if list == nil {
		list = []repository.TeacherCourse{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "list": list, "data_source": "real"})
}

// GradeStatsByCourse 课程成绩只读统计（P1，按课程维度）
// GET /api/v1/teacher/homework/:courseId/grade-stats?semester=...  (teacher.grade.write)
func (h *HomeworkHandler) GradeStatsByCourse(c *gin.Context) {
	opID, _ := homeworkTeacherID(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	courseID := c.Param("courseId")
	semester := c.Query("semester")
	stats, err := h.svc.GradeStatsByCourse(c.Request.Context(), opID, courseID, semester)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if stats == nil {
		stats = &repository.CourseGradeStats{
			CourseID:     courseID,
			Semester:     semester,
			NotAvailable: true,
			Levels:       map[string]int{"优秀": 0, "良好": 0, "及格": 0, "不及格": 0},
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "stats": stats, "data_source": "real"})
}
