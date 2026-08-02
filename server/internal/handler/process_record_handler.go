package handler

import (
	"log"
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// ProcessRecordHandler 办事流程办理记录 HTTP handler
type ProcessRecordHandler struct {
	svc *service.ProcessRecordService
}

// NewProcessRecordHandler 创建 handler
func NewProcessRecordHandler(svc *service.ProcessRecordService) *ProcessRecordHandler {
	return &ProcessRecordHandler{svc: svc}
}

// ListMine GET /api/v1/process/records 查询当前用户的全部办事记录
func (h *ProcessRecordHandler) ListMine(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	items, err := h.svc.ListMine(userCtx.UserID, 50)
	if err != nil {
		log.Printf("process_record ListMine err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询办事记录失败，请稍后重试",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    items,
	})
}

// StartOrResume POST /api/v1/process/records/:flow/start
// body: {"flow_label": "新生入学", "total_steps": 5}
func (h *ProcessRecordHandler) StartOrResume(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	flowType := c.Param("flow")

	var body struct {
		FlowLabel  string `json:"flow_label"`
		TotalSteps int    `json:"total_steps"`
	}
	_ = c.ShouldBindJSON(&body)

	rec, err := h.svc.StartOrResume(userCtx.UserID, flowType, body.FlowLabel, body.TotalSteps)
	if err != nil {
		log.Printf("process_record StartOrResume err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "操作失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    rec,
	})
}

// UpdateProgress POST /api/v1/process/records/:flow/progress
// body: {"current_step": 2, "completed_steps": [0,1], "notes": "..."}
func (h *ProcessRecordHandler) UpdateProgress(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	flowType := c.Param("flow")

	var body struct {
		CurrentStep    int    `json:"current_step"`
		CompletedSteps []int  `json:"completed_steps"`
		Notes          string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		log.Printf("process_record UpdateProgress bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数错误",
		})
		return
	}

	rec, err := h.svc.UpdateProgress(userCtx.UserID, flowType, body.CurrentStep, body.CompletedSteps, body.Notes)
	if err != nil {
		log.Printf("process_record UpdateProgress err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "操作失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    rec,
	})
}
