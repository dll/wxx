package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"math"
	"net/http"
	"time"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

type healthActivityItem struct {
	ActivityID     string `json:"activity_id"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	StartAt        string `json:"start_at"`
	EndAt          string `json:"end_at"`
	Venue          string `json:"venue"`
	Organizer      string `json:"organizer"`
	Capacity       int    `json:"capacity"`
	SignupDeadline string `json:"signup_deadline"`
	Status         string `json:"status"`
	CreatorRole    string `json:"creator_role"`
	// 关注/报名统计与当前用户状态
	FavoriteCount int  `json:"favorite_count"`
	SignupCount   int  `json:"signup_count"`
	IsFavorite    bool `json:"is_favorite"`
	IsSignup      bool `json:"is_signup"`
}

// ListHealthActivities 获取活动列表（按开始时间排序，含关注/报名状态）
// GET /api/v1/health/activities?category=sports
// student_union/管理员 可看全部；普通学生看 active
func (h *EducationHandler) ListHealthActivities(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	category := c.Query("category")

	where := "WHERE a.status = 'active'"
	var args []interface{}
	if category != "" {
		where += " AND a.category = ?"
		args = append(args, category)
	}

	rows, err := h.db.Query(
		`SELECT a.activity_id, a.title, a.category, a.description, a.start_at, a.end_at,
		        a.venue, a.organizer, a.capacity, a.signup_deadline, a.status, a.creator_role,
		        (SELECT COUNT(*) FROM health_activity_favorites f WHERE f.activity_id = a.activity_id) AS fav,
		        (SELECT COUNT(*) FROM health_activity_signups s WHERE s.activity_id = a.activity_id AND s.status='registered') AS sg,
		        EXISTS(SELECT 1 FROM health_activity_favorites f2 WHERE f2.activity_id = a.activity_id AND f2.user_id = ?) AS is_fav,
		        EXISTS(SELECT 1 FROM health_activity_signups s2 WHERE s2.activity_id = a.activity_id AND s2.user_id = ? AND s2.status='registered') AS is_sg
		 FROM health_activities a `+where+` ORDER BY a.start_at DESC LIMIT 200`,
		append(args, userID, userID)...,
	)
	if err != nil {
		log.Printf("health ListHealthActivities err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询活动失败"})
		return
	}
	defer rows.Close()

	var list []*healthActivityItem
	for rows.Next() {
		item := &healthActivityItem{}
		var isFav, isSg int
		if err := rows.Scan(&item.ActivityID, &item.Title, &item.Category, &item.Description,
			&item.StartAt, &item.EndAt, &item.Venue, &item.Organizer, &item.Capacity,
			&item.SignupDeadline, &item.Status, &item.CreatorRole,
			&item.FavoriteCount, &item.SignupCount, &isFav, &isSg); err != nil {
			continue
		}
		item.IsFavorite = isFav == 1
		item.IsSignup = isSg == 1
		list = append(list, item)
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

	_, err := h.db.Exec(
		`INSERT INTO health_activities
		   (activity_id, title, category, description, start_at, end_at, venue, organizer, capacity, signup_deadline, status, creator_id, creator_role)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
		req.ActivityID, req.Title, req.Category, req.Description, req.StartAt, req.EndAt,
		req.Venue, req.Organizer, req.Capacity, req.SignupDeadline, userCtx.UserID, userCtx.Role,
	)
	if err != nil {
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

	if req.Favorite {
		_, err := h.db.Exec(
			dbutil.InsertIgnore(dbutil.DriverOf(h.db))+` health_activity_favorites (user_id, activity_id) VALUES (?, ?)`,
			userID, activityID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "操作失败"})
			return
		}
	} else {
		_, err := h.db.Exec(
			`DELETE FROM health_activity_favorites WHERE user_id = ? AND activity_id = ?`,
			userID, activityID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "操作失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// UpdateHealthActivityStatus 更新活动状态（进行中→已结束/已关闭），用于活动生命周期的"结束"阶段
// POST /api/v1/health/activities/:id/status  body: {status: "active"|"closed"|"ended"}
func (h *EducationHandler) UpdateHealthActivityStatus(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
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
	res, err := h.db.Exec(`UPDATE health_activities SET status=? WHERE activity_id=?`, req.Status, c.Param("id"))
	if err != nil {
		log.Printf("health UpdateHealthActivityStatus err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新活动状态失败"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
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
	res, err := h.db.Exec(
		`UPDATE health_activity_signups SET attended=? WHERE activity_id=? AND user_id=? AND status='registered'`,
		attach, c.Param("id"), c.Param("uid"))
	if err != nil {
		log.Printf("health AttendActivitySignup err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "签到失败"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "该用户未报名或活动不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "签到状态已更新", "attended": req.Attended})
}

// ActivityReviewStats 活动复盘指标（真实统计，非 mock）
// GET /api/v1/health/activities/review-stats
// 返回：每个活动 报名/到场/到场率 + 汇总（总报名/总到场/平均到场率/分类分布/组织方排行）
func (h *EducationHandler) ActivityReviewStats(c *gin.Context) {
	type row struct {
		ActivityID  string  `json:"activity_id"`
		Title       string  `json:"title"`
		Category    string  `json:"category"`
		Venue       string  `json:"venue"`
		Organizer   string  `json:"organizer"`
		Status      string  `json:"status"`
		SignupCount int     `json:"signup_count"`
		AttendCount int     `json:"attend_count"`
		AttendRate  float64 `json:"attend_rate"`
	}
	rows, err := h.db.Query(`
		SELECT a.activity_id, a.title, a.category, a.venue, a.organizer, a.status,
		       (SELECT COUNT(*) FROM health_activity_signups s WHERE s.activity_id=a.activity_id AND s.status='registered') AS sg,
		       (SELECT COUNT(*) FROM health_activity_signups s WHERE s.activity_id=a.activity_id AND s.status='registered' AND s.attended=1) AS at
		FROM health_activities a
		ORDER BY sg DESC`)
	if err != nil {
		log.Printf("health ActivityReviewStats err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "获取复盘数据失败"})
		return
	}
	defer rows.Close()
	var items []row
	totalSignup, totalAttend := 0, 0
	catCount := map[string]int{}
	orgStat := map[string][2]int{} // organizer -> [报名, 到场]
	for rows.Next() {
		var r row
		var sg, at int
		if err := rows.Scan(&r.ActivityID, &r.Title, &r.Category, &r.Venue, &r.Organizer, &r.Status, &sg, &at); err != nil {
			continue
		}
		r.SignupCount = sg
		r.AttendCount = at
		if sg > 0 {
			r.AttendRate = math.Round(float64(at)/float64(sg)*1000) / 10
		}
		items = append(items, r)
		totalSignup += sg
		totalAttend += at
		catCount[r.Category]++
		o := r.Organizer
		if o == "" {
			o = "未知组织方"
		}
		orgStat[o] = [2]int{orgStat[o][0] + sg, orgStat[o][1] + at}
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
	rows, err := h.db.Query(`
		SELECT s.user_id, u.username, u.display_name, s.attended, s.created_at
		FROM health_activity_signups s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE s.activity_id=? AND s.status='registered'
		ORDER BY s.attended ASC, s.created_at ASC`, c.Param("id"))
	if err != nil {
		log.Printf("health ListActivitySignups err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "获取名单失败"})
		return
	}
	defer rows.Close()
	var list []gin.H
	for rows.Next() {
		var uid int
		var uname, disp string
		var att int
		var created string
		if err := rows.Scan(&uid, &uname, &disp, &att, &created); err != nil {
			continue
		}
		show := disp
		if show == "" {
			show = uname
		}
		list = append(list, gin.H{"user_id": uid, "name": show, "username": uname, "attended": att == 1, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "items": list})
}

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

	if req.Signup {
		_, err := h.db.Exec(
			dbutil.InsertIgnore(dbutil.DriverOf(h.db))+` health_activity_signups (user_id, activity_id, status) VALUES (?, ?, 'registered')`,
			userID, activityID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "报名失败"})
			return
		}
	} else {
		_, err := h.db.Exec(
			`UPDATE health_activity_signups SET status = 'cancelled' WHERE user_id = ? AND activity_id = ?`,
			userID, activityID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "取消失败"})
			return
		}
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
