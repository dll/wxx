package handler

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// DocumentHandler 文档解析 HTTP handler
type DocumentHandler struct {
	docSvc *service.DocumentService
}

// NewDocumentHandler 创建文档解析 handler
func NewDocumentHandler(docSvc *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{docSvc: docSvc}
}

// ParseDocument 上传并解析文档
// POST /api/v1/documents/parse
// multipart/form-data，字段名 file
// 权限：counselor.kb.write 以上
func (h *DocumentHandler) ParseDocument(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未认证",
		})
		return
	}

	if !auth.HasCapability(userCtx.Role, auth.CounselorKBWrite) {
		if !auth.HasCapability(userCtx.Role, auth.UnionKBSubmit) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "无文档解析权限",
			})
			return
		}
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请选择要上传的文件",
		})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".txt":  true,
		".md":   true,
		".pdf":  true,
		".docx": true,
		".csv":  true,
		".xlsx": true,
	}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不支持的文件类型: " + ext + "。支持的格式：TXT, MD, PDF, DOCX, CSV, XLSX",
		})
		return
	}

	result, err := h.docSvc.ParseDocument(file)
	if err != nil {
		log.Printf("document ParseDocument err: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "文档解析失败，请检查文件格式或内容",
		})
		return
	}

	// refine=true 时用 LLM 精修标题/摘要/关键词，解析结果可直接用于表单回填。
	// 未配置模型 / 调用失败 / 输出不合法时静默回退启发式结果。
	if c.Query("refine") == "1" || strings.EqualFold(c.Query("refine"), "true") {
		refined := h.docSvc.RefineMetadata(c.Request.Context(), result.Title, result.Summary, result.Keywords, result.Content)
		if refined != nil && !refined.Fallback {
			result.Title = refined.Title
			result.Summary = refined.Summary
			result.Keywords = refined.Keywords
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "解析成功",
		"data":    result,
	})
}

// SupportedFormats 获取支持的文档格式
// GET /api/v1/documents/formats
func (h *DocumentHandler) SupportedFormats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"formats": []string{
				"txt", "md", "pdf", "docx", "csv", "xlsx",
			},
			"max_size_mb": 100,
			"note":        "支持的文档格式，解析后返回标题、摘要、关键词、字数、段落数等信息",
		},
	})
}

// RefineDocument 使用 LLM 精修文档元数据（标题/摘要/关键词）。
// POST /api/v1/documents/refine
// body: {"content": "...", "title": "...", "summary": "...", "keywords": [...]}
// 说明：content 为清洗后的正文文本；title/summary/keywords 作为精修失败时的兜底值。
// 权限：counselor.kb.write 以上（与解析一致）
func (h *DocumentHandler) RefineDocument(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未认证",
		})
		return
	}

	if !auth.HasCapability(userCtx.Role, auth.CounselorKBWrite) {
		if !auth.HasCapability(userCtx.Role, auth.UnionKBSubmit) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "无文档精修权限",
			})
			return
		}
	}

	var req struct {
		Content  string   `json:"content" binding:"required"`
		Title    string   `json:"title"`
		Summary  string   `json:"summary"`
		Keywords []string `json:"keywords"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("document RefineDocument bind err: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
		})
		return
	}

	result := h.docSvc.RefineMetadata(c.Request.Context(), req.Title, req.Summary, req.Keywords, req.Content)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "精修完成",
		"data":    result,
	})
}
