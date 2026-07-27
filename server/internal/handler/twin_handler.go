package handler

import (
	"log"
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// TwinHandler 个人数字孪生 HTTP handler
type TwinHandler struct {
	svc *service.TwinService
}

// NewTwinHandler 创建数字孪生 handler
func NewTwinHandler(svc *service.TwinService) *TwinHandler {
	return &TwinHandler{svc: svc}
}

// GetDigitalTwin 获取当前学生的数字孪生画像
// GET /api/v1/student/digital-twin
// 能力门控：self.twin.read（学生本人可读）
func (h *TwinHandler) GetDigitalTwin(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "缺少用户上下文，请重新登录", TraceID: middleware.GetTraceID(c)})
		return
	}

	result, err := h.svc.GetDigitalTwin(c.Request.Context(), userCtx.UserID)
	if err != nil {
		log.Printf("生成数字孪生画像失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "生成数字孪生画像失败", TraceID: middleware.GetTraceID(c)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}
