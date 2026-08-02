package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// SessionHandler 会话 handler
type SessionHandler struct {
	sessionSvc *service.SessionService
}

// NewSessionHandler 创建会话 handler
func NewSessionHandler(sessionSvc *service.SessionService) *SessionHandler {
	return &SessionHandler{sessionSvc: sessionSvc}
}

// ListSessions 查询会话列表
// GET /api/v1/sessions?limit=20
func (h *SessionHandler) ListSessions(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	// 解析 limit 参数
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	sessions, err := h.sessionSvc.ListSessions(userCtx.UserID, limit)
	if err != nil {
		log.Printf("session ListSessions err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询会话列表失败，请稍后重试",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    sessions,
	})
}

// GetMessages 查询会话消息历史
// GET /api/v1/sessions/:id/messages?limit=100
func (h *SessionHandler) GetMessages(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "缺少会话 ID",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	// 解析 limit 参数
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	messages, err := h.sessionSvc.GetSessionMessages(userCtx.UserID, sessionID, limit)
	if err != nil {
		log.Printf("session GetSessionMessages err: %v", err)
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    403,
			Message: "查询失败",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    messages,
	})
}

// DeleteSession 删除会话
// DELETE /api/v1/sessions/:id
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "缺少会话 ID",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	if err := h.sessionSvc.DeleteSession(userCtx.UserID, sessionID); err != nil {
		log.Printf("session DeleteSession err: %v", err)
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    403,
			Message: "操作失败",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

// RenameSession 重命名会话标题
// PATCH /api/v1/sessions/:id  body: {"title": "新标题"}
func (h *SessionHandler) RenameSession(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "缺少会话 ID",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	var body struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		log.Printf("session RenameSession bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数错误",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	if err := h.sessionSvc.RenameSession(userCtx.UserID, sessionID, body.Title); err != nil {
		log.Printf("session RenameSession err: %v", err)
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    403,
			Message: "操作失败",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}
