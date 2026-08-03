package handler

import (
	"net/http"

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

// ImportSchedules 批量导入课表（JSON）
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
