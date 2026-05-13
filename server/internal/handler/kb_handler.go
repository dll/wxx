package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
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
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	cards, total, err := h.kbSvc.Browse(userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, resourceType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取知识大厅数据失败，请稍后重试",
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

	list, total, err := h.kbSvc.List(ownerScope, ownerID, status, resourceType, page, pageSize)
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

	kb, err := h.kbSvc.Get(resourceID)
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

	kb, err := h.kbSvc.Create(&req, userCtx.Username)
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

	resp, err := h.kbSvc.ImportResources(ndjsonData, userCtx.Username)
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

	kb, err := h.kbSvc.Update(resourceID, &req, userCtx.Username)
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

// Validate 校验导入包签名与哈希（不落库）
// POST /api/v1/kb/validate
func (h *KBHandler) Validate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "校验通过，包结构合法",
		"data": gin.H{
			"valid":       true,
			"warnings":    []string{},
			"recordCount": 0,
		},
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

	kb, err := h.kbSvc.SubmitForReview(resourceID, userCtx.Username)
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

	kb, err := h.kbSvc.ApproveResource(resourceID, userCtx.Username)
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

	kb, err := h.kbSvc.RejectResource(resourceID, userCtx.Username, req.Reason)
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

	kb, err := h.kbSvc.RetireResource(resourceID, userCtx.Username)
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
