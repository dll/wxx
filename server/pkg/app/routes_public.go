package app

import (
	"github.com/dll/wxx/server/internal/handler"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// registerPublicRoutes 注册无需业务 JWT 的公开 API。
// 认证、版本、校园报到和公开知识浏览集中在本文件，避免与受保护路由混杂。
func registerPublicRoutes(v1 *gin.RouterGroup, d *deps) {
	internalRepair := v1.Group("/internal/repair-tasks")
	internalRepair.Use(middleware.RepairAgentTokenAuth())
	internalRepair.POST("/next", d.feedbackRepairTaskH.NextTask)
	internalRepair.POST("/:no/verify", d.feedbackRepairTaskH.VerifyTask)

	authGroup := v1.Group("/auth")
	authGroup.POST("/login", middleware.LoginIPRateLimiter(), d.authH.Login)
	authGroup.POST("/sso/callback", middleware.LoginIPRateLimiter(), d.authH.SSOCallback)
	authGroup.POST("/qr-login", handler.CreateQRSession)
	authGroup.GET("/qr-status", handler.GetQRSessionStatus)
	authGroup.PUT("/qr-scan", handler.ScanQRSession)
	authGroup.POST("/send-code", middleware.LoginIPRateLimiter(), d.authH.SendCode)
	authGroup.POST("/guest-register", middleware.LoginIPRateLimiter(), d.authH.GuestRegister)

	v1.GET("/version/check", d.appVersionH.CheckUpdate)
	v1.GET("/version/latest", d.appVersionH.GetLatestVersion)
	v1.GET("/campus/steps", d.campusH.ListPublicSteps)
	v1.GET("/knowledge/public", d.kbH.BrowseKnowledgePublic)
	v1.GET("/public/feature-switches", d.adminH.GetPublicFeatureSwitches)
}
