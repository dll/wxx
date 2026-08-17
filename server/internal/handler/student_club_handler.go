package handler

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// 社团生活
// ══════════════════════════════════════════════════════════════

// ListClubs 获取社团列表
// GET /api/v1/club/list?category=&page=&page_size=
func (h *StudentFeaturesHandler) ListClubs(c *gin.Context) {
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, total, err := h.svc.ListClubs(category, page, pageSize)
	if err != nil {
		log.Printf("查询社团列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items, "total": total, "page": page, "page_size": pageSize})
}

// GetClub 获取社团详情
// GET /api/v1/club/:id
func (h *StudentFeaturesHandler) GetClub(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	item, err := h.svc.GetClub(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "社团不存在"})
			return
		}
		log.Printf("查询社团详情失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": item})
}

// JoinClub 加入社团
// POST /api/v1/club/join
func (h *StudentFeaturesHandler) JoinClub(c *gin.Context) {
	var req struct {
		ClubID int64 `json:"club_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	id, err := h.svc.JoinClub(req.ClubID, userCtx.UserID, userCtx.Username, userCtx.DisplayName, "member")
	if err != nil {
		log.Printf("加入社团失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "加入失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "加入成功", "data": gin.H{"id": id}})
}

// GetMyClubs 获取我加入的社团
// GET /api/v1/club/my-clubs
func (h *StudentFeaturesHandler) GetMyClubs(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, err := h.svc.GetMyClubs(userCtx.UserID)
	if err != nil {
		log.Printf("查询我的社团失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

// ListClubActivities 获取社团活动列表
// GET /api/v1/club/activities?club_id=&status=&page=&page_size=
func (h *StudentFeaturesHandler) ListClubActivities(c *gin.Context) {
	clubID, _ := strconv.ParseInt(c.Query("club_id"), 10, 64)
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, total, err := h.svc.ListClubActivities(clubID, status, page, pageSize)
	if err != nil {
		log.Printf("查询社团活动失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items, "total": total, "page": page, "page_size": pageSize})
}

// RegisterClubActivity 报名社团活动
// POST /api/v1/club/activity/register
func (h *StudentFeaturesHandler) RegisterClubActivity(c *gin.Context) {
	var req struct {
		ActivityID int64 `json:"activity_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	id, err := h.svc.RegisterClubActivity(req.ActivityID, userCtx.UserID, userCtx.DisplayName)
	if err != nil {
		log.Printf("报名社团活动失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "报名失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "报名成功", "data": gin.H{"id": id}})
}
