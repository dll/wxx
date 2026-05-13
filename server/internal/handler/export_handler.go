package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// exportService 知识导出服务接口（用于测试 mock）
type exportService interface {
	ExportResources(resourceType, sinceCursor string) ([]*model.KBResource, error)
}

// ExportHandler 知识导出 HTTP handler
type ExportHandler struct {
	kbSvc     exportService
	exportSvc *service.ExportService
}

// NewExportHandler 创建导出 handler
func NewExportHandler(kbSvc *service.KBService, exportSvc *service.ExportService) *ExportHandler {
	return &ExportHandler{kbSvc: kbSvc, exportSvc: exportSvc}
}

// ExportAnswer 导出回答卡片为指定格式
// POST /api/v1/export/answer
func (h *ExportHandler) ExportAnswer(c *gin.Context) {
	if h.exportSvc == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Code:    503,
			Message: "导出服务暂不可用",
		})
		return
	}

	var req model.ExportAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数有误：" + err.Error(),
		})
		return
	}

	format := service.ExportPDF
	switch req.Format {
	case "json":
		format = service.ExportJSON
	case "md":
		format = service.ExportMD
	case "pdf", "":
		format = service.ExportPDF
	default:
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "不支持的导出格式：" + req.Format + "，可选：pdf / json / md",
		})
		return
	}

	data, mime, err := h.exportSvc.ExportAnswer(req.AnswerCard, format, req.Watermark)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "导出失败：" + err.Error(),
		})
		return
	}

	c.Header("Content-Type", mime)
	c.Header("Content-Disposition", "attachment; filename=answer."+string(format))
	c.Data(http.StatusOK, mime, data)
}

// Export 导出知识资源
// GET /api/v1/export?resource_type=Policy&since=2026-01-01T00:00:00Z
func (h *ExportHandler) Export(c *gin.Context) {
	resourceType := c.Query("resource_type")
	sinceCursor := c.Query("since")

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	resources, err := h.kbSvc.ExportResources(resourceType, sinceCursor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "导出服务暂不可用，请稍后重试",
		})
		return
	}

	// 记录导出日志
	log.Printf("知识导出 user=%s role=%s resource_type=%s count=%d since=%s",
		userCtx.Username, userCtx.Role, resourceType, len(resources), sinceCursor)

	// 生成导出游标（当前时间戳，供下次增量使用）
	cursor := time.Now().UTC().Format(time.RFC3339)

	c.JSON(http.StatusOK, model.ExportResponse{
		Code:    0,
		Message: "导出成功",
		Manifest: model.ExportManifest{
			ExportedAt: cursor,
			Format:     "json",
			Count:      len(resources),
			Cursor:     cursor,
			Version:    "1.0",
		},
		Data: resources,
	})
}
