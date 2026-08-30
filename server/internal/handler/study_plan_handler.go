package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// StudyPlanHandler 学习计划与校历 HTTP handler（P4-d：SQL 已下沉 StudyPlanRepo）
type StudyPlanHandler struct {
	studyPlanRepo *repository.StudyPlanRepo
	studyPlanSvc  *service.StudyPlanService
}

// NewStudyPlanHandler 创建学习计划 handler
func NewStudyPlanHandler(studyPlanRepo *repository.StudyPlanRepo, studyPlanSvc *service.StudyPlanService) *StudyPlanHandler {
	return &StudyPlanHandler{studyPlanRepo: studyPlanRepo, studyPlanSvc: studyPlanSvc}
}

// ═══════════════════════════════════════════════
// 一、校历接口 /api/v1/study/calendar
// ═══════════════════════════════════════════════

// GetCurrentCalendar 获取当前学期校历（含近期事件）
// GET /api/v1/study/calendar/current
func (h *StudyPlanHandler) GetCurrentCalendar(c *gin.Context) {
	calendar, currentWeek, err := h.studyPlanRepo.ResolveCurrentCalendar()
	if err != nil {
		log.Printf("study_plan GetCurrentCalendar err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询当前校历失败，请稍后重试"})
		return
	}

	// 近期事件：当前日期前后 30 天内
	today := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	to := time.Now().AddDate(0, 0, 30).Format("2006-01-02")

	events := make([]*model.CalendarEvent, 0)
	if calendar != nil {
		events, err = h.studyPlanRepo.ListRecentEvents(calendar.SemesterCode, from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询校历事件失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":         0,
		"message":      "success",
		"data":         calendar,
		"current_week": currentWeek,
		"today":        today,
		"events":       events,
	})
}

// GetCalendarBySemester 获取指定学期校历（含全部事件）
// GET /api/v1/study/calendar/:semester_code
func (h *StudyPlanHandler) GetCalendarBySemester(c *gin.Context) {
	semesterCode := c.Param("semester_code")
	if semesterCode == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "semester_code 不能为空"})
		return
	}

	calendar, err := h.studyPlanRepo.GetCalendarBySemester(semesterCode)
	if err == repository.ErrCalendarNotFound {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "学期校历不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询校历失败"})
		return
	}

	events, err := h.studyPlanRepo.ListEventsBySemester(semesterCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询校历事件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    calendar,
		"events":  events,
	})
}

// ═══════════════════════════════════════════════
// 二、课表接口 /api/v1/study/timetable
// ═══════════════════════════════════════════════

// GetMyTimetable 获取我的课表（按 weekday 分组）
// GET /api/v1/study/timetable?semester_code=
func (h *StudyPlanHandler) GetMyTimetable(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	semesterCode := c.Query("semester_code")
	// 默认使用当前学期
	if semesterCode == "" {
		calendar, _, err := h.studyPlanRepo.ResolveCurrentCalendar()
		if err != nil || calendar == nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "无法确定当前学期"})
			return
		}
		semesterCode = calendar.SemesterCode
	}

	all, err := h.studyPlanRepo.ListTimetable(userCtx.UserID, semesterCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询课表失败"})
		return
	}

	// 按 weekday 分组：1-7 → []
	grouped := make(map[int][]*model.CourseScheduleItem)
	for _, item := range all {
		grouped[item.Weekday] = append(grouped[item.Weekday], item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":          0,
		"message":       "success",
		"semester_code": semesterCode,
		"data":          all,
		"grouped":       grouped,
		"total":         len(all),
	})
}

// ═══════════════════════════════════════════════
// 请求校验辅助（保持 handler 层职责：入参合法性）
// ═══════════════════════════════════════════════

// isValidPlanType 校验计划类型
func isValidPlanType(t string) bool {
	switch t {
	case "weekly", "monthly", "quarterly", "semester", "yearly", "four_year":
		return true
	}
	return false
}

// isValidPlanStatus 校验计划状态
func isValidPlanStatus(s string) bool {
	switch s {
	case "active", "completed", "archived":
		return true
	}
	return false
}

// isValidTaskStatus 校验任务状态
func isValidTaskStatus(s string) bool {
	switch s {
	case "pending", "in_progress", "done", "skipped":
		return true
	}
	return false
}

// calcDefaultEndDate 根据 plan_type 推算默认结束日期
func calcDefaultEndDate(planType string, today time.Time) string {
	switch planType {
	case "weekly":
		return today.AddDate(0, 0, 7).Format("2006-01-02")
	case "monthly":
		return today.AddDate(0, 1, 0).Format("2006-01-02")
	case "quarterly":
		return today.AddDate(0, 3, 0).Format("2006-01-02")
	case "semester":
		return today.AddDate(0, 5, 0).Format("2006-01-02")
	case "yearly":
		return today.AddDate(1, 0, 0).Format("2006-01-02")
	case "four_year":
		return today.AddDate(4, 0, 0).Format("2006-01-02")
	default:
		return today.AddDate(0, 0, 7).Format("2006-01-02")
	}
}
