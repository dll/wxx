package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/util"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	svc *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

type createNotificationReq struct {
	Title        string `json:"title" binding:"required"`
	Content      string `json:"content" binding:"required"`
	AudienceType string `json:"audience_type"`
	PushQQ       bool   `json:"push_qq"`
	PushWechat   bool   `json:"push_wechat"`
}

func (h *NotificationHandler) Create(c *gin.Context) {
	var req createNotificationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "标题和内容不能为空", TraceID: middleware.GetTraceID(c)})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证", TraceID: middleware.GetTraceID(c)})
		return
	}
	notif, err := h.svc.Create(c.Request.Context(), userCtx, req.Title, req.Content, req.AudienceType, req.PushQQ, req.PushWechat)
	if err != nil {
		util.FailInternalError(c, "创建通知失败")
		return
	}
	c.JSON(http.StatusOK, notif)
}

func (h *NotificationHandler) List(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证", TraceID: middleware.GetTraceID(c)})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	notifs, total, err := h.svc.List(c.Request.Context(), userCtx, page, limit)
	if err != nil {
		util.FailInternalError(c, "查询通知列表失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": notifs, "total": total, "page": page, "limit": limit})
}

func (h *NotificationHandler) Publish(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "无效ID", TraceID: middleware.GetTraceID(c)})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证", TraceID: middleware.GetTraceID(c)})
		return
	}
	notif, err := h.svc.Publish(c.Request.Context(), userCtx, id)
	if err != nil {
		util.FailInternalError(c, "发布通知失败")
		return
	}
	c.JSON(http.StatusOK, notif)
}

func (h *NotificationHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "无效ID", TraceID: middleware.GetTraceID(c)})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证", TraceID: middleware.GetTraceID(c)})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), userCtx, id); err != nil {
		util.FailInternalError(c, "删除通知失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// WebhookStatus 返回已配置的 webhook 状态
func (h *NotificationHandler) WebhookStatus(c *gin.Context) {
	status := h.svc.GetWebhookStatus()
	c.JSON(http.StatusOK, gin.H{"webhooks": status})
}
