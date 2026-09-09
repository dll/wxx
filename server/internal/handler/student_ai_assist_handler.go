package handler

import (
	"net/http"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// InteractiveAIAssist is the shared, read-only AI helper for student pages.
func (h *StudentHandler) InteractiveAIAssist(c *gin.Context) {
	if h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "AI 助手暂时不可用"})
		return
	}
	var in struct {
		Feature string `json:"feature"`
		Input   string `json:"input"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Feature = strings.TrimSpace(in.Feature)
	in.Input = strings.TrimSpace(in.Input)
	if in.Input == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"code": 422, "message": "请描述你希望 AI 协助的内容"})
		return
	}
	if len([]rune(in.Input)) > 2000 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"code": 422, "message": "输入内容不超过 2000 字"})
		return
	}
	u := middleware.GetUserContext(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	result := h.svc.GenerateInteractiveAssist(c.Request.Context(), in.Feature, u.UserID, in.Input)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}
