package handler

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// StudentFeaturesHandler 学生功能 HTTP handler（竞赛+规划+入党+社团）
type StudentFeaturesHandler struct {
	svc *service.StudentFeaturesService
}

// NewStudentFeaturesHandler 创建学生功能 handler
func NewStudentFeaturesHandler(svc *service.StudentFeaturesService) *StudentFeaturesHandler {
	return &StudentFeaturesHandler{svc: svc}
}

// ══════════════════════════════════════════════════════════════
// 学科竞赛
// ══════════════════════════════════════════════════════════════

// ListCompetitions 获取竞赛列表
// GET /api/v1/competition/list?level=&category=&status=&page=&page_size=
func (h *StudentFeaturesHandler) ListCompetitions(c *gin.Context) {
	level := c.Query("level")
	category := c.Query("category")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	items, total, err := h.svc.ListCompetitions(level, category, status, page, pageSize)
	if err != nil {
		log.Printf("查询竞赛列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items, "total": total, "page": page, "page_size": pageSize})
}

// GetCompetition 获取竞赛详情
// GET /api/v1/competition/:id
func (h *StudentFeaturesHandler) GetCompetition(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	item, err := h.svc.GetCompetition(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "竞赛不存在"})
			return
		}
		log.Printf("查询竞赛详情失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": item})
}

// RegisterCompetition 报名竞赛
// POST /api/v1/competition/register
func (h *StudentFeaturesHandler) RegisterCompetition(c *gin.Context) {
	var req struct {
		CompetitionID int64  `json:"competition_id" binding:"required"`
		TeamName      string `json:"team_name"`
		TeamMembers   string `json:"team_members"`
		AdvisorName   string `json:"advisor_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	id, err := h.svc.RegisterCompetition(req.CompetitionID, userCtx.UserID, userCtx.Username, userCtx.DisplayName, userCtx.OwnerScope, "", "", req.TeamName, req.TeamMembers, req.AdvisorName)
	if err != nil {
		log.Printf("竞赛报名失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "报名失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "报名成功", "data": gin.H{"id": id}})
}

// GetMyCompetitionRegistrations 获取我的竞赛报名
// GET /api/v1/competition/my-registrations
func (h *StudentFeaturesHandler) GetMyCompetitionRegistrations(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, err := h.svc.GetMyCompetitionRegistrations(userCtx.UserID)
	if err != nil {
		log.Printf("查询竞赛报名失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

// SubmitWork 提交作品
// POST /api/v1/competition/submit-work
func (h *StudentFeaturesHandler) SubmitWork(c *gin.Context) {
	var req struct {
		RegistrationID int64  `json:"registration_id" binding:"required"`
		WorkTitle      string `json:"work_title" binding:"required"`
		WorkDesc       string `json:"work_description"`
		WorkFileURL    string `json:"work_file_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	if err := h.svc.SubmitWork(req.RegistrationID, req.WorkTitle, req.WorkDesc, req.WorkFileURL); err != nil {
		log.Printf("作品提交失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "提交失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "作品提交成功"})
}

// GetCompetitionStats 竞赛统计
// GET /api/v1/competition/stats
func (h *StudentFeaturesHandler) GetCompetitionStats(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	stats, err := h.svc.GetCompetitionStats()
	if err != nil {
		log.Printf("查询竞赛统计失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询统计失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": stats})
}

// CompetitionMatch 基于学生专业/学院对真实竞赛做个性化匹配推荐
// GET /api/v1/competition/match?major=&college=&limit=
// major/college 可选：前端从登录资料带入；为空则退化为按级别+报名状态排序的通用推荐。
func (h *StudentFeaturesHandler) CompetitionMatch(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	major := c.Query("major")
	college := c.Query("college")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	items, err := h.svc.MatchCompetitions(major, college, limit)
	if err != nil {
		log.Printf("竞赛个性化匹配失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "匹配失败"})
		return
	}
	source := "real"
	if len(items) == 0 {
		source = "empty"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items, "total": len(items), "data_source": source})
}

// ══════════════════════════════════════════════════════════════
// 大学规划
// ══════════════════════════════════════════════════════════════

// ListPlanTemplates 获取规划模板列表
// GET /api/v1/plan/templates?category=
func (h *StudentFeaturesHandler) ListPlanTemplates(c *gin.Context) {
	category := c.Query("category")
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, err := h.svc.ListPlanTemplates(category)
	if err != nil {
		log.Printf("查询规划模板失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

// ListMyPlans 获取我的规划列表
// GET /api/v1/plan/my-plans
func (h *StudentFeaturesHandler) ListMyPlans(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, err := h.svc.ListMyPlans(userCtx.UserID)
	if err != nil {
		log.Printf("查询我的规划失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

// CreatePlan 创建规划
// POST /api/v1/plan/create
func (h *StudentFeaturesHandler) CreatePlan(c *gin.Context) {
	var req struct {
		TemplateID   int    `json:"template_id"`
		Title        string `json:"title" binding:"required"`
		Category     string `json:"category" binding:"required"`
		AcademicYear int    `json:"academic_year"`
		Semester     int    `json:"semester"`
		Goals        string `json:"goals"`
		Content      string `json:"content"` // 前端兼容字段
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	goals := req.Goals
	if goals == "" && req.Content != "" {
		goals = req.Content
	}
	id, err := h.svc.CreatePlan(userCtx.UserID, req.TemplateID, req.Title, req.Category, req.AcademicYear, req.Semester, goals)
	if err != nil {
		log.Printf("创建规划失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "规划创建成功", "data": gin.H{"id": id}})
}

// SubmitPlan 提交规划审核
// PUT /api/v1/plan/:id/submit
func (h *StudentFeaturesHandler) SubmitPlan(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	if err := h.svc.SubmitPlan(id); err != nil {
		log.Printf("提交规划失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "提交失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已提交审核"})
}

// ReviewPlan 审核规划（管理员）
// PUT /api/v1/plan/:id/review
func (h *StudentFeaturesHandler) ReviewPlan(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Status  string `json:"status" binding:"required"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	if userCtx.Role != "sys_admin" && userCtx.Role != "college_admin" && userCtx.Role != "counselor" {
		c.JSON(http.StatusForbidden, model.ErrorResponse{Code: 403, Message: "无权操作"})
		return
	}
	if err := h.svc.ReviewPlan(id, req.Status, req.Comment); err != nil {
		log.Printf("审核规划失败: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审核完成"})
}

// ══════════════════════════════════════════════════════════════
// 入党教育
// ══════════════════════════════════════════════════════════════

// ListPartyStages 获取入党阶段列表
// GET /api/v1/party/stages
func (h *StudentFeaturesHandler) ListPartyStages(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, err := h.svc.ListPartyStages()
	if err != nil {
		log.Printf("查询入党阶段失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

// GetMyPartyProgress 获取我的入党进度
// GET /api/v1/party/my-progress
func (h *StudentFeaturesHandler) GetMyPartyProgress(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	item, err := h.svc.GetMyPartyProgress(userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "暂无入党进度记录", "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": item})
}

// UpdatePartyProgress 更新入党进度
// PUT /api/v1/party/my-progress
func (h *StudentFeaturesHandler) UpdatePartyProgress(c *gin.Context) {
	var req struct {
		Stage string `json:"stage" binding:"required"`
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	if err := h.svc.UpdatePartyProgress(userCtx.UserID, req.Stage, req.Notes); err != nil {
		log.Printf("更新入党进度失败: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "进度已更新"})
}

// ListMyStudyRecords 获取我的学习记录
// GET /api/v1/party/my-study-records
func (h *StudentFeaturesHandler) ListMyStudyRecords(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, err := h.svc.ListMyStudyRecords(userCtx.UserID)
	if err != nil {
		log.Printf("查询学习记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

// AddStudyRecord 添加学习记录
// POST /api/v1/party/study-record
func (h *StudentFeaturesHandler) AddStudyRecord(c *gin.Context) {
	var req struct {
		StudyType   string `json:"study_type"`
		Title       string `json:"title" binding:"required"`
		Content     string `json:"content"`
		Duration    int    `json:"duration"`
		StudyDate   string `json:"study_date"`
		Certificate string `json:"certificate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	id, err := h.svc.AddStudyRecord(userCtx.UserID, req.StudyType, req.Title, req.Content, req.Duration, req.StudyDate, req.Certificate)
	if err != nil {
		log.Printf("添加学习记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "添加失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "记录添加成功", "data": gin.H{"id": id}})
}

// GetPartyStats 入党统计
// GET /api/v1/party/stats
func (h *StudentFeaturesHandler) GetPartyStats(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	stats, err := h.svc.GetPartyStats()
	if err != nil {
		log.Printf("查询入党统计失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询统计失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": stats})
}

// ══════════════════════════════════════════════════════════════
// 社团生活
// ══════════════════════════════════════════════════════════════

// ListClubs 获取社团列表
// GET /api/v1/club/list?category=&page=&page_size=
func (h *StudentFeaturesHandler) ListClubs(c *gin.Context) {
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, total, err := h.svc.ListClubs(category, page, pageSize)
	if err != nil {
		log.Printf("查询社团列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items, "total": total, "page": page, "page_size": pageSize})
}

// GetClub 获取社团详情
// GET /api/v1/club/:id
func (h *StudentFeaturesHandler) GetClub(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	item, err := h.svc.GetClub(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "社团不存在"})
			return
		}
		log.Printf("查询社团详情失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": item})
}

// JoinClub 加入社团
// POST /api/v1/club/join
func (h *StudentFeaturesHandler) JoinClub(c *gin.Context) {
	var req struct {
		ClubID int64 `json:"club_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	id, err := h.svc.JoinClub(req.ClubID, userCtx.UserID, userCtx.Username, userCtx.DisplayName, "member")
	if err != nil {
		log.Printf("加入社团失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "加入失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "加入成功", "data": gin.H{"id": id}})
}

// GetMyClubs 获取我加入的社团
// GET /api/v1/club/my-clubs
func (h *StudentFeaturesHandler) GetMyClubs(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, err := h.svc.GetMyClubs(userCtx.UserID)
	if err != nil {
		log.Printf("查询我的社团失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

// ListClubActivities 获取社团活动列表
// GET /api/v1/club/activities?club_id=&status=&page=&page_size=
func (h *StudentFeaturesHandler) ListClubActivities(c *gin.Context) {
	clubID, _ := strconv.ParseInt(c.Query("club_id"), 10, 64)
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, total, err := h.svc.ListClubActivities(clubID, status, page, pageSize)
	if err != nil {
		log.Printf("查询社团活动失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items, "total": total, "page": page, "page_size": pageSize})
}

// RegisterClubActivity 报名社团活动
// POST /api/v1/club/activity/register
func (h *StudentFeaturesHandler) RegisterClubActivity(c *gin.Context) {
	var req struct {
		ActivityID int64 `json:"activity_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	id, err := h.svc.RegisterClubActivity(req.ActivityID, userCtx.UserID, userCtx.DisplayName)
	if err != nil {
		log.Printf("报名社团活动失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "报名失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "报名成功", "data": gin.H{"id": id}})
}
