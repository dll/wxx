package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// ChatHandler 对话 handler
type ChatHandler struct {
	chatSvc *service.ChatService
}

// NewChatHandler 创建对话 handler
func NewChatHandler(chatSvc *service.ChatService) *ChatHandler {
	return &ChatHandler{chatSvc: chatSvc}
}

// Ask 处理对话请求
// POST /api/v1/chat
func (h *ChatHandler) Ask(c *gin.Context) {
	// 解析请求
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数错误：" + err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	// 获取用户上下文
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	// 调用 service 层
	card, sessionID, err := h.chatSvc.Ask(c.Request.Context(), userCtx, req.SessionID, req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "问答处理失败：" + err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, model.ChatResponse{
		Code:      0,
		Message:   "success",
		Data:      card,
		SessionID: sessionID,
	})
}
