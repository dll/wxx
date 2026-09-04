package app

import (
	"github.com/dll/wxx/server/internal/handler"
	"github.com/gin-gonic/gin"
)

// registerUserRoutes 注册当前用户资料、凭证、偏好与安全设置路由。
func registerUserRoutes(secured *gin.RouterGroup, d *deps) {
	secured.GET("/user/profile", d.authH.Profile)
	secured.GET("/user/profile/detail", d.authH.ProfileDetail)
	secured.POST("/user/switch-role", d.authH.SwitchRole)
	secured.GET("/user/ai-key", d.authH.GetAIKey)
	secured.PUT("/user/ai-key", d.authH.SaveAIKey)
	secured.DELETE("/user/ai-key", d.authH.ClearAIKey)
	secured.GET("/user/logs", d.adminH.MyLogs)
	secured.DELETE("/user/logs/:id", d.adminH.DeleteMyLog)
	secured.GET("/user/portal-credential", d.portalCredH.Get)
	secured.PUT("/user/portal-credential", d.portalCredH.Save)
	secured.DELETE("/user/portal-credential", d.portalCredH.Delete)
	secured.GET("/user/portal/*path", d.portalProxyH.Proxy)
	secured.GET("/user/portal", d.portalProxyH.Proxy)
	secured.POST("/auth/qr-confirm", handler.ConfirmQRSession)
	secured.POST("/user/consent", d.authH.Consent)
	secured.PUT("/user/password", d.authH.ChangePassword)
	secured.GET("/user/voice-config", d.authH.GetVoiceConfig)
	secured.PUT("/user/voice-config", d.authH.UpdateVoiceConfig)
	secured.GET("/user/capabilities", d.authH.GetCapabilities)
	secured.GET("/user/model-config", d.modelConfigH.Get)
	secured.PUT("/user/model-config", d.modelConfigH.Save)
}
