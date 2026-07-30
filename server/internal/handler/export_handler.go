package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// exportService 知识导出服务接口（用于测试 mock）
type exportService interface {
	ExportResources(ctx context.Context, resourceType, sinceCursor, callerScope, callerOwnerID string) ([]*model.KBResource, error)
}

// ExportHandler 知识导出 HTTP handler
type ExportHandler struct {
	kbSvc      exportService
	exportSvc  *service.ExportService
	hmacSecret string // HMAC-SHA256 签名密钥（Q-05：知识导出包签名）
}

// NewExportHandler 创建导出 handler
func NewExportHandler(kbSvc *service.KBService, exportSvc *service.ExportService) *ExportHandler {
	return &ExportHandler{kbSvc: kbSvc, exportSvc: exportSvc}
}

// SetHMACSecret 设置知识导出包 HMAC-SHA256 签名密钥（可选；空值 = 不签名）
func (h *ExportHandler) SetHMACSecret(secret string) {
	h.hmacSecret = secret
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

	// 安全修复 RB-01：以 JWT 上下文的 scope 强制过滤导出范围，忽略客户端可能伪造的参数
	resources, err := h.kbSvc.ExportResources(c.Request.Context(), resourceType, sinceCursor, userCtx.OwnerScope, userCtx.OwnerID)
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

	resp := model.ExportResponse{
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
	}

	// Q-05：序列化响应体并计算 HMAC-SHA256 签名，供接收方（如蔚园智答）校验包完整性
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "响应序列化失败",
		})
		return
	}

	if h.hmacSecret != "" {
		sig := computeHMACSHA256(jsonBytes, h.hmacSecret)
		c.Header("X-Signature", "sha256="+sig)
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", jsonBytes)
}

// computeHMACSHA256 计算 HMAC-SHA256 并返回十六进制字符串
func computeHMACSHA256(data []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyExportSignature 校验知识导出包的 HMAC-SHA256 签名（供接收方调用）
// body 为响应体 JSON 字节；sigHeader 为 X-Signature 响应头的值（格式：sha256=<hex>）；secret 为共享密钥。
// 使用恒时比较，防止时序攻击。
func VerifyExportSignature(body []byte, sigHeader, secret string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(sigHeader, prefix) {
		return false
	}
	expected := computeHMACSHA256(body, secret)
	// hmac.Equal 使用恒时比较，防止时序攻击
	return hmac.Equal([]byte(sigHeader[len(prefix):]), []byte(expected))
}
