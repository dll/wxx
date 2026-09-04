package app

import (
	"github.com/dll/wxx/server/internal/auth"
	"github.com/gin-gonic/gin"
)

// registerAdminCoreRoutes 注册管理端统计、用户、审计、设置和版本管理路由。
func registerAdminCoreRoutes(secured *gin.RouterGroup, d *deps) {
	admin := secured.Group("/admin")
	admin.GET("/stats/dashboard", auth.RequireCapability(auth.CollegeMetricsRead), d.statsH.GetDashboardStats)
	admin.GET("/stats/user-activity", auth.RequireCapability(auth.CollegeMetricsRead), d.userActivityStatsH.GetStats)
	admin.POST("/stats/user-activity/notify", auth.RequireCapability(auth.CollegeMetricsRead), d.userActivityStatsH.Notify)
	admin.GET("/metrics", auth.RequireCapability(auth.CollegeMetricsRead), d.adminH.GetMetrics)
	admin.GET("/metrics/fallback-questions", auth.RequireCapability(auth.CollegeMetricsRead), d.adminH.TopFallbackQuestions)
	admin.GET("/users", auth.RequireCapability(auth.CollegeUserRead), d.adminH.ListUsers)
	admin.GET("/audit", auth.RequireCapability(auth.CollegeAuditRead), d.adminH.ListAudit)
	admin.DELETE("/audit", auth.RequireCapability(auth.CollegeAuditRead), d.adminH.DeleteAudit)
	admin.GET("/audit/snapshots", auth.RequireCapability(auth.CollegeAuditRead), d.adminH.ListSnapshots)
	admin.POST("/audit/snapshots/:id/restore", auth.RequireCapability(auth.SystemAuditAll), d.adminH.RestoreSnapshot)
	admin.PUT("/users/:id", auth.RequireCapability(auth.SchoolUserUpdate), d.adminH.UpdateUser)
	admin.DELETE("/users/:id", auth.RequireCapability(auth.SchoolUserUpdate), d.adminH.DeleteUser)
	admin.PUT("/users/:id/password", auth.RequireCapability(auth.SystemPasswordReset), d.adminH.ResetUserPassword)
	admin.GET("/users/advanced", auth.RequireCapability(auth.CollegeUserRead), d.adminH.ListUsersAdvanced)
	admin.GET("/users/dict", auth.RequireCapability(auth.CollegeUserRead), d.adminH.GetUserDict)
	admin.POST("/users/batch/status", auth.RequireCapability(auth.SchoolUserUpdate), d.adminH.BatchUpdateStatus)
	admin.POST("/users/batch/password", auth.RequireCapability(auth.SystemPasswordReset), d.adminH.BatchResetPassword)
	admin.POST("/users/batch/delete", auth.RequireCapability(auth.SchoolUserUpdate), d.adminH.BatchDelete)
	admin.GET("/settings", auth.RequireCapability(auth.SystemSettingsWrite), d.adminH.GetSettings)
	admin.PUT("/settings", auth.RequireCapability(auth.SystemSettingsWrite), d.adminH.UpdateSettings)
	admin.GET("/app-versions", auth.RequireCapability(auth.SystemSettingsWrite), d.appVersionH.ListVersions)
	admin.POST("/app-versions", auth.RequireCapability(auth.SystemSettingsWrite), d.appVersionH.CreateVersion)
	admin.PUT("/app-versions", auth.RequireCapability(auth.SystemSettingsWrite), d.appVersionH.UpdateVersion)
	admin.DELETE("/app-versions/:id", auth.RequireCapability(auth.SystemSettingsWrite), d.appVersionH.DeleteVersion)
}
