package handler

import (
	"net/http"
	"strconv"

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
			Message: "查询知识列表失败: " + err.Error(),
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
			Message: "创建知识资源失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, model.KBDetailResponse{
		Code:    0,
		Message: "创建成功",
		Data:    kb,
	})
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
			Message: "更新知识资源失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.KBDetailResponse{
		Code:    0,
		Message: "更新成功",
		Data:    kb,
	})
}
