package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// AIBriefingHandler AI 简讯 HTTP handler
type AIBriefingHandler struct {
	svc *service.AIBriefingService
}

// NewAIBriefingHandler 创建 AI 简讯 handler
func NewAIBriefingHandler(svc *service.AIBriefingService) *AIBriefingHandler {
	return &AIBriefingHandler{svc: svc}
}

// ListUser 用户端资讯列表 GET /api/v1/ai-briefings
// 支持 category / q / hot（hot=1 按热度排序）/ limit；返回含当前用户收藏态
func (h *AIBriefingHandler) ListUser(c *gin.Context) {
	category := c.Query("category")
	q := c.Query("q")
	hot := c.Query("hot") == "1" || c.Query("hot") == "true"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	userCtx := middleware.GetUserContext(c)
	list, err := h.svc.ListUserVisible(category, q, limit, userCtx.UserID, hot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询资讯失败"})
		return
	}
	if list == nil {
		list = []*model.AIBriefing{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// ListUserHot 热度榜 GET /api/v1/ai-briefings/hot
func (h *AIBriefingHandler) ListUserHot(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	userCtx := middleware.GetUserContext(c)
	list, err := h.svc.ListUserVisible("", "", limit, userCtx.UserID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询热度榜失败"})
		return
	}
	if list == nil {
		list = []*model.AIBriefing{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// ListFavorites 我的收藏 GET /api/v1/ai-briefings/favorites
func (h *AIBriefingHandler) ListFavorites(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	userCtx := middleware.GetUserContext(c)
	list, err := h.svc.ListFavorites(userCtx.UserID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询收藏失败"})
		return
	}
	if list == nil {
		list = []*model.AIBriefing{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// Favorite 收藏 POST /api/v1/ai-briefings/:id/favorite
func (h *AIBriefingHandler) Favorite(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if err := h.svc.Favorite(userCtx.UserID, id); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "收藏失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "收藏成功"})
}

// Unfavorite 取消收藏 DELETE /api/v1/ai-briefings/:id/favorite
func (h *AIBriefingHandler) Unfavorite(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if err := h.svc.Unfavorite(userCtx.UserID, id); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "取消收藏失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已取消收藏"})
}

// List 管理端列表 GET /api/v1/admin/ai-briefings
func (h *AIBriefingHandler) List(c *gin.Context) {
	status := c.Query("status")
	category := c.Query("category")
	q := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, err := h.svc.List(status, category, q, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询资讯失败"})
		return
	}
	if list == nil {
		list = []*model.AIBriefing{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"list": list, "total": total}})
}

// Create 新增 POST /api/v1/admin/ai-briefings
func (h *AIBriefingHandler) Create(c *gin.Context) {
	var b model.AIBriefing
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	id, err := h.svc.Create(&b, userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": gin.H{"id": id}})
}

// Update 更新 PUT /api/v1/admin/ai-briefings/:id
func (h *AIBriefingHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var b model.AIBriefing
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	b.ID = id
	if err := h.svc.Update(&b); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// UpdateStatus 上下架 PUT /api/v1/admin/ai-briefings/:id/status
func (h *AIBriefingHandler) UpdateStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	if err := h.svc.UpdateStatus(id, req.Status); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "操作成功"})
}

// Delete 删除 DELETE /api/v1/admin/ai-briefings/:id
func (h *AIBriefingHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// DeleteMany 批量删除 POST /api/v1/admin/ai-briefings/batch-delete
func (h *AIBriefingHandler) DeleteMany(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	n, err := h.svc.DeleteMany(req.IDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功", "data": gin.H{"deleted": n}})
}

// ClearAll 清空全部 DELETE /api/v1/admin/ai-briefings/clear
func (h *AIBriefingHandler) ClearAll(c *gin.Context) {
	n, err := h.svc.ClearAll()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "清空成功", "data": gin.H{"deleted": n}})
}

// Stats 汇总统计 GET /api/v1/admin/ai-briefings/stats
func (h *AIBriefingHandler) Stats(c *gin.Context) {
	s, err := h.svc.Stats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "统计失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": s})
}

// Export 导出 GET /api/v1/admin/ai-briefings/export?format=md|pdf&all=1&ids=1,2,3
// 支持 GET query 参数（浏览器直接下载）与 POST body（前端选择导出）两种方式
func (h *AIBriefingHandler) Export(c *gin.Context) {
	format := c.DefaultQuery("format", "md")
	if format != "md" && format != "pdf" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "导出格式仅支持 md/pdf"})
		return
	}
	var ids []int64
	all := c.Query("all") == "1" || c.Query("all") == "true"
	if idsStr := c.Query("ids"); idsStr != "" {
		for _, p := range strings.Split(idsStr, ",") {
			if id, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
	}
	// POST body 兼容
	if c.Request.Method == http.MethodPost {
		var req struct {
			IDs []int64 `json:"ids"`
			All bool    `json:"all"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			ids = req.IDs
			all = req.All
		}
	}

	var items []*model.AIBriefing
	if all {
		list, _, err := h.svc.List("", "", "", 1, 1000)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "导出失败"})
			return
		}
		items = list
	} else if len(ids) > 0 {
		items, _ = h.svc.GetByIDs(ids)
	} else {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请选择要导出的资讯"})
		return
	}

	filename := "ai-briefings." + format
	c.Header("Content-Disposition", "attachment; filename="+filename)
	if format == "md" {
		data, err := h.svc.ExportBriefingsMarkdown(items)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "导出失败"})
			return
		}
		c.Data(http.StatusOK, "text/markdown; charset=utf-8", data)
		return
	}
	data, err := h.svc.ExportBriefingsPDF(items)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "导出失败"})
		return
	}
	c.Data(http.StatusOK, "application/pdf", data)
}

// FetchNow 立即抓取 POST /api/v1/admin/ai-briefings/fetch
func (h *AIBriefingHandler) FetchNow(c *gin.Context) {
	result := h.svc.FetchNow()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "抓取完成", "data": result})
}

// ── 来源管理 ──

// ListSources GET /api/v1/admin/ai-briefings/sources
func (h *AIBriefingHandler) ListSources(c *gin.Context) {
	list, err := h.svc.ListSources()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询来源失败"})
		return
	}
	if list == nil {
		list = []*model.AIBriefingSource{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// CreateSource POST /api/v1/admin/ai-briefings/sources
func (h *AIBriefingHandler) CreateSource(c *gin.Context) {
	var src model.AIBriefingSource
	if err := c.ShouldBindJSON(&src); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	id, err := h.svc.CreateSource(&src)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": gin.H{"id": id}})
}

// UpdateSource PUT /api/v1/admin/ai-briefings/sources/:id
func (h *AIBriefingHandler) UpdateSource(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var src model.AIBriefingSource
	if err := c.ShouldBindJSON(&src); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	src.ID = id
	if err := h.svc.UpdateSource(&src); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteSource DELETE /api/v1/admin/ai-briefings/sources/:id
func (h *AIBriefingHandler) DeleteSource(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.DeleteSource(id); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
