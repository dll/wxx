package handler

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

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
		log.Printf("competition RegisterCompetition bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
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
		log.Printf("competition SubmitWork bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
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

// ======================== 竞赛管理（管理端） ========================

// AdminListCompetitions 管理端竞赛列表
// GET /api/v1/competition/admin/list?page=&page_size=&level=&category=&status=
func (h *StudentFeaturesHandler) AdminListCompetitions(c *gin.Context) {
	level := c.Query("level")
	category := c.Query("category")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	items, total, err := h.svc.AdminListCompetitions(level, category, status, page, pageSize)
	if err != nil {
		log.Printf("管理端竞赛列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询竞赛失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items, "total": total})
}

// AdminCreateCompetition 新增竞赛
// POST /api/v1/competition/admin
func (h *StudentFeaturesHandler) AdminCreateCompetition(c *gin.Context) {
	var req struct {
		Name              string `json:"name"`
		Level             string `json:"level"`
		Category          string `json:"category"`
		Organizer         string `json:"organizer"`
		Description       string `json:"description"`
		Requirements      string `json:"requirements"`
		RegistrationStart string `json:"registration_start"`
		RegistrationEnd   string `json:"registration_end"`
		CompetitionDate   string `json:"competition_date"`
		Status            string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "竞赛名称不能为空"})
		return
	}
	if req.Level == "" {
		req.Level = "school"
	}
	if req.Status == "" {
		req.Status = "upcoming"
	}
	fields := map[string]interface{}{
		"name": req.Name, "level": req.Level, "category": req.Category,
		"organizer": req.Organizer, "description": req.Description,
		"requirements": req.Requirements, "registration_start": req.RegistrationStart,
		"registration_end": req.RegistrationEnd, "competition_date": req.CompetitionDate,
		"status": req.Status,
	}
	id, err := h.svc.AdminCreateCompetition(fields)
	if err != nil {
		log.Printf("新增竞赛失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "新增竞赛失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"id": id}})
}

// AdminUpdateCompetition 更新竞赛
// PUT /api/v1/competition/admin/:id
func (h *StudentFeaturesHandler) AdminUpdateCompetition(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var fields map[string]interface{}
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}
	if err := h.svc.AdminUpdateCompetition(id, fields); err != nil {
		log.Printf("更新竞赛失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新竞赛失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// AdminDeleteCompetition 删除竞赛
// DELETE /api/v1/competition/admin/:id
func (h *StudentFeaturesHandler) AdminDeleteCompetition(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.AdminDeleteCompetition(id); err != nil {
		log.Printf("删除竞赛失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "删除竞赛失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
