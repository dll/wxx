package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/gin-gonic/gin"
)

// UserActivityStatsHandler 学生注册/登录/打卡统计 handler（任务5+7，2026-09-01）
type UserActivityStatsHandler struct {
	repo         *repository.UserActivityStatsRepo
	notification *repository.UserNotificationRepo
	userRepo     *repository.UserRepo
}

// NewUserActivityStatsHandler 创建学生活动统计 handler
func NewUserActivityStatsHandler(repo *repository.UserActivityStatsRepo, notification *repository.UserNotificationRepo, userRepo *repository.UserRepo) *UserActivityStatsHandler {
	return &UserActivityStatsHandler{repo: repo, notification: notification, userRepo: userRepo}
}

// notifyTargetUsername 统计播报默认接收人（任务7：工号 120001 胡少启）
const notifyTargetUsername = "120001"

// GetStats 获取学生注册/登录/打卡统计
// GET /api/v1/admin/stats/user-activity
func (h *UserActivityStatsHandler) GetStats(c *gin.Context) {
	scopeType, scopeID := h.resolveScope(c)
	stats, err := h.repo.GetUserActivityStats(scopeType, scopeID)
	if err != nil {
		log.Printf("查询学生活动统计失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询学生活动统计失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

// Notify 将统计摘要以站内通知推送给 120001（胡老师）或指定用户
// POST /api/v1/admin/stats/user-activity/notify  body: {"target_users":[...]}（可省略）
func (h *UserActivityStatsHandler) Notify(c *gin.Context) {
	scopeType, scopeID := h.resolveScope(c)
	stats, err := h.repo.GetUserActivityStats(scopeType, scopeID)
	if err != nil {
		log.Printf("查询学生活动统计失败（推送）: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询学生活动统计失败",
		})
		return
	}

	// 默认推送给 120001；body 可指定 target_users 覆盖
	var req struct {
		TargetUsers []int64 `json:"target_users"`
	}
	_ = c.ShouldBindJSON(&req) // body 可选

	targets := req.TargetUsers
	if len(targets) == 0 {
		user, err := h.userRepo.GetByUsername(notifyTargetUsername)
		if err != nil || user == nil {
			log.Printf("统计推送目标用户不存在: username=%s err=%v", notifyTargetUsername, err)
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "推送目标用户（" + notifyTargetUsername + "）不存在",
			})
			return
		}
		targets = []int64{user.ID}
	}

	title := "学生活动统计日报"
	content := buildUserActivityBrief(stats)
	sent := h.notification.SendBulk(title, content, targets)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"title":      title,
			"content":    content,
			"send_count": sent,
		},
	})
}

// buildUserActivityBrief 生成统计摘要文本（供通知推送复用）
func buildUserActivityBrief(s *repository.UserActivityStats) string {
	return "学生注册/登录/打卡统计（近7日）：\n" +
		"· 累计注册学生 " + strconv.Itoa(s.RegisteredTotal) + " 人，待审核 " + strconv.Itoa(s.PendingApproval) + " 人\n" +
		"· 今日新增注册 " + strconv.Itoa(s.RegisteredToday) + " 人，本月新增 " + strconv.Itoa(s.RegisteredMonth) + " 人\n" +
		"· 今日登录 " + strconv.Itoa(s.LoginTodayUsers) + " 人（" + strconv.Itoa(s.LoginTodayCount) + " 次），近7日活跃 " + strconv.Itoa(s.Login7dUsers) + " 人\n" +
		"· 今日打卡 " + strconv.Itoa(s.CheckinToday) + " 人，昨日 " + strconv.Itoa(s.CheckinYesterday) + " 人，近7日日均 " + strconv.FormatFloat(s.Checkin7dAvg, 'f', 1, 64) + " 人"
}

// RequireUserActivityStatsRead 学生活动统计查看权限（college_admin 及以上）
func RequireUserActivityStatsRead() gin.HandlerFunc {
	return auth.RequireCapability(auth.CollegeMetricsRead)
}

// resolveScope 依据当前操作者解析统计数据的范围：
//   - college_admin：仅本院（按 users.college 过滤），防止跨学院统计泄漏
//   - sys/school_admin：全校（空范围）
//
// 返回 (scopeType, scopeID)；scopeType=college 时 scopeID 为操作者学院名。
func (h *UserActivityStatsHandler) resolveScope(c *gin.Context) (string, string) {
	uc := middleware.GetUserContext(c)
	if uc == nil || uc.Role != "college_admin" {
		// 未登录或 sys/school_admin 及更高：全校数据
		return "", ""
	}
	// 解析操作者学院归属
	op, err := h.userRepo.GetByID(uc.UserID)
	if err != nil || op == nil {
		return "college", ""
	}
	college := strings.TrimSpace(op.College)
	if college == "" {
		college = strings.TrimSpace(op.OwnerID)
	}
	return "college", college
}
