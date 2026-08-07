package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
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
	kbSvc        exportService
	exportSvc    *service.ExportService
	hmacSecret   string // HMAC-SHA256 签名密钥（Q-05：知识导出包签名）
	pkgSvc       *service.KnowledgePackageService
	exportLogSvc *service.ExportLogService
}

// NewExportHandler 创建导出 handler
func NewExportHandler(kbSvc *service.KBService, exportSvc *service.ExportService) *ExportHandler {
	return &ExportHandler{kbSvc: kbSvc, exportSvc: exportSvc}
}

// SetHMACSecret 设置知识导出包 HMAC-SHA256 签名密钥（可选；空值 = 不签名）
func (h *ExportHandler) SetHMACSecret(secret string) {
	h.hmacSecret = secret
}

// SetPackageService 注入标准知识包服务。
func (h *ExportHandler) SetPackageService(svc *service.KnowledgePackageService) {
	h.pkgSvc = svc
}

func (h *ExportHandler) SetExportLogService(svc *service.ExportLogService) {
	h.exportLogSvc = svc
}

// ExportPackage 导出标准 zip 知识包（manifest.json + resources.ndjson）。
func (h *ExportHandler) ExportPackage(c *gin.Context) {
	if h.pkgSvc == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Code:    503,
			Message: "知识包服务暂不可用",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	sinceCursor := c.Query("sinceCursor")
	if sinceCursor == "" {
		sinceCursor = c.Query("since")
	}
	limit := 1000
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	data, manifest, err := h.pkgSvc.ExportPackage(
		c.Request.Context(),
		c.Query("resource_type"),
		sinceCursor,
		userCtx.OwnerScope,
		userCtx.OwnerID,
		limit,
	)
	if err != nil {
		log.Printf("知识包导出失败 user=%s err=%v", userCtx.Username, err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "知识包导出失败",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=kb-package.zip")
	c.Header("X-Export-Batch-Id", manifest.ExportBatchID)
	c.Header("X-Next-Cursor", manifest.NextCursor)
	if manifest.HasMore {
		c.Header("X-Has-More", "true")
	} else {
		c.Header("X-Has-More", "false")
	}
	if h.exportLogSvc != nil {
		_ = h.exportLogSvc.Log(
			c.Request.Context(),
			userCtx.UserID,
			userCtx.Role,
			"kb-zip",
			"",
			middleware.GetTraceID(c),
			false,
		)
	}
	c.Data(http.StatusOK, "application/zip", data)
}

// ImportPackage 导入标准 zip 知识包并校验 manifest/hash/签名。
func (h *ExportHandler) ImportPackage(c *gin.Context) {
	if h.pkgSvc == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Code:    503,
			Message: "知识包服务暂不可用",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "读取请求体失败",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}
	resp, err := h.pkgSvc.ImportPackage(c.Request.Context(), body, userCtx.Username, middleware.GetTraceID(c))
	if err != nil {
		log.Printf("知识包导入失败 user=%s err=%v", userCtx.Username, err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "知识包导入失败",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// InitChunkUpload 初始化知识包分片上传。
// POST /api/v1/kb/import/package/chunk/init
func (h *ExportHandler) InitChunkUpload(c *gin.Context) {
	if h.pkgSvc == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{Code: 503, Message: "知识包服务暂不可用", TraceID: middleware.GetTraceID(c)})
		return
	}
	var req model.KBImportChunkInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误", TraceID: middleware.GetTraceID(c)})
		return
	}
	resp, err := h.pkgSvc.InitChunkUpload(req.TotalChunks, req.ExpectedSha256)
	if err != nil {
		log.Printf("init chunk upload err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "初始化分片上传失败", TraceID: middleware.GetTraceID(c)})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// UploadChunk 上传单个分片。
// PUT /api/v1/kb/import/package/chunk/:upload_id/:chunk_index
func (h *ExportHandler) UploadChunk(c *gin.Context) {
	if h.pkgSvc == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{Code: 503, Message: "知识包服务暂不可用", TraceID: middleware.GetTraceID(c)})
		return
	}
	uploadID := c.Param("upload_id")
	chunkIndex, err := strconv.Atoi(c.Param("chunk_index"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "chunk_index 不合法", TraceID: middleware.GetTraceID(c)})
		return
	}
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "读取分片失败", TraceID: middleware.GetTraceID(c)})
		return
	}
	if err := h.pkgSvc.PutChunk(uploadID, chunkIndex, body, c.GetHeader("X-Chunk-Sha256")); err != nil {
		log.Printf("upload chunk err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "分片上传失败", TraceID: middleware.GetTraceID(c)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "分片上传成功"})
}

// ChunkUploadStatus 查询分片上传状态。
// GET /api/v1/kb/import/package/chunk/status/:upload_id
func (h *ExportHandler) ChunkUploadStatus(c *gin.Context) {
	if h.pkgSvc == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{Code: 503, Message: "知识包服务暂不可用", TraceID: middleware.GetTraceID(c)})
		return
	}
	status, err := h.pkgSvc.ChunkStatus(c.Param("upload_id"))
	if err != nil {
		log.Printf("chunk status err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "查询分片状态失败", TraceID: middleware.GetTraceID(c)})
		return
	}
	c.JSON(http.StatusOK, status)
}

// CompleteChunkUpload 汇总分片并导入。
// POST /api/v1/kb/import/package/chunk/complete/:upload_id
func (h *ExportHandler) CompleteChunkUpload(c *gin.Context) {
	if h.pkgSvc == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{Code: 503, Message: "知识包服务暂不可用", TraceID: middleware.GetTraceID(c)})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息", TraceID: middleware.GetTraceID(c)})
		return
	}
	resp, err := h.pkgSvc.CompleteChunkUpload(c.Request.Context(), c.Param("upload_id"), userCtx.Username, middleware.GetTraceID(c))
	if err != nil {
		log.Printf("complete chunk upload err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "分片汇总导入失败", TraceID: middleware.GetTraceID(c)})
		return
	}
	c.JSON(http.StatusOK, resp)
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
		log.Printf("export ExportAnswer bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数有误",
		})
		return
	}

	format := service.ExportPDF
	switch req.Format {
	case "json":
		format = service.ExportJSON
	case "md":
		format = service.ExportMD
	case "docx":
		format = service.ExportDOCX
	case "xlsx":
		format = service.ExportXLSX
	case "png":
		format = service.ExportPNG
	case "ics":
		format = service.ExportICS
	case "pdf", "":
		format = service.ExportPDF
	default:
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "不支持的导出格式：" + req.Format + "，可选：pdf / json / md / docx / xlsx / png / ics",
		})
		return
	}

	data, mime, err := h.exportSvc.ExportAnswer(req.AnswerCard, format, req.Watermark)
	if err != nil {
		log.Printf("export ExportAnswer err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "导出失败，请稍后重试",
		})
		return
	}
	if h.exportLogSvc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			_ = h.exportLogSvc.Log(
				c.Request.Context(),
				userCtx.UserID,
				userCtx.Role,
				req.Format,
				req.AnswerCard.TraceID,
				middleware.GetTraceID(c),
				false,
			)
		}
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
