package app

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dll/wxx/server"
	"github.com/dll/wxx/server/internal/agent"
	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/handler"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"

	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动（含 FTS5）
)

var (
	instance http.Handler
	initOnce sync.Once
	initErr  error
)

// New 返回完全初始化的 http.Handler，首次调用执行初始化，后续调用返回缓存实例。
// 内部调用 config.Load() 加载环境变量配置。
func New() (http.Handler, error) {
	initOnce.Do(func() {
		instance, initErr = initApp()
	})
	return instance, initErr
}

// NewWithConfig 使用外部提供的配置初始化（供本地 main.go 复用）。
func NewWithConfig(cfg *config.Config) (http.Handler, error) {
	initOnce.Do(func() {
		instance, initErr = initAppWithConfig(cfg)
	})
	return instance, initErr
}

func initApp() (http.Handler, error) {
	return initAppWithConfig(config.Load())
}

func initAppWithConfig(cfg *config.Config) (http.Handler, error) {
	log.Println("蔚小芯后端启动中...")

	// ── 1. 设置运行模式 ──
	if cfg.AppMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// ── 2. 初始化 SQLite 数据库 ──
	dbPath := cfg.SQLitePath
	if os.Getenv("VERCEL") != "" {
		dbPath = "/tmp/wxx.db"
		log.Printf("Vercel 环境：数据库路径 %s", dbPath)
	}

	db, err := initDB(dbPath)
	if err != nil {
		return nil, err
	}

	// ── 3. 自动迁移 ──
	if err := runMigrations(db); err != nil {
		return nil, err
	}

	// ── 4. 初始化各层依赖 ──

	// Repository 层
	userRepo := repository.NewUserRepo(db)
	sessionRepo := repository.NewSessionRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	kbRepo := repository.NewKBRepo(db)
	emotionRepo := repository.NewEmotionRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	auditRepo := repository.NewAuditRepo(db)
	feedbackRepo := repository.NewFeedbackRepo(db)
	settingsRepo := repository.NewSettingsRepo(db)
	modelConfigRepo := repository.NewModelConfigRepo(db)

	// LLM 客户端（优先 DeepSeek，备选智谱）
	var llmClient llm.ChatClient
	if cfg.DeepSeekAPIKey != "" {
		llmClient = llm.NewDeepSeekClient(cfg)
		log.Println("LLM 客户端: DeepSeek")
	} else if cfg.ZhipuAPIKey != "" {
		llmClient = llm.NewZhipuClient(cfg)
		log.Println("LLM 客户端: 智谱清言")
	} else {
		log.Println("警告：未配置任何 LLM API Key，问答功能不可用")
	}

	// 讯飞语音客户端
	var xfyunClient *llm.XfyunClient
	if cfg.XfyunAPIKey != "" && cfg.XfyunAPISecret != "" {
		xfyunClient = llm.NewXfyunClient(cfg)
		log.Println("讯飞语音客户端已启用")
	}

	// ── Temporal（Vercel 环境禁用）──
	if os.Getenv("VERCEL") == "" && cfg.TemporalHostPort != "" {
		log.Printf("Temporal 已配置: %s", cfg.TemporalHostPort)
	} else if os.Getenv("VERCEL") != "" {
		log.Println("Vercel 环境：Temporal 工作流引擎已禁用")
	}

	// Service 层
	authSvc := service.NewAuthService(cfg, userRepo)
	sessionSvc := service.NewSessionService(sessionRepo, messageRepo)
	kbSvc := service.NewKBService(kbRepo)

	var chatSvc *service.ChatService
	if llmClient != nil {
		chatSvc = service.NewChatService(sessionRepo, messageRepo, kbRepo, agentRepo, llmClient)
		chatSvc.SetOrchestrator(agent.NewOrchestrator(kbRepo, llmClient))
	}

	var emotionSvc *service.EmotionService
	if llmClient != nil {
		emotionSvc = service.NewEmotionService(emotionRepo, llmClient)
		log.Println("情感预警服务已启用")
	}

	agentSvc := service.NewAgentService(agentRepo)
	studentSvc := service.NewStudentService(userRepo, sessionRepo, messageRepo, emotionRepo, llmClient)
	counselorSvc := service.NewCounselorService(userRepo, emotionRepo, llmClient)
	integrationSvc := service.NewIntegrationService(cfg)
	adminSvc := service.NewAdminService(userRepo, auditRepo, settingsRepo)
	feedbackSvc := service.NewFeedbackService(feedbackRepo)
	modelConfigSvc := service.NewModelConfigService(modelConfigRepo)

	// Handler 层
	authHandler := handler.NewAuthHandler(authSvc)
	sessionHandler := handler.NewSessionHandler(sessionSvc)
	kbHandler := handler.NewKBHandler(kbSvc)

	var chatHandler *handler.ChatHandler
	if chatSvc != nil {
		chatHandler = handler.NewChatHandler(chatSvc)
		if emotionSvc != nil {
			chatHandler.SetEmotionService(emotionSvc)
		}
	}

	var voiceHandler *handler.VoiceHandler
	if xfyunClient != nil {
		voiceHandler = handler.NewVoiceHandler(xfyunClient)
	}

	var emotionHandler *handler.EmotionHandler
	if emotionSvc != nil {
		emotionHandler = handler.NewEmotionHandler(emotionSvc)
	}

	agentHandler := handler.NewAgentHandler(agentSvc)
	recSvc := service.NewRecommendationService(kbRepo, messageRepo)
	recHandler := handler.NewRecommendationHandler(recSvc)
	exportSvc := service.NewExportService()
	exportHandler := handler.NewExportHandler(kbSvc, exportSvc)
	integrationHandler := handler.NewIntegrationHandler(integrationSvc)
	adminHandler := handler.NewAdminHandler(adminSvc, authSvc)
	feedbackHandler := handler.NewFeedbackHandler(feedbackSvc)
	modelConfigHandler := handler.NewModelConfigHandler(modelConfigSvc)
	studentHandler := handler.NewStudentHandler(studentSvc)
	counselorHandler := handler.NewCounselorHandler(counselorSvc)
	teacherHandler := handler.NewTeacherHandler()
	assistantHandler := handler.NewAssistantHandler()
	unionHandler := handler.NewUnionHandler()
	collegeHandler := handler.NewCollegeHandler()
	cultureHandler := handler.NewCultureHandler()

	// ── 5. 构建路由 ──
	router := setupRouter(cfg, db, authHandler, sessionHandler, chatHandler, kbHandler,
		voiceHandler, emotionHandler, agentHandler, exportHandler, integrationHandler, recHandler,
		adminHandler, feedbackHandler, modelConfigHandler,
		studentHandler, counselorHandler, teacherHandler, assistantHandler, unionHandler, collegeHandler,
		cultureHandler)

	return router, nil
}

// initDB 初始化 SQLite 连接
func initDB(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Printf("SQLite 数据库已连接: %s", dbPath)
	return db, nil
}

// runMigrations 从嵌入的迁移文件执行数据库迁移
func runMigrations(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL UNIQUE,
		executed_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}

	entries, err := server.Migrations.ReadDir("migrations")
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	executed := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM _migrations WHERE filename = ?", entry.Name()).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		content, err := server.Migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}

		if err := execSQL(db, string(content), entry.Name()); err != nil {
			return err
		}

		if _, err := db.Exec("INSERT INTO _migrations (filename) VALUES (?)", entry.Name()); err != nil {
			return err
		}

		log.Printf("已执行迁移: %s", entry.Name())
		executed++
	}

	if executed == 0 {
		log.Println("所有迁移已是最新状态")
	} else {
		log.Printf("成功执行 %d 个迁移文件", executed)
	}
	return nil
}

// execSQL 解析并执行 SQL 内容（按分号分割，处理触发器复合语句）
func execSQL(db *sql.DB, content, filename string) error {
	statements := splitSQL(content)
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			log.Printf("迁移 %s 第 %d 条语句失败: %v", filename, i+1, err)
			return err
		}
	}
	return nil
}

// splitSQL 按分号分割 SQL 语句，正确处理触发器复合语句
func splitSQL(content string) []string {
	var statements []string
	var current strings.Builder
	inTrigger := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}

		upper := strings.ToUpper(trimmed)
		if strings.Contains(upper, "CREATE TRIGGER") {
			inTrigger = true
		}

		current.WriteString(line)
		current.WriteString("\n")

		if inTrigger && strings.HasSuffix(trimmed, "END;") {
			statements = append(statements, current.String())
			current.Reset()
			inTrigger = false
			continue
		}

		if !inTrigger && strings.HasSuffix(trimmed, ";") {
			statements = append(statements, current.String())
			current.Reset()
		}
	}

	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		statements = append(statements, remaining)
	}

	return statements
}

// setupRouter 构建 Gin 路由树
func setupRouter(cfg *config.Config, db *sql.DB,
	authH *handler.AuthHandler,
	sessionH *handler.SessionHandler,
	chatH *handler.ChatHandler,
	kbH *handler.KBHandler,
	voiceH *handler.VoiceHandler,
	emotionH *handler.EmotionHandler,
	agentH *handler.AgentHandler,
	exportH *handler.ExportHandler,
	integrationH *handler.IntegrationHandler,
	recH *handler.RecommendationHandler,
	adminH *handler.AdminHandler,
	feedbackH *handler.FeedbackHandler,
	modelConfigH *handler.ModelConfigHandler,
	studentH *handler.StudentHandler,
	counselorH *handler.CounselorHandler,
	teacherH *handler.TeacherHandler,
	assistantH *handler.AssistantHandler,
	unionH *handler.UnionHandler,
	collegeH *handler.CollegeHandler,
	cultureH *handler.CultureHandler,
) *gin.Engine {
	router := gin.New()

	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.TraceID())
	router.Use(gin.Logger())
	router.Use(middleware.AuditLog(db))

	// 静态文件服务：上传文件（截图等）
	router.Static("/uploads", "data/uploads")

	// 根路由
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "蔚小芯",
			"version": "0.1.0",
			"docs":    "/health",
		})
	})

	// 健康检查
	router.GET("/health", healthHandler(db))

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// 认证（公开）
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", authH.Login)
			authGroup.POST("/qr-login", handler.CreateQRSession)
			authGroup.GET("/qr-status", handler.GetQRSessionStatus)
			authGroup.PUT("/qr-scan", handler.ScanQRSession)
		}

		// 需要 JWT 认证
		secured := v1.Group("/")
		secured.Use(middleware.JWTAuth(cfg))
		{
			// ── AI 对话（self.chat）──
			if chatH != nil {
				secured.POST("/chat", auth.RequireCapability(auth.SelfChat), chatH.Ask)
			} else {
				secured.POST("/chat", placeholderHandler("对话接口（LLM 未配置）"))
			}

			// ── 会话/知识/推荐（self.* 能力）──
			secured.GET("/sessions", auth.RequireCapability(auth.SelfSessionRead), sessionH.ListSessions)
			secured.GET("/sessions/:id/messages", auth.RequireCapability(auth.SelfSessionRead), sessionH.GetMessages)
			secured.DELETE("/sessions/:id", auth.RequireCapability(auth.SelfSessionDelete), sessionH.DeleteSession)
			secured.GET("/knowledge", auth.RequireCapability(auth.SelfKnowledgeRead), kbH.BrowseKnowledge)
			secured.GET("/recommendations", auth.RequireCapability(auth.SelfRecommendRead), recH.GetRecommendations)

			// ── 情感数据 ──
			if emotionH != nil {
				// 自身情感统计：所有用户都可看自己（self.emotion.stats 由 student 起继承）
				secured.GET("/emotion/stats", auth.RequireCapability(auth.SelfEmotionStats), emotionH.GetStats)
			}

			if emotionH != nil {
				emotion := secured.Group("/emotion")
				{
					emotion.POST("/analyze", auth.RequireCapability(auth.CounselorAlertAnalyze), emotionH.Analyze)
					emotion.GET("/alerts", auth.RequireCapability(auth.CounselorAlertRead), emotionH.ListAlerts)
					emotion.PUT("/alerts/:id", auth.RequireCapability(auth.CounselorAlertHandle), emotionH.UpdateAlert)
					emotion.GET("/trends", auth.RequireCapability(auth.CounselorEmotionTrends), emotionH.Trends)
				}
			}

			// ── 知识库 CRUD（counselor.kb.write）──
			kb := secured.Group("/kb")
			{
				kb.GET("/resources", auth.RequireCapability(auth.CounselorKBWrite), kbH.ListResources)
				kb.POST("/resources", auth.RequireCapability(auth.CounselorKBWrite), kbH.CreateResource)
				kb.PUT("/resources/:id", auth.RequireCapability(auth.CounselorKBWrite), kbH.UpdateResource)
				kb.GET("/resources/:id", auth.RequireCapability(auth.CounselorKBWrite), kbH.GetResource)
				kb.POST("/import", auth.RequireCapability(auth.CounselorKBWrite), kbH.Import)
				kb.POST("/validate", auth.RequireCapability(auth.CounselorKBWrite), kbH.Validate)

				// 知识审核（counselor.kb.review）
				kb.POST("/resources/:id/approve", auth.RequireCapability(auth.CounselorKBReview), kbH.ApproveResource)
				kb.POST("/resources/:id/reject", auth.RequireCapability(auth.CounselorKBReview), kbH.RejectResource)
				kb.POST("/resources/:id/retire", auth.RequireCapability(auth.CounselorKBReview), kbH.RetireResource)

				// 知识提交（union.kb.submit，student_union 起）
				kb.POST("/resources/:id/submit", auth.RequireCapability(auth.UnionKBSubmit), kbH.SubmitForReview)
			}

			// ── 知识导出（self.export.self，所有人）──
			secured.GET("/kb/export", auth.RequireCapability(auth.SelfExportSelf), exportH.Export)

			// ── 智能体管理（school.agent.write）──
			agents := secured.Group("/agents")
			{
				agents.GET("", auth.RequireCapability(auth.SchoolAgentWrite), agentH.List)
				agents.POST("", auth.RequireCapability(auth.SchoolAgentWrite), agentH.Create)
				agents.GET("/:id", auth.RequireCapability(auth.SchoolAgentWrite), agentH.Get)
				agents.PUT("/:id", auth.RequireCapability(auth.SchoolAgentWrite), agentH.Update)
				agents.DELETE("/:id", auth.RequireCapability(auth.SchoolAgentWrite), agentH.Delete)
			}

			// ── 语音 ASR/TTS（self.voice）──
			if voiceH != nil {
				secured.POST("/voice/asr", auth.RequireCapability(auth.SelfVoice), voiceH.ASR)
				secured.POST("/voice/tts", auth.RequireCapability(auth.SelfVoice), voiceH.TTS)
			}

			// ── 通用导出（self.export.self）──
			secured.GET("/export", auth.RequireCapability(auth.SelfExportSelf), exportH.Export)
			secured.POST("/export/answer", auth.RequireCapability(auth.SelfExportSelf), exportH.ExportAnswer)

			// ── 校外系统对接（counselor.integration.read）──
			integration := secured.Group("/integration")
			{
				integration.GET("/status", auth.RequireCapability(auth.CounselorIntegrationRead), integrationH.Status)
				integration.GET("/xuegong/*path", auth.RequireCapability(auth.CounselorIntegrationRead), integrationH.ProxyXuegong)
				integration.GET("/ybt/*path", auth.RequireCapability(auth.CounselorIntegrationRead), integrationH.ProxyYBT)
			}

			secured.GET("/user/profile", authH.Profile)
			secured.POST("/auth/qr-confirm", handler.ConfirmQRSession)
			secured.POST("/user/consent", authH.Consent)
			secured.PUT("/user/password", authH.ChangePassword)

			// 语音配置
			secured.GET("/user/voice-config", authH.GetVoiceConfig)
			secured.PUT("/user/voice-config", authH.UpdateVoiceConfig)

			// 当前用户能力清单（基于角色继承自动展开）
			secured.GET("/user/capabilities", authH.GetCapabilities)

			// AI 模型配置
			secured.GET("/user/model-config", modelConfigH.Get)
			secured.PUT("/user/model-config", modelConfigH.Save)

			// ── 管理端 ──
			admin := secured.Group("/admin")
			{
				admin.GET("/metrics", auth.RequireCapability(auth.CollegeMetricsRead), adminH.GetMetrics)
				admin.GET("/users", auth.RequireCapability(auth.CollegeUserRead), adminH.ListUsers)
				admin.GET("/audit", auth.RequireCapability(auth.CollegeAuditRead), adminH.ListAudit)

				admin.PUT("/users/:id", auth.RequireCapability(auth.SchoolUserUpdate), adminH.UpdateUser)
				admin.PUT("/users/:id/password", auth.RequireCapability(auth.SystemPasswordReset), adminH.ResetUserPassword)
				admin.GET("/settings", auth.RequireCapability(auth.SystemSettingsWrite), adminH.GetSettings)
				admin.PUT("/settings", auth.RequireCapability(auth.SystemSettingsWrite), adminH.UpdateSettings)
			}

			// ── 知识审核 ──
			review := secured.Group("/review")
			{
				review.GET("/pending", auth.RequireCapability(auth.CounselorReviewPending), kbH.ListPendingReviews)
			}

			// ── 反馈 ──
			secured.POST("/feedback", auth.RequireCapability(auth.SelfFeedbackSubmit), feedbackH.Submit)
			secured.POST("/feedback/screenshot", auth.RequireCapability(auth.SelfFeedbackSubmit), feedbackH.UploadScreenshot)

			feedback := secured.Group("/feedback")
			{
				feedback.GET("", auth.RequireCapability(auth.UnionFeedbackList), feedbackH.List)
				feedback.PUT("/:id", auth.RequireCapability(auth.UnionFeedbackList), feedbackH.Resolve)
			}

			// ── 学生 AI 功能（个人能力，所有角色继承自 student 都可用）──
			student := secured.Group("/student")
			{
				student.GET("/daily-briefing", auth.RequireCapability(auth.SelfBriefingRead), studentH.DailyBriefing)
				student.GET("/learning-diary", auth.RequireCapability(auth.SelfDiaryRead), studentH.LearningDiary)
				student.POST("/checkin", auth.RequireCapability(auth.SelfCheckinWrite), studentH.Checkin)
				student.GET("/checkin/history", auth.RequireCapability(auth.SelfCheckinWrite), studentH.CheckinHistory)
				student.GET("/digital-twin", auth.RequireCapability(auth.SelfTwinRead), studentH.DigitalTwin)
				student.GET("/personality", auth.RequireCapability(auth.SelfPersonalityRead), studentH.Personality)
				student.GET("/achievements", auth.RequireCapability(auth.SelfAchievements), studentH.Achievements)
				student.GET("/course-map", auth.RequireCapability(auth.SelfCourseMapRead), studentH.CourseMap)
				student.GET("/course-analytics", auth.RequireCapability(auth.SelfCourseAnalytics), studentH.CourseAnalytics)
				student.GET("/weekly-report", auth.RequireCapability(auth.SelfWeeklyReport), studentH.WeeklyReport)
				student.GET("/freshman-plan", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("freshman-plan"))
				student.GET("/growth-path", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("growth-path"))
				student.GET("/political-study", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("political-study"))
				student.GET("/ideological-record", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("ideological-record"))
				student.GET("/party-progress", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("party-progress"))
				student.GET("/campus-life", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("campus-life"))
				student.GET("/schedule", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("schedule"))
				student.GET("/competition-match", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("competition-match"))
				student.GET("/study-buddy", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("study-buddy"))
				student.GET("/mental-health", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("mental-health"))
				student.GET("/digital-mentor", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("digital-mentor"))
				student.GET("/qa-plaza", auth.RequireCapability(auth.SelfCommunityRead), studentH.QAPlaza)
				student.GET("/hot-topics", auth.RequireCapability(auth.SelfCommunityRead), studentH.HotTopics)
				student.GET("/qa-leaderboard", auth.RequireCapability(auth.SelfCommunityRead), studentH.QALeaderboard)
				student.GET("/private-chat", auth.RequireCapability(auth.SelfPrivateChat), studentH.PrivateChat)
				student.GET("/process-enhanced", auth.RequireCapability(auth.SelfProcessRead), studentH.ProcessEnhanced)
			}

			// ── 个人通用入口 /me/* —— 与 /student/* 个人能力等价的语义化别名 ──
			// 适用于"任何角色访问自己的"场景，避免高阶用户访问 /student/ 路径的语义违和
			me := secured.Group("/me")
			{
				me.GET("/daily-briefing", auth.RequireCapability(auth.SelfBriefingRead), studentH.DailyBriefing)
				me.GET("/learning-diary", auth.RequireCapability(auth.SelfDiaryRead), studentH.LearningDiary)
				me.GET("/digital-twin", auth.RequireCapability(auth.SelfTwinRead), studentH.DigitalTwin)
				me.GET("/personality", auth.RequireCapability(auth.SelfPersonalityRead), studentH.Personality)
				me.GET("/achievements", auth.RequireCapability(auth.SelfAchievements), studentH.Achievements)
				me.GET("/weekly-report", auth.RequireCapability(auth.SelfWeeklyReport), studentH.WeeklyReport)
				me.POST("/checkin", auth.RequireCapability(auth.SelfCheckinWrite), studentH.Checkin)
				me.GET("/checkin/history", auth.RequireCapability(auth.SelfCheckinWrite), studentH.CheckinHistory)
			}

			// ── 辅导员 AI 功能 ──
			counselor := secured.Group("/counselor")
			{
				counselor.GET("/daily-focus", auth.RequireCapability(auth.CounselorDailyFocusRead), counselorH.DailyFocus)
				counselor.GET("/class-report", auth.RequireCapability(auth.CounselorClassReport), counselorH.ClassReport)
				counselor.GET("/twin-board", auth.RequireCapability(auth.CounselorTwinBoard), counselorH.TwinBoard)
				counselor.GET("/prediction", auth.RequireCapability(auth.CounselorPredictionRead), counselorH.Prediction)
				counselor.POST("/intervention", auth.RequireCapability(auth.CounselorInterventionWrite), counselorH.Intervention)
				counselor.GET("/talk-record", auth.RequireCapability(auth.CounselorTalkRecord), counselorH.TalkRecord)
				counselor.POST("/talk-record", auth.RequireCapability(auth.CounselorTalkRecord), counselorH.TalkRecord)
				counselor.GET("/talk-tips", auth.RequireCapability(auth.CounselorTalkTips), counselorH.TalkTips)
				counselor.GET("/ideological", auth.RequireCapability(auth.CounselorIdeological), counselorH.Ideological)
				counselor.GET("/class-profile", auth.RequireCapability(auth.CounselorClassProfile), counselorH.ClassProfile)
				counselor.GET("/community-manage", auth.RequireCapability(auth.CounselorCommunityManage), counselorH.CommunityManage)
				counselor.GET("/hot-topic-sense", auth.RequireCapability(auth.CounselorHotTopicSense), counselorH.HotTopicSense)
				counselor.GET("/process-edit", auth.RequireCapability(auth.CounselorProcessEdit), counselorH.ProcessEdit)
				counselor.GET("/student-list", auth.RequireCapability(auth.CounselorStudentList), counselorH.StudentList)
			}

			// ── 教师 AI 功能 ──
			teacher := secured.Group("/teacher")
			{
				teacher.GET("/daily-overview", auth.RequireCapability(auth.TeacherDailyOverview), teacherH.DailyOverview)
				teacher.POST("/lesson-prep", auth.RequireCapability(auth.TeacherLessonPrep), teacherH.LessonPrep)
				teacher.POST("/exam-gen", auth.RequireCapability(auth.TeacherExamGen), teacherH.ExamGen)
				teacher.POST("/class-interact", auth.RequireCapability(auth.TeacherClassInteract), teacherH.ClassInteract)
				teacher.POST("/grading", auth.RequireCapability(auth.TeacherGrading), teacherH.Grading)
				teacher.GET("/heatmap", auth.RequireCapability(auth.TeacherHeatmapRead), teacherH.Heatmap)
				teacher.GET("/reflection", auth.RequireCapability(auth.TeacherReflection), teacherH.Reflection)
				teacher.GET("/style-distribution", auth.RequireCapability(auth.TeacherStyleDist), teacherH.StyleDist)
				teacher.GET("/community-qa", auth.RequireCapability(auth.TeacherCommunityQA), teacherH.CommunityQA)
			}

			// ── 教辅 AI 功能 ──
			assistantGroup := secured.Group("/assistant")
			{
				assistantGroup.GET("/schedule-check", auth.RequireCapability(auth.AssistantScheduleCheck), assistantH.ScheduleCheck)
				assistantGroup.GET("/graduation-audit", auth.RequireCapability(auth.AssistantGradAudit), assistantH.GradAudit)
				assistantGroup.GET("/exam-arrange", auth.RequireCapability(auth.AssistantExamArrange), assistantH.ExamArrange)
			}

			// ── 学生会 AI 功能 ──
			unionGroup := secured.Group("/union")
			{
				unionGroup.GET("/event-plan", auth.RequireCapability(auth.UnionEventPlan), unionH.EventPlan)
				unionGroup.GET("/poster-gen", auth.RequireCapability(auth.UnionPosterGen), unionH.PosterGen)
			}

			// ── 学院管理员 AI 功能 ──
			collegeGroup := secured.Group("/college")
			{
				collegeGroup.GET("/twin-screen", auth.RequireCapability(auth.CollegeTwinScreen), collegeH.TwinScreen)
				collegeGroup.GET("/data-analysis", auth.RequireCapability(auth.CollegeDataAnalysis), collegeH.DataAnalysis)
			}

			// ── 校园文化智能体（全员可见）──
			cultureGroup := secured.Group("/culture")
			{
				cultureGroup.GET("/anthems", auth.RequireCapability(auth.SelfCultureAnthem), cultureH.Anthems)
				cultureGroup.GET("/radio", auth.RequireCapability(auth.SelfCultureRadio), cultureH.Radio)
				cultureGroup.GET("/lectures", auth.RequireCapability(auth.SelfCultureLectures), cultureH.Lectures)
				cultureGroup.GET("/events", auth.RequireCapability(auth.SelfCultureEvents), cultureH.Events)
				cultureGroup.GET("/volunteer", auth.RequireCapability(auth.SelfCultureVolunteer), cultureH.Volunteer)
			}
		}
	}

	return router
}

func healthHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbOK := "ok"
		if err := db.Ping(); err != nil {
			dbOK = "error: " + err.Error()
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "running",
			"service": "蔚小芯",
			"version": "0.1.0",
			"db":      dbOK,
			"time":    time.Now().Format(time.RFC3339),
		})
	}
}

func placeholderHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"code":    501,
			"message": name + " 待实现",
		})
	}
}
