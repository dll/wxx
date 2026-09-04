package app

import (
	"github.com/dll/wxx/server/internal/auth"
	"github.com/gin-gonic/gin"
)

// registerPlatformRoutes 注册知识库、知识包同步、智能体、语音和校外系统能力路由。
func registerPlatformRoutes(secured *gin.RouterGroup, d *deps) {
	kb := secured.Group("/kb")
	kb.GET("/resources/advanced", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.ListResourcesAdvanced)
	kb.GET("/dict", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.GetDictValues)
	kb.GET("/stats", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.GetStats)
	kb.GET("/resources", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.ListResources)
	kb.POST("/resources", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.CreateResource)
	kb.PUT("/resources/:id", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.UpdateResource)
	kb.GET("/resources/:id", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.GetResource)
	kb.POST("/import", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.Import)
	kb.POST("/validate", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.Validate)
	kb.POST("/batch/approve", auth.RequireCapability(auth.CounselorKBReview), d.kbH.BatchApprove)
	kb.POST("/batch/reject", auth.RequireCapability(auth.CounselorKBReview), d.kbH.BatchReject)
	kb.POST("/batch/retire", auth.RequireCapability(auth.CounselorKBReview), d.kbH.BatchRetire)
	kb.POST("/batch/delete", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.BatchDelete)
	kb.POST("/batch/refine", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.BatchRefine)
	kb.GET("/governance", auth.RequireCapability(auth.CounselorKBReview), d.kgH.GovernanceRun)
	kb.POST("/resources/:id/approve", auth.RequireCapability(auth.CounselorKBReview), d.kbH.ApproveResource)
	kb.POST("/resources/:id/reject", auth.RequireCapability(auth.CounselorKBReview), d.kbH.RejectResource)
	kb.POST("/resources/:id/retire", auth.RequireCapability(auth.CounselorKBReview), d.kbH.RetireResource)
	kb.POST("/resources/:id/submit", auth.RequireCapability(auth.UnionKBSubmit), d.kbH.SubmitForReview)

	secured.GET("/kb/export", auth.RequireCapability(auth.SchoolKBSyncExport), d.exportH.Export)
	secured.GET("/kb/export/package", auth.RequireCapability(auth.SchoolKBSyncExport), d.exportH.ExportPackage)
	secured.POST("/kb/import/package", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), d.exportH.ImportPackage)
	secured.POST("/kb/import/package/chunk/init", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), d.exportH.InitChunkUpload)
	secured.PUT("/kb/import/package/chunk/:upload_id/:chunk_index", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), d.exportH.UploadChunk)
	secured.GET("/kb/import/package/chunk/status/:upload_id", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), d.exportH.ChunkUploadStatus)
	secured.POST("/kb/import/package/chunk/complete/:upload_id", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), d.exportH.CompleteChunkUpload)

	agents := secured.Group("/agents")
	agents.GET("", auth.RequireCapability(auth.SchoolAgentWrite), d.agentH.List)
	agents.POST("", auth.RequireCapability(auth.SchoolAgentWrite), d.agentH.Create)
	agents.GET("/:id", auth.RequireCapability(auth.SchoolAgentWrite), d.agentH.Get)
	agents.PUT("/:id", auth.RequireCapability(auth.SchoolAgentWrite), d.agentH.Update)
	agents.DELETE("/:id", auth.RequireCapability(auth.SchoolAgentWrite), d.agentH.Delete)
	secured.GET("/agents/active", d.agentH.ListActive)

	if d.voiceH != nil {
		secured.POST("/voice/asr", auth.RequireCapability(auth.SelfVoice), d.voiceH.ASR)
		secured.POST("/voice/tts", auth.RequireCapability(auth.SelfVoice), d.voiceH.TTS)
	}
	secured.GET("/export", auth.RequireCapability(auth.SchoolKBSyncExport), d.exportH.Export)
	secured.POST("/export/answer", auth.RequireCapability(auth.SelfExportSelf), d.exportH.ExportAnswer)

	integration := secured.Group("/integration")
	integration.GET("/status", auth.RequireCapability(auth.CounselorIntegrationRead), d.integrationH.Status)
	integration.GET("/xuegong/*path", auth.RequireCapability(auth.CounselorIntegrationRead), d.integrationH.ProxyXuegong)
	integration.GET("/ybt/*path", auth.RequireCapability(auth.CounselorIntegrationRead), d.integrationH.ProxyYBT)

	secured.GET("/token-stats/my", auth.RequireCapability(auth.SelfTokenStats), d.tokenStatsH.GetMyStats)
	secured.GET("/token-stats/subordinates", auth.RequireCapability(auth.CounselorTokenSubordinates), d.tokenStatsH.GetSubordinateStats)
}
