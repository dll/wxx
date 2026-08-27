package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// FeedbackRepairTaskHandler 反馈修复任务（闭环 MVP）HTTP handler。
// 分两类入口：
//  1. 管理端（/api/v1/admin/feedback/repair-tasks*）：走 JWT + UnionFeedbackWrite/List 能力门控；
//  2. 内部执行端（/api/v1/internal/repair-tasks*）：走 WXX_REPAIR_AGENT_TOKEN 专用 token 鉴权，
//     与本机 Runner 交互，服务器绝不执行改码/构建/部署。
type FeedbackRepairTaskHandler struct {
	svc *service.FeedbackRepairTaskService
}

// NewFeedbackRepairTaskHandler 创建修复任务 handler
func NewFeedbackRepairTaskHandler(svc *service.FeedbackRepairTaskService) *FeedbackRepairTaskHandler {
	return &FeedbackRepairTaskHandler{svc: svc}
}

// ── 管理端 ──

// CreateTask 管理端创建修复任务（单条/批量） POST /api/v1/admin/feedback/repair-tasks
func (h *FeedbackRepairTaskHandler) CreateTask(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	var req model.RepairTaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}
	task, err := h.svc.Create(c.Request.Context(), userCtx.Username, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": task})
}

// ListTasks 管理端任务分页列表 GET /api/v1/admin/feedback/repair-tasks
func (h *FeedbackRepairTaskHandler) ListTasks(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	dtos, total, err := h.svc.List(status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询任务列表失败"})
		return
	}
	c.JSON(http.StatusOK, model.RepairTaskListResponse{
		Code:     0,
		Message:  "success",
		Data:     *dtoSliceToPtr(dtos),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetTask 管理端任务详情 GET /api/v1/admin/feedback/repair-tasks/:no
func (h *FeedbackRepairTaskHandler) GetTask(c *gin.Context) {
	dto, err := h.svc.Get(c.Param("no"))
	if err != nil {
		h.notFoundOrErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": dto})
}

// CancelTask 管理端取消任务 POST /api/v1/admin/feedback/repair-tasks/:no/cancel
func (h *FeedbackRepairTaskHandler) CancelTask(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	operator := ""
	if userCtx != nil {
		operator = userCtx.Username
	}
	dto, err := h.svc.Cancel(c.Param("no"), operator)
	if err != nil {
		h.badStateOrErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": dto})
}

// AcceptTask 管理端验收 POST /api/v1/admin/feedback/repair-tasks/:no/accept
func (h *FeedbackRepairTaskHandler) AcceptTask(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	operator := ""
	if userCtx != nil {
		operator = userCtx.Username
	}
	var req model.RepairTaskAcceptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}
	dto, err := h.svc.Accept(c.Param("no"), operator, req.Note)
	if err != nil {
		h.badStateOrErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": dto})
}

// RejectTask 管理端驳回 POST /api/v1/admin/feedback/repair-tasks/:no/reject
func (h *FeedbackRepairTaskHandler) RejectTask(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	operator := ""
	if userCtx != nil {
		operator = userCtx.Username
	}
	var req model.RepairTaskRejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}
	dto, err := h.svc.Reject(c.Param("no"), operator, req.Reason)
	if err != nil {
		h.badStateOrErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": dto})
}

// DeployConfirmTask 管理端部署确认（仅标记） POST /api/v1/admin/feedback/repair-tasks/:no/deploy-confirm
func (h *FeedbackRepairTaskHandler) DeployConfirmTask(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	operator := ""
	if userCtx != nil {
		operator = userCtx.Username
	}
	var req model.RepairTaskDeployConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}
	dto, err := h.svc.DeployConfirm(c.Param("no"), operator, req.DeployRef)
	if err != nil {
		h.badStateOrErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": dto})
}

// DeployDoneTask 管理端部署完成 POST /api/v1/admin/feedback/repair-tasks/:no/deploy-done
func (h *FeedbackRepairTaskHandler) DeployDoneTask(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	operator := ""
	if userCtx != nil {
		operator = userCtx.Username
	}
	var req model.RepairTaskDeployDoneRequest
	_ = c.ShouldBindJSON(&req) // 可选，缺失也不报错（reply/resolve_feedback 可空）
	dto, err := h.svc.DeployDone(c.Param("no"), operator, req.Reply, req.ResolveFeedback)
	if err != nil {
		h.badStateOrErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": dto})
}

// ── 内部执行端（token 鉴权，见 middleware.RepairAgentTokenAuth）──

// NextTask 执行端认领/领取下一个可执行任务 GET /api/v1/internal/repair-tasks/next
// body: {worker_host, base_commit(optional), branch(optional)}
func (h *FeedbackRepairTaskHandler) NextTask(c *gin.Context) {
	var req model.RepairTaskClaimRequest
	_ = c.ShouldBindJSON(&req) // body 可为空（无 payload 时读取空值）
	if req.WorkerHost == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "worker_host 必填"})
		return
	}
	payload, err := h.svc.Claim(c.Request.Context(), req.WorkerHost, req.BaseCommit, req.Branch)
	if err != nil {
		if errors.Is(err, service.ErrRepairTaskNotFound) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
			return
		}
		if errors.Is(err, service.ErrRepairTaskConcurrency) {
			c.JSON(http.StatusConflict, model.ErrorResponse{Code: 409, Message: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": payload})
}

// VerifyTask 执行端验证结果上报 POST /api/v1/internal/repair-tasks/:no/verify
func (h *FeedbackRepairTaskHandler) VerifyTask(c *gin.Context) {
	var req model.RepairTaskVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}
	dto, err := h.svc.SubmitVerify(c.Param("no"), &req, "repair-agent")
	if err != nil {
		if errors.Is(err, service.ErrRepairTaskNotFound) {
			c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": dto})
}

// ── 辅助 ──

func (h *FeedbackRepairTaskHandler) notFoundOrErr(c *gin.Context, err error) {
	if errors.Is(err, service.ErrRepairTaskNotFound) {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: err.Error()})
}

func (h *FeedbackRepairTaskHandler) badStateOrErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrRepairTaskNotFound):
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: err.Error()})
	case errors.Is(err, service.ErrRepairTaskBadState):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: err.Error()})
	}
}

// dtoSliceToPtr 将 []RepairTaskDTO 切片转指针切片以复用 RepairTaskListResponse.Data
func dtoSliceToPtr(dtos []*model.RepairTaskDTO) *[]model.RepairTaskDTO {
	out := make([]model.RepairTaskDTO, 0, len(dtos))
	for _, d := range dtos {
		if d != nil {
			out = append(out, *d)
		}
	}
	return &out
}
