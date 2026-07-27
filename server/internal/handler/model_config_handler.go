package handler

import (
	"log"
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// ModelConfigHandler 用户 AI 模型配置 HTTP handler
type ModelConfigHandler struct {
	svc *service.ModelConfigService
}

// NewModelConfigHandler 创建模型配置 handler
func NewModelConfigHandler(svc *service.ModelConfigService) *ModelConfigHandler {
	return &ModelConfigHandler{svc: svc}
}

// Get 获取当前用户的模型配置 GET /api/v1/user/model-config
func (h *ModelConfigHandler) Get(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	cfg, err := h.svc.Get(userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取模型配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    cfg,
	})
}

// Save 保存模型配置 PUT /api/v1/user/model-config
func (h *ModelConfigHandler) Save(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	var req model.ModelConfigSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败: " + err.Error(),
		})
		return
	}

	cfg, err := h.svc.Save(userCtx.UserID, &req)
	if err != nil {
		log.Printf("model_config Save err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "保存失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "模型配置已保存",
		"data":    cfg,
	})
}
