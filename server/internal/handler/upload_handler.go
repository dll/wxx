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
		".txt": true, ".csv": true, ".pdf": true, ".docx": true,
		".xlsx": true, ".png": true, ".jpg": true, ".jpeg": true,
		".gif": true, ".bmp": true, ".webp": true,
		".mp4": true, ".avi": true, ".mov": true, ".mkv": true,
	}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("不支持的文件类型: %s。支持的格式：TXT, CSV, PDF, DOCX, XLSX, PNG, JPG, GIF, BMP, WEBP, MP4, AVI, MOV, MKV", ext),
		})
		return
	}

	result, err := h.docSvc.ProcessUpload(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	roleScopeJSON, _ := json.Marshal([]string{"student", "counselor", "student_union", "college_admin"})
	resourceTypeModel := resourceType
	if resourceTypeModel != "Policy" && resourceTypeModel != "Process" && resourceTypeModel != "Activity" {
		resourceTypeModel = "FAQ"
	}
	tagsJSON, _ := json.Marshal([]string{"上传文档", ext})

	req := &model.KBCreateRequest{
		ResourceType: resourceTypeModel,
		OwnerScope:   userCtx.OwnerScope,
		OwnerID:      userCtx.OwnerID,
		RoleScope:    string(roleScopeJSON),
		Title:        strings.TrimSuffix(result.FileName, filepath.Ext(result.FileName)),
		Summary:      fmt.Sprintf("上传文档：%s（%s, %s）", result.FileName, result.FileType, service.BytesToContentSize(result.FileSize)),
		Content:      result.TextContent,
		Tags:         string(tagsJSON),
	}

	created, saveErr := h.kbSvc.Create(req, userCtx.Username)
	if saveErr != nil {
		result.Error = fmt.Sprintf("知识入库失败: %v", saveErr)
	}

	_ = h.docSvc.SaveResult(result)

	resourceID := ""
	if created != nil {
		resourceID = created.ResourceID
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "上传成功",
		"file":              result.FileName,
		"file_type":         result.FileType,
		"file_size":         service.BytesToContentSize(result.FileSize),
		"content_preview":   service.GetTextPreview(result.TextContent, 200),
		"content":           result.TextContent,
		"title":             strings.TrimSuffix(result.FileName, filepath.Ext(result.FileName)),
		"summary":           fmt.Sprintf("上传文档：%s（%s, %s）", result.FileName, result.FileType, service.BytesToContentSize(result.FileSize)),
		"pages":             result.Pages,
		"in_knowledge_base": saveErr == nil,
		"resource_id":       resourceID,
	})
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
