package handler

import (
	"io"
	"net/http"
	"os"
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
			Message: "参数校验失败: " + err.Error(),
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
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "提交反馈失败: " + err.Error(),
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
			Message: "查询反馈列表失败: " + err.Error(),
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
			Message: "参数校验失败: " + err.Error(),
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
			Message: err.Error(),
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
func (h *FeedbackHandler) UploadScreenshot(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "未获取到上传文件: " + err.Error(),
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

	// 确保上传目录存在（Vercel serverless 只有 /tmp 可写）
	uploadDir := "data/uploads/feedback"
	if os.Getenv("VERCEL") != "" {
		uploadDir = "/tmp/data/uploads/feedback"
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "服务器存储初始化失败",
		})
		return
	}

	dstPath := filepath.Join(uploadDir, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "创建文件失败",
		})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "写入文件失败",
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
