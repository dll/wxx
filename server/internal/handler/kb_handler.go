package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/util"
	"github.com/gin-gonic/gin"
)

// KBHandler 知识库管理 HTTP handler
type KBHandler struct {
	kbSvc *service.KBService
}

// NewKBHandler 创建知识库 handler
func NewKBHandler(kbSvc *service.KBService) *KBHandler {
	return &KBHandler{kbSvc: kbSvc}
}

// BrowseKnowledge 知识大厅浏览（面向所有已认证用户，无需管理角色）
// GET /api/v1/knowledge?type=Policy&page=1&page_size=20
// 兼容小程序端使用 resource_type 参数名
func (h *KBHandler) BrowseKnowledge(c *gin.Context) {
	resourceType := c.Query("type")
	if resourceType == "" {
		resourceType = c.Query("resource_type")
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未获取到用户信息")
		return
	}

	cards, total, err := h.kbSvc.Browse(c.Request.Context(), userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, resourceType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取知识大厅数据失败，请稍后重试",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, model.KnowledgeBrowseResponse{
		Code:     0,
		Message:  "success",
		Data:     cards,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// BrowseKnowledgePublic 公开知识大厅浏览（游客也可访问，仅返回全校公开的已发布资源）
// GET /api/v1/knowledge/public?type=Policy&page=1&page_size=20
func (h *KBHandler) BrowseKnowledgePublic(c *gin.Context) {
	resourceType := c.Query("type")
	if resourceType == "" {
		resourceType = c.Query("resource_type")
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	cards, total, err := h.kbSvc.Browse(c.Request.Context(), "school", "", "guest", resourceType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取知识大厅数据失败，请稍后重试",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, model.KnowledgeBrowseResponse{
		Code:     0,
		Message:  "success",
		Data:     cards,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// ListResources 知识列表（分页 + 过滤）
// GET /api/v1/kb/resources?page=1&page_size=20&status=published&resource_type=Policy&owner_scope=school
func (h *KBHandler) ListResources(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	ownerScope := c.Query("owner_scope")
	ownerID := c.Query("owner_id")
	status := c.Query("status")
	resourceType := c.Query("resource_type")

	list, total, err := h.kbSvc.List(c.Request.Context(), ownerScope, ownerID, status, resourceType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询知识列表失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, model.KBListResponse{
		Code:     0,
		Message:  "success",
		Data:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetResource 知识详情
// GET /api/v1/kb/resources/:id
func (h *KBHandler) GetResource(c *gin.Context) {
	resourceID := c.Param("id")
	if resourceID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "资源 ID 不能为空",
		})
		return
	}

	kb, err := h.kbSvc.Get(c.Request.Context(), resourceID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Code:    404,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.KBDetailResponse{
		Code:    0,
		Message: "success",
		Data:    kb,
	})
}

// CreateResource 创建知识资源
// POST /api/v1/kb/resources
func (h *KBHandler) CreateResource(c *gin.Context) {
	var req model.KBCreateRequest
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

	kb, err := h.kbSvc.Create(c.Request.Context(), &req, userCtx.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "创建知识资源失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusCreated, model.KBDetailResponse{
		Code:    0,
		Message: "创建成功",
		Data:    kb,
	})
}

// Import 导入知识资源（支持 NDJSON 文本 或 JSON 包裹格式 {"resources": [...]}）
// POST /api/v1/kb/import
func (h *KBHandler) Import(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "读取请求体失败: " + err.Error(),
		})
		return
	}
	if len(body) == 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求体为空",
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

	// 根据 Content-Type 判断格式，未知时回退到试探解析
	contentType := c.GetHeader("Content-Type")
	isJSONWrapper := strings.HasPrefix(contentType, "application/json") && !strings.Contains(contentType, "ndjson")
	ndjsonData := string(body)
	if isJSONWrapper {
		var importReq model.KBImportRequest
		if err := json.Unmarshal(body, &importReq); err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{
				Code:    400,
				Message: "JSON 格式错误，无法解析请求体",
			})
			return
		}
		if len(importReq.Resources) == 0 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{
				Code:    400,
				Message: "resources 数组为空",
			})
			return
		}
		var lines []string
		for _, r := range importReq.Resources {
			b, _ := json.Marshal(r)
			lines = append(lines, string(b))
		}
		ndjsonData = strings.Join(lines, "\n")
	}

	resp, err := h.kbSvc.ImportResources(c.Request.Context(), ndjsonData, userCtx.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "导入失败，请检查数据格式后重试",
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateResource 更新知识资源
// PUT /api/v1/kb/resources/:id
func (h *KBHandler) UpdateResource(c *gin.Context) {
	resourceID := c.Param("id")
	if resourceID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "资源 ID 不能为空",
		})
		return
	}

	var req model.KBUpdateRequest
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

	kb, err := h.kbSvc.Update(c.Request.Context(), resourceID, &req, userCtx.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "更新知识资源失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, model.KBDetailResponse{
		Code:    0,
		Message: "更新成功",
		Data:    kb,
	})
}

// Validate 校验导入包（不落库），逐行解析 NDJSON 并校验字段完整性
// POST /api/v1/kb/validate
func (h *KBHandler) Validate(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "读取请求体失败",
		})
		return
	}

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	var warnings []string
	validCount := 0
	totalCount := 0

	validTypes := map[string]bool{"Policy": true, "Process": true, "FAQ": true, "Activity": true}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		totalCount++

		var kb model.KBResource
		if err := json.Unmarshal([]byte(line), &kb); err != nil {
			warnings = append(warnings, "第"+strconv.Itoa(totalCount)+"行JSON解析失败: "+err.Error())
			continue
		}

		var missing []string
		if kb.ResourceID == "" {
			missing = append(missing, "resource_id")
		}
		if kb.Title == "" {
			missing = append(missing, "title")
		}
		if kb.Content == "" {
			missing = append(missing, "content")
		}
		if kb.ResourceType == "" {
			missing = append(missing, "resource_type")
		}
		if len(missing) > 0 {
			warnings = append(warnings, "第"+strconv.Itoa(totalCount)+"行缺少必填字段: "+strings.Join(missing, ", "))
			continue
		}

		if !validTypes[kb.ResourceType] {
			warnings = append(warnings, "第"+strconv.Itoa(totalCount)+"行无效资源类型: "+kb.ResourceType)
			continue
		}

		validCount++
	}

	valid := len(warnings) == 0 && validCount > 0

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "校验完成",
		"data": gin.H{
			"valid":       valid,
			"warnings":    warnings,
			"recordCount": validCount,
			"totalCount":  totalCount,
		},
	})
}

// ListPendingReviews 待审核知识列表 GET /api/v1/review/pending
// 角色限制：counselor 及以上
func (h *KBHandler) ListPendingReviews(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	resources, total, err := h.kbSvc.ListPending(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询待审核列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.ReviewPendingResponse{
		Code:    0,
		Message: "success",
		Data:    resources,
		Total:   total,
	})
}

// ═══ 知识库运维工作流 ═══

// SubmitForReview 提交知识资源进入审核（draft → pending）
// POST /api/v1/kb/resources/:id/submit
// 角色限制：student_union 及以上
func (h *KBHandler) SubmitForReview(c *gin.Context) {
	resourceID := c.Param("id")
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	kb, err := h.kbSvc.SubmitForReview(c.Request.Context(), resourceID, userCtx.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.KBDetailResponse{
		Code:    0,
		Message: "已提交审核",
		Data:    kb,
	})
}

// ApproveResource 审核通过知识资源（pending → published）
// POST /api/v1/kb/resources/:id/approve
// 角色限制：counselor 及以上
func (h *KBHandler) ApproveResource(c *gin.Context) {
	resourceID := c.Param("id")
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	kb, err := h.kbSvc.ApproveResource(c.Request.Context(), resourceID, userCtx.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.KBDetailResponse{
		Code:    0,
		Message: "审核通过，已发布",
		Data:    kb,
	})
}

// rejectRequest 驳回请求体
type rejectRequest struct {
	Reason string `json:"reason"` // 驳回理由
}

// RejectResource 驳回知识资源（pending → draft）
// POST /api/v1/kb/resources/:id/reject
// 角色限制：counselor 及以上
func (h *KBHandler) RejectResource(c *gin.Context) {
	resourceID := c.Param("id")
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	var req rejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = "未提供理由"
	}

	kb, err := h.kbSvc.RejectResource(c.Request.Context(), resourceID, userCtx.Username, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.KBDetailResponse{
		Code:    0,
		Message: "已驳回，退回草稿状态",
		Data:    kb,
	})
}

// RetireResource 下架知识资源（published → retired）
// POST /api/v1/kb/resources/:id/retire
// 角色限制：counselor 及以上
func (h *KBHandler) RetireResource(c *gin.Context) {
	resourceID := c.Param("id")
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	kb, err := h.kbSvc.RetireResource(c.Request.Context(), resourceID, userCtx.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.KBDetailResponse{
		Code:    0,
		Message: "已下架",
		Data:    kb,
	})
}

// ════════ 高级查询与批量操作 ════════

// ListResourcesAdvanced 高级知识资源查询
// GET /api/v1/kb/resources/advanced
func (h *KBHandler) ListResourcesAdvanced(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	q := &model.KBQuery{
		Keyword:      c.Query("keyword"),
		ResourceType: c.Query("resource_type"),
		Status:       c.Query("status"),
		OwnerScope:   c.Query("owner_scope"),
		OwnerID:      c.Query("owner_id"),
		UpdatedBy:    c.Query("updated_by"),
		Tag:          c.Query("tag"),
		SortBy:       c.Query("sort_by"),
		SortOrder:    c.Query("sort_order"),
		Page:         page,
		PageSize:     pageSize,
	}

	list, total, err := h.kbSvc.ListAdvanced(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询知识资源失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, model.KBListResponse{
		Code:     0,
		Message:  "success",
		Data:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetDictValues 获取字典值（用于筛选下拉）
// GET /api/v1/kb/dict?column=resource_type
func (h *KBHandler) GetDictValues(c *gin.Context) {
	column := c.Query("column")
	if column == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "column 参数不能为空",
		})
		return
	}

	values, err := h.kbSvc.GetDictValues(c.Request.Context(), column)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取字典值失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    values,
	})
}

// batchIDsRequest 批量操作请求体
type batchIDsRequest struct {
	IDs []string `json:"ids"`
}

// BatchApprove 批量审核通过
// POST /api/v1/kb/batch/approve
func (h *KBHandler) BatchApprove(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	var req batchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败：ids 不能为空",
		})
		return
	}

	count, err := h.kbSvc.BatchApprove(c.Request.Context(), req.IDs, userCtx.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量审核通过成功",
		"data":    gin.H{"count": count},
	})
}

// BatchReject 批量驳回
// POST /api/v1/kb/batch/reject
func (h *KBHandler) BatchReject(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	var req batchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败：ids 不能为空",
		})
		return
	}

	count, err := h.kbSvc.BatchReject(c.Request.Context(), req.IDs, userCtx.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量驳回成功",
		"data":    gin.H{"count": count},
	})
}

// BatchRetire 批量下架
// POST /api/v1/kb/batch/retire
func (h *KBHandler) BatchRetire(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	var req batchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败：ids 不能为空",
		})
		return
	}

	count, err := h.kbSvc.BatchRetire(c.Request.Context(), req.IDs, userCtx.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量下架成功",
		"data":    gin.H{"count": count},
	})
}

// BatchDelete 批量删除
// POST /api/v1/kb/batch/delete
func (h *KBHandler) BatchDelete(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	var req batchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败：ids 不能为空",
		})
		return
	}

	count, err := h.kbSvc.BatchDelete(c.Request.Context(), req.IDs, userCtx.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量删除成功",
		"data":    gin.H{"count": count},
	})
}

// GetStats 获取知识资源统计
// GET /api/v1/kb/stats
func (h *KBHandler) GetStats(c *gin.Context) {
	stats, err := h.kbSvc.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取统计数据失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}
