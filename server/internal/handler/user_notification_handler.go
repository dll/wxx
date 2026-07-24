package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// UserNotificationHandler 用户站内通知 HTTP handler
type UserNotificationHandler struct {
	db *sql.DB
}

// NewUserNotificationHandler 创建用户站内通知 handler
func NewUserNotificationHandler(db *sql.DB) *UserNotificationHandler {
	return &UserNotificationHandler{db: db}
}

// UserNotification 站内通知结构体
type UserNotification struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Type        string `json:"type"`
	RelatedType string `json:"related_type"`
	RelatedID   int64  `json:"related_id"`
	IsRead      int    `json:"is_read"`
	CreatedAt   string `json:"created_at"`
}

// ═══════════════════════════════════════════════
// 一、用户端接口
// ═══════════════════════════════════════════════

// ListNotifications 我的通知列表
// GET /api/v1/notifications?page=1&page_size=20&type=system
func (h *UserNotificationHandler) ListNotifications(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	notifType := c.Query("type")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var where []string
	var args []interface{}
	where = append(where, "user_id = ?")
	args = append(args, userCtx.UserID)

	if notifType != "" {
		where = append(where, "type = ?")
		args = append(args, notifType)
	}
	whereSQL := strings.Join(where, " AND ")

	// 查询未读总数
	var unreadCount int
	err := h.db.QueryRow("SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND is_read = 0", userCtx.UserID).Scan(&unreadCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询未读数量失败",
		})
		return
	}

	// 查询总数
	var total int
	err = h.db.QueryRow("SELECT COUNT(*) FROM user_notifications WHERE "+whereSQL, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询通知列表失败",
		})
		return
	}

	// 查询列表
	rows, err := h.db.Query(
		"SELECT id, user_id, title, content, type, related_type, related_id, is_read, created_at "+
			"FROM user_notifications WHERE "+whereSQL+" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询通知列表失败",
		})
		return
	}
	defer rows.Close()

	items := make([]UserNotification, 0)
	for rows.Next() {
		var notif UserNotification
		err := rows.Scan(&notif.ID, &notif.UserID, &notif.Title, &notif.Content, &notif.Type,
			&notif.RelatedType, &notif.RelatedID, &notif.IsRead, &notif.CreatedAt)
		if err != nil {
			continue
		}
		items = append(items, notif)
	}

	c.JSON(http.StatusOK, gin.H{
		"items":        items,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
		"unread_count": unreadCount,
	})
}

// GetUnreadCount 获取未读通知数量
// GET /api/v1/notifications/unread-count
func (h *UserNotificationHandler) GetUnreadCount(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	var unreadCount int
	err := h.db.QueryRow("SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND is_read = 0", userCtx.UserID).Scan(&unreadCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询未读数量失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unread_count": unreadCount,
	})
}

// MarkAsRead 标记单条通知为已读
// PUT /api/v1/notifications/:id/read
func (h *UserNotificationHandler) MarkAsRead(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "无效的通知ID",
		})
		return
	}

	result, err := h.db.Exec(
		"UPDATE user_notifications SET is_read = 1 WHERE id = ? AND user_id = ?",
		id, userCtx.UserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "标记已读失败",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Code:    404,
			Message: "通知不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "已标记为已读",
	})
}

// MarkAllAsRead 全部标记为已读
// PUT /api/v1/notifications/read-all
func (h *UserNotificationHandler) MarkAllAsRead(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	_, err := h.db.Exec(
		"UPDATE user_notifications SET is_read = 1 WHERE user_id = ? AND is_read = 0",
		userCtx.UserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "标记全部已读失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "全部标记为已读",
	})
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
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "标题和内容不能为空",
		})
		return
	}

	if len(req.TargetUsers) == 0 {
		// 全体用户：从 users 表中获取所有用户ID
		rows, err := h.db.Query("SELECT id FROM users WHERE status = 'active'")
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{
				Code:    500,
				Message: "获取用户列表失败",
			})
			return
		}
		defer rows.Close()

		var userIDs []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err == nil {
				userIDs = append(userIDs, id)
			}
		}
		req.TargetUsers = userIDs
	}

	// 批量插入通知
	sendCount := 0
	for _, userID := range req.TargetUsers {
		_, err := h.db.Exec(
			"INSERT INTO user_notifications (user_id, title, content, type, related_type, related_id, is_read) VALUES (?, ?, ?, 'system', '', 0, 0)",
			userID, req.Title, req.Content,
		)
		if err == nil {
			sendCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "发送成功",
		"send_count": sendCount,
	})
}

// 确保 auth 包被引用（用于权限中间件）
var _ = auth.RequireCapability
