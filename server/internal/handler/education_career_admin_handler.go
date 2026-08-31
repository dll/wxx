package handler

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// 管理端：就业指导内容管理（学校/学院管理员）；SQL 已下沉 CareerRepo（P4-d）

// AdminListCareerPolicies 管理端就业政策列表（含非 active）
// GET /api/v1/career/admin/policies?page=&page_size=&category=
func (h *EducationHandler) AdminListCareerPolicies(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	list, total, err := h.careerRepo.AdminListPolicies(c.Query("category"), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询就业政策失败"})
		return
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

	if err := h.careerRepo.AdminCreatePolicy(req.PolicyID, req.Title, req.Category, req.Level, req.Source, req.Content, req.Summary, req.Tags, req.Status); err != nil {
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
	if err := h.careerRepo.AdminDeletePolicy(id); err != nil {
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

	list, total, err := h.careerRepo.AdminListJobs(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询招聘信息失败"})
		return
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

	jobID := "job-" + strconv.FormatInt(time.Now().Unix(), 10)
	if err := h.careerRepo.AdminCreateJob(jobID, req.CompanyName, req.PositionName, req.PositionType, req.Industry, req.Location, req.Education, req.Description, req.Requirement, req.Status); err != nil {
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
	if err := h.careerRepo.AdminDeleteJob(id); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "删除招聘信息失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
