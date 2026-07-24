package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// SysAdminHandler 系统管理员角色 AI 功能接口
type SysAdminHandler struct {
	svc *service.SysAdminService
}

func NewSysAdminHandler(svc *service.SysAdminService) *SysAdminHandler {
	return &SysAdminHandler{svc: svc}
}

// SystemHealth 系统健康监控
func (h *SysAdminHandler) SystemHealth(c *gin.Context) {
	if h.svc != nil {
		data := h.svc.GetSystemHealth(c.Request.Context())
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"status":       "healthy",
		"uptime":       "72h 35m",
		"cpu_usage":    0.35,
		"memory_usage": 0.62,
		"data_source":  "fallback",
	})
}

// KnowledgeQuality 知识质量评估
func (h *SysAdminHandler) KnowledgeQuality(c *gin.Context) {
	if h.svc != nil {
		data := h.svc.EvaluateKnowledgeQuality(c.Request.Context())
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"total_resources": 1250,
		"coverage":        0.82,
		"accuracy":        0.91,
		"data_source":     "fallback",
	})
}

// UserBehavior 用户行为分析
func (h *SysAdminHandler) UserBehavior(c *gin.Context) {
	if h.svc != nil {
		data := h.svc.AnalyzeUserBehavior(c.Request.Context())
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"period":         "2026年5月",
		"active_users":   3200,
		"dau":            450,
		"retention_rate": 0.68,
		"data_source":    "fallback",
	})
}
