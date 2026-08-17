package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// ListAudit 审计日志列表 GET /api/v1/admin/audit?username=&action=&resource=&start_date=&end_date=&page=&page_size=
func (h *AdminHandler) ListAudit(c *gin.Context) {
	username := c.Query("username")
	action := c.Query("action")
	resource := c.Query("resource")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	logs, total, err := h.adminSvc.ListAudit(username, action, resource, startDate, endDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询审计日志失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.AuditListResponse{
		Code:     0,
		Message:  "success",
		Data:     logs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// DeleteAudit 清理审计日志 DELETE /api/v1/admin/audit
// 支持按 username/action/resource/start_date/end_date 过滤；不带参数则清空全部。
func (h *AdminHandler) DeleteAudit(c *gin.Context) {
	username := c.Query("username")
	action := c.Query("action")
	resource := c.Query("resource")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var n int64
	var err error
	if username == "" && action == "" && resource == "" && startDate == "" && endDate == "" {
		// 无过滤条件 → 清空全部（需明确二次确认由前端承担）
		err = h.adminSvc.ClearAllAudit()
	} else {
		n, err = h.adminSvc.DeleteAudit(username, action, resource, startDate, endDate)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "清理审计日志失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "deleted": n})
}

// ListSnapshots 列出可恢复操作快照 GET /api/v1/admin/audit/snapshots
func (h *AdminHandler) ListSnapshots(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	snaps, err := h.adminSvc.ListSnapshots(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询快照失败"})
		return
	}
	if snaps == nil {
		snaps = []*model.AuditSnapshot{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": snaps})
}

// RestoreSnapshot 恢复操作 POST /api/v1/admin/audit/snapshots/:id/restore
func (h *AdminHandler) RestoreSnapshot(c *gin.Context) {
	snapID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	n, err := h.adminSvc.RestoreSnapshot(snapID, userCtx.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "恢复成功", "data": gin.H{"restored": n}})
}

// MyLogs 当前用户自己的操作日志 GET /api/v1/user/logs
// action_type: 默认 ""=仅用户操作(写操作)；all=全部
func (h *AdminHandler) MyLogs(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	actionType := c.Query("action_type")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	logs, total, err := h.adminSvc.ListMyAudit(userCtx.UserID, actionType, startDate, endDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询日志失败"})
		return
	}
	if logs == nil {
		logs = []*model.AuditLog{}
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "success",
		"data": gin.H{"list": logs, "total": total},
	})
}

// DeleteMyLog 删除当前用户自己的日志 DELETE /api/v1/user/logs/:id
// 不带 id 时清空自己的操作日志
func (h *AdminHandler) DeleteMyLog(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	n, err := h.adminSvc.DeleteMyLog(userCtx.UserID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "删除日志失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"deleted": n}})
}
