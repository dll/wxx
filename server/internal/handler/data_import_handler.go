package handler

import (
	"net/http"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// DataImportHandler 数据底座导入接口（成绩 / 课表）
type DataImportHandler struct {
	phase3 *service.Phase3Service
}

// NewDataImportHandler 创建数据导入 handler
func NewDataImportHandler(phase3 *service.Phase3Service) *DataImportHandler {
	return &DataImportHandler{phase3: phase3}
}

// ImportGrades 批量导入成绩（JSON）
func (h *DataImportHandler) ImportGrades(c *gin.Context) {
	if h.phase3 == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据服务未就绪"})
		return
	}
	var req struct {
		Grades []*repository.GradeRow `json:"grades"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Grades) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：需要 grades 数组"})
		return
	}
	if len(req.Grades) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "单次最多导入 2000 条"})
		return
	}
	res := h.phase3.ImportGrades(req.Grades)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": res})
}

// ImportSchedules 批量导入课表（JSON）。
// 每条 ScheduleRow 推荐提供 username（学号/工号，稳定键）而非内部自增 user_id，
// 后端按 username 解析真实账号，避免填错 user_id 使课程挂到错误账号（2026-09-01 修复）。
func (h *DataImportHandler) ImportSchedules(c *gin.Context) {
	if h.phase3 == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据服务未就绪"})
		return
	}
	var req struct {
		Schedules []*repository.ScheduleRow `json:"schedules"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Schedules) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：需要 schedules 数组"})
		return
	}
	if len(req.Schedules) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "单次最多导入 2000 条"})
		return
	}
	res := h.phase3.ImportSchedules(req.Schedules)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": res})
}

// ReassignSchedules 按工号归位课表（彻底修复历史错挂的课程显示）。
// POST /api/v1/admin/schedules/reassign  body: {"username":"120001"}
// 把 course_schedules.owner_username=该工号 的所有课表 user_id 改为工号解析出的正确账号，
// 覆盖历史按错误 user_id 导入的课表。返回受影响条数与该工号名下课表总数。
func (h *DataImportHandler) ReassignSchedules(c *gin.Context) {
	if h.phase3 == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据服务未就绪"})
		return
	}
	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Username) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：需要 username(工号/学号)"})
		return
	}
	affected, total, err := h.phase3.ReassignSchedulesByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"reassigned": affected, "total": total, "username": req.Username}})
}

// ImportMySchedule 学生个人导入自己的课表（角色化导入：仅限导入本人课表）
// POST /api/v1/student/schedule/import
// 学生从门户登录查到自己的课表后，按格式填入，由本接口导入本人课表；
// 后端强制 user_id = 当前登录学生，杜绝越权改他人课表。
func (h *DataImportHandler) ImportMySchedule(c *gin.Context) {
	if h.phase3 == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据服务未就绪"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil || userCtx.UserID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	// 角色化：仅学生本人可导入自己的课表（学生会/教辅/辅导员走 /admin/schedules/import 批量）
	var req struct {
		Schedules []*repository.ScheduleRow `json:"schedules"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Schedules) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：需要 schedules 数组"})
		return
	}
	if len(req.Schedules) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "单次最多导入 2000 条"})
		return
	}
	// 强制本人：忽略请求里的 user_id，全部写当前登录学生。
	// 2026-09-01：同时清空 username（课表导入按 username 解析归属），
	// 防止学生误/恶意传他人 username 把课表写到别的账号。
	for _, s := range req.Schedules {
		s.UserID = userCtx.UserID
		s.Username = "" // 归属强制为本人，杜绝跨账号挂课表
	}
	res := h.phase3.ImportSchedules(req.Schedules)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "导入成功(仅本人课表)", "data": res})
}

// ImportTeacherGrades 教师自助录入所授班级成绩（方案 A：教师自主声明授课关系）
// POST /api/v1/teacher/grades/import  （能力门控 TeacherGradeWrite）
// 教师在前端选定课程 + 学生学号集合 + 真实成绩，本接口校验：
//   - 每条记录 target 必须为 role='student' 的学生（不得对教师/管理员等写成绩）
//   - created_by 强制写当前教师 user_id（审计可追溯谁的声明）
//
// 幂等沿用 UpsertGrade。
func (h *DataImportHandler) ImportTeacherGrades(c *gin.Context) {
	if h.phase3 == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据服务未就绪"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil || userCtx.UserID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	var req struct {
		Grades []*repository.GradeRow `json:"grades"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Grades) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误：需要 grades 数组"})
		return
	}
	if len(req.Grades) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "单次最多导入 2000 条"})
		return
	}
	res := h.phase3.ImportTeacherGrades(req.Grades, userCtx.UserID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "教师成绩录入完成", "data": res})
}

// ListMyTeacherGrades 教师查看自己声明录入的成绩记录（读取边界=created_by）
// GET /api/v1/teacher/grades/mine   （能力门控 TeacherGradeWrite）
func (h *DataImportHandler) ListMyTeacherGrades(c *gin.Context) {
	if h.phase3 == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据服务未就绪"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil || userCtx.UserID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	list, err := h.phase3.ListTeacherGrades(userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}
	// 诚实空态：无数据时直接返回空数组，不填假成绩
	if list == nil {
		list = []*repository.ListedGrade{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}
