package handler

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/util"
	"github.com/gin-gonic/gin"
)

// GetSettings 获取系统配置 GET /api/v1/admin/settings
func (h *AdminHandler) GetSettings(c *gin.Context) {
	settings, err := h.adminSvc.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取系统配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.SettingsResponse{
		Code:    0,
		Message: "success",
		Data:    settings,
	})
}

// UpdateSettings 更新系统配置 PUT /api/v1/admin/settings
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var req model.SettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("admin UpdateSettings bind err: %v", err)
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

	if err := h.adminSvc.UpdateSettings(req.Settings, userCtx.Username); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "更新系统配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "系统配置已更新",
	})
}

// ResetUserPassword 管理员重置用户密码 PUT /api/v1/admin/users/:id/password
func (h *AdminHandler) ResetUserPassword(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "用户 ID 格式错误",
		})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("admin ResetUserPassword bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	if err := h.authSvc.ResetPassword(userCtx.UserID, userID, req.Password); err != nil {
		log.Printf("重置密码失败 user_id=%d: %v", userID, err)
		util.FailBadRequest(c, "重置密码失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "密码已重置",
	})
}

// ListPendingGuests 列出待审核游客 GET /api/v1/admin/guests/pending
func (h *AdminHandler) ListPendingGuests(c *gin.Context) {
	guests, err := h.authSvc.ListPendingGuests()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询待审核游客失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    guests,
	})
}

// ApproveGuest 审核通过游客 PUT /api/v1/admin/guests/:id/approve
func (h *AdminHandler) ApproveGuest(c *gin.Context) {
	guestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "用户 ID 格式错误",
		})
		return
	}

	var req struct {
		StudentID string `json:"student_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("admin ApproveGuest bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	if err := h.authSvc.ApproveGuest(guestID, req.StudentID); err != nil {
		log.Printf("审核通过游客失败 guest_id=%d: %v", guestID, err)
		util.FailBadRequest(c, "审核通过失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "游客审核通过，已升级为学生",
	})
}

// RejectGuest 拒绝游客申请 PUT /api/v1/admin/guests/:id/reject
func (h *AdminHandler) RejectGuest(c *gin.Context) {
	guestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "用户 ID 格式错误",
		})
		return
	}

	if err := h.authSvc.RejectGuest(guestID); err != nil {
		log.Printf("拒绝游客失败 guest_id=%d: %v", guestID, err)
		util.FailBadRequest(c, "拒绝游客失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "已拒绝该游客申请",
	})
}

// ImportStudentsRequest 导入学生请求体（JSON 格式）
type importStudentsRequest struct {
	Students        []importStudentItem `json:"students" binding:"required"`
	DefaultPassword string              `json:"default_password"` // 默认密码，空则用学号
}

type importStudentItem struct {
	Username       string `json:"username" binding:"required"`
	DisplayName    string `json:"display_name"`
	College        string `json:"college"`
	Major          string `json:"major"`
	ClassName      string `json:"class_name"`
	EnrollmentDate string `json:"enrollment_date"`
	EnrollmentYear string `json:"enrollment_year"`
	Role           string `json:"role"`
}

// ImportStudents 批量导入学生
// POST /api/v1/admin/users/import
func (h *AdminHandler) ImportStudents(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	contentType := c.ContentType()

	// 支持 multipart/form-data（上传 xlsx 文件）和 application/json（直接传 JSON）
	if strings.HasPrefix(contentType, "multipart/form-data") {
		h.importStudentsFromFile(c)
	} else {
		h.importStudentsFromJSON(c)
	}
}

// importStudentsFromFile 通过上传 xlsx 文件导入
func (h *AdminHandler) importStudentsFromFile(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 12<<20)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请选择要上传的 xlsx 文件"})
		return
	}
	defer file.Close()
	if !strings.EqualFold(filepath.Ext(header.Filename), ".xlsx") {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "仅支持 .xlsx 格式"})
		return
	}
	if header.Size <= 0 || header.Size > 10<<20 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "文件大小必须在 10MB 以内"})
		return
	}

	// 读取文件到内存
	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("读取导入文件失败: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "读取文件失败"})
		return
	}

	defaultPassword := c.DefaultPostForm("default_password", "")

	// 解析 xlsx
	rows, err := h.adminSvc.ParseStudentXLSX(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		log.Printf("解析学生Excel失败: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "解析文件失败，请检查格式"})
		return
	}

	result, err := h.adminSvc.ImportStudents(
		rows, defaultPassword, userCtx.Role, userCtx.OwnerScope, userCtx.OwnerID,
	)
	if err != nil {
		log.Printf("导入学生失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "导入失败，请检查数据"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "导入完成",
		"data":    result,
	})
}

// importStudentsFromJSON 通过 JSON 数组导入
func (h *AdminHandler) importStudentsFromJSON(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	var req importStudentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("admin importStudentsFromJSON bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	rows := make([]*service.ImportStudentRow, 0, len(req.Students))
	for _, s := range req.Students {
		rows = append(rows, &service.ImportStudentRow{
			Username:       s.Username,
			DisplayName:    s.DisplayName,
			College:        s.College,
			Major:          s.Major,
			ClassName:      s.ClassName,
			EnrollmentDate: s.EnrollmentDate,
			EnrollmentYear: s.EnrollmentYear,
			Role:           s.Role,
		})
	}

	result, err := h.adminSvc.ImportStudents(
		rows, req.DefaultPassword, userCtx.Role, userCtx.OwnerScope, userCtx.OwnerID,
	)
	if err != nil {
		log.Printf("JSON导入学生失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "导入失败，请检查数据"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "导入完成",
		"data":    result,
	})
}
