package handler

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// 就业指导相关 handler（从 education_handler.go 按业务域拆分）
func (h *EducationHandler) ListCareerPolicies(c *gin.Context) {
	category := c.Query("category")
	level := c.Query("level")
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
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	if level != "" {
		where = append(where, "level = ?")
		args = append(args, level)
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM career_policies WHERE "+whereSQL, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询就业政策失败",
		})
		return
	}

	rows, err := h.db.Query(
		"SELECT id, policy_id, title, category, level, source, summary, publish_date, tags, view_count, created_at "+
			"FROM career_policies WHERE "+whereSQL+" ORDER BY publish_date DESC, id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询就业政策失败",
		})
		return
	}
	defer rows.Close()

	type PolicyItem struct {
		ID          int64   `json:"id"`
		PolicyID    string  `json:"policy_id"`
		Title       string  `json:"title"`
		Category    string  `json:"category"`
		Level       string  `json:"level"`
		Source      string  `json:"source"`
		Summary     string  `json:"summary"`
		PublishDate *string `json:"publish_date"`
		Tags        string  `json:"tags"`
		ViewCount   int     `json:"view_count"`
		CreatedAt   string  `json:"created_at"`
	}

	var list []*PolicyItem
	for rows.Next() {
		item := &PolicyItem{}
		if err := rows.Scan(&item.ID, &item.PolicyID, &item.Title, &item.Category, &item.Level,
			&item.Source, &item.Summary, &item.PublishDate, &item.Tags, &item.ViewCount, &item.CreatedAt); err != nil {
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

// GetCareerPolicy 就业政策详情
// GET /api/v1/career/policies/:id
func (h *EducationHandler) GetCareerPolicy(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "政策ID不能为空"})
		return
	}

	type PolicyDetail struct {
		ID            int64   `json:"id"`
		PolicyID      string  `json:"policy_id"`
		Title         string  `json:"title"`
		Category      string  `json:"category"`
		Level         string  `json:"level"`
		Source        string  `json:"source"`
		Content       string  `json:"content"`
		Summary       string  `json:"summary"`
		PublishDate   *string `json:"publish_date"`
		EffectiveDate *string `json:"effective_date"`
		ExpiryDate    *string `json:"expiry_date"`
		Tags          string  `json:"tags"`
		Status        string  `json:"status"`
		ViewCount     int     `json:"view_count"`
		CreatedAt     string  `json:"created_at"`
		UpdatedAt     string  `json:"updated_at"`
	}

	detail := &PolicyDetail{}
	err := h.db.QueryRow(
		"SELECT id, policy_id, title, category, level, source, content, summary, publish_date, "+
			"effective_date, expiry_date, tags, status, view_count, created_at, updated_at "+
			"FROM career_policies WHERE policy_id = ? AND status = 'active'", id,
	).Scan(&detail.ID, &detail.PolicyID, &detail.Title, &detail.Category, &detail.Level,
		&detail.Source, &detail.Content, &detail.Summary, &detail.PublishDate,
		&detail.EffectiveDate, &detail.ExpiryDate, &detail.Tags, &detail.Status,
		&detail.ViewCount, &detail.CreatedAt, &detail.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "政策不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询政策详情失败"})
		return
	}

	_, _ = h.db.Exec("UPDATE career_policies SET view_count = view_count + 1 WHERE id = ?", detail.ID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    detail,
	})
}

// ListJobPostings 招聘信息列表
// GET /api/v1/career/jobs?position_type=&industry=&location=&page=1&page_size=20
func (h *EducationHandler) ListJobPostings(c *gin.Context) {
	positionType := c.Query("position_type")
	industry := c.Query("industry")
	location := c.Query("location")
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
	if positionType != "" {
		where = append(where, "position_type = ?")
		args = append(args, positionType)
	}
	if industry != "" {
		where = append(where, "industry = ?")
		args = append(args, industry)
	}
	if location != "" {
		where = append(where, "location LIKE ?")
		args = append(args, "%"+location+"%")
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM job_postings WHERE "+whereSQL, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询招聘信息失败"})
		return
	}

	rows, err := h.db.Query(
		"SELECT id, job_id, company_name, company_logo, position_name, position_type, industry, "+
			"salary_min, salary_max, salary_unit, location, education, deadline, view_count, apply_count, created_at "+
			"FROM job_postings WHERE "+whereSQL+" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询招聘信息失败"})
		return
	}
	defer rows.Close()

	type JobItem struct {
		ID           int64   `json:"id"`
		JobID        string  `json:"job_id"`
		CompanyName  string  `json:"company_name"`
		CompanyLogo  string  `json:"company_logo"`
		PositionName string  `json:"position_name"`
		PositionType string  `json:"position_type"`
		Industry     string  `json:"industry"`
		SalaryMin    int     `json:"salary_min"`
		SalaryMax    int     `json:"salary_max"`
		SalaryUnit   string  `json:"salary_unit"`
		Location     string  `json:"location"`
		Education    string  `json:"education"`
		Deadline     *string `json:"deadline"`
		ViewCount    int     `json:"view_count"`
		ApplyCount   int     `json:"apply_count"`
		CreatedAt    string  `json:"created_at"`
	}

	var list []*JobItem
	for rows.Next() {
		item := &JobItem{}
		if err := rows.Scan(&item.ID, &item.JobID, &item.CompanyName, &item.CompanyLogo,
			&item.PositionName, &item.PositionType, &item.Industry, &item.SalaryMin,
			&item.SalaryMax, &item.SalaryUnit, &item.Location, &item.Education,
			&item.Deadline, &item.ViewCount, &item.ApplyCount, &item.CreatedAt); err != nil {
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

// GetJobPosting 职位详情
// GET /api/v1/career/jobs/:id
func (h *EducationHandler) GetJobPosting(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "职位ID不能为空"})
		return
	}

	type JobDetail struct {
		ID               int64   `json:"id"`
		JobID            string  `json:"job_id"`
		CompanyName      string  `json:"company_name"`
		CompanyLogo      string  `json:"company_logo"`
		CompanyIntro     string  `json:"company_intro"`
		PositionName     string  `json:"position_name"`
		PositionType     string  `json:"position_type"`
		Industry         string  `json:"industry"`
		SalaryMin        int     `json:"salary_min"`
		SalaryMax        int     `json:"salary_max"`
		SalaryUnit       string  `json:"salary_unit"`
		Location         string  `json:"location"`
		Education        string  `json:"education"`
		MajorRequirement string  `json:"major_requirement"`
		Description      string  `json:"description"`
		Requirement      string  `json:"requirement"`
		Benefits         string  `json:"benefits"`
		ApplicationURL   string  `json:"application_url"`
		Deadline         *string `json:"deadline"`
		Source           string  `json:"source"`
		Status           string  `json:"status"`
		ViewCount        int     `json:"view_count"`
		ApplyCount       int     `json:"apply_count"`
		CreatedAt        string  `json:"created_at"`
		UpdatedAt        string  `json:"updated_at"`
	}

	detail := &JobDetail{}
	err := h.db.QueryRow(
		"SELECT id, job_id, company_name, company_logo, company_intro, position_name, position_type, "+
			"industry, salary_min, salary_max, salary_unit, location, education, major_requirement, "+
			"description, requirement, benefits, application_url, deadline, source, status, "+
			"view_count, apply_count, created_at, updated_at "+
			"FROM job_postings WHERE job_id = ? AND status = 'active'", id,
	).Scan(&detail.ID, &detail.JobID, &detail.CompanyName, &detail.CompanyLogo, &detail.CompanyIntro,
		&detail.PositionName, &detail.PositionType, &detail.Industry, &detail.SalaryMin,
		&detail.SalaryMax, &detail.SalaryUnit, &detail.Location, &detail.Education,
		&detail.MajorRequirement, &detail.Description, &detail.Requirement, &detail.Benefits,
		&detail.ApplicationURL, &detail.Deadline, &detail.Source, &detail.Status,
		&detail.ViewCount, &detail.ApplyCount, &detail.CreatedAt, &detail.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "职位不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询职位详情失败"})
		return
	}

	_, _ = h.db.Exec("UPDATE job_postings SET view_count = view_count + 1 WHERE id = ?", detail.ID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    detail,
	})
}

// ListInfoSessions 宣讲会列表
// GET /api/v1/career/sessions?date=&start_date=&end_date=
func (h *EducationHandler) ListInfoSessions(c *gin.Context) {
	date := c.Query("date")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var where []string
	var args []interface{}
	where = append(where, "status = 'active'")
	if date != "" {
		where = append(where, "date = ?")
		args = append(args, date)
	} else {
		if startDate != "" {
			where = append(where, "date >= ?")
			args = append(args, startDate)
		}
		if endDate != "" {
			where = append(where, "date <= ?")
			args = append(args, endDate)
		}
	}
	whereSQL := strings.Join(where, " AND ")

	rows, err := h.db.Query(
		"SELECT id, session_id, company_name, company_logo, title, date, time_start, time_end, "+
			"location, campus, description, registration_url, capacity, registered_count, created_at "+
			"FROM info_sessions WHERE "+whereSQL+" ORDER BY date ASC, time_start ASC, id ASC",
		args...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询宣讲会失败"})
		return
	}
	defer rows.Close()

	type SessionItem struct {
		ID              int64  `json:"id"`
		SessionID       string `json:"session_id"`
		CompanyName     string `json:"company_name"`
		CompanyLogo     string `json:"company_logo"`
		Title           string `json:"title"`
		Date            string `json:"date"`
		TimeStart       string `json:"time_start"`
		TimeEnd         string `json:"time_end"`
		Location        string `json:"location"`
		Campus          string `json:"campus"`
		Description     string `json:"description"`
		RegistrationURL string `json:"registration_url"`
		Capacity        int    `json:"capacity"`
		RegisteredCount int    `json:"registered_count"`
		CreatedAt       string `json:"created_at"`
	}

	var list []*SessionItem
	for rows.Next() {
		item := &SessionItem{}
		if err := rows.Scan(&item.ID, &item.SessionID, &item.CompanyName, &item.CompanyLogo,
			&item.Title, &item.Date, &item.TimeStart, &item.TimeEnd, &item.Location,
			&item.Campus, &item.Description, &item.RegistrationURL, &item.Capacity,
			&item.RegisteredCount, &item.CreatedAt); err != nil {
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

// ListInterviewQuestions 面试题库列表
// GET /api/v1/career/interview/questions?category=&industry=&page=1&page_size=20
func (h *EducationHandler) ListInterviewQuestions(c *gin.Context) {
	category := c.Query("category")
	industry := c.Query("industry")
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
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	if industry != "" {
		where = append(where, "industry = ?")
		args = append(args, industry)
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM interview_questions WHERE "+whereSQL, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询面试题失败"})
		return
	}

	rows, err := h.db.Query(
		"SELECT id, question_id, category, industry, position, question, answer_hint, keywords, difficulty, created_at "+
			"FROM interview_questions WHERE "+whereSQL+" ORDER BY difficulty ASC, id ASC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询面试题失败"})
		return
	}
	defer rows.Close()

	type QuestionItem struct {
		ID         int64  `json:"id"`
		QuestionID string `json:"question_id"`
		Category   string `json:"category"`
		Industry   string `json:"industry"`
		Position   string `json:"position"`
		Question   string `json:"question"`
		AnswerHint string `json:"answer_hint"`
		Keywords   string `json:"keywords"`
		Difficulty int    `json:"difficulty"`
		CreatedAt  string `json:"created_at"`
	}

	var list []*QuestionItem
	for rows.Next() {
		item := &QuestionItem{}
		if err := rows.Scan(&item.ID, &item.QuestionID, &item.Category, &item.Industry,
			&item.Position, &item.Question, &item.AnswerHint, &item.Keywords,
			&item.Difficulty, &item.CreatedAt); err != nil {
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

// ═══════════════════════════════════════════════
// 二、学业学习模块 /api/v1/study
// ═══════════════════════════════════════════════

// ListCourses 课程列表
// GET /api/v1/study/courses?department=&category=&semester=

// ═══════════════════════════════════════════════
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
