package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/util"
	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	docSvc *service.DocumentService
	kbSvc  *service.KBService
}

func NewUploadHandler(docSvc *service.DocumentService, kbSvc *service.KBService) *UploadHandler {
	return &UploadHandler{docSvc: docSvc, kbSvc: kbSvc}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证", TraceID: middleware.GetTraceID(c)})
		return
	}

	resourceType := c.DefaultPostForm("resource_type", "FAQ")
	if !auth.HasCapability(userCtx.Role, auth.CounselorKBWrite) {
		if !auth.HasCapability(userCtx.Role, auth.UnionKBSubmit) {
			c.JSON(http.StatusForbidden, model.ErrorResponse{Code: 403, Message: "无上传权限", TraceID: middleware.GetTraceID(c)})
			return
		}
		resourceType = "FAQ"
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请选择要上传的文件", TraceID: middleware.GetTraceID(c)})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".txt": true, ".md": true, ".csv": true, ".pdf": true, ".docx": true,
		".xlsx": true, ".png": true, ".jpg": true, ".jpeg": true,
		".gif": true, ".bmp": true, ".webp": true,
		".mp4": true, ".avi": true, ".mov": true, ".mkv": true,
	}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: fmt.Sprintf("不支持的文件类型: %s。支持的格式：TXT, MD, CSV, PDF, DOCX, XLSX, PNG, JPG, GIF, BMP, WEBP, MP4, AVI, MOV, MKV", ext),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	isTextDoc := map[string]bool{
		".txt": true, ".md": true, ".csv": true, ".pdf": true, ".docx": true, ".xlsx": true,
	}

	var parseResult *service.DocumentParseResult
	var result *service.DocumentResult

	if isTextDoc[ext] {
		parseResult, err = h.docSvc.ParseDocument(file)
		if err != nil {
			log.Printf("文档解析失败: %v", err)
			// 无文本层（扫描件/图片型 PDF/DOCX）不是服务端故障，返回 422 + 引导
			if errors.Is(err, service.ErrNoTextLayer) {
				c.JSON(http.StatusUnprocessableEntity, model.ErrorResponse{
					Code:    422,
					Message: err.Error() + "。可先转成带文字层的版本（如从 Word/WPS「另存为 PDF」），或上传文本型 DOCX。",
					TraceID: middleware.GetTraceID(c),
				})
				return
			}
			util.FailInternalError(c, "文档解析失败")
			return
		}

		// 解析质量门槛：正文过短/无中文/疑似乱码时拒绝自动入库，防止污染知识库。
		// force=1 可显式覆盖（如确为英文文档等边缘场景）。
		if parseResult.Quality != nil && !parseResult.Quality.OK {
			force := c.PostForm("force")
			if force != "1" && force != "true" {
				c.JSON(http.StatusUnprocessableEntity, model.ErrorResponse{
					Code:    422,
					Message: "解析内容质量不达标，已拒绝自动入库：" + strings.Join(parseResult.Quality.Reasons, "；"),
					TraceID: middleware.GetTraceID(c),
				})
				return
			}
		}

		// 入库前自动 LLM 精修元数据（标题/摘要/关键词），让入库内容可直接使用。
		// 未配置模型 / 调用失败 / 输出不合法时静默回退启发式结果，不影响上传成功。
		refined := h.docSvc.RefineMetadata(c.Request.Context(), parseResult.Title, parseResult.Summary, parseResult.Keywords, parseResult.Content)
		if refined != nil && !refined.Fallback {
			parseResult.Title = refined.Title
			parseResult.Summary = refined.Summary
			parseResult.Keywords = refined.Keywords
		}
	} else {
		result, err = h.docSvc.ProcessUpload(file)
		if err != nil {
			log.Printf("文件上传处理失败: %v", err)
			util.FailInternalError(c, "文件上传处理失败")
			return
		}
	}

	roleScopeJSON, _ := json.Marshal([]string{"student", "counselor", "student_union", "college_admin"})
	resourceTypeModel := resourceType
	if resourceTypeModel != "Policy" && resourceTypeModel != "Process" && resourceTypeModel != "Activity" {
		resourceTypeModel = "FAQ"
	}

	var title, summary, content string
	var pages int
	var keywords []string
	var wordCount, paragraphs int
	var fileName, fileType string
	var fileSize int64

	if parseResult != nil {
		title = parseResult.Title
		summary = parseResult.Summary
		content = parseResult.Content
		pages = parseResult.Pages
		keywords = parseResult.Keywords
		wordCount = parseResult.WordCount
		paragraphs = parseResult.Paragraphs
		fileName = parseResult.FileName
		fileType = parseResult.FileType
		fileSize = parseResult.FileSize
	} else {
		title = strings.TrimSuffix(result.FileName, filepath.Ext(result.FileName))
		summary = fmt.Sprintf("上传文档：%s（%s, %s）", result.FileName, result.FileType, service.BytesToContentSize(result.FileSize))
		content = result.TextContent
		pages = result.Pages
		fileName = result.FileName
		fileType = result.FileType
		fileSize = result.FileSize
	}

	tags := []string{"上传文档", ext}
	if len(keywords) > 0 {
		tags = append(tags, keywords...)
	}
	if len(tags) > 10 {
		tags = tags[:10]
	}
	tagsJSON, _ := json.Marshal(tags)

	req := &model.KBCreateRequest{
		ResourceType: resourceTypeModel,
		OwnerScope:   userCtx.OwnerScope,
		OwnerID:      userCtx.OwnerID,
		RoleScope:    string(roleScopeJSON),
		Title:        title,
		Summary:      summary,
		Content:      content,
		Tags:         string(tagsJSON),
	}

	created, saveErr := h.kbSvc.Create(c.Request.Context(), req, userCtx.Username)

	resourceID := ""
	if created != nil {
		resourceID = created.ResourceID
	}

	if parseResult != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":              0,
			"message":           "上传成功",
			"file":              fileName,
			"file_type":         fileType,
			"file_size":         service.BytesToContentSize(fileSize),
			"title":             title,
			"summary":           summary,
			"content":           content,
			"keywords":          keywords,
			"word_count":        wordCount,
			"paragraphs":        paragraphs,
			"pages":             pages,
			"in_knowledge_base": saveErr == nil,
			"resource_id":       resourceID,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":              0,
			"message":           "上传成功",
			"file":              fileName,
			"file_type":         fileType,
			"file_size":         service.BytesToContentSize(fileSize),
			"content_preview":   service.GetTextPreview(content, 200),
			"content":           content,
			"title":             title,
			"summary":           summary,
			"pages":             pages,
			"in_knowledge_base": saveErr == nil,
			"resource_id":       resourceID,
		})
	}
}

func (h *UploadHandler) SupportedFormats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"formats": []string{
			"txt", "csv", "pdf", "docx", "xlsx",
			"png", "jpg", "jpeg", "gif", "bmp", "webp",
			"mp4", "avi", "mov", "mkv",
		},
		"max_size_mb": 100,
		"note":        "上传后自动提取文本并入库至知识库",
	})
}
