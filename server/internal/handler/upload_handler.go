package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	resourceType := c.DefaultPostForm("resource_type", "FAQ")
	if !auth.HasCapability(userCtx.Role, auth.CounselorKBWrite) {
		if !auth.HasCapability(userCtx.Role, auth.UnionKBSubmit) {
			c.JSON(http.StatusForbidden, gin.H{"error": "无上传权限"})
			return
		}
		resourceType = "FAQ"
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的文件"})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("不支持的文件类型: %s。支持的格式：TXT, MD, CSV, PDF, DOCX, XLSX, PNG, JPG, GIF, BMP, WEBP, MP4, AVI, MOV, MKV", ext),
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "文档解析失败: " + err.Error()})
			return
		}
	} else {
		result, err = h.docSvc.ProcessUpload(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		"max_size_mb": 50,
		"note":        "上传后自动提取文本并入库至知识库",
	})
}
