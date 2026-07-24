package handler

import (
	"encoding/base64"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FeedbackHandler 用户反馈 HTTP handler
type FeedbackHandler struct {
	feedbackSvc *service.FeedbackService
}

// NewFeedbackHandler 创建反馈 handler
func NewFeedbackHandler(feedbackSvc *service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{feedbackSvc: feedbackSvc}
}

// Submit 提交反馈 POST /api/v1/feedback
func (h *FeedbackHandler) Submit(c *gin.Context) {
	var req model.FeedbackCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败",
		})
		return
	}

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	fb, err := h.feedbackSvc.Submit(userCtx.UserID, userCtx.Username, &req)
	if err != nil {
		// Vercel /tmp 数据库冷启动后，旧 token 对应的用户可能已不存在
		if err == service.ErrUserNotFound {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Code:    401,
				Message: "登录状态已失效，请重新登录",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "提交反馈失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "反馈已提交",
		"data":    fb,
	})
}

// List 反馈列表 GET /api/v1/feedback?status=&page=&page_size=
func (h *FeedbackHandler) List(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	items, total, err := h.feedbackSvc.List(status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询反馈列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.FeedbackListResponse{
		Code:     0,
		Message:  "success",
		Data:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Mine 我的反馈 GET /api/v1/feedback/mine?status=&page=&page_size=
// 仅返回当前登录用户提交的反馈，所有角色可用
func (h *FeedbackHandler) Mine(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	items, total, err := h.feedbackSvc.ListMine(userCtx.UserID, status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询我的反馈失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.FeedbackListResponse{
		Code:     0,
		Message:  "success",
		Data:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Resolve 处理反馈 PUT /api/v1/feedback/:id
func (h *FeedbackHandler) Resolve(c *gin.Context) {
	feedbackID := c.Param("id")

	var req model.FeedbackUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败",
		})
		return
	}

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	fb, err := h.feedbackSvc.Resolve(feedbackID, userCtx.Username, req.Status, req.Reply)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "处理反馈失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "反馈已处理",
		"data":    fb,
	})
}

// UploadScreenshot 上传反馈截图 POST /api/v1/feedback/screenshot
// 写入 SQLite blob 表（跨 Vercel 实例可读），返回 /uploads/feedback/{filename} URL
func (h *FeedbackHandler) UploadScreenshot(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "未获取到上传文件",
		})
		return
	}
	defer file.Close()

	// 限制文件大小（5MB）
	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "文件大小超过 5MB 限制",
		})
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".png"
	}
	filename := "fb-screenshot-" + uuid.New().String()[:8] + ext

	// 读取全部字节
	bytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "读取文件失败",
		})
		return
	}

	// 推断 MIME（仅按扩展名简单判断）
	mime := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	}

	// base64 编码后入库（跨实例持久；SQLite 文本字段更稳）
	encoded := base64.StdEncoding.EncodeToString(bytes)

	uploader := ""
	if u := middleware.GetUserContext(c); u != nil {
		uploader = u.Username
	}

	if err := h.feedbackSvc.SaveScreenshot(filename, mime, encoded, uploader, header.Size); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "保存截图失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "截图上传成功",
		"data": gin.H{
			"url":       "/uploads/feedback/" + filename,
			"filename":  filename,
			"size":      header.Size,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})
}

// ServeScreenshot GET /uploads/feedback/:filename — 从 SQLite blob 返回图片字节
func (h *FeedbackHandler) ServeScreenshot(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	dataB64, mime, err := h.feedbackSvc.GetScreenshot(filename)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if dataB64 == "" {
		c.Status(http.StatusNotFound)
		return
	}
	bytes, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if mime == "" {
		mime = "image/png"
	}
	// 浏览器缓存：截图不可变，缓存 1 天
	c.Header("Cache-Control", "public, max-age=86400, immutable")
	c.Data(http.StatusOK, mime, bytes)
}
