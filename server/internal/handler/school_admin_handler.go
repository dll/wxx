package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// SchoolAdminHandler 学校管理员角色 AI 功能接口
type SchoolAdminHandler struct {
	svc *service.SchoolAdminService
}

func NewSchoolAdminHandler(svc *service.SchoolAdminService) *SchoolAdminHandler {
	return &SchoolAdminHandler{svc: svc}
}

// Panorama 全校数字孪生全景
func (h *SchoolAdminHandler) Panorama(c *gin.Context) {
	if h.svc != nil {
		data := h.svc.GenerateSchoolPanorama(c.Request.Context())
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"total_students": 8000, "total_colleges": 8, "health_score": 82.5,
		"data_source": "fallback",
	})
}

// PolicySimulation 政策影响模拟
func (h *SchoolAdminHandler) PolicySimulation(c *gin.Context) {
	var req struct {
		Policy     string `json:"policy"`
		Adjustment string `json:"adjustment"`
	}
	_ = c.ShouldBindJSON(&req)
	if h.svc != nil {
		data := h.svc.SimulatePolicy(c.Request.Context(), req.Policy, req.Adjustment)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"policy":             req.Policy,
		"adjustment":         req.Adjustment,
		"beneficiary_change": "预计受益学生从1200人增加至1500人(+25%)",
		"data_source":        "fallback",
	})
}

// CollegeComparison 跨学院对比
func (h *SchoolAdminHandler) CollegeComparison(c *gin.Context) {
	metric := c.Query("metric")
	if h.svc != nil {
		data := h.svc.CompareColleges(c.Request.Context(), metric)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"metric":     metric,
		"rankings":   []gin.H{{"rank": 1, "college": "文学院", "score": 88.0}},
		"data_source": "fallback",
	})
}

// AcademicOverview 校级学情总览
func (h *SchoolAdminHandler) AcademicOverview(c *gin.Context) {
	if h.svc != nil {
		data := h.svc.GenerateAcademicOverview(c.Request.Context())
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"date":            "2026-05-18",
		"college_rankings": []gin.H{{"college": "计算机学院", "health": 85.2}},
		"data_source":     "fallback",
	})
}
