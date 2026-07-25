package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
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
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "缺少用户上下文，请重新登录"})
		return
	}

	result, err := h.svc.GetDigitalTwin(c.Request.Context(), userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成数字孪生画像失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}
