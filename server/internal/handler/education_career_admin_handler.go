package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// 管理端：就业指导内容管理（学校/学院管理员）
// ═══════════════════════════════════════════════

// AdminListCareerPolicies 管理端就业政策列表（含非 active）
// GET /api/v1/career/admin/policies?page=&page_size=&category=
func (h *EducationHandler) AdminListCareerPolicies(c *gin.Context) {
	category := c.Query("category")
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
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM career_policies `+whereSQL, args...).Scan(&total)

	rows, err := h.db.Query(
		`SELECT id, policy_id, title, category, level, source, summary, status, view_count, created_at
		 FROM career_policies `+whereSQL+` ORDER BY id DESC LIMIT ? OFFSET ?`,
		append(args, pageSize, offset)...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询就业政策失败"})
		return
	}
	defer rows.Close()

	type item struct {
		ID        int64  `json:"id"`
		PolicyID  string `json:"policy_id"`
		Title     string `json:"title"`
		Category  string `json:"category"`
		Level     string `json:"level"`
		Source    string `json:"source"`
		Summary   string `json:"summary"`
		Status    string `json:"status"`
		ViewCount int    `json:"view_count"`
		CreatedAt string `json:"created_at"`
	}
	var list []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.ID, &it.PolicyID, &it.Title, &it.Category, &it.Level, &it.Source, &it.Summary, &it.Status, &it.ViewCount, &it.CreatedAt); err == nil {
			list = append(list, it)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list, "total": total})
}

// AdminCreateCareerPolicy 新增就业政策
// POST /api/v1/career/admin/policies
func (h *EducationHandler) AdminCreateCareerPolicy(c *gin.Context) {
	var req struct {
		PolicyID string `json:"policy_id"`
		Title    string `json:"title"`
		Category string `json:"category"`
		Level    string `json:"level"`
		Source   string `json:"source"`
		Content  string `json:"content"`
		Summary  string `json:"summary"`
		Tags     string `json:"tags"`
		Status   string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "政策标题不能为空"})
		return
	}
	if req.PolicyID == "" {
		req.PolicyID = "career-" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	if req.Category == "" {
		req.Category = "employment_policy"
	}
	if req.Level == "" {
		req.Level = "school"
	}
	if req.Status == "" {
		req.Status = "active"
	}
	_, err := h.db.Exec(
		`INSERT INTO career_policies (policy_id, title, category, level, source, content, summary, tags, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.PolicyID, req.Title, req.Category, req.Level, req.Source, req.Content, req.Summary, req.Tags, req.Status,
	)
	if err != nil {
		log.Printf("新增就业政策失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "新增政策失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// AdminDeleteCareerPolicy 删除就业政策
// DELETE /api/v1/career/admin/policies/:id
func (h *EducationHandler) AdminDeleteCareerPolicy(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if _, err := h.db.Exec(`DELETE FROM career_policies WHERE id = ?`, id); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "删除政策失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// AdminListJobPostings 管理端招聘信息列表
// GET /api/v1/career/admin/jobs?page=&page_size=
func (h *EducationHandler) AdminListJobPostings(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM job_postings`).Scan(&total)

	rows, err := h.db.Query(
		`SELECT id, job_id, company_name, position_name, position_type, industry, salary_min, salary_max, location, education, status, created_at
		 FROM job_postings ORDER BY id DESC LIMIT ? OFFSET ?`,
		pageSize, offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询招聘信息失败"})
		return
	}
	defer rows.Close()

	type item struct {
		ID           int64  `json:"id"`
		JobID        string `json:"job_id"`
		CompanyName  string `json:"company_name"`
		PositionName string `json:"position_name"`
		PositionType string `json:"position_type"`
		Industry     string `json:"industry"`
		SalaryMin    int    `json:"salary_min"`
		SalaryMax    int    `json:"salary_max"`
		Location     string `json:"location"`
		Education    string `json:"education"`
		Status       string `json:"status"`
		CreatedAt    string `json:"created_at"`
	}
	var list []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.ID, &it.JobID, &it.CompanyName, &it.PositionName, &it.PositionType, &it.Industry, &it.SalaryMin, &it.SalaryMax, &it.Location, &it.Education, &it.Status, &it.CreatedAt); err == nil {
			list = append(list, it)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list, "total": total})
}

// AdminCreateJobPosting 新增招聘信息
// POST /api/v1/career/admin/jobs
func (h *EducationHandler) AdminCreateJobPosting(c *gin.Context) {
	var req struct {
		CompanyName  string `json:"company_name"`
		PositionName string `json:"position_name"`
		PositionType string `json:"position_type"`
		Industry     string `json:"industry"`
		Location     string `json:"location"`
		Education    string `json:"education"`
		Description  string `json:"description"`
		Requirement  string `json:"requirement"`
		Status       string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}
	if req.CompanyName == "" || req.PositionName == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "公司名和职位名不能为空"})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	_, err := h.db.Exec(
		`INSERT INTO job_postings (job_id, company_name, position_name, position_type, industry, location, education, description, requirement, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"job-"+strconv.FormatInt(time.Now().Unix(), 10), req.CompanyName, req.PositionName, req.PositionType, req.Industry, req.Location, req.Education, req.Description, req.Requirement, req.Status,
	)
	if err != nil {
		log.Printf("新增招聘信息失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "新增招聘信息失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// AdminDeleteJobPosting 删除招聘信息
// DELETE /api/v1/career/admin/jobs/:id
func (h *EducationHandler) AdminDeleteJobPosting(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if _, err := h.db.Exec(`DELETE FROM job_postings WHERE id = ?`, id); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "删除招聘信息失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
