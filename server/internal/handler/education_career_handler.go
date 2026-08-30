package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/gin-gonic/gin"
)

// 就业指导相关 handler（从 education_handler.go 按业务域拆分；SQL 已下沉 CareerRepo）
func (h *EducationHandler) ListCareerPolicies(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	list, total, err := h.careerRepo.ListCareerPolicies(c.Query("category"), c.Query("level"), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询就业政策失败"})
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

// GetCareerPolicy 就业政策详情
// GET /api/v1/career/policies/:id
func (h *EducationHandler) GetCareerPolicy(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "政策ID不能为空"})
		return
	}

	detail, err := h.careerRepo.GetCareerPolicyDetail(id)
	if err == repository.ErrCareerNotFound {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "政策不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询政策详情失败"})
		return
	}

	h.careerRepo.IncrementPolicyView(detail.ID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    detail,
	})
}

// ListJobPostings 招聘信息列表
// GET /api/v1/career/jobs?position_type=&industry=&location=&page=1&page_size=20
func (h *EducationHandler) ListJobPostings(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	list, total, err := h.careerRepo.ListJobPostings(c.Query("position_type"), c.Query("industry"), c.Query("location"), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询招聘信息失败"})
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

// GetJobPosting 职位详情
// GET /api/v1/career/jobs/:id
func (h *EducationHandler) GetJobPosting(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "职位ID不能为空"})
		return
	}

	detail, err := h.careerRepo.GetJobPostingDetail(id)
	if err == repository.ErrCareerNotFound {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "职位不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询职位详情失败"})
		return
	}

	h.careerRepo.IncrementJobView(detail.ID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    detail,
	})
}

// ListInfoSessions 宣讲会列表
// GET /api/v1/career/sessions?date=&start_date=&end_date=
func (h *EducationHandler) ListInfoSessions(c *gin.Context) {
	list, err := h.careerRepo.ListInfoSessions(c.Query("date"), c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询宣讲会失败"})
		return
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	list, total, err := h.careerRepo.ListInterviewQuestions(c.Query("category"), c.Query("industry"), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询面试题失败"})
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
