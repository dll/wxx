package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// 健康活动 handler（SQL 已下沉 HealthActivityRepo，P4-d）

// ListHealthActivities 获取活动列表（按开始时间排序，含关注/报名状态）
// GET /api/v1/health/activities?category=sports
// student_union/管理员 可看全部；普通学生看 active
func (h *EducationHandler) ListHealthActivities(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	list, err := h.activityRepo.ListActivities(userID, c.Query("category"))
	if err != nil {
		log.Printf("health ListHealthActivities err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询活动失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// CreateHealthActivity 发起健身/竞技活动（学生会 student_union 及以上）
// POST /api/v1/health/activities
func (h *EducationHandler) CreateHealthActivity(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	var req struct {
		ActivityID     string `json:"activity_id"`
		Title          string `json:"title" binding:"required"`
		Category       string `json:"category"`
		Description    string `json:"description"`
		StartAt        string `json:"start_at"`
		EndAt          string `json:"end_at"`
		Venue          string `json:"venue"`
		Organizer      string `json:"organizer"`
		Capacity       int    `json:"capacity"`
		SignupDeadline string `json:"signup_deadline"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}
	if req.ActivityID == "" {
		req.ActivityID = "act-" + time.Now().Format("20060102") + "-" + randHex(6)
	}
	if req.Category == "" {
		req.Category = "sports"
	}
	if req.Organizer == "" {
		req.Organizer = userCtx.Username
	}

	if err := h.activityRepo.CreateActivity(req.ActivityID, req.Title, req.Category, req.Description,
		req.StartAt, req.EndAt, req.Venue, req.Organizer, req.Capacity, req.SignupDeadline,
		userCtx.UserID, userCtx.Role); err != nil {
		log.Printf("health CreateHealthActivity err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "发布活动失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "活动已发布", "data": gin.H{"activity_id": req.ActivityID}})
}

// ToggleActivityFavorite 关注/取消关注活动
// POST /api/v1/health/activities/:id/favorite  body: {favorite: bool}
func (h *EducationHandler) ToggleActivityFavorite(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	activityID := c.Param("id")
	var req struct {
		Favorite bool `json:"favorite"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	var err error
	if req.Favorite {
		err = h.activityRepo.AddFavorite(userID, activityID)
	} else {
		err = h.activityRepo.RemoveFavorite(userID, activityID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "操作失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// UpdateHealthActivityStatus 更新活动状态（进行中→已结束/已关闭），用于活动生命周期的"结束"阶段
// POST /api/v1/health/activities/:id/status  body: {status: "active"|"closed"|"ended"}
func (h *EducationHandler) UpdateHealthActivityStatus(c *gin.Context) {
	if middleware.GetUserContext(c) == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required,oneof=active closed ended"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "status 需为 active/closed/ended"})
		return
	}
	affected, err := h.activityRepo.UpdateStatus(c.Param("id"), req.Status)
	if err != nil {
		log.Printf("health UpdateHealthActivityStatus err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新活动状态失败"})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "活动不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "活动状态已更新", "status": req.Status})
}

// AttendActivitySignup 活动签到：学生会将某报名者标记为到场（复盘用）
// POST /api/v1/health/activities/:id/attend/:uid  body: {attended: bool}
func (h *EducationHandler) AttendActivitySignup(c *gin.Context) {
	var req struct {
		Attended bool `json:"attended"`
	}
	_ = c.ShouldBindJSON(&req)
	attach := 0
	if req.Attended {
		attach = 1
	}
	affected, err := h.activityRepo.MarkAttended(c.Param("id"), c.Param("uid"), attach)
	if err != nil {
		log.Printf("health AttendActivitySignup err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "签到失败"})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "该用户未报名或活动不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "签到状态已更新", "attended": req.Attended})
}

// ActivityReviewStats 活动复盘指标（真实统计，非 mock）
// GET /api/v1/health/activities/review-stats
func (h *EducationHandler) ActivityReviewStats(c *gin.Context) {
	rows, err := h.activityRepo.ListReviewRows()
	if err != nil {
		log.Printf("health ActivityReviewStats err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "获取复盘数据失败"})
		return
	}

	items := make([]*model.ActivityReviewRow, 0, len(rows))
	totalSignup, totalAttend := 0, 0
	catCount := map[string]int{}
	orgStat := map[string][2]int{} // organizer -> [报名, 到场]
	for _, r := range rows {
		if r.SignupCount > 0 {
			r.AttendRate = math.Round(float64(r.AttendCount)/float64(r.SignupCount)*1000) / 10
		}
		items = append(items, r)
		totalSignup += r.SignupCount
		totalAttend += r.AttendCount
		catCount[r.Category]++
		o := r.Organizer
		if o == "" {
			o = "未知组织方"
		}
		orgStat[o] = [2]int{orgStat[o][0] + r.SignupCount, orgStat[o][1] + r.AttendCount}
	}
	orgRank := make([]gin.H, 0, len(orgStat))
	for k, v := range orgStat {
		rate := 0.0
		if v[0] > 0 {
			rate = math.Round(float64(v[1])/float64(v[0])*1000) / 10
		}
		orgRank = append(orgRank, gin.H{"organizer": k, "signup": v[0], "attend": v[1], "attend_rate": rate})
	}
	// 到场率降序
	for i := 0; i < len(orgRank); i++ {
		for j := i + 1; j < len(orgRank); j++ {
			if orgRank[j]["attend_rate"].(float64) > orgRank[i]["attend_rate"].(float64) {
				orgRank[i], orgRank[j] = orgRank[j], orgRank[i]
			}
		}
	}
	avgRate := 0.0
	if totalSignup > 0 {
		avgRate = math.Round(float64(totalAttend)/float64(totalSignup)*1000) / 10
	}
	c.JSON(http.StatusOK, gin.H{
		"code":            0,
		"total_signup":    totalSignup,
		"total_attend":    totalAttend,
		"avg_attend_rate": avgRate,
		"category_count":  catCount,
		"organizer_rank":  orgRank,
		"activities":      items,
	})
}

// ListActivitySignups 活动报名/到场名单（学生会签到用）
// GET /api/v1/health/activities/:id/signups
func (h *EducationHandler) ListActivitySignups(c *gin.Context) {
	list, err := h.activityRepo.ListSignups(c.Param("id"))
	if err != nil {
		log.Printf("health ListActivitySignups err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "获取名单失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "items": list})
}

// ToggleActivitySignup 报名/取消报名
// POST /api/v1/health/activities/:id/signup  body: {signup: bool}
func (h *EducationHandler) ToggleActivitySignup(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	activityID := c.Param("id")
	var req struct {
		Signup bool `json:"signup"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	var err error
	if req.Signup {
		err = h.activityRepo.AddSignup(userID, activityID)
	} else {
		err = h.activityRepo.CancelSignup(userID, activityID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "操作失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// randHex 生成随机 hex 后缀（用于 activity_id）
func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "000000"
	}
	return hex.EncodeToString(b)
}
