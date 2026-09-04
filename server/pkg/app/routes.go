package app

// Gin 路由树构建 setupRouter（从原 app.go 拆分，行为不变）

import (
	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/handler"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// setupRouter 构建 Gin 路由树
//
// 依赖已收敛为 deps 结构体（见 deps.go）：cfg/db/userRepo 与 45 个 handler 全部由
// app.go 装配层构造后打包传入，行为不变、仅结构重构。
func setupRouter(d *deps) *gin.Engine {
	router := gin.New()

	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(middleware.CORSWithConfig(d.cfg.CORSAllowedOrigins, d.cfg.IsRelease()))
	router.Use(middleware.GlobalIPRateLimiter()) // 全局限流（IP 维度，100 req/min/IP）
	router.Use(middleware.TraceID())
	router.Use(middleware.PIIMask()) // PII 检测与脱敏（在请求进入 handler 前检测并脱敏）
	router.Use(gin.Logger())
	router.Use(middleware.AuditLog(d.db))

	// 静态资源实现位于 routes_static.go 的 registerBaseRoutes：
	// router.Static("/assets", ...)、router.Static("/canvaskit", ...)，
	// 并在 NoRoute 中豁免 !strings.HasPrefix(c.Request.URL.Path, "/api/")
	// 与 !strings.HasPrefix(c.Request.URL.Path, "/health")。
	// 这些标记同时供回归测试确认静态服务与 API 注册顺序契约。
	// NoCache 入口：/main.dart.js、/index.html、/flutter_bootstrap.js、
	// /flutter_service_worker.js。
	registerBaseRoutes(router, d.cfg, healthHandler(d.db))

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// 公开路由集中在 routes_public.go；受保护路由继续按领域注册。
		registerPublicRoutes(v1, d)

		// 需要 JWT 认证
		secured := v1.Group("/")
		secured.Use(middleware.JWTAuth(d.cfg))
		secured.Use(middleware.EnsureUserExists(d.userRepo))
		{
			registerVOPCRoutes(secured, d)
			registerSelfRoutes(secured, d)
			registerForecastRoutes(secured, d)
			registerGraduationRoutes(secured, d)
			registerStudentFeatureRoutes(secured, d)
			/*
				// vOPC：学院准入、系统 capability 与项目角色三层边界缺一不可。
				vopc := secured.Group("/vopc")
				vopc.Use(handler.CollegeAccess(d.cfg.VOPCCollegeID))
				{
					vopc.GET("/access", d.vopcH.AccessStatus)
					vopc.GET("/learning", d.vopcH.Learning)
					vopc.GET("/guides", d.vopcH.Guides)
					vopc.GET("/users/search", d.vopcH.SearchUsers)
					vopc.GET("/projects", d.vopcH.ListProjects)
					vopc.POST("/projects", auth.RequireCapability(auth.VOPCProjectCreate), d.vopcH.CreateProject)
					vopc.POST("/demo-projects", auth.RequireCapability(auth.VOPCProjectCreate), d.vopcH.CreateDemoProject)
					vopc.POST("/projects/:id/simulation/advance", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.AdvanceSimulation)
					vopc.GET("/projects/:id", d.vopcH.GetProject)
					vopc.PUT("/projects/:id", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.UpdateProject)
					vopc.GET("/projects/:id/tasks", d.vopcH.ListTasks)
					vopc.GET("/projects/:id/decisions", d.vopcH.ListDecisions)
					vopc.POST("/projects/:id/decisions", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.CreateDecision)
					vopc.PUT("/projects/:id/decisions/:decisionId", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.ActDecision)
					vopc.GET("/projects/:id/members", d.vopcH.ListMembers)
					vopc.POST("/projects/:id/members", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.InviteMember)
					vopc.GET("/invitations", auth.RequireCapability(auth.VOPCProjectJoin), d.vopcH.ListMyInvitations)
					vopc.POST("/invitations/:invitationId/respond", auth.RequireCapability(auth.VOPCProjectJoin), d.vopcH.RespondInvitation)
					vopc.GET("/projects/:id/artifacts", d.vopcH.ListArtifacts)
					vopc.POST("/projects/:id/artifacts", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.CreateArtifact)
					vopc.GET("/projects/:id/artifacts/:artifactId/versions", d.vopcH.ListArtifactVersions)
					vopc.POST("/projects/:id/artifacts/:artifactId/versions", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.CreateArtifactVersion)
					// vOPC 私有文件受控上传与鉴权下载：上传限项目写权限，下载走项目读权限 + 学院准入复检。
					vopc.POST("/projects/:id/files", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.UploadFile)
					vopc.GET("/projects/:id/files/:key", d.vopcH.DownloadFile)
					vopc.GET("/projects/:id/milestone-submissions", d.vopcH.ListMilestoneSubmissions)
					vopc.POST("/projects/:id/milestone-submissions", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.SubmitMilestone)
					vopc.GET("/projects/:id/ai-roles", d.vopcH.ListAIRoles)
					vopc.GET("/projects/:id/timeline", d.vopcH.ListTimeline)
					vopc.POST("/projects/:id/milestone-submissions/:submissionId/review", auth.RequireCapability(auth.VOPCMilestoneReview), d.vopcH.ReviewMilestone)
					// vOPC A4 里程碑完整业务门禁：评分量表 / 条件闭环 / 豁免 / 甲方结构化证据
					vopc.GET("/projects/:id/rubrics", d.vopcH.ListRubrics)
					vopc.GET("/projects/:id/milestone-submissions/:submissionId/review", d.vopcH.GetSubmissionReview)
					vopc.PUT("/projects/:id/milestone-submissions/:submissionId/conditions/:conditionId", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.MarkConditionSatisfied)
					vopc.POST("/projects/:id/milestone-submissions/:submissionId/finalize", auth.RequireCapability(auth.VOPCMilestoneReview), d.vopcH.FinalizeMilestone)
					vopc.GET("/projects/:id/milestone-waivers", d.vopcH.ListMilestoneWaivers)
					vopc.POST("/projects/:id/milestone-waivers", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.CreateMilestoneWaiver)
					vopc.POST("/projects/:id/milestone-waivers/:waiverId/review", auth.RequireAnyCapability(auth.VOPCMentorReview, auth.VOPCMilestoneReview, auth.VOPCRiskManage), d.vopcH.ReviewMilestoneWaiver)
					vopc.GET("/projects/:id/client-evidence", d.vopcH.ListClientEvidence)
					vopc.POST("/projects/:id/client-evidence", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.CreateClientEvidence)
					vopc.PUT("/projects/:id/client-evidence/:evidenceId", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.UpdateClientEvidence)
					// vOPC B1 AI 任务真实执行闭环
					vopc.POST("/projects/:id/ai-tasks", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.CreateAITask)
					vopc.GET("/projects/:id/ai-tasks", d.vopcH.ListAITasks)
					vopc.GET("/projects/:id/ai-tasks/:taskId", d.vopcH.GetAITask)
					vopc.POST("/projects/:id/ai-tasks/:taskId/review", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.ReviewAITask)
					vopc.POST("/projects/:id/ai-tasks/:taskId/retry", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.RetryAITask)
					vopc.POST("/projects/:id/tasks", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.CreateTask)
					vopc.PUT("/projects/:id/tasks/:taskId", d.vopcH.UpdateTask)
					vopc.POST("/projects/:id/submit", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.SubmitProject)

					// 结项与异常状态机
					vopc.POST("/projects/:id/close", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.CloseProject)
					vopc.GET("/projects/:id/close-records", d.vopcH.ListCloseRecords)
					// 项目生命周期：删除（仅草稿）——归档走既有 close archive 动作。
					vopc.POST("/projects/:id/delete", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.DeleteProject)

					// 风险治理
					vopc.GET("/projects/:id/risks", d.vopcH.ListRisks)
					vopc.POST("/projects/:id/risks", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.CreateRisk)
					vopc.POST("/projects/:id/risks/:riskId/approve", auth.RequireAnyCapability(auth.VOPCRiskManage, auth.VOPCMentorReview), d.vopcH.ApproveRisk)
					vopc.GET("/projects/:id/special-approvals", d.vopcH.ListSpecialApprovals)
					vopc.POST("/projects/:id/special-approvals", auth.RequireCapability(auth.VOPCRiskManage), d.vopcH.CreateSpecialApproval)
					vopc.POST("/projects/:id/freeze", auth.RequireCapability(auth.VOPCRiskManage), d.vopcH.FreezeProject)
					vopc.POST("/projects/:id/risk-appeals", auth.RequireCapability(auth.VOPCProjectManage), d.vopcH.CreateRiskAppeal)
					vopc.POST("/projects/:id/risk-appeals/:appealId/resolve", auth.RequireCapability(auth.VOPCRiskManage), d.vopcH.ResolveRiskAppeal)

					// 治理角色受控授予/撤销：仅平台治理系统角色（college_admin/school_admin/sys_admin）可调用。
					vopc.POST("/projects/:id/governance-roles", auth.RequireAnyCapability(auth.VOPCRiskManage, auth.VOPCAudit), d.vopcH.GrantGovernanceRole)
				}
			*/

			// ── AI 对话（self.chat）──
			// 安全修复 SEC-02：对话为主要 PII 输入入口，要求已同意隐私政策/用户协议方可访问
			/* secured.POST("/chat", middleware.RequireConsent(), auth.RequireCapability(auth.SelfChat), middleware.ChatUserRateLimiter(), d.chatH.Ask)
			secured.POST("/chat/stream", middleware.RequireConsent(), auth.RequireCapability(auth.SelfChat), middleware.ChatUserRateLimiter(), d.chatH.Stream)

			// ── 会话/知识/推荐（self.* 能力）──
			secured.GET("/sessions", auth.RequireCapability(auth.SelfSessionRead), d.sessionH.ListSessions)
			secured.GET("/sessions/:id/messages", auth.RequireCapability(auth.SelfSessionRead), d.sessionH.GetMessages)
			secured.DELETE("/sessions/:id", auth.RequireCapability(auth.SelfSessionDelete), d.sessionH.DeleteSession)
			secured.PATCH("/sessions/:id", auth.RequireCapability(auth.SelfSessionRead), d.sessionH.RenameSession)
			secured.GET("/knowledge", auth.RequireCapability(auth.SelfKnowledgeRead), d.kbH.BrowseKnowledge)
			secured.GET("/recommendations", auth.RequireCapability(auth.SelfRecommendRead), d.recH.GetRecommendations) */

			// ── 情感数据 ──
			/* if d.emotionH != nil {
				// 自身情感统计：所有用户都可看自己。
				// 独立授权语义：需同时拥有 self.emotion.consent（独立于通用隐私 consent）
				secured.GET("/emotion/stats", auth.RequireAnyCapability(auth.SelfEmotionStats, auth.SelfEmotionConsent), d.emotionH.GetStats)
			} */

			/* if d.emotionH != nil {
				emotion := secured.Group("/emotion")
				{
					emotion.POST("/analyze", auth.RequireCapability(auth.CounselorAlertAnalyze), d.emotionH.Analyze)
					emotion.GET("/alerts", auth.RequireCapability(auth.CounselorAlertRead), d.emotionH.ListAlerts)
					emotion.PUT("/alerts/:id", auth.RequireCapability(auth.CounselorAlertHandle), d.emotionH.UpdateAlert)
					emotion.GET("/trends", auth.RequireCapability(auth.CounselorEmotionTrends), d.emotionH.Trends)
				}
			} */

			/* ── 问题预案（forecast.*）──
			forecast := secured.Group("/forecast")
			{
				forecast.POST("/analysis", auth.RequireCapability(auth.CollegeForecast), d.forecastH.Analyze)
				forecast.GET("/issues", auth.RequireCapability(auth.CollegeForecast), d.forecastH.ListForecasts)
				forecast.GET("/issues/:id", auth.RequireCapability(auth.CollegeForecast), d.forecastH.GetForecast)
				forecast.PUT("/issues/:id/status", auth.RequireCapability(auth.CollegeForecast), d.forecastH.UpdateStatus)
				forecast.GET("/statistics", auth.RequireCapability(auth.CollegeForecast), d.forecastH.GetStatistics)
			} */

			/* ── 毕设选题（graduation.*）──
			graduation := secured.Group("/graduation")
			{
				graduation.GET("/advisors", auth.RequireCapability(auth.SelfGraduationRead), d.graduationH.ListAdvisors)
				graduation.GET("/topics", auth.RequireCapability(auth.SelfGraduationRead), d.graduationH.ListTopics)
				graduation.GET("/topics/:id", auth.RequireCapability(auth.SelfGraduationRead), d.graduationH.GetTopic)
				graduation.POST("/select", auth.RequireCapability(auth.SelfGraduationWrite), d.graduationH.SelectTopic)
				graduation.GET("/my-selection", auth.RequireCapability(auth.SelfGraduationRead), d.graduationH.GetMySelection)
				graduation.GET("/milestones", auth.RequireCapability(auth.SelfGraduationRead), d.graduationH.ListMilestones)
				graduation.GET("/stats", auth.RequireCapability(auth.SelfGraduationRead), d.graduationH.GetStats)
				graduation.GET("/selections", auth.RequireCapability(auth.CollegeGraduationRead), d.graduationH.ListSelections)
				graduation.PUT("/selections/:id/confirm", auth.RequireCapability(auth.CollegeGraduationWrite), d.graduationH.ConfirmSelection)
				// ── 毕设选题管理（学院管理员）──
				graduation.POST("/admin/topics", auth.RequireCapability(auth.CollegeGraduationWrite), d.graduationH.CreateTopic)
				graduation.PUT("/admin/topics/:id", auth.RequireCapability(auth.CollegeGraduationWrite), d.graduationH.UpdateTopic)
				graduation.DELETE("/admin/topics/:id", auth.RequireCapability(auth.CollegeGraduationWrite), d.graduationH.DeleteTopic)
			} */

			/* ── 学科竞赛 ──
			competition := secured.Group("/competition")
			{
				competition.GET("/list", auth.RequireCapability(auth.SelfCompetitionRead), d.studentFeaturesH.ListCompetitions)
				competition.GET("/match", auth.RequireCapability(auth.SelfCompetitionRead), d.studentFeaturesH.CompetitionMatch)
				competition.GET("/:id", auth.RequireCapability(auth.SelfCompetitionRead), d.studentFeaturesH.GetCompetition)
				competition.POST("/register", auth.RequireCapability(auth.SelfCompetitionWrite), d.studentFeaturesH.RegisterCompetition)
				competition.GET("/my-registrations", auth.RequireCapability(auth.SelfCompetitionRead), d.studentFeaturesH.GetMyCompetitionRegistrations)
				competition.POST("/submit-work", auth.RequireCapability(auth.SelfCompetitionWrite), d.studentFeaturesH.SubmitWork)
				competition.GET("/stats", auth.RequireCapability(auth.SelfCompetitionRead), d.studentFeaturesH.GetCompetitionStats)
				// ── 竞赛管理（学校/学院管理员）──
				competition.GET("/admin/list", auth.RequireCapability(auth.SystemSettingsWrite), d.studentFeaturesH.AdminListCompetitions)
				competition.POST("/admin", auth.RequireCapability(auth.SystemSettingsWrite), d.studentFeaturesH.AdminCreateCompetition)
				competition.PUT("/admin/:id", auth.RequireCapability(auth.SystemSettingsWrite), d.studentFeaturesH.AdminUpdateCompetition)
				competition.DELETE("/admin/:id", auth.RequireCapability(auth.SystemSettingsWrite), d.studentFeaturesH.AdminDeleteCompetition)
			}

			// ── 大学规划 ──
			plan := secured.Group("/plan")
			{
				plan.GET("/templates", auth.RequireCapability(auth.SelfPlanRead), d.studentFeaturesH.ListPlanTemplates)
				plan.GET("/my-plans", auth.RequireCapability(auth.SelfPlanRead), d.studentFeaturesH.ListMyPlans)
				plan.POST("/create", auth.RequireCapability(auth.SelfPlanWrite), d.studentFeaturesH.CreatePlan)
				plan.PUT("/:id/submit", auth.RequireCapability(auth.SelfPlanWrite), d.studentFeaturesH.SubmitPlan)
				plan.PUT("/:id/review", auth.RequireCapability(auth.CounselorKBWrite), d.studentFeaturesH.ReviewPlan)
			}

			// ── 入党教育 ──
			party := secured.Group("/party")
			{
				party.GET("/stages", auth.RequireCapability(auth.SelfPartyRead), d.studentFeaturesH.ListPartyStages)
				party.GET("/my-progress", auth.RequireCapability(auth.SelfPartyRead), d.studentFeaturesH.GetMyPartyProgress)
				party.PUT("/my-progress", auth.RequireCapability(auth.SelfPartyWrite), d.studentFeaturesH.UpdatePartyProgress)
				party.GET("/my-study-records", auth.RequireCapability(auth.SelfPartyRead), d.studentFeaturesH.ListMyStudyRecords)
				party.POST("/study-record", auth.RequireCapability(auth.SelfPartyWrite), d.studentFeaturesH.AddStudyRecord)
				party.GET("/stats", auth.RequireCapability(auth.SelfPartyRead), d.studentFeaturesH.GetPartyStats)
			}

			// ── 社团生活 ──
			club := secured.Group("/club")
			{
				club.GET("/list", auth.RequireCapability(auth.SelfClubRead), d.studentFeaturesH.ListClubs)
				club.GET("/:id", auth.RequireCapability(auth.SelfClubRead), d.studentFeaturesH.GetClub)
				club.POST("/join", auth.RequireCapability(auth.SelfClubWrite), d.studentFeaturesH.JoinClub)
				club.GET("/my-clubs", auth.RequireCapability(auth.SelfClubRead), d.studentFeaturesH.GetMyClubs)
				club.GET("/activities", auth.RequireCapability(auth.SelfClubRead), d.studentFeaturesH.ListClubActivities)
				club.POST("/activity/register", auth.RequireCapability(auth.SelfClubWrite), d.studentFeaturesH.RegisterClubActivity)
			} */

			// ── 知识库 CRUD（counselor.kb.write）──
			kb := secured.Group("/kb")
			{
				// 高级查询与字典（必须在 /resources/:id 之前注册，避免 "advanced" 被匹配为 :id）
				kb.GET("/resources/advanced", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.ListResourcesAdvanced)
				kb.GET("/dict", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.GetDictValues)
				kb.GET("/stats", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.GetStats)

				kb.GET("/resources", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.ListResources)
				kb.POST("/resources", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.CreateResource)
				kb.PUT("/resources/:id", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.UpdateResource)
				kb.GET("/resources/:id", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.GetResource)
				kb.POST("/import", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.Import)
				kb.POST("/validate", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.Validate)

				// 批量操作（counselor.kb.review）
				kb.POST("/batch/approve", auth.RequireCapability(auth.CounselorKBReview), d.kbH.BatchApprove)
				kb.POST("/batch/reject", auth.RequireCapability(auth.CounselorKBReview), d.kbH.BatchReject)
				kb.POST("/batch/retire", auth.RequireCapability(auth.CounselorKBReview), d.kbH.BatchRetire)
				kb.POST("/batch/delete", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.BatchDelete)
				kb.POST("/batch/refine", auth.RequireCapability(auth.CounselorKBWrite), d.kbH.BatchRefine)

				// 知识治理智能体（counselor.kb.review）：确定性检查 + LLM 准确性审计报告
				kb.GET("/governance", auth.RequireCapability(auth.CounselorKBReview), d.kgH.GovernanceRun)

				// 知识审核（counselor.kb.review）
				kb.POST("/resources/:id/approve", auth.RequireCapability(auth.CounselorKBReview), d.kbH.ApproveResource)
				kb.POST("/resources/:id/reject", auth.RequireCapability(auth.CounselorKBReview), d.kbH.RejectResource)
				kb.POST("/resources/:id/retire", auth.RequireCapability(auth.CounselorKBReview), d.kbH.RetireResource)

				// 知识提交（union.kb.submit，student_union 起）
				kb.POST("/resources/:id/submit", auth.RequireCapability(auth.UnionKBSubmit), d.kbH.SubmitForReview)
			}

			// ── 知识同步导出（school.kb.sync.export，学校级运维）──
			// 安全修复 RB-01：知识全量导出不再对所有登录用户开放，仅学校级同步能力可用，并在服务层按 scope 过滤
			secured.GET("/kb/export", auth.RequireCapability(auth.SchoolKBSyncExport), d.exportH.Export)
			secured.GET("/kb/export/package", auth.RequireCapability(auth.SchoolKBSyncExport), d.exportH.ExportPackage)
			secured.POST("/kb/import/package", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), d.exportH.ImportPackage)
			secured.POST("/kb/import/package/chunk/init", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), d.exportH.InitChunkUpload)
			secured.PUT("/kb/import/package/chunk/:upload_id/:chunk_index", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), d.exportH.UploadChunk)
			secured.GET("/kb/import/package/chunk/status/:upload_id", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), d.exportH.ChunkUploadStatus)
			secured.POST("/kb/import/package/chunk/complete/:upload_id", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), d.exportH.CompleteChunkUpload)

			// ── 智能体管理（school.agent.write）──
			agents := secured.Group("/agents")
			{
				agents.GET("", auth.RequireCapability(auth.SchoolAgentWrite), d.agentH.List)
				agents.POST("", auth.RequireCapability(auth.SchoolAgentWrite), d.agentH.Create)
				agents.GET("/:id", auth.RequireCapability(auth.SchoolAgentWrite), d.agentH.Get)
				agents.PUT("/:id", auth.RequireCapability(auth.SchoolAgentWrite), d.agentH.Update)
				agents.DELETE("/:id", auth.RequireCapability(auth.SchoolAgentWrite), d.agentH.Delete)
			}

			// 已启用智能体列表（对话页选择器用，普通登录用户即可访问）
			secured.GET("/agents/active", d.agentH.ListActive)

			// ── 语音 ASR/TTS（self.voice）──
			if d.voiceH != nil {
				secured.POST("/voice/asr", auth.RequireCapability(auth.SelfVoice), d.voiceH.ASR)
				secured.POST("/voice/tts", auth.RequireCapability(auth.SelfVoice), d.voiceH.TTS)
			}

			// ── 知识资源导出（school.kb.sync.export，同 /kb/export，学校级同步）──
			// 安全修复 RB-01：与 /kb/export 同一 handler，统一收敛到学校级同步能力
			secured.GET("/export", auth.RequireCapability(auth.SchoolKBSyncExport), d.exportH.Export)
			// 导出本人回答卡片（self.export.self）——仅处理调用者自行提交的卡片数据，无越权风险
			secured.POST("/export/answer", auth.RequireCapability(auth.SelfExportSelf), d.exportH.ExportAnswer)

			// ── 校外系统对接（counselor.integration.read）──
			integration := secured.Group("/integration")
			{
				integration.GET("/status", auth.RequireCapability(auth.CounselorIntegrationRead), d.integrationH.Status)
				integration.GET("/xuegong/*path", auth.RequireCapability(auth.CounselorIntegrationRead), d.integrationH.ProxyXuegong)
				integration.GET("/ybt/*path", auth.RequireCapability(auth.CounselorIntegrationRead), d.integrationH.ProxyYBT)
			}

			secured.GET("/user/profile", d.authH.Profile)
			secured.GET("/user/profile/detail", d.authH.ProfileDetail)
			secured.POST("/user/switch-role", d.authH.SwitchRole)
			secured.GET("/user/ai-key", d.authH.GetAIKey)
			secured.PUT("/user/ai-key", d.authH.SaveAIKey)
			secured.DELETE("/user/ai-key", d.authH.ClearAIKey)
			// 我的操作日志（普通用户查看自己的日志）
			secured.GET("/user/logs", d.adminH.MyLogs)
			secured.DELETE("/user/logs/:id", d.adminH.DeleteMyLog)
			// 学校门户凭证（加密存储，密码不回显）
			secured.GET("/user/portal-credential", d.portalCredH.Get)
			secured.PUT("/user/portal-credential", d.portalCredH.Save)
			secured.DELETE("/user/portal-credential", d.portalCredH.Delete)
			// 学校门户页面代理（持用户门户凭证登录后访问校内各级网站）
			secured.GET("/user/portal/*path", d.portalProxyH.Proxy)
			secured.GET("/user/portal", d.portalProxyH.Proxy)
			secured.POST("/auth/qr-confirm", handler.ConfirmQRSession)
			secured.POST("/user/consent", d.authH.Consent)
			secured.PUT("/user/password", d.authH.ChangePassword)

			// 语音配置
			secured.GET("/user/voice-config", d.authH.GetVoiceConfig)
			secured.PUT("/user/voice-config", d.authH.UpdateVoiceConfig)

			// 当前用户能力清单（基于角色继承自动展开）
			secured.GET("/user/capabilities", d.authH.GetCapabilities)

			// AI 模型配置
			secured.GET("/user/model-config", d.modelConfigH.Get)
			secured.PUT("/user/model-config", d.modelConfigH.Save)

			// ── 词元统计 ──
			secured.GET("/token-stats/my", auth.RequireCapability(auth.SelfTokenStats), d.tokenStatsH.GetMyStats)
			secured.GET("/token-stats/subordinates", auth.RequireCapability(auth.CounselorTokenSubordinates), d.tokenStatsH.GetSubordinateStats)

			// ── 管理端 ──
			admin := secured.Group("/admin")
			{
				admin.GET("/stats/dashboard", auth.RequireCapability(auth.CollegeMetricsRead), d.statsH.GetDashboardStats)
				// 学生活动统计（任务5，2026-09-01）：注册/登录/打卡聚合
				admin.GET("/stats/user-activity", auth.RequireCapability(auth.CollegeMetricsRead), d.userActivityStatsH.GetStats)
				// 统计播报（任务7）：站内通知推送给 120001（默认）或指定用户
				admin.POST("/stats/user-activity/notify", auth.RequireCapability(auth.CollegeMetricsRead), d.userActivityStatsH.Notify)
				admin.GET("/metrics", auth.RequireCapability(auth.CollegeMetricsRead), d.adminH.GetMetrics)
				admin.GET("/metrics/fallback-questions", auth.RequireCapability(auth.CollegeMetricsRead), d.adminH.TopFallbackQuestions)
				admin.GET("/users", auth.RequireCapability(auth.CollegeUserRead), d.adminH.ListUsers)
				admin.GET("/audit", auth.RequireCapability(auth.CollegeAuditRead), d.adminH.ListAudit)
				admin.DELETE("/audit", auth.RequireCapability(auth.CollegeAuditRead), d.adminH.DeleteAudit)
				// 审计恢复快照（college_admin+ 可查看，sys_admin 可恢复）
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
				// 应用版本管理（sys_admin）
				admin.GET("/app-versions", auth.RequireCapability(auth.SystemSettingsWrite), d.appVersionH.ListVersions)
				admin.POST("/app-versions", auth.RequireCapability(auth.SystemSettingsWrite), d.appVersionH.CreateVersion)
				admin.PUT("/app-versions", auth.RequireCapability(auth.SystemSettingsWrite), d.appVersionH.UpdateVersion)
				admin.DELETE("/app-versions/:id", auth.RequireCapability(auth.SystemSettingsWrite), d.appVersionH.DeleteVersion)

				// 第三方应用管理（sys_admin）
				admin.GET("/apps", auth.RequireCapability(auth.SystemSettingsWrite), d.externalAppH.ListAdmin)
				admin.POST("/apps", auth.RequireCapability(auth.SystemSettingsWrite), d.externalAppH.Create)
				admin.PUT("/apps/:id", auth.RequireCapability(auth.SystemSettingsWrite), d.externalAppH.Update)
				admin.DELETE("/apps/:id", auth.RequireCapability(auth.SystemSettingsWrite), d.externalAppH.Delete)
				// 应用中心（登录用户可见，按角色过滤）
				secured.GET("/apps", d.externalAppH.ListVisible)

				// ── AI 简讯（sys_admin 管理）──
				admin.GET("/ai-briefings", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.List)
				admin.POST("/ai-briefings", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.Create)
				admin.PUT("/ai-briefings/:id", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.Update)
				admin.PUT("/ai-briefings/:id/status", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.UpdateStatus)
				admin.DELETE("/ai-briefings/:id", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.Delete)
				admin.POST("/ai-briefings/batch-delete", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.DeleteMany)
				admin.DELETE("/ai-briefings/clear", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.ClearAll)
				admin.GET("/ai-briefings/stats", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.Stats)
				admin.POST("/ai-briefings/export", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.Export)
				admin.POST("/ai-briefings/fetch", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.FetchNow)
				admin.GET("/ai-briefings/sources", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.ListSources)
				admin.POST("/ai-briefings/sources", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.CreateSource)
				admin.PUT("/ai-briefings/sources/:id", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.UpdateSource)
				admin.DELETE("/ai-briefings/sources/:id", auth.RequireCapability(auth.SystemAIBriefing), d.aiBriefingH.DeleteSource)
				// AI 简讯（登录用户可见）
				secured.GET("/ai-briefings", d.aiBriefingH.ListUser)
				secured.GET("/ai-briefings/hot", d.aiBriefingH.ListUserHot)
				secured.GET("/ai-briefings/favorites", d.aiBriefingH.ListFavorites)
				secured.POST("/ai-briefings/:id/favorite", d.aiBriefingH.Favorite)
				secured.DELETE("/ai-briefings/:id/favorite", d.aiBriefingH.Unfavorite)

				// 数字孪生画像（登录用户可见，文生图/图生图）
				secured.GET("/twin-portraits", auth.RequireCapability(auth.SelfTwinRead), d.twinPortraitH.List)
				secured.GET("/twin-portraits/:type", auth.RequireCapability(auth.SelfTwinRead), d.twinPortraitH.Get)
				secured.POST("/twin-portraits/generate", auth.RequireCapability(auth.SelfTwinWrite), d.twinPortraitH.Generate)

				// 游客管理（college_admin+）
				admin.GET("/guests/pending", auth.RequireCapability(auth.CollegeUserRead), d.adminH.ListPendingGuests)
				admin.PUT("/guests/:id/approve", auth.RequireCapability(auth.CollegeUserRead), d.adminH.ApproveGuest)
				admin.PUT("/guests/:id/reject", auth.RequireCapability(auth.CollegeUserRead), d.adminH.RejectGuest)
				// 学生导入（除学生和游客外的组织角色均可用）
				admin.POST("/users/import", auth.RequireCapability(auth.CounselorImportStudent), d.adminH.ImportStudents)
				// ── 数据底座导入（成绩/课表，college_admin+）──
				admin.POST("/grades/import", auth.RequireCapability(auth.CollegeUserRead), d.dataImportH.ImportGrades)
				admin.POST("/schedules/import", auth.RequireCapability(auth.BatchScheduleImport), d.dataImportH.ImportSchedules)
				// 按工号归位课表（彻底修复历史错挂课程显示）
				admin.POST("/schedules/reassign", auth.RequireCapability(auth.BatchScheduleImport), d.dataImportH.ReassignSchedules)

				// ── 校园报到步骤管理（college_admin+）──
				campusAdmin := admin.Group("/campus")
				{
					campusAdmin.GET("/steps", auth.RequireCapability(auth.CollegeUserRead), d.campusH.ListAdminSteps)
					campusAdmin.POST("/steps", auth.RequireCapability(auth.CollegeUserRead), d.campusH.CreateStep)
					campusAdmin.PUT("/steps/:id", auth.RequireCapability(auth.CollegeUserRead), d.campusH.UpdateStep)
					campusAdmin.POST("/steps/:id/submit", auth.RequireCapability(auth.CollegeUserRead), d.campusH.SubmitStep)
					campusAdmin.POST("/steps/:id/publish", auth.RequireCapability(auth.CollegeDataAnalysis), d.campusH.PublishStep)
					// 管理员拖拽校正坐标（college_admin+，不走审核流程，已发布步骤也可调整）
					campusAdmin.PATCH("/steps/:id/coords", auth.RequireCapability(auth.CollegeUserRead), d.campusH.UpdateStepCoords)
					campusAdmin.DELETE("/steps/:id", auth.RequireCapability(auth.CollegeUserRead), d.campusH.DeleteStep)
					// 管理员强制更新/删除（不限状态，已发布步骤也可直接修改内容或删除）
					campusAdmin.PUT("/steps/:id/force", auth.RequireCapability(auth.CollegeUserRead), d.campusH.UpdateStepForce)
					campusAdmin.DELETE("/steps/:id/force", auth.RequireCapability(auth.CollegeUserRead), d.campusH.DeleteStepForce)
				}
			}

			// ── 知识审核 ──
			review := secured.Group("/review")
			{
				review.GET("/pending", auth.RequireCapability(auth.CounselorReviewPending), d.kbH.ListPendingReviews)
			}

			// ── 反馈 ──
			secured.POST("/feedback", auth.RequireCapability(auth.SelfFeedbackSubmit), d.feedbackH.Submit)
			secured.POST("/feedback/screenshot", auth.RequireCapability(auth.SelfFeedbackSubmit), d.feedbackH.UploadScreenshot)
			// 我的反馈：所有登录用户都能查看自己提交的反馈（按 user_id 过滤）
			secured.GET("/feedback/mine", auth.RequireCapability(auth.SelfFeedbackSubmit), d.feedbackH.Mine)
			// 反馈详情（所有登录用户可查自己的，管理端可查所有；权限由 handler 内 user_id 校验）
			secured.GET("/feedback/:id", d.feedbackH.Get)
			// 反馈满意度评价（用户对已解决的反馈打分）
			secured.PUT("/feedback/:id/rate", auth.RequireCapability(auth.SelfFeedbackSubmit), d.feedbackH.Rate)
			// 反馈处理记录
			secured.GET("/feedback/:id/logs", d.feedbackH.GetLogs)
			// 管理员反馈列表和处理（注意：直接注册避免 Group 产生的尾部斜杠重定向丢 Authorization 头）
			secured.GET("/feedback", auth.RequireCapability(auth.UnionFeedbackList), d.feedbackH.List)
			secured.PUT("/feedback/:id", auth.RequireCapability(auth.UnionFeedbackList), d.feedbackH.Resolve)
			secured.POST("/feedback/:id/ai-repair", auth.RequireCapability(auth.UnionFeedbackWrite), d.feedbackH.AIRepair)
			// 修复工单轮询/审计（管理端）
			secured.GET("/feedback/:id/ai-repair/job", auth.RequireCapability(auth.UnionFeedbackWrite), d.feedbackH.LatestRepairJob)
			// 管理端反馈统计
			secured.GET("/admin/feedback/stats", auth.RequireCapability(auth.UnionFeedbackRead), d.feedbackH.Stats)
			// 管理端关联知识资源
			secured.PUT("/admin/feedback/:id/link-resource", auth.RequireCapability(auth.UnionFeedbackWrite), d.feedbackH.LinkResource)
			// 反馈截图：从 SQLite blob 流式输出（需认证，原公开路由已移入 secured 组修复越权）
			secured.GET("/uploads/feedback/:filename", d.feedbackH.ServeScreenshot)

			// ── 反馈修复任务（闭环 MVP，管理端）──
			// 审核创建/列表/详情/取消/验收/驳回/部署确认/完成，全部走 JWT + UnionFeedback 能力门控。
			// 服务器绝不执行改码/构建/部署；仅做状态机与审计。
			repairTasks := secured.Group("/admin/feedback/repair-tasks")
			{
				repairTasks.POST("", auth.RequireCapability(auth.UnionFeedbackWrite), d.feedbackRepairTaskH.CreateTask)
				repairTasks.GET("", auth.RequireCapability(auth.UnionFeedbackList), d.feedbackRepairTaskH.ListTasks)
				repairTasks.GET("/:no", auth.RequireCapability(auth.UnionFeedbackList), d.feedbackRepairTaskH.GetTask)
				repairTasks.POST("/:no/cancel", auth.RequireCapability(auth.UnionFeedbackWrite), d.feedbackRepairTaskH.CancelTask)
				repairTasks.POST("/:no/accept", auth.RequireCapability(auth.UnionFeedbackWrite), d.feedbackRepairTaskH.AcceptTask)
				repairTasks.POST("/:no/reject", auth.RequireCapability(auth.UnionFeedbackWrite), d.feedbackRepairTaskH.RejectTask)
				repairTasks.POST("/:no/deploy-confirm", auth.RequireCapability(auth.UnionFeedbackWrite), d.feedbackRepairTaskH.DeployConfirmTask)
				repairTasks.POST("/:no/deploy-done", auth.RequireCapability(auth.UnionFeedbackWrite), d.feedbackRepairTaskH.DeployDoneTask)
			}

			// ── 办事流程办理记录 ──
			process := secured.Group("/process/records")
			{
				process.GET("", auth.RequireCapability(auth.SelfProcessRead), d.processRecordH.ListMine)
				process.POST("/:flow/start", auth.RequireCapability(auth.SelfProcessRead), d.processRecordH.StartOrResume)
				process.POST("/:flow/progress", auth.RequireCapability(auth.SelfProcessRead), d.processRecordH.UpdateProgress)
			}

			// ── 办事流程定义（学生端动态列表）──
			processDef := secured.Group("/process/definitions")
			processDef.Use(auth.RequireCapability(auth.SelfProcessRead))
			{
				processDef.GET("", d.processH.ListDefinitions)
				processDef.GET("/:id", d.processH.GetDefinition)
			}

			// ── 办事流程管理/审核（counselor+，学校/学院管理员继承）──
			processAdmin := secured.Group("/process/admin")
			processAdmin.Use(auth.RequireAnyCapability(auth.CounselorKBWrite, auth.CounselorKBReview))
			{
				processAdmin.GET("", auth.RequireCapability(auth.CounselorKBWrite), d.processH.ListAdmin)
				processAdmin.GET("/pending", auth.RequireCapability(auth.CounselorKBReview), d.processH.ListPending)
				processAdmin.POST("", auth.RequireCapability(auth.CounselorKBWrite), d.processH.Create)
				processAdmin.GET("/:id", auth.RequireCapability(auth.CounselorKBWrite), d.processH.GetAdmin)
				processAdmin.PUT("/:id", auth.RequireCapability(auth.CounselorKBWrite), d.processH.Update)
				processAdmin.DELETE("/:id", auth.RequireCapability(auth.CounselorKBWrite), d.processH.Delete)
				processAdmin.POST("/:id/submit", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), d.processH.Submit)
				processAdmin.POST("/:id/approve", auth.RequireCapability(auth.CounselorKBReview), d.processH.Approve)
				processAdmin.POST("/:id/reject", auth.RequireCapability(auth.CounselorKBReview), d.processH.Reject)
				processAdmin.POST("/:id/retire", auth.RequireCapability(auth.CounselorKBReview), d.processH.Retire)
			}

			// ── 学生 AI 功能（个人能力，所有角色继承自 student 都可用）──
			student := secured.Group("/student")
			{
				student.GET("/home", auth.RequireCapability(auth.SelfStudyRead), d.studentH.Home)
				student.GET("/profile", auth.RequireCapability(auth.SelfTwinRead), d.studentH.PersonalProfile)
				student.GET("/twin-profile", auth.RequireCapability(auth.SelfTwinRead), d.studentH.TwinProfile)
				student.GET("/personality-profile", auth.RequireCapability(auth.SelfPersonalityRead), d.studentH.PersonalityProfile)
				// 头像上传与服务（GET 供前端 <img> 直接加载，认证保护）
				student.POST("/avatar", auth.RequireCapability(auth.SelfProfileWrite), d.studentH.UploadAvatar)
				student.GET("/avatar/:user_id", auth.RequireCapability(auth.SelfTwinRead), d.studentH.ServeAvatar)
				student.GET("/daily-briefing", auth.RequireCapability(auth.SelfBriefingRead), d.studentH.DailyBriefing)
				student.GET("/learning-diary", auth.RequireCapability(auth.SelfDiaryRead), d.studentH.LearningDiary)
				student.POST("/checkin", auth.RequireCapability(auth.SelfCheckinWrite), d.studentH.Checkin)
				student.GET("/checkin/history", auth.RequireCapability(auth.SelfCheckinWrite), d.studentH.CheckinHistory)
				student.POST("/checkin/makeup", auth.RequireCapability(auth.SelfCheckinWrite), d.studentH.CheckinMakeup)
				// 毕业去向自报（学生，待教辅审核；2026-08-15）
				student.POST("/outcome/self-report", auth.RequireCapability(auth.OutcomeRecordWrite), d.secretaryH.SubmitOutcome)
				student.GET("/outcome/my", auth.RequireCapability(auth.OutcomeRecordRead), d.secretaryH.ListOutcomes)
				student.POST("/schedule/import", auth.RequireCapability(auth.SelfProfileWrite), d.dataImportH.ImportMySchedule)
				student.GET("/digital-twin", auth.RequireCapability(auth.SelfTwinRead), d.studentH.DigitalTwin)
				student.GET("/personality", auth.RequireCapability(auth.SelfPersonalityRead), d.studentH.Personality)
				student.GET("/achievements", auth.RequireCapability(auth.SelfAchievements), d.studentH.Achievements)
				student.GET("/course-map", auth.RequireCapability(auth.SelfCourseMapRead), d.studentH.CourseMap)
				student.GET("/course-analytics", auth.RequireCapability(auth.SelfCourseAnalytics), d.studentH.CourseAnalytics)
				student.GET("/weekly-report", auth.RequireCapability(auth.SelfWeeklyReport), d.studentH.WeeklyReport)
				student.GET("/freshman-plan", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GenericAI("freshman-plan"))
				student.GET("/growth-path", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GrowthPath)
				student.GET("/political-study", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GenericAI("political-study"))
				student.GET("/ideological-record", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GenericAI("ideological-record"))
				student.GET("/party-progress", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GenericAI("party-progress"))
				student.GET("/campus-life", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GenericAI("campus-life"))
				student.GET("/schedule", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GenericAI("schedule"))
				student.GET("/competition-match", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GenericAI("competition-match"))
				student.GET("/study-buddy", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GenericAI("study-buddy"))
				student.GET("/mental-health", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GenericAI("mental-health"))
				student.GET("/digital-mentor", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GenericAI("digital-mentor"))
				student.GET("/qa-plaza", auth.RequireCapability(auth.SelfCommunityRead), d.studentH.QAPlaza)
				student.GET("/qa/posts", auth.RequireCapability(auth.SelfCommunityRead), d.studentH.ListQAPosts)
				student.POST("/qa/posts", auth.RequireCapability(auth.SelfCommunityRead), d.studentH.CreateQAPost)
				student.GET("/qa/posts/:id", auth.RequireCapability(auth.SelfCommunityRead), d.studentH.GetQAPostDetail)
				student.POST("/qa/posts/:id/answer", auth.RequireCapability(auth.SelfCommunityRead), d.studentH.AnswerQAPost)
				student.GET("/hot-topics", auth.RequireCapability(auth.SelfCommunityRead), d.studentH.HotTopics)
				student.GET("/qa-leaderboard", auth.RequireCapability(auth.SelfCommunityRead), d.studentH.QALeaderboard)
				student.GET("/private-chat", auth.RequireCapability(auth.SelfPrivateChat), d.studentH.PrivateChat)
				student.GET("/process-enhanced", auth.RequireCapability(auth.SelfProcessRead), d.studentH.ProcessEnhanced)
				student.GET("/freshmen-guide", auth.RequireCapability(auth.SelfKnowledgeRead), d.studentH.FreshmenGuide)
				// ── P2 学生深度分析 ──
				student.GET("/values-guidance", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GenericAI("values-guidance"))
				student.GET("/classroom-extension", auth.RequireCapability(auth.SelfGenericAI), d.studentH.GenericAI("classroom-extension"))
				student.GET("/mock-interview", auth.RequireCapability(auth.SelfGenericAI), d.studentH.MockInterview)
				student.GET("/resume", auth.RequireCapability(auth.SelfGenericAI), d.studentH.Resume)
				student.GET("/career-simulation", auth.RequireCapability(auth.SelfGenericAI), d.studentH.CareerSimulation)
				student.GET("/study-buddy-match", auth.RequireCapability(auth.SelfGenericAI), d.studentH.StudyBuddyMatch)
				student.GET("/mental-health-report", auth.RequireCapability(auth.SelfGenericAI), d.studentH.MentalHealthReport)
				student.GET("/note-assistant", auth.RequireCapability(auth.SelfGenericAI), d.studentH.NoteAssistant)
				student.GET("/alumni-match", auth.RequireCapability(auth.SelfGenericAI), d.studentH.AlumniMatch)
				// ── P3 生态扩展 ──
				student.GET("/dynamic-mentor", auth.RequireCapability(auth.SelfGenericAI), d.studentH.DynamicMentor)
				student.GET("/career-sim-enhanced", auth.RequireCapability(auth.SelfGenericAI), d.studentH.EnhancedCareerSim)
			}

			// ── 个人通用入口 /me/* —— 与 /student/* 个人能力等价的语义化别名 ──
			// 适用于"任何角色访问自己的"场景，避免高阶用户访问 /student/ 路径的语义违和
			me := secured.Group("/me")
			{
				me.GET("/daily-briefing", auth.RequireCapability(auth.SelfBriefingRead), d.studentH.DailyBriefing)
				me.GET("/learning-diary", auth.RequireCapability(auth.SelfDiaryRead), d.studentH.LearningDiary)
				me.GET("/digital-twin", auth.RequireCapability(auth.SelfTwinRead), d.studentH.DigitalTwin)
				me.GET("/personality", auth.RequireCapability(auth.SelfPersonalityRead), d.studentH.Personality)
				me.GET("/achievements", auth.RequireCapability(auth.SelfAchievements), d.studentH.Achievements)
				me.GET("/weekly-report", auth.RequireCapability(auth.SelfWeeklyReport), d.studentH.WeeklyReport)
				me.POST("/checkin", auth.RequireCapability(auth.SelfCheckinWrite), d.studentH.Checkin)
				me.GET("/checkin/history", auth.RequireCapability(auth.SelfCheckinWrite), d.studentH.CheckinHistory)
				me.POST("/checkin/makeup", auth.RequireCapability(auth.SelfCheckinWrite), d.studentH.CheckinMakeup)
			}

			// ── 辅导员 AI 功能 ──
			counselor := secured.Group("/counselor")
			{
				counselor.GET("/daily-focus", auth.RequireCapability(auth.CounselorDailyFocusRead), d.counselorH.DailyFocus)
				counselor.GET("/class-report", auth.RequireCapability(auth.CounselorClassReport), d.counselorH.ClassReport)
				counselor.GET("/twin-board", auth.RequireCapability(auth.CounselorTwinBoard), d.counselorH.TwinBoard)
				counselor.GET("/prediction", auth.RequireCapability(auth.CounselorPredictionRead), d.counselorH.Prediction)
				counselor.POST("/intervention", auth.RequireCapability(auth.CounselorInterventionWrite), d.counselorH.Intervention)
				counselor.GET("/talk-record", auth.RequireCapability(auth.CounselorTalkRecord), d.counselorH.TalkRecord)
				counselor.POST("/talk-record", auth.RequireCapability(auth.CounselorTalkRecord), d.counselorH.TalkRecord)
				counselor.GET("/talk-tips", auth.RequireCapability(auth.CounselorTalkTips), d.counselorH.TalkTips)
				counselor.GET("/ideological", auth.RequireCapability(auth.CounselorIdeological), d.counselorH.Ideological)
				counselor.GET("/class-profile", auth.RequireCapability(auth.CounselorClassProfile), d.counselorH.ClassProfile)
				counselor.GET("/community-manage", auth.RequireCapability(auth.CounselorCommunityManage), d.counselorH.CommunityManage)
				counselor.GET("/hot-topic-sense", auth.RequireCapability(auth.CounselorHotTopicSense), d.counselorH.HotTopicSense)
				counselor.GET("/process-edit", auth.RequireCapability(auth.CounselorProcessEdit), d.counselorH.ProcessEdit)
				counselor.GET("/student-list", auth.RequireCapability(auth.CounselorStudentList), d.counselorH.StudentList)
				// 第二课堂班级看板（辅导员）：真实聚合名下学生活动参与/积分
				counselor.GET("/second-class-board", auth.RequireCapability(auth.CounselorSecondClassBoard), d.counselorH.SecondClassBoard)
				// ── P2 辅导员深度分析 ──
				counselor.GET("/follow-up-reminders", auth.RequireCapability(auth.CounselorTalkRecord), d.counselorH.FollowUpReminders)
				counselor.GET("/checkin-stats", auth.RequireCapability(auth.CounselorClassReport), d.counselorH.CheckinStats)
				counselor.POST("/smart-notify", auth.RequireCapability(auth.CounselorInterventionWrite), d.counselorH.SmartNotify)
				counselor.GET("/monthly-brief", auth.RequireCapability(auth.CounselorClassReport), d.counselorH.MonthlyBrief)
				counselor.POST("/session-insight", auth.RequireCapability(auth.CounselorTalkRecord), d.counselorH.SessionInsight)
			}

			// ── 通知推送（辅导员及以上角色） ──
			// 移动到 /admin/notifications/push 路径下，避免与用户站内通知冲突
			notificationPush := secured.Group("/admin/notifications/push")
			notificationPush.Use(auth.RequireCapability(auth.CounselorNotify))
			{
				notificationPush.POST("", d.notificationH.Create)
				notificationPush.GET("", d.notificationH.List)
				notificationPush.POST("/:id/publish", d.notificationH.Publish)
				notificationPush.DELETE("/:id", d.notificationH.Delete)
				notificationPush.GET("/webhook-status", d.notificationH.WebhookStatus)
			}

			// ── 用户站内通知（所有登录用户） ──
			secured.GET("/notifications", d.userNotificationH.ListNotifications)
			secured.GET("/notifications/unread-count", d.userNotificationH.GetUnreadCount)
			secured.PUT("/notifications/:id/read", d.userNotificationH.MarkAsRead)
			secured.PUT("/notifications/read-all", d.userNotificationH.MarkAllAsRead)

			// ── 管理员发送系统通知 ──
			secured.POST("/admin/notifications/send", auth.RequireCapability(auth.SystemSettingsWrite), d.userNotificationH.SendSystemNotification)

			// ── 管理员站内通知管理（查看/删除/清空） ──
			secured.GET("/admin/notifications/list", auth.RequireCapability(auth.SystemSettingsWrite), d.userNotificationH.AdminListNotifications)
			secured.DELETE("/admin/notifications/:id", auth.RequireCapability(auth.SystemSettingsWrite), d.userNotificationH.AdminDeleteNotification)
			secured.DELETE("/admin/notifications", auth.RequireCapability(auth.SystemSettingsWrite), d.userNotificationH.AdminClearNotifications)

			// ── 文档解析 ──
			secured.POST("/documents/parse", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), d.documentH.ParseDocument)
			secured.POST("/documents/refine", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), d.documentH.RefineDocument)
			secured.GET("/documents/formats", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), d.documentH.SupportedFormats)

			// ── 文档上传与知识入库 ──
			secured.POST("/kb/upload", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), d.uploadH.Upload)
			secured.GET("/kb/formats", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), d.uploadH.SupportedFormats)

			// ── 教师 AI 功能 ──
			teacher := secured.Group("/teacher")
			{
				teacher.GET("/daily-overview", auth.RequireCapability(auth.TeacherDailyOverview), d.teacherH.DailyOverview)
				teacher.POST("/lesson-prep", auth.RequireCapability(auth.TeacherLessonPrep), d.teacherH.LessonPrep)
				teacher.POST("/exam-gen", auth.RequireCapability(auth.TeacherExamGen), d.teacherH.ExamGen)
				teacher.POST("/class-interact", auth.RequireCapability(auth.TeacherClassInteract), d.teacherH.ClassInteract)
				teacher.POST("/grading", auth.RequireCapability(auth.TeacherGrading), d.teacherH.Grading)
				teacher.GET("/heatmap", auth.RequireCapability(auth.TeacherHeatmapRead), d.teacherH.Heatmap)
				teacher.GET("/reflection", auth.RequireCapability(auth.TeacherReflection), d.teacherH.Reflection)
				teacher.GET("/style-distribution", auth.RequireCapability(auth.TeacherStyleDist), d.teacherH.StyleDist)
				teacher.GET("/community-qa", auth.RequireCapability(auth.TeacherCommunityQA), d.teacherH.CommunityQA)
				// ── P2 教师深度分析 ──
				teacher.GET("/faq-knowledge", auth.RequireCapability(auth.TeacherCommunityQA), d.teacherH.FAQKnowledge)
				teacher.GET("/student-twin", auth.RequireCapability(auth.TeacherHeatmapRead), d.teacherH.StudentTwin)
				teacher.GET("/knowledge-coverage", auth.RequireCapability(auth.TeacherLessonPrep), d.teacherH.KnowledgeCoverage)
				teacher.GET("/ideological-suggestions", auth.RequireCapability(auth.TeacherReflection), d.teacherH.IdeologicalSuggestions)
				teacher.GET("/personalized-teaching", auth.RequireCapability(auth.TeacherStyleDist), d.teacherH.PersonalizedTeaching)
				// ── 党课/活动登记（2026-08-16，蓝图第3块）：教师/教辅登记党课与积极分子活动 ──
				teacher.POST("/party/register", auth.RequireCapability(auth.PartyRecordWrite), d.secretaryH.CreatePartyRecord)
				teacher.GET("/party/records", auth.RequireCapability(auth.PartyRecordRead), d.secretaryH.ListPartyRecords)
				teacher.DELETE("/party/records/:id", auth.RequireCapability(auth.PartyRecordWrite), d.secretaryH.DeletePartyRecord)
				// ── 教师录入所授班级成绩（2026-08-17，P0-1，方案A：教师自主声明授课）──
				teacher.POST("/grades/import", auth.RequireCapability(auth.TeacherGradeWrite), d.dataImportH.ImportTeacherGrades)
				teacher.GET("/grades/mine", auth.RequireCapability(auth.TeacherGradeWrite), d.dataImportH.ListMyTeacherGrades)
				// ── 教师授课关系申报（2026-08-17，R3 越权边界升级）：申报→教辅审核→approved 后录入 ──
				teacher.POST("/courses/apply", auth.RequireCapability(auth.TeacherGradeWrite), d.teacherCourseH.SubmitTeacherCourse)
				teacher.GET("/courses/mine", auth.RequireCapability(auth.TeacherGradeWrite), d.teacherCourseH.ListMyTeacherCourses)
				// ── 教师作业信息发布+成绩统计（2026-08-17，P2 轻量版）：复用 TeacherGradeWrite，不新增 capability ──
				teacher.POST("/homework", auth.RequireCapability(auth.TeacherGradeWrite), d.homeworkH.PublishHomework)
				teacher.PUT("/homework/:id", auth.RequireCapability(auth.TeacherGradeWrite), d.homeworkH.UpdateHomework)
				teacher.DELETE("/homework/:id", auth.RequireCapability(auth.TeacherGradeWrite), d.homeworkH.ArchiveHomework)
				teacher.GET("/homework/mine", auth.RequireCapability(auth.TeacherGradeWrite), d.homeworkH.ListMyHomework)
				teacher.GET("/homework/courses", auth.RequireCapability(auth.TeacherGradeWrite), d.homeworkH.ListApprovedCourses)
				teacher.GET("/homework/:courseId/grade-stats", auth.RequireCapability(auth.TeacherGradeWrite), d.homeworkH.GradeStatsByCourse)
			}

			// ── 教辅 AI 功能 ──
			assistantGroup := secured.Group("/assistant")
			{
				assistantGroup.GET("/schedule-check", auth.RequireCapability(auth.AssistantScheduleCheck), d.assistantH.ScheduleCheck)
				assistantGroup.GET("/graduation-audit", auth.RequireCapability(auth.AssistantGradAudit), d.assistantH.GradAudit)
				assistantGroup.GET("/exam-arrange", auth.RequireCapability(auth.AssistantExamArrange), d.assistantH.ExamArrange)
				// ── P2 教辅深度分析 ──
				assistantGroup.POST("/notification", auth.RequireCapability(auth.AssistantScheduleCheck), d.assistantH.Notification)
				assistantGroup.GET("/teaching-calendar", auth.RequireCapability(auth.AssistantScheduleCheck), d.assistantH.TeachingCalendar)
				assistantGroup.GET("/student-info", auth.RequireCapability(auth.AssistantGradAudit), d.assistantH.StudentInfoQuery)
				// ── P2 补充功能 ──
				assistantGroup.GET("/material-templates", auth.RequireCapability(auth.AssistantScheduleCheck), d.assistantH.MaterialTemplates)
				assistantGroup.POST("/doc-process", auth.RequireCapability(auth.AssistantScheduleCheck), d.assistantH.DocProcess)
				assistantGroup.GET("/workflow-automation", auth.RequireCapability(auth.AssistantScheduleCheck), d.assistantH.WorkflowAutomation)
				assistantGroup.GET("/process-steps-manage", auth.RequireCapability(auth.AssistantGradAudit), d.assistantH.ProcessStepsManage)
				assistantGroup.GET("/music-radio", auth.RequireCapability(auth.AssistantScheduleCheck), d.assistantH.MusicRadio)
				assistantGroup.GET("/activity-register", auth.RequireCapability(auth.AssistantScheduleCheck), d.assistantH.ActivityRegister)
				// ── 后勤服务台（并入教辅，2026-08-15）──
				assistantGroup.GET("/facility/roles", auth.RequireCapability(auth.FacilityRecordRead), d.facilityH.RoleMeta)
				assistantGroup.POST("/facility/record", auth.RequireCapability(auth.FacilityRecordWrite), d.facilityH.CreateRecord)
				assistantGroup.GET("/facility/records", auth.RequireCapability(auth.FacilityRecordRead), d.facilityH.ListRecords)
				assistantGroup.GET("/facility/dashboard", auth.RequireCapability(auth.FacilityDashboard), d.facilityH.Dashboard)

				// ── 毕业去向登记与审核（教辅，2026-08-15 书记教育成果闭环）──
				assistantGroup.GET("/outcome/meta", auth.RequireCapability(auth.OutcomeRecordRead), d.secretaryH.OutcomeMeta)
				assistantGroup.POST("/outcome/record", auth.RequireCapability(auth.OutcomeRecordWrite), d.secretaryH.SubmitOutcome)
				assistantGroup.GET("/outcome/records", auth.RequireCapability(auth.OutcomeRecordRead), d.secretaryH.ListOutcomes)
				assistantGroup.GET("/outcome/pending", auth.RequireCapability(auth.OutcomeReview), d.secretaryH.CountPending)
				assistantGroup.PUT("/outcome/review/:id", auth.RequireCapability(auth.OutcomeReview), d.secretaryH.ReviewOutcome)

				// ── 教师授课关系审核（2026-08-17，R3）：教辅/教务审核 + 待审角标（teacher.course.review）──
				assistantGroup.GET("/courses/pending", auth.RequireCapability(auth.TeacherCourseReview), d.teacherCourseH.ListPendingTeacherCourses)
				assistantGroup.PUT("/courses/review/:id", auth.RequireCapability(auth.TeacherCourseReview), d.teacherCourseH.ReviewTeacherCourse)
				assistantGroup.GET("/courses/pending-count", auth.RequireCapability(auth.TeacherCourseReview), d.teacherCourseH.CountPending)
			}

			// ── 学生会 AI 功能 ──
			unionGroup := secured.Group("/union")
			{
				unionGroup.POST("/event-plan", auth.RequireCapability(auth.UnionEventPlan), d.unionH.EventPlan)
				unionGroup.POST("/poster-gen", auth.RequireCapability(auth.UnionPosterGen), d.unionH.PosterGen)
				// ── P2 学生会深度分析 ──
				unionGroup.GET("/recruitment", auth.RequireCapability(auth.UnionEventPlan), d.unionH.Recruitment)
				unionGroup.GET("/member-manage", auth.RequireCapability(auth.UnionEventPlan), d.unionH.MemberManage)
				unionGroup.GET("/questionnaire", auth.RequireCapability(auth.UnionPosterGen), d.unionH.Questionnaire)
				unionGroup.GET("/hot-topic-track", auth.RequireCapability(auth.UnionEventPlan), d.unionH.HotTopicTrack)
				unionGroup.GET("/activity-analysis", auth.RequireCapability(auth.UnionEventPlan), d.unionH.ActivityAnalysis)
			}

			// ── 学院管理员 AI 功能 ──
			collegeGroup := secured.Group("/college")
			{
				collegeGroup.GET("/twin-screen", auth.RequireCapability(auth.CollegeTwinScreen), d.collegeH.TwinScreen)
				collegeGroup.GET("/data-analysis", auth.RequireCapability(auth.CollegeDataAnalysis), d.collegeH.DataAnalysis)
				// ── P2 学院管理员深度分析 ──
				collegeGroup.GET("/decision-advice", auth.RequireCapability(auth.CollegeDataAnalysis), d.collegeH.DecisionAdvice)
				collegeGroup.GET("/teacher-efficiency", auth.RequireCapability(auth.CollegeTwinScreen), d.collegeH.TeacherEfficiency)
				collegeGroup.GET("/course-quality", auth.RequireCapability(auth.CollegeDataAnalysis), d.collegeH.CourseQuality)
				collegeGroup.GET("/college-report", auth.RequireCapability(auth.CollegeTwinScreen), d.collegeH.CollegeReport)
				collegeGroup.GET("/process-step-edit", auth.RequireCapability(auth.CollegeDataAnalysis), d.collegeH.ProcessStepEdit)
				// ── 书记教育成果大屏（2026-08-15）：school_admin 全校/college_admin 本院 ──
				// college 参数空=全校（学校书记），传学院=本院（学院书记）
				collegeGroup.GET("/education-outcome", auth.RequireCapability(auth.OutcomeDashboard), d.secretaryH.OutcomeDashboard)
				collegeGroup.GET("/party-dashboard", auth.RequireCapability(auth.OutcomeDashboard), d.secretaryH.PartyDashboard)
				// ── 协同育人总览（2026-08-16，蓝图第2块）：书记视角聚合教师/教辅育人动作 ──
				collegeGroup.GET("/collab-dashboard", auth.RequireCapability(auth.CollabDashboard), d.secretaryH.CollabDashboard)
				// ── 育人成效 KPI 指标卡（D5-1 功能补齐，2026-08-16）：量化 KPI + 诚实数据来源标注，复用 outcome.dashboard 能力 ──
				collegeGroup.GET("/nurture-kpi", auth.RequireCapability(auth.OutcomeDashboard), d.secretaryH.NurtureKPI)
				// ── 督办工单（D5-3「洞察→工单」治理回环，2026-08-16）──
				// 书记/学院从治理洞察/KPI 生成督办工单分派给辅导员/教辅/党群，责任人推进本人分派。
				collegeGroup.POST("/tickets", auth.RequireCapability(auth.GovTicketManage), d.govTicketH.Create)
				collegeGroup.POST("/tickets/from-kpi", auth.RequireCapability(auth.GovTicketManage), d.govTicketH.CreateFromKPI) // D5-1 联动：补料督办
				collegeGroup.GET("/tickets", auth.RequireCapability(auth.GovTicketManage), d.govTicketH.List)
				collegeGroup.GET("/tickets/mine", auth.RequireCapability(auth.GovTicketAssignee), d.govTicketH.ListMine)
				collegeGroup.GET("/tickets/mine/:id", auth.RequireCapability(auth.GovTicketAssignee), d.govTicketH.Get)
				collegeGroup.PUT("/tickets/mine/:id/status", auth.RequireCapability(auth.GovTicketAssignee), d.govTicketH.UpdateStatus)
				collegeGroup.GET("/tickets/stats", auth.RequireCapability(auth.GovTicketManage), d.govTicketH.Stats)
				collegeGroup.GET("/tickets/:id", auth.RequireCapability(auth.GovTicketManage), d.govTicketH.Get)
				collegeGroup.PUT("/tickets/:id/assign", auth.RequireCapability(auth.GovTicketManage), d.govTicketH.Assign)
				collegeGroup.PUT("/tickets/:id/status", auth.RequireCapability(auth.GovTicketManage), d.govTicketH.UpdateStatus)
			}

			// ── 学校管理员 AI 功能（P2）──
			schoolAdminGroup := secured.Group("/school-admin")
			{
				schoolAdminGroup.GET("/panorama", auth.RequireCapability(auth.CollegeTwinScreen), d.schoolAdminH.Panorama)
				schoolAdminGroup.POST("/policy-simulation", auth.RequireCapability(auth.CollegeDataAnalysis), d.schoolAdminH.PolicySimulation)
				schoolAdminGroup.GET("/college-comparison", auth.RequireCapability(auth.CollegeTwinScreen), d.schoolAdminH.CollegeComparison)
				schoolAdminGroup.GET("/academic-overview", auth.RequireCapability(auth.CollegeTwinScreen), d.schoolAdminH.AcademicOverview)
			}

			// ── 系统管理员 AI 功能（P2）──
			sysAdminGroup := secured.Group("/sys-admin")
			{
				sysAdminGroup.GET("/system-health", auth.RequireCapability(auth.CollegeTwinScreen), d.sysAdminH.SystemHealth)
				sysAdminGroup.GET("/knowledge-quality", auth.RequireCapability(auth.CollegeDataAnalysis), d.sysAdminH.KnowledgeQuality)
				sysAdminGroup.GET("/user-behavior", auth.RequireCapability(auth.CollegeTwinScreen), d.sysAdminH.UserBehavior)
			}

			// ── 就业指导模块（全员可见）──
			career := secured.Group("/career")
			{
				career.GET("/policies", auth.RequireCapability(auth.SelfCareerRead), d.educationH.ListCareerPolicies)
				career.GET("/policies/:id", auth.RequireCapability(auth.SelfCareerRead), d.educationH.GetCareerPolicy)
				career.GET("/jobs", auth.RequireCapability(auth.SelfCareerRead), d.educationH.ListJobPostings)
				career.GET("/jobs/:id", auth.RequireCapability(auth.SelfCareerRead), d.educationH.GetJobPosting)
				career.GET("/sessions", auth.RequireCapability(auth.SelfCareerRead), d.educationH.ListInfoSessions)
				career.GET("/interview/questions", auth.RequireCapability(auth.SelfCareerRead), d.educationH.ListInterviewQuestions)
				// ── 就业指导管理（学校/学院管理员）──
				career.GET("/admin/policies", auth.RequireCapability(auth.SystemSettingsWrite), d.educationH.AdminListCareerPolicies)
				career.POST("/admin/policies", auth.RequireCapability(auth.SystemSettingsWrite), d.educationH.AdminCreateCareerPolicy)
				career.DELETE("/admin/policies/:id", auth.RequireCapability(auth.SystemSettingsWrite), d.educationH.AdminDeleteCareerPolicy)
				career.GET("/admin/jobs", auth.RequireCapability(auth.SystemSettingsWrite), d.educationH.AdminListJobPostings)
				career.POST("/admin/jobs", auth.RequireCapability(auth.SystemSettingsWrite), d.educationH.AdminCreateJobPosting)
				career.DELETE("/admin/jobs/:id", auth.RequireCapability(auth.SystemSettingsWrite), d.educationH.AdminDeleteJobPosting)
			}

			// ── 学业学习模块（全员可见）──
			study := secured.Group("/study")
			{
				study.GET("/courses", auth.RequireCapability(auth.SelfStudyRead), d.educationH.ListCourses)
				study.GET("/courses/:id", auth.RequireCapability(auth.SelfStudyRead), d.educationH.GetCourse)
				study.GET("/grades", auth.RequireCapability(auth.SelfStudyRead), d.educationH.ListMyGrades)
				study.GET("/grades/summary", auth.RequireCapability(auth.SelfStudyRead), d.educationH.GetGradeSummary)
				study.GET("/resources", auth.RequireCapability(auth.SelfStudyRead), d.educationH.ListLearningResources)
				study.GET("/exams", auth.RequireCapability(auth.SelfStudyRead), d.educationH.ListExamSchedules)

				// ── 校历 / 课表 / 学习计划（study_plan_handler）──
				// 校历
				study.GET("/calendar/current", auth.RequireCapability(auth.SelfStudyRead), d.studyPlanH.GetCurrentCalendar)
				study.GET("/calendar/:semester_code", auth.RequireCapability(auth.SelfStudyRead), d.studyPlanH.GetCalendarBySemester)
				// 课表
				study.GET("/timetable", auth.RequireCapability(auth.SelfStudyRead), d.studyPlanH.GetMyTimetable)
				// 学习计划概览（用于多 Tab 首页）
				study.GET("/plans/overview", auth.RequireCapability(auth.SelfStudyRead), d.studyPlanH.GetPlansOverview)
				// AI 生成学习计划
				study.POST("/plans/ai-generate", auth.RequireCapability(auth.SelfStudyWrite), d.studyPlanH.AIGeneratePlan)
				// 学习计划 CRUD
				study.GET("/plans", auth.RequireCapability(auth.SelfStudyRead), d.studyPlanH.ListMyPlans)
				study.POST("/plans", auth.RequireCapability(auth.SelfStudyWrite), d.studyPlanH.CreatePlan)
				study.GET("/plans/:id", auth.RequireCapability(auth.SelfStudyRead), d.studyPlanH.GetPlan)
				study.PUT("/plans/:id", auth.RequireCapability(auth.SelfStudyWrite), d.studyPlanH.UpdatePlan)
				study.DELETE("/plans/:id", auth.RequireCapability(auth.SelfStudyWrite), d.studyPlanH.DeletePlan)
				// 计划任务
				study.POST("/plans/:id/tasks", auth.RequireCapability(auth.SelfStudyWrite), d.studyPlanH.AddTask)
				study.PUT("/plans/:id/tasks/:task_id", auth.RequireCapability(auth.SelfStudyWrite), d.studyPlanH.UpdateTask)
			}

			// ── 心理健康模块（全员可见）──
			mental := secured.Group("/mental")
			{
				mental.GET("/scales", auth.RequireCapability(auth.SelfMentalRead), d.educationH.ListPsychScales)
				mental.GET("/scales/:id", auth.RequireCapability(auth.SelfMentalRead), d.educationH.GetPsychScale)
				mental.POST("/assessments", auth.RequireCapability(auth.SelfMentalWrite), d.educationH.SubmitAssessment)
				mental.GET("/assessments", auth.RequireCapability(auth.SelfMentalRead), d.educationH.ListMyAssessments)
				mental.GET("/counselors", auth.RequireCapability(auth.SelfMentalRead), d.educationH.ListCounselors)
				mental.GET("/appointments", auth.RequireCapability(auth.SelfMentalRead), d.educationH.ListMyAppointments)
				mental.POST("/appointments", auth.RequireCapability(auth.SelfMentalWrite), d.educationH.CreateAppointment)
				mental.GET("/articles", auth.RequireCapability(auth.SelfMentalRead), d.educationH.ListPsychArticles)
				mental.GET("/articles/:id", auth.RequireCapability(auth.SelfMentalRead), d.educationH.GetPsychArticle)
				mental.GET("/hotlines", auth.RequireCapability(auth.SelfMentalRead), d.educationH.ListCrisisHotlines)
				mental.GET("/mood", auth.RequireCapability(auth.SelfMentalRead), d.educationH.ListMyMoodDiary)
				mental.POST("/mood", auth.RequireCapability(auth.SelfMentalWrite), d.educationH.CreateMoodDiary)
			}

			// ── 身体健康模块（学生本人）──
			health := secured.Group("/health")
			{
				health.GET("/basic", auth.RequireCapability(auth.SelfHealthRead), d.educationH.GetHealthBasicInfo)
				health.PUT("/basic", auth.RequireCapability(auth.SelfHealthWrite), d.educationH.UpsertHealthBasicInfo)
				health.GET("/checkups", auth.RequireCapability(auth.SelfHealthRead), d.educationH.ListHealthCheckups)
				health.POST("/checkups", auth.RequireCapability(auth.SelfHealthWrite), d.educationH.CreateHealthCheckup)
				health.PUT("/checkups/:id", auth.RequireCapability(auth.SelfHealthWrite), d.educationH.UpdateHealthCheckup)
				health.DELETE("/checkups/:id", auth.RequireCapability(auth.SelfHealthWrite), d.educationH.DeleteHealthCheckup)
				health.GET("/records", auth.RequireCapability(auth.SelfHealthRead), d.educationH.ListHealthRecords)
				health.POST("/records", auth.RequireCapability(auth.SelfHealthWrite), d.educationH.CreateHealthRecord)
				health.PUT("/records/:id", auth.RequireCapability(auth.SelfHealthWrite), d.educationH.UpdateHealthRecord)
				health.DELETE("/records/:id", auth.RequireCapability(auth.SelfHealthWrite), d.educationH.DeleteHealthRecord)
				health.GET("/daily", auth.RequireCapability(auth.SelfHealthRead), d.educationH.ListHealthDaily)
				health.PUT("/daily", auth.RequireCapability(auth.SelfHealthWrite), d.educationH.UpsertHealthDaily)
				health.DELETE("/daily/:date", auth.RequireCapability(auth.SelfHealthWrite), d.educationH.DeleteHealthDaily)
				health.GET("/activities", auth.RequireCapability(auth.SelfHealthRead), d.educationH.ListHealthActivities)
				health.POST("/activities", auth.RequireCapability(auth.UnionEventPlan), d.educationH.CreateHealthActivity)
				health.POST("/activities/:id/favorite", auth.RequireCapability(auth.SelfHealthWrite), d.educationH.ToggleActivityFavorite)
				health.POST("/activities/:id/signup", auth.RequireCapability(auth.SelfHealthWrite), d.educationH.ToggleActivitySignup)
				health.POST("/activities/:id/status", auth.RequireCapability(auth.UnionEventPlan), d.educationH.UpdateHealthActivityStatus)
				health.POST("/activities/:id/attend/:uid", auth.RequireCapability(auth.UnionEventPlan), d.educationH.AttendActivitySignup)
				health.GET("/activities/review-stats", auth.RequireCapability(auth.UnionEventPlan), d.educationH.ActivityReviewStats)
				health.GET("/activities/:id/signups", auth.RequireCapability(auth.UnionEventPlan), d.educationH.ListActivitySignups)
			}

			// ── 校园文化智能体（全员可见）──
			cultureGroup := secured.Group("/culture")
			{
				cultureGroup.GET("/anthems", auth.RequireCapability(auth.SelfCultureAnthem), d.cultureH.Anthems)
				cultureGroup.GET("/radio", auth.RequireCapability(auth.SelfCultureRadio), d.cultureH.Radio)
				cultureGroup.GET("/lectures", auth.RequireCapability(auth.SelfCultureLectures), d.cultureH.Lectures)
				cultureGroup.GET("/events", auth.RequireCapability(auth.SelfCultureEvents), d.cultureH.Events)
				cultureGroup.GET("/volunteer", auth.RequireCapability(auth.SelfCultureVolunteer), d.cultureH.Volunteer)
			}
		}
	}

	return router
}
