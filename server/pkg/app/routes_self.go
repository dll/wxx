package app

import (
	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// registerSelfRoutes 注册问答、会话、知识浏览和情感相关路由。
func registerSelfRoutes(secured *gin.RouterGroup, d *deps) {
	secured.POST("/chat", middleware.RequireConsent(), auth.RequireCapability(auth.SelfChat), middleware.ChatUserRateLimiter(), d.chatH.Ask)
	secured.POST("/chat/stream", middleware.RequireConsent(), auth.RequireCapability(auth.SelfChat), middleware.ChatUserRateLimiter(), d.chatH.Stream)
	secured.GET("/sessions", auth.RequireCapability(auth.SelfSessionRead), d.sessionH.ListSessions)
	secured.GET("/sessions/:id/messages", auth.RequireCapability(auth.SelfSessionRead), d.sessionH.GetMessages)
	secured.DELETE("/sessions/:id", auth.RequireCapability(auth.SelfSessionDelete), d.sessionH.DeleteSession)
	secured.PATCH("/sessions/:id", auth.RequireCapability(auth.SelfSessionRead), d.sessionH.RenameSession)
	secured.GET("/knowledge", auth.RequireCapability(auth.SelfKnowledgeRead), d.kbH.BrowseKnowledge)
	secured.GET("/recommendations", auth.RequireCapability(auth.SelfRecommendRead), d.recH.GetRecommendations)
	if d.emotionH == nil {
		return
	}
	secured.GET("/emotion/stats", auth.RequireAnyCapability(auth.SelfEmotionStats, auth.SelfEmotionConsent), d.emotionH.GetStats)
	emotion := secured.Group("/emotion")
	emotion.POST("/analyze", auth.RequireCapability(auth.CounselorAlertAnalyze), d.emotionH.Analyze)
	emotion.GET("/alerts", auth.RequireCapability(auth.CounselorAlertRead), d.emotionH.ListAlerts)
	emotion.PUT("/alerts/:id", auth.RequireCapability(auth.CounselorAlertHandle), d.emotionH.UpdateAlert)
	emotion.GET("/trends", auth.RequireCapability(auth.CounselorEmotionTrends), d.emotionH.Trends)
}
