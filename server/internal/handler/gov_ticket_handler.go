package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// GovTicketHandler 督办工单接口（D5-3「洞察→工单」治理回环）
// 书记/学院管理员创建、分派、督办；辅导员/教辅/党群责任人查看并推进本人分派的工单。
type GovTicketHandler struct {
	svc *service.GovTicketService
}

// NewGovTicketHandler 创建督办工单 handler
func NewGovTicketHandler(svc *service.GovTicketService) *GovTicketHandler {
	return &GovTicketHandler{svc: svc}
}

// govTicketCtx 取当前用户上下文（id/name/role/ownerID/ownerScope）
func govTicketCtx(c *gin.Context) (id int64, name, role, ownerID, ownerScope string) {
	if u := middleware.GetUserContext(c); u != nil {
		return u.UserID, u.Username, u.Role, u.OwnerID, u.OwnerScope
	}
	return 0, "", "", "", ""
}

// govResolveCollege 对齐既有书记看板范围语义：学院书记落本院，学校书记落全校（college 空）
func govResolveCollege(c *gin.Context, college string) string {
	if college != "" {
		return college
	}
	if _, _, role, ownerID, ownerScope := govTicketCtx(c); role == "college_admin" && ownerScope == "college" {
		return ownerID
	}
	return ""
}

// Create 创建督办工单（治理洞察/手工）
// POST /api/v1/college/tickets  (college.ticket.manage)
func (h *GovTicketHandler) Create(c *gin.Context) {
	var req repository.GovTicketCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, opName, opRole, _, _ := govTicketCtx(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	req.College = govResolveCollege(c, req.College)
	req.CreatedBy = opID
	req.CreatedByRole = opRole
	req.CreatedByName = opName
	id, err := h.svc.CreateTicket(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "督办工单创建成功", "id": id})
}

// CreateFromKPI 从育人 KPI 生成补料督办工单（D5-1 联动）
// POST /api/v1/college/tickets/from-kpi  (college.ticket.manage)
func (h *GovTicketHandler) CreateFromKPI(c *gin.Context) {
	var req struct {
		KPIKey   string                        `json:"kpi_key"`
		OwnerID  string                        `json:"owner_id"`
		Assignee repository.GovTicketCreateReq `json:"assignee"` // 分派信息（可选，缺省先创建待分派）
		Priority string                        `json:"priority"`
		Deadline string                        `json:"deadline"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, opName, opRole, _, _ := govTicketCtx(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	ownerID := govResolveCollege(c, req.OwnerID)
	createReq := &repository.GovTicketCreateReq{
		College:       ownerID,
		Priority:      req.Priority,
		Deadline:      req.Deadline,
		CreatedBy:     opID,
		CreatedByRole: opRole,
		CreatedByName: opName,
	}
	if req.Assignee.AssigneeID > 0 {
		createReq.AssigneeID = req.Assignee.AssigneeID
		createReq.AssigneeName = req.Assignee.AssigneeName
		createReq.AssigneeRole = req.Assignee.AssigneeRole
	}
	id, err := h.svc.CreateFromKPI(c.Request.Context(), req.KPIKey, ownerID, createReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已从育人指标生成补料督办工单", "id": id, "category": "supplement"})
}

// List 督办工单列表（书记/学院管理端）
// GET /api/v1/college/tickets  (college.ticket.manage)
func (h *GovTicketHandler) List(c *gin.Context) {
	status := c.Query("status")
	college := c.Query("college")
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	college = govResolveCollege(c, college)
	list, total, err := h.svc.List(c.Request.Context(), status, college, category, (page-1)*pageSize, pageSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if list == nil {
		list = []*model.GovTicket{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "list": list, "total": total, "page": page, "page_size": pageSize})
}

// ListMine 分派给本人的督办工单（责任人视角）
// GET /api/v1/college/tickets/mine  (college.ticket.assignee)
func (h *GovTicketHandler) ListMine(c *gin.Context) {
	opID, _, _, _, _ := govTicketCtx(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	list, total, err := h.svc.ListMine(c.Request.Context(), opID, status, (page-1)*pageSize, pageSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if list == nil {
		list = []*model.GovTicket{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "list": list, "total": total})
}

// Get 工单详情（含操作记录）
// GET /api/v1/college/tickets/:id  (college.ticket.manage 或 责任人本人)
func (h *GovTicketHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, _, role, _, _ := govTicketCtx(c)
	isManager := role == "college_admin" || role == "school_admin"
	t, logs, err := h.svc.Get(c.Request.Context(), id, opID, isManager)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "ticket": t, "logs": logs})
}

// Assign 分派/改派责任人
// PUT /api/v1/college/tickets/:id/assign  (college.ticket.manage)
func (h *GovTicketHandler) Assign(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		AssigneeID   int64  `json:"assignee_id"`
		AssigneeRole string `json:"assignee_role"`
		AssigneeName string `json:"assignee_name"`
		Deadline     string `json:"deadline"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, opName, _, _, _ := govTicketCtx(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	if err := h.svc.Assign(c.Request.Context(), id, req.AssigneeID, req.AssigneeRole, req.AssigneeName, req.Deadline, opID, opName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "分派成功"})
}

// UpdateStatus 推进工单状态（书记可推进任意；责任人推进本人）
// PUT /api/v1/college/tickets/:id/status  (college.ticket.manage 或 责任人本人)
func (h *GovTicketHandler) UpdateStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Status string `json:"status"`
		Detail string `json:"detail"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, opName, role, _, _ := govTicketCtx(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	// 责任人视角：非管理端仅允许推进分派给本人的工单（service.Get 之外的轻鉴权）
	isManager := role == "college_admin" || role == "school_admin"
	if !isManager {
		t, _, err := h.svc.Get(c.Request.Context(), id, opID, false)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
			return
		}
		if t.AssigneeID != opID {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权推进该督办工单"})
			return
		}
	}
	if err := h.svc.UpdateStatus(c.Request.Context(), id, opID, opName, req.Status, req.Detail); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "状态已更新"})
}

// Stats 督办总览
// GET /api/v1/college/tickets/stats  (college.ticket.manage)
func (h *GovTicketHandler) Stats(c *gin.Context) {
	college := govResolveCollege(c, c.Query("college"))
	stats, err := h.svc.Stats(c.Request.Context(), college)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "stats": stats})
}
