package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"

	"github.com/gin-gonic/gin"
)

// AppVersionHandler 应用版本管理 HTTP handler
type AppVersionHandler struct {
	service *service.AppVersionService
}

// NewAppVersionHandler 创建版本管理 handler
func NewAppVersionHandler(svc *service.AppVersionService) *AppVersionHandler {
	return &AppVersionHandler{service: svc}
}

// CheckUpdateRequest 检查更新请求
type CheckUpdateRequest struct {
	Platform    string `form:"platform" json:"platform"`
	VersionCode int    `form:"version_code" json:"version_code"`
	VersionName string `form:"version_name" json:"version_name"`
}

// CheckUpdateResponse 检查更新响应
type CheckUpdateResponse struct {
	HasUpdate bool              `json:"has_update"`
	IsForce   bool              `json:"is_force"`
	Latest    *model.AppVersion `json:"latest,omitempty"`
}

// CheckUpdate 检查版本更新（公开接口，无需登录）
func (h *AppVersionHandler) CheckUpdate(c *gin.Context) {
	var req CheckUpdateRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
		})
		return
	}

	if req.Platform == "" {
		req.Platform = "all"
	}

	latest, hasUpdate, err := h.service.CheckUpdate(req.Platform, req.VersionCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "检查更新失败",
		})
		return
	}

	isForce := false
	if latest != nil && latest.ForceUpdate == 1 {
		isForce = true
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": CheckUpdateResponse{
			HasUpdate: hasUpdate,
			IsForce:   isForce,
			Latest:    latest,
		},
		"message": "ok",
	})
}

// GetLatestVersion 获取最新版本信息
func (h *AppVersionHandler) GetLatestVersion(c *gin.Context) {
	platform := c.DefaultQuery("platform", "all")

	latest, err := h.service.GetLatestVersion(platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取最新版本失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    latest,
		"message": "ok",
	})
}

// ListVersions 版本列表（管理用）
func (h *AppVersionHandler) ListVersions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	list, total, err := h.service.ListVersions(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取版本列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
		"message": "ok",
	})
}

// CreateVersion 创建新版本（管理用）
func (h *AppVersionHandler) CreateVersion(c *gin.Context) {
	var v model.AppVersion
	if err := c.ShouldBindJSON(&v); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	if v.VersionCode <= 0 || v.VersionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "版本号和版本名称不能为空",
		})
		return
	}

	if v.Platform == "" {
		v.Platform = "all"
	}

	if err := h.service.CreateVersion(&v); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建版本失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    v,
		"message": "创建成功",
	})
}

// UpdateVersion 更新版本信息（管理用）
func (h *AppVersionHandler) UpdateVersion(c *gin.Context) {
	var v model.AppVersion
	if err := c.ShouldBindJSON(&v); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	if v.ID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "版本ID不能为空",
		})
		return
	}

	if err := h.service.UpdateVersion(&v); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// DeleteVersion 删除版本（管理用）
func (h *AppVersionHandler) DeleteVersion(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "ID格式错误",
		})
		return
	}

	if err := h.service.DeleteVersion(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}
