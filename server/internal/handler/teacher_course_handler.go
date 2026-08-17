package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// TeacherCourseHandler 教师授课关系申报+教辅审核接口（R3）
// 端点为新增附加，不触碰存量 outcome / gov_ticket 逻辑。
type TeacherCourseHandler struct {
	svc *service.TeacherCourseService
}

func NewTeacherCourseHandler(svc *service.TeacherCourseService) *TeacherCourseHandler {
	return &TeacherCourseHandler{svc: svc}
}

func teacherCourseRole(c *gin.Context) (int64, string, string) {
	if u := middleware.GetUserContext(c); u != nil {
		return u.UserID, u.Username, u.Role
	}
	return 0, "", ""
}

// SubmitTeacherCourse 教师申报授课关系
// POST /api/v1/teacher/courses/apply  (teacher.grade.write)
func (h *TeacherCourseHandler) SubmitTeacherCourse(c *gin.Context) {
	var req struct {
		CourseID   string `json:"course_id"`
		CourseName string `json:"course_name"`
		Semester   string `json:"semester"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, _, opRole := teacherCourseRole(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	// 仅教师本人可申报（TeacherGradeWrite 门控在上层，此处进一步禁止其他角色代报）
	if opRole != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅教师本人可申报授课关系"})
		return
	}
	if req.CourseID == "" || req.Semester == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "课程和学期必填"})
		return
	}
	id, status, err := h.svc.SubmitTeacherCourse(c.Request.Context(), opID, req.CourseID, req.CourseName, req.Semester)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	msg := "申报成功，待教辅/教务审核"
	if status == repository.CourseStatusApproved {
		msg = "该授课关系已通过，可录入成绩"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": msg, "id": id, "status": status, "data_source": "real"})
}

// ListMyTeacherCourses 教师查询本人申报列表
// GET /api/v1/teacher/courses/mine  (teacher.grade.write)
func (h *TeacherCourseHandler) ListMyTeacherCourses(c *gin.Context) {
	opID, _, _ := teacherCourseRole(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	list, err := h.svc.ListMyTeacherCourses(c.Request.Context(), opID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if list == nil {
		list = []repository.TeacherCourse{}
	}
	// 诚实空态：0 申报返回空数组
	c.JSON(http.StatusOK, gin.H{"code": 0, "list": list, "data_source": "real"})
}

// ListPendingTeacherCourses 教辅/教务查询待审核申报
// GET /api/v1/assistant/courses/pending  (teacher.course.review)
func (h *TeacherCourseHandler) ListPendingTeacherCourses(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	list, err := h.svc.ListPendingTeacherCourses(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if list == nil {
		list = []repository.TeacherCourse{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "list": list, "data_source": "real"})
}

// ReviewTeacherCourse 教辅/教务审核申报
// PUT /api/v1/assistant/courses/review/:id  (teacher.course.review)
func (h *TeacherCourseHandler) ReviewTeacherCourse(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Status string `json:"status"` // approved/rejected
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, opName, _ := teacherCourseRole(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	if err := h.svc.ReviewTeacherCourse(c.Request.Context(), id, opID, opName, req.Status, req.Note); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	msg := "已通过（该教师可录入此课程成绩）"
	if req.Status == repository.CourseStatusRejected {
		msg = "已驳回（教师可重新申报）"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": msg})
}

// CountPending 待审核申报角标
// GET /api/v1/assistant/courses/pending-count  (teacher.course.review)
func (h *TeacherCourseHandler) CountPending(c *gin.Context) {
	n, err := h.svc.CountPending(c.Request.Context())
	if err != nil {
		n = 0
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "pending": n, "data_source": "real"})
}
