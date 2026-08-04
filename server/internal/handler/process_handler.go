package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/util"
	"github.com/gin-gonic/gin"
)

// ProcessHandler 办事流程定义 HTTP handler
type ProcessHandler struct {
	svc *service.ProcessService
}

// NewProcessHandler 创建办事流程 handler
func NewProcessHandler(svc *service.ProcessService) *ProcessHandler {
	return &ProcessHandler{svc: svc}
}

type processListResponse struct {
	Code     int                          `json:"code"`
	Message  string                       `json:"message"`
	Data     []*service.ProcessDefinition `json:"data"`
	Total    int                          `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
}

type processDetailResponse struct {
	Code    int                        `json:"code"`
	Message string                     `json:"message"`
	Data    *service.ProcessDefinition `json:"data"`
}

func pagination(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	return
}

// ListDefinitions GET /api/v1/process/definitions
func (h *ProcessHandler) ListDefinitions(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未获取到用户信息")
		return
	}
	page, pageSize := pagination(c)
	items, total, err := h.svc.ListForUser(c.Request.Context(), userCtx, page, pageSize)
	if err != nil {
		log.Printf("process ListDefinitions err: %v", err)
		util.FailInternalError(c, "获取办事流程列表失败")
		return
	}
	c.JSON(http.StatusOK, processListResponse{
		Code: 0, Message: "success", Data: items, Total: total, Page: page, PageSize: pageSize,
	})
}

// GetDefinition GET /api/v1/process/definitions/:id
func (h *ProcessHandler) GetDefinition(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未获取到用户信息")
		return
	}
	def, err := h.svc.GetForUser(c.Request.Context(), userCtx, c.Param("id"))
	if err != nil {
		log.Printf("process GetDefinition err: %v", err)
		util.FailNotFound(c, "办事流程不存在或未发布")
		return
	}
	c.JSON(http.StatusOK, processDetailResponse{Code: 0, Message: "success", Data: def})
}

// ListAdmin GET /api/v1/process/admin
func (h *ProcessHandler) ListAdmin(c *gin.Context) {
	page, pageSize := pagination(c)
	items, total, err := h.svc.ListAdmin(c.Request.Context(), c.Query("keyword"), c.Query("status"), page, pageSize)
	if err != nil {
		log.Printf("process ListAdmin err: %v", err)
		util.FailInternalError(c, "获取办事流程管理列表失败")
		return
	}
	c.JSON(http.StatusOK, processListResponse{
		Code: 0, Message: "success", Data: items, Total: total, Page: page, PageSize: pageSize,
	})
}

// ListPending GET /api/v1/process/admin/pending
func (h *ProcessHandler) ListPending(c *gin.Context) {
	page, pageSize := pagination(c)
	items, total, err := h.svc.ListPending(c.Request.Context(), page, pageSize)
	if err != nil {
		log.Printf("process ListPending err: %v", err)
		util.FailInternalError(c, "获取待审核流程失败")
		return
	}
	c.JSON(http.StatusOK, processListResponse{
		Code: 0, Message: "success", Data: items, Total: total, Page: page, PageSize: pageSize,
	})
}

// GetAdmin GET /api/v1/process/admin/:id
func (h *ProcessHandler) GetAdmin(c *gin.Context) {
	def, err := h.svc.GetAdmin(c.Request.Context(), c.Param("id"))
	if err != nil {
		log.Printf("process GetAdmin err: %v", err)
		util.FailNotFound(c, "办事流程不存在")
		return
	}
	c.JSON(http.StatusOK, processDetailResponse{Code: 0, Message: "success", Data: def})
}

// Create POST /api/v1/process/admin
func (h *ProcessHandler) Create(c *gin.Context) {
	var req model.ProcessUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("process Create bind err: %v", err)
		util.FailBadRequest(c, "参数校验失败："+err.Error())
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未获取到用户信息")
		return
	}
	def, err := h.svc.Create(c.Request.Context(), userCtx, &req)
	if err != nil {
		log.Printf("process Create err: %v", err)
		util.FailBadRequest(c, "创建办事流程失败："+err.Error())
		return
	}
	c.JSON(http.StatusCreated, processDetailResponse{Code: 0, Message: "创建成功", Data: def})
}

// Update PUT /api/v1/process/admin/:id
func (h *ProcessHandler) Update(c *gin.Context) {
	var req model.ProcessUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.FailBadRequest(c, "参数校验失败："+err.Error())
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未获取到用户信息")
		return
	}
	def, err := h.svc.Update(c.Request.Context(), userCtx, c.Param("id"), &req)
	if err != nil {
		log.Printf("process Update err: %v", err)
		util.FailBadRequest(c, "更新办事流程失败："+err.Error())
		return
	}
	c.JSON(http.StatusOK, processDetailResponse{Code: 0, Message: "更新成功", Data: def})
}

// Delete DELETE /api/v1/process/admin/:id
func (h *ProcessHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		log.Printf("process Delete err: %v", err)
		util.FailInternalError(c, "删除办事流程失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// Submit POST /api/v1/process/admin/:id/submit
func (h *ProcessHandler) Submit(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未获取到用户信息")
		return
	}
	def, err := h.svc.SubmitForReview(c.Request.Context(), c.Param("id"), userCtx.Username)
	if err != nil {
		util.FailBadRequest(c, "提交审核失败："+err.Error())
		return
	}
	c.JSON(http.StatusOK, processDetailResponse{Code: 0, Message: "已提交审核", Data: def})
}

// Approve POST /api/v1/process/admin/:id/approve
func (h *ProcessHandler) Approve(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未获取到用户信息")
		return
	}
	def, err := h.svc.Approve(c.Request.Context(), c.Param("id"), userCtx.Username)
	if err != nil {
		util.FailBadRequest(c, "审核通过失败："+err.Error())
		return
	}
	c.JSON(http.StatusOK, processDetailResponse{Code: 0, Message: "审核通过，已发布", Data: def})
}

// Reject POST /api/v1/process/admin/:id/reject
func (h *ProcessHandler) Reject(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未获取到用户信息")
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	def, err := h.svc.Reject(c.Request.Context(), c.Param("id"), userCtx.Username, body.Reason)
	if err != nil {
		util.FailBadRequest(c, "驳回失败："+err.Error())
		return
	}
	c.JSON(http.StatusOK, processDetailResponse{Code: 0, Message: "已驳回", Data: def})
}

// Retire POST /api/v1/process/admin/:id/retire
func (h *ProcessHandler) Retire(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未获取到用户信息")
		return
	}
	def, err := h.svc.Retire(c.Request.Context(), c.Param("id"), userCtx.Username)
	if err != nil {
		util.FailBadRequest(c, "下架失败："+err.Error())
		return
	}
	c.JSON(http.StatusOK, processDetailResponse{Code: 0, Message: "已下架", Data: def})
}
