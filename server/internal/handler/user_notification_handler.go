package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/gin-gonic/gin"
)

// UserNotificationHandler 用户站内通知 HTTP handler（P4-d：SQL 已下沉 UserNotificationRepo）
type UserNotificationHandler struct {
	repo *repository.UserNotificationRepo
}

// NewUserNotificationHandler 创建用户站内通知 handler
func NewUserNotificationHandler(repo *repository.UserNotificationRepo) *UserNotificationHandler {
	return &UserNotificationHandler{repo: repo}
}

// ═══════════════════════════════════════════════
// 一、用户端接口
// ═══════════════════════════════════════════════

// ListNotifications 我的通知列表
// GET /api/v1/notifications?page=1&page_size=20&type=system
func (h *UserNotificationHandler) ListNotifications(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	items, total, unread, err := h.repo.ListByUser(userCtx.UserID, c.Query("type"), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询通知列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":        items,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
		"unread_count": unread,
	})
}

// GetUnreadCount 获取未读通知数量
// GET /api/v1/notifications/unread-count
func (h *UserNotificationHandler) GetUnreadCount(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	unread, err := h.repo.CountUnread(userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询未读数量失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": unread})
}

// MarkAsRead 标记单条通知为已读
// PUT /api/v1/notifications/:id/read
func (h *UserNotificationHandler) MarkAsRead(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "无效的通知ID"})
		return
	}

	affected, err := h.repo.MarkRead(id, userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "标记已读失败"})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "通知不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已标记为已读"})
}

// MarkAllAsRead 全部标记为已读
// PUT /api/v1/notifications/read-all
func (h *UserNotificationHandler) MarkAllAsRead(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	if err := h.repo.MarkAllRead(userCtx.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "标记全部已读失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "全部标记为已读"})
}

// ═══════════════════════════════════════════════
// 二、管理端接口
// ═══════════════════════════════════════════════

type sendNotificationReq struct {
	Title       string  `json:"title" binding:"required"`
	Content     string  `json:"content" binding:"required"`
	TargetUsers []int64 `json:"target_users"`
}

// SendSystemNotification 管理员发送系统通知
// POST /api/v1/admin/notifications/send
func (h *UserNotificationHandler) SendSystemNotification(c *gin.Context) {
	var req sendNotificationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "标题和内容不能为空"})
		return
	}

	if len(req.TargetUsers) == 0 {
		ids, err := h.repo.ActiveUserIDs()
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "获取用户列表失败"})
			return
		}
		req.TargetUsers = ids
	}

	sendCount := h.repo.SendBulk(req.Title, req.Content, req.TargetUsers)

	c.JSON(http.StatusOK, gin.H{
		"message":    "发送成功",
		"send_count": sendCount,
	})
}

// AdminListNotifications 管理端查看全部通知（供管理页使用）
// GET /api/v1/admin/notifications/list?page=1&page_size=20&type=system
func (h *UserNotificationHandler) AdminListNotifications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	list, total, err := h.repo.AdminList(c.Query("type"), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询通知列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "success", "data": list,
		"total": total, "page": page, "page_size": pageSize,
	})
}

// AdminDeleteNotification 删除一条通知（管理端）
// DELETE /api/v1/admin/notifications/:id
func (h *UserNotificationHandler) AdminDeleteNotification(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "无效的通知ID"})
		return
	}
	affected, err := h.repo.AdminDelete(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "删除通知失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "deleted": affected})
}

// AdminClearNotifications 清空全部通知（管理端）
// DELETE /api/v1/admin/notifications
func (h *UserNotificationHandler) AdminClearNotifications(c *gin.Context) {
	affected, err := h.repo.AdminClear()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "清空通知失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "deleted": affected})
}

// 确保 auth 包被引用（用于权限中间件）
var _ = auth.RequireCapability
