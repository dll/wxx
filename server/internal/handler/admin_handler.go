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

// AdminHandler 管理端 HTTP handler
type AdminHandler struct {
	adminSvc *service.AdminService
	authSvc  *service.AuthService
}

// NewAdminHandler 创建管理端 handler
func NewAdminHandler(adminSvc *service.AdminService, authSvc *service.AuthService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc, authSvc: authSvc}
}

// GetMetrics 质量看板 GET /api/v1/admin/metrics
func (h *AdminHandler) GetMetrics(c *gin.Context) {
	metrics, err := h.adminSvc.GetMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取质量看板数据失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.AdminMetricsResponse{
		Code:    0,
		Message: "success",
		Data:    metrics,
	})
}

// ListUsers 用户列表 GET /api/v1/admin/users?role=&owner_scope=&owner_id=&page=&page_size=
func (h *AdminHandler) ListUsers(c *gin.Context) {
	role := c.Query("role")
	ownerScope := c.Query("owner_scope")
	ownerID := c.Query("owner_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	// 范围过滤：college_admin 只能看本院用户
	queryScope := ownerScope
	queryOwnerID := ownerID
	if userCtx.Role == "college_admin" {
		queryScope = userCtx.OwnerScope
		queryOwnerID = userCtx.OwnerID
	}

	users, total, err := h.adminSvc.ListUsers(role, queryScope, queryOwnerID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询用户列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.UserListResponse{
		Code:     0,
		Message:  "success",
		Data:     users,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// UpdateUser 修改用户 PUT /api/v1/admin/users/:id
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "用户 ID 格式错误",
		})
		return
	}

	var req model.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("admin UpdateUser bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败",
		})
		return
	}

	if req.DisplayName == nil && req.Role == nil && req.OwnerScope == nil && req.OwnerID == nil && req.Status == nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "至少需要修改一项用户信息",
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

	user, err := h.adminSvc.UpdateUser(userID, &req, userCtx.Username)
	if err != nil {
		log.Printf("更新用户失败 user_id=%d: %v", userID, err)
		util.FailBadRequest(c, "更新用户失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "用户信息已更新",
		"data":    user,
	})
}

// DeleteUser 删除用户 DELETE /api/v1/admin/users/:id
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "用户 ID 格式错误",
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

	// 不允许删除自己
	if userCtx.UserID == userID {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "不允许删除当前登录账户",
		})
		return
	}

	if err := h.adminSvc.DeleteUser(userID, userCtx.Username); err != nil {
		log.Printf("删除用户失败 user_id=%d: %v", userID, err)
		util.FailBadRequest(c, "删除用户失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "用户已删除",
	})
}

// ListUsersAdvanced 高级用户查询 GET /api/v1/admin/users/advanced
// ?keyword=&role=&owner_scope=&owner_id=&college=&major=&class_name=&enrollment_year=&status=&sort_by=&sort_order=&page=&page_size=
func (h *AdminHandler) ListUsersAdvanced(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}

	ownerScope := c.Query("owner_scope")
	ownerID := c.Query("owner_id")
	if userCtx.Role == "college_admin" {
		ownerScope = userCtx.OwnerScope
		ownerID = userCtx.OwnerID
	}

	q := &model.UserQuery{
		Keyword:        c.Query("keyword"),
		Role:           c.Query("role"),
		OwnerScope:     ownerScope,
		OwnerID:        ownerID,
		College:        c.Query("college"),
		Major:          c.Query("major"),
		ClassName:      c.Query("class_name"),
		EnrollmentYear: c.Query("enrollment_year"),
		Status:         c.Query("status"),
		SortBy:         c.DefaultQuery("sort_by", "id"),
		SortOrder:      c.DefaultQuery("sort_order", "asc"),
		Offset:         offset,
		Limit:          pageSize,
	}

	users, total, err := h.adminSvc.ListUsersAdvanced(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询用户列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.UserListResponse{
		Code:     0,
		Message:  "success",
		Data:     users,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetUserDict 获取用户字典值 GET /api/v1/admin/users/dict?column=college
func (h *AdminHandler) GetUserDict(c *gin.Context) {
	column := c.Query("column")
	role := c.Query("role")

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	ownerScope := ""
	ownerID := ""
	if userCtx.Role == "college_admin" {
		ownerScope = userCtx.OwnerScope
		ownerID = userCtx.OwnerID
	}

	values, err := h.adminSvc.GetUserDictValues(column, role, ownerScope, ownerID)
	if err != nil {
		log.Printf("获取字典值失败 column=%s: %v", column, err)
		util.FailBadRequest(c, "获取字典值失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    values,
	})
}

// BatchUpdateStatus 批量更新用户状态 POST /api/v1/admin/users/batch/status
func (h *AdminHandler) BatchUpdateStatus(c *gin.Context) {
	var req struct {
		Ids    []int64 `json:"ids" binding:"required,min=1"`
		Status string  `json:"status" binding:"required,oneof=active disabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("admin BatchUpdateStatus bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数错误",
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

	count, err := h.adminSvc.BatchUpdateStatus(req.Ids, req.Status, userCtx.Username)
	if err != nil {
		log.Printf("批量更新用户状态失败: %v", err)
		util.FailBadRequest(c, "批量更新状态失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "操作成功",
		"data":    gin.H{"affected": count},
	})
}

// BatchResetPassword 批量重置密码 POST /api/v1/admin/users/batch/password
func (h *AdminHandler) BatchResetPassword(c *gin.Context) {
	var req struct {
		Ids         []int64 `json:"ids" binding:"required,min=1"`
		NewPassword string  `json:"new_password" binding:"required,min=6,max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("admin BatchResetPassword bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数错误",
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

	count, err := h.adminSvc.BatchResetPassword(req.Ids, req.NewPassword, userCtx.Username)
	if err != nil {
		log.Printf("批量重置密码失败: %v", err)
		util.FailBadRequest(c, "批量重置密码失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "密码重置成功",
		"data":    gin.H{"affected": count},
	})
}

// BatchDelete 批量删除用户 POST /api/v1/admin/users/batch/delete
func (h *AdminHandler) BatchDelete(c *gin.Context) {
	var req struct {
		Ids []int64 `json:"ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("admin BatchDelete bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数错误",
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

	// 检查是否包含自己
	for _, id := range req.Ids {
		if id == userCtx.UserID {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{
				Code:    400,
				Message: "不允许删除当前登录账户",
			})
			return
		}
	}

	count, err := h.adminSvc.BatchDelete(req.Ids, userCtx.Username)
	if err != nil {
		log.Printf("批量删除用户失败: %v", err)
		util.FailBadRequest(c, "批量删除用户失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data":    gin.H{"affected": count},
	})
}

// ListAudit 审计日志列表 GET /api/v1/admin/audit?username=&action=&resource=&start_date=&end_date=&page=&page_size=
func (h *AdminHandler) ListAudit(c *gin.Context) {
	username := c.Query("username")
	action := c.Query("action")
	resource := c.Query("resource")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	logs, total, err := h.adminSvc.ListAudit(username, action, resource, startDate, endDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询审计日志失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.AuditListResponse{
		Code:     0,
		Message:  "success",
		Data:     logs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// DeleteAudit 清理审计日志 DELETE /api/v1/admin/audit
// 支持按 username/action/resource/start_date/end_date 过滤；不带参数则清空全部。
func (h *AdminHandler) DeleteAudit(c *gin.Context) {
	username := c.Query("username")
	action := c.Query("action")
	resource := c.Query("resource")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var n int64
	var err error
	if username == "" && action == "" && resource == "" && startDate == "" && endDate == "" {
		// 无过滤条件 → 清空全部（需明确二次确认由前端承担）
		err = h.adminSvc.ClearAllAudit()
	} else {
		n, err = h.adminSvc.DeleteAudit(username, action, resource, startDate, endDate)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "清理审计日志失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "deleted": n})
}

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
