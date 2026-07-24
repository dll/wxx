package app

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	tokenUsageRepo := repository.NewTokenUsageRepo(db)
	processRecordRepo := repository.NewProcessRecordRepo(db)
	feedbackScreenshotRepo := repository.NewFeedbackScreenshotRepo(db)
	forecastRepo := repository.NewForecastRepo(db)
	graduationRepo := repository.NewGraduationRepo(db)
	studentFeaturesRepo := repository.NewStudentFeaturesRepo(db)

	// ── 服务层 ──
	graduationService := service.NewGraduationService(graduationRepo)
	studentFeaturesService := service.NewStudentFeaturesService(studentFeaturesRepo)

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
	forecastSvc := service.NewForecastService(db, forecastRepo, emotionRepo, feedbackRepo, llmClient)

	chatSvc := service.NewChatService(sessionRepo, messageRepo, kbRepo, agentRepo, llmClient)
	if llmClient != nil {
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
	feedbackSvc := service.NewFeedbackService(feedbackRepo, userRepo, feedbackScreenshotRepo)
	modelConfigSvc := service.NewModelConfigService(modelConfigRepo)
	tokenStatsSvc := service.NewTokenStatsService(tokenUsageRepo, userRepo)
	processRecordSvc := service.NewProcessRecordService(processRecordRepo, kbRepo)
	notificationSvc := service.NewNotificationService(db, cfg.QQWebhookURL, cfg.WechatWebhookURL)
	uploadDir := "./data/uploads"
	if os.Getenv("VERCEL") != "" {
		uploadDir = "/tmp/uploads"
		log.Printf("Vercel 环境：上传目录 %s", uploadDir)
	}
	docSvc := service.NewDocumentService(uploadDir, 50)
	if chatSvc != nil {
		chatSvc.SetTokenStatsService(tokenStatsSvc)
		// 反馈"回答有误"时，立即把对应 FAQ 缓存标为 retired
		feedbackSvc.SetAnswerErrorHook(func(messageID, _ string) {
			if messageID == "" {
				return
			}
			id, err := strconv.ParseInt(messageID, 10, 64)
			if err != nil {
				return
			}
			question, err := messageRepo.GetUserQuestionByMessageID(id)
			if err != nil || question == "" {
				return
			}
			_ = chatSvc.RetireFAQ(question)
		})
	}

	// Handler 层
	authHandler := handler.NewAuthHandler(authSvc)
	sessionHandler := handler.NewSessionHandler(sessionSvc)
	kbHandler := handler.NewKBHandler(kbSvc)

	chatHandler := handler.NewChatHandler(chatSvc)
	if emotionSvc != nil {
		chatHandler.SetEmotionService(emotionSvc)
	}

	var voiceHandler *handler.VoiceHandler
	if xfyunClient != nil {
		voiceSvc := service.NewVoiceService(xfyunClient)
		voiceHandler = handler.NewVoiceHandler(voiceSvc)
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
	tokenStatsHandler := handler.NewTokenStatsHandler(tokenStatsSvc)
	processRecordHandler := handler.NewProcessRecordHandler(processRecordSvc)
	studentHandler := handler.NewStudentHandler(studentSvc)
	studentHandler.SetKBRepo(kbRepo)
	counselorHandler := handler.NewCounselorHandler(counselorSvc)

	var teacherSvc *service.TeacherService
	if llmClient != nil {
		teacherSvc = service.NewTeacherService(llmClient)
		log.Println("教师 AI 服务已启用")
	}
	teacherHandler := handler.NewTeacherHandler(teacherSvc)

	var assistantSvc *service.AssistantService
	if llmClient != nil {
		assistantSvc = service.NewAssistantService(llmClient)
	}
	assistantHandler := handler.NewAssistantHandler(assistantSvc)

	var unionSvc *service.UnionService
	if llmClient != nil {
		unionSvc = service.NewUnionService(llmClient)
	}
	unionHandler := handler.NewUnionHandler(unionSvc)

	var collegeSvc *service.CollegeService
	if llmClient != nil {
		collegeSvc = service.NewCollegeService(llmClient)
	}
	collegeHandler := handler.NewCollegeHandler(collegeSvc)
	cultureHandler := handler.NewCultureHandler()

	var schoolAdminSvc *service.SchoolAdminService
	if llmClient != nil {
		schoolAdminSvc = service.NewSchoolAdminService(llmClient)
	}
	schoolAdminHandler := handler.NewSchoolAdminHandler(schoolAdminSvc)

	var sysAdminSvc *service.SysAdminService
	if llmClient != nil {
		sysAdminSvc = service.NewSysAdminService(llmClient)
	}
	sysAdminHandler := handler.NewSysAdminHandler(sysAdminSvc)
	forecastHandler := handler.NewForecastHandler(forecastSvc)
	notificationHandler := handler.NewNotificationHandler(notificationSvc)
	uploadHandler := handler.NewUploadHandler(docSvc, kbSvc)
	graduationHandler := handler.NewGraduationHandler(graduationService)
	studentFeaturesHandler := handler.NewStudentFeaturesHandler(studentFeaturesService)
	educationHandler := handler.NewEducationHandler(db)
	studyPlanHandler := handler.NewStudyPlanHandler(db, llmClient)

	// ── 5. 构建路由 ──
	router := setupRouter(cfg, db, userRepo, authHandler, sessionHandler, chatHandler, kbHandler,
		voiceHandler, emotionHandler, agentHandler, exportHandler, integrationHandler, recHandler,
		adminHandler, feedbackHandler, modelConfigHandler, tokenStatsHandler,
		studentHandler, counselorHandler, teacherHandler, assistantHandler, unionHandler, collegeHandler,
		cultureHandler, schoolAdminHandler, sysAdminHandler, processRecordHandler, forecastHandler, graduationHandler, studentFeaturesHandler, notificationHandler, uploadHandler, educationHandler, studyPlanHandler)

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
			// ALTER TABLE ADD COLUMN 重复列名视为非致命错误（列已存在 = 目标状态）
			if isDuplicateColumnError(err) && strings.Contains(strings.ToUpper(stmt), "ALTER TABLE") {
				log.Printf("迁移 %s 第 %d 条语句跳过（列已存在）: %v", filename, i+1, err)
				continue
			}
			log.Printf("迁移 %s 第 %d 条语句失败: %v", filename, i+1, err)
			return err
		}
	}
	return nil
}

// isDuplicateColumnError 检测 SQLite "duplicate column name" 错误
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate column name") ||
		strings.Contains(msg, "duplicate column")
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
	userRepo *repository.UserRepo,
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
	tokenStatsH *handler.TokenStatsHandler,
	studentH *handler.StudentHandler,
	counselorH *handler.CounselorHandler,
	teacherH *handler.TeacherHandler,
	assistantH *handler.AssistantHandler,
	unionH *handler.UnionHandler,
	collegeH *handler.CollegeHandler,
	cultureH *handler.CultureHandler,
	schoolAdminH *handler.SchoolAdminHandler,
	sysAdminH *handler.SysAdminHandler,
	processRecordH *handler.ProcessRecordHandler,
	forecastH *handler.ForecastHandler,
	graduationH *handler.GraduationHandler,
	studentFeaturesH *handler.StudentFeaturesHandler,
	notificationH *handler.NotificationHandler,
	uploadH *handler.UploadHandler,
	educationH *handler.EducationHandler,
	studyPlanH *handler.StudyPlanHandler,
) *gin.Engine {
	router := gin.New()

	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.TraceID())
	router.Use(middleware.PIIMask()) // PII 检测与脱敏（在请求进入 handler 前检测并脱敏）
	router.Use(gin.Logger())
	router.Use(middleware.AuditLog(db))

	// 反馈截图：从 SQLite blob 流式输出，跨 Vercel 实例可读（取代曾经的本地文件 /uploads）
	router.GET("/uploads/feedback/:filename", feedbackH.ServeScreenshot)

	// 根路由
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "蔚小芯",
			"version": "0.0.1",
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
			authGroup.POST("/send-code", authH.SendCode)
			authGroup.POST("/guest-register", authH.GuestRegister)
		}

		// 需要 JWT 认证
		secured := v1.Group("/")
		secured.Use(middleware.JWTAuth(cfg))
		secured.Use(middleware.EnsureUserExists(userRepo))
		{
			// ── AI 对话（self.chat）──
			secured.POST("/chat", auth.RequireCapability(auth.SelfChat), chatH.Ask)

			// ── 会话/知识/推荐（self.* 能力）──
			secured.GET("/sessions", auth.RequireCapability(auth.SelfSessionRead), sessionH.ListSessions)
			secured.GET("/sessions/:id/messages", auth.RequireCapability(auth.SelfSessionRead), sessionH.GetMessages)
			secured.DELETE("/sessions/:id", auth.RequireCapability(auth.SelfSessionDelete), sessionH.DeleteSession)
			secured.PATCH("/sessions/:id", auth.RequireCapability(auth.SelfSessionRead), sessionH.RenameSession)
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

			// ── 问题预案（forecast.*）──
			forecast := secured.Group("/forecast")
			{
				forecast.POST("/analysis", auth.RequireCapability(auth.CollegeForecast), forecastH.Analyze)
				forecast.GET("/issues", auth.RequireCapability(auth.CollegeForecast), forecastH.ListForecasts)
				forecast.GET("/issues/:id", auth.RequireCapability(auth.CollegeForecast), forecastH.GetForecast)
				forecast.PUT("/issues/:id/status", auth.RequireCapability(auth.CollegeForecast), forecastH.UpdateStatus)
				forecast.GET("/statistics", auth.RequireCapability(auth.CollegeForecast), forecastH.GetStatistics)
			}

			// ── 毕设选题（graduation.*）──
			graduation := secured.Group("/graduation")
			{
				graduation.GET("/advisors", auth.RequireCapability(auth.SelfGraduationRead), graduationH.ListAdvisors)
				graduation.GET("/topics", auth.RequireCapability(auth.SelfGraduationRead), graduationH.ListTopics)
				graduation.GET("/topics/:id", auth.RequireCapability(auth.SelfGraduationRead), graduationH.GetTopic)
				graduation.POST("/select", auth.RequireCapability(auth.SelfGraduationWrite), graduationH.SelectTopic)
				graduation.GET("/my-selection", auth.RequireCapability(auth.SelfGraduationRead), graduationH.GetMySelection)
				graduation.GET("/milestones", auth.RequireCapability(auth.SelfGraduationRead), graduationH.ListMilestones)
				graduation.GET("/stats", auth.RequireCapability(auth.SelfGraduationRead), graduationH.GetStats)
				graduation.GET("/selections", auth.RequireCapability(auth.CollegeGraduationRead), graduationH.ListSelections)
				graduation.PUT("/selections/:id/confirm", auth.RequireCapability(auth.CollegeGraduationWrite), graduationH.ConfirmSelection)
			}

			// ── 学科竞赛 ──
			competition := secured.Group("/competition")
			{
				competition.GET("/list", auth.RequireCapability(auth.SelfCompetitionRead), studentFeaturesH.ListCompetitions)
				competition.GET("/:id", auth.RequireCapability(auth.SelfCompetitionRead), studentFeaturesH.GetCompetition)
				competition.POST("/register", auth.RequireCapability(auth.SelfCompetitionWrite), studentFeaturesH.RegisterCompetition)
				competition.GET("/my-registrations", auth.RequireCapability(auth.SelfCompetitionRead), studentFeaturesH.GetMyCompetitionRegistrations)
				competition.POST("/submit-work", auth.RequireCapability(auth.SelfCompetitionWrite), studentFeaturesH.SubmitWork)
				competition.GET("/stats", auth.RequireCapability(auth.SelfCompetitionRead), studentFeaturesH.GetCompetitionStats)
			}

			// ── 大学规划 ──
			plan := secured.Group("/plan")
			{
				plan.GET("/templates", auth.RequireCapability(auth.SelfPlanRead), studentFeaturesH.ListPlanTemplates)
				plan.GET("/my-plans", auth.RequireCapability(auth.SelfPlanRead), studentFeaturesH.ListMyPlans)
				plan.POST("/create", auth.RequireCapability(auth.SelfPlanWrite), studentFeaturesH.CreatePlan)
				plan.PUT("/:id/submit", auth.RequireCapability(auth.SelfPlanWrite), studentFeaturesH.SubmitPlan)
				plan.PUT("/:id/review", auth.RequireCapability(auth.CounselorKBWrite), studentFeaturesH.ReviewPlan)
			}

			// ── 入党教育 ──
			party := secured.Group("/party")
			{
				party.GET("/stages", auth.RequireCapability(auth.SelfPartyRead), studentFeaturesH.ListPartyStages)
				party.GET("/my-progress", auth.RequireCapability(auth.SelfPartyRead), studentFeaturesH.GetMyPartyProgress)
				party.PUT("/my-progress", auth.RequireCapability(auth.SelfPartyWrite), studentFeaturesH.UpdatePartyProgress)
				party.GET("/my-study-records", auth.RequireCapability(auth.SelfPartyRead), studentFeaturesH.ListMyStudyRecords)
				party.POST("/study-record", auth.RequireCapability(auth.SelfPartyWrite), studentFeaturesH.AddStudyRecord)
				party.GET("/stats", auth.RequireCapability(auth.SelfPartyRead), studentFeaturesH.GetPartyStats)
			}

			// ── 社团生活 ──
			club := secured.Group("/club")
			{
				club.GET("/list", auth.RequireCapability(auth.SelfClubRead), studentFeaturesH.ListClubs)
				club.GET("/:id", auth.RequireCapability(auth.SelfClubRead), studentFeaturesH.GetClub)
				club.POST("/join", auth.RequireCapability(auth.SelfClubWrite), studentFeaturesH.JoinClub)
				club.GET("/my-clubs", auth.RequireCapability(auth.SelfClubRead), studentFeaturesH.GetMyClubs)
				club.GET("/activities", auth.RequireCapability(auth.SelfClubRead), studentFeaturesH.ListClubActivities)
				club.POST("/activity/register", auth.RequireCapability(auth.SelfClubWrite), studentFeaturesH.RegisterClubActivity)
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

				// 高级查询与字典
				kb.GET("/resources/advanced", auth.RequireCapability(auth.CounselorKBWrite), kbH.ListResourcesAdvanced)
				kb.GET("/dict", auth.RequireCapability(auth.CounselorKBWrite), kbH.GetDictValues)
				kb.GET("/stats", auth.RequireCapability(auth.CounselorKBWrite), kbH.GetStats)

				// 批量操作（counselor.kb.review）
				kb.POST("/batch/approve", auth.RequireCapability(auth.CounselorKBReview), kbH.BatchApprove)
				kb.POST("/batch/reject", auth.RequireCapability(auth.CounselorKBReview), kbH.BatchReject)
				kb.POST("/batch/retire", auth.RequireCapability(auth.CounselorKBReview), kbH.BatchRetire)
				kb.POST("/batch/delete", auth.RequireCapability(auth.CounselorKBWrite), kbH.BatchDelete)

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

			// ── 词元统计 ──
			secured.GET("/token-stats/my", auth.RequireCapability(auth.SelfTokenStats), tokenStatsH.GetMyStats)
			secured.GET("/token-stats/subordinates", auth.RequireCapability(auth.CounselorTokenSubordinates), tokenStatsH.GetSubordinateStats)

			// ── 管理端 ──
			admin := secured.Group("/admin")
			{
				admin.GET("/metrics", auth.RequireCapability(auth.CollegeMetricsRead), adminH.GetMetrics)
				admin.GET("/users", auth.RequireCapability(auth.CollegeUserRead), adminH.ListUsers)
				admin.GET("/audit", auth.RequireCapability(auth.CollegeAuditRead), adminH.ListAudit)

				admin.PUT("/users/:id", auth.RequireCapability(auth.SchoolUserUpdate), adminH.UpdateUser)
				admin.DELETE("/users/:id", auth.RequireCapability(auth.SchoolUserUpdate), adminH.DeleteUser)
				admin.PUT("/users/:id/password", auth.RequireCapability(auth.SystemPasswordReset), adminH.ResetUserPassword)
				admin.GET("/users/advanced", auth.RequireCapability(auth.CollegeUserRead), adminH.ListUsersAdvanced)
				admin.GET("/users/dict", auth.RequireCapability(auth.CollegeUserRead), adminH.GetUserDict)
				admin.POST("/users/batch/status", auth.RequireCapability(auth.SchoolUserUpdate), adminH.BatchUpdateStatus)
				admin.POST("/users/batch/password", auth.RequireCapability(auth.SystemPasswordReset), adminH.BatchResetPassword)
				admin.POST("/users/batch/delete", auth.RequireCapability(auth.SchoolUserUpdate), adminH.BatchDelete)
				admin.GET("/settings", auth.RequireCapability(auth.SystemSettingsWrite), adminH.GetSettings)
				admin.PUT("/settings", auth.RequireCapability(auth.SystemSettingsWrite), adminH.UpdateSettings)

				// 游客管理（college_admin+）
				admin.GET("/guests/pending", auth.RequireCapability(auth.CollegeUserRead), adminH.ListPendingGuests)
				admin.PUT("/guests/:id/approve", auth.RequireCapability(auth.CollegeUserRead), adminH.ApproveGuest)
				admin.PUT("/guests/:id/reject", auth.RequireCapability(auth.CollegeUserRead), adminH.RejectGuest)
				// 学生导入（除学生和游客外的组织角色均可用）
				admin.POST("/users/import", auth.RequireCapability(auth.CounselorImportStudent), adminH.ImportStudents)
			}

			// ── 知识审核 ──
			review := secured.Group("/review")
			{
				review.GET("/pending", auth.RequireCapability(auth.CounselorReviewPending), kbH.ListPendingReviews)
			}

			// ── 反馈 ──
			secured.POST("/feedback", auth.RequireCapability(auth.SelfFeedbackSubmit), feedbackH.Submit)
			secured.POST("/feedback/screenshot", auth.RequireCapability(auth.SelfFeedbackSubmit), feedbackH.UploadScreenshot)
			// 我的反馈：所有登录用户都能查看自己提交的反馈（按 user_id 过滤）
			secured.GET("/feedback/mine", auth.RequireCapability(auth.SelfFeedbackSubmit), feedbackH.Mine)
			// 管理员反馈列表和处理（注意：直接注册避免 Group 产生的尾部斜杠重定向丢 Authorization 头）
			secured.GET("/feedback", auth.RequireCapability(auth.UnionFeedbackList), feedbackH.List)
			secured.PUT("/feedback/:id", auth.RequireCapability(auth.UnionFeedbackList), feedbackH.Resolve)

			// ── 办事流程办理记录 ──
			process := secured.Group("/process/records")
			{
				process.GET("", auth.RequireCapability(auth.SelfProcessRead), processRecordH.ListMine)
				process.POST("/:flow/start", auth.RequireCapability(auth.SelfProcessRead), processRecordH.StartOrResume)
				process.POST("/:flow/progress", auth.RequireCapability(auth.SelfProcessRead), processRecordH.UpdateProgress)
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
				// ── P2 学生深度分析 ──
				student.GET("/values-guidance", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("values-guidance"))
				student.GET("/classroom-extension", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("classroom-extension"))
				student.GET("/mock-interview", auth.RequireCapability(auth.SelfGenericAI), studentH.MockInterview)
				student.GET("/resume", auth.RequireCapability(auth.SelfGenericAI), studentH.Resume)
				student.GET("/career-simulation", auth.RequireCapability(auth.SelfGenericAI), studentH.CareerSimulation)
				student.GET("/study-buddy-match", auth.RequireCapability(auth.SelfGenericAI), studentH.StudyBuddyMatch)
				student.GET("/mental-health-report", auth.RequireCapability(auth.SelfGenericAI), studentH.MentalHealthReport)
				student.GET("/note-assistant", auth.RequireCapability(auth.SelfGenericAI), studentH.NoteAssistant)
				student.GET("/alumni-match", auth.RequireCapability(auth.SelfGenericAI), studentH.AlumniMatch)
				// ── P3 生态扩展 ──
				student.GET("/dynamic-mentor", auth.RequireCapability(auth.SelfGenericAI), studentH.DynamicMentor)
				student.GET("/career-sim-enhanced", auth.RequireCapability(auth.SelfGenericAI), studentH.EnhancedCareerSim)
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
				// ── P2 辅导员深度分析 ──
				counselor.GET("/follow-up-reminders", auth.RequireCapability(auth.CounselorTalkRecord), counselorH.FollowUpReminders)
				counselor.GET("/checkin-stats", auth.RequireCapability(auth.CounselorClassReport), counselorH.CheckinStats)
				counselor.POST("/smart-notify", auth.RequireCapability(auth.CounselorInterventionWrite), counselorH.SmartNotify)
				counselor.GET("/monthly-brief", auth.RequireCapability(auth.CounselorClassReport), counselorH.MonthlyBrief)
				counselor.POST("/session-insight", auth.RequireCapability(auth.CounselorTalkRecord), counselorH.SessionInsight)
			}

			// ── 通知推送（辅导员及以上角色） ──
			secured.POST("/notifications", auth.RequireCapability(auth.CounselorNotify), notificationH.Create)
			secured.GET("/notifications", auth.RequireCapability(auth.CounselorNotify), notificationH.List)
			secured.POST("/notifications/:id/publish", auth.RequireCapability(auth.CounselorNotify), notificationH.Publish)
			secured.DELETE("/notifications/:id", auth.RequireCapability(auth.CounselorNotify), notificationH.Delete)
			secured.GET("/notifications/webhook-status", auth.RequireCapability(auth.CounselorNotify), notificationH.WebhookStatus)

			// ── 文档上传与知识入库 ──
			secured.POST("/kb/upload", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), uploadH.Upload)
			secured.GET("/kb/formats", uploadH.SupportedFormats)

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
				// ── P2 教师深度分析 ──
				teacher.GET("/faq-knowledge", auth.RequireCapability(auth.TeacherCommunityQA), teacherH.FAQKnowledge)
				teacher.GET("/student-twin", auth.RequireCapability(auth.TeacherHeatmapRead), teacherH.StudentTwin)
				teacher.GET("/knowledge-coverage", auth.RequireCapability(auth.TeacherLessonPrep), teacherH.KnowledgeCoverage)
				teacher.GET("/ideological-suggestions", auth.RequireCapability(auth.TeacherReflection), teacherH.IdeologicalSuggestions)
				teacher.GET("/personalized-teaching", auth.RequireCapability(auth.TeacherStyleDist), teacherH.PersonalizedTeaching)
			}

			// ── 教辅 AI 功能 ──
			assistantGroup := secured.Group("/assistant")
			{
				assistantGroup.GET("/schedule-check", auth.RequireCapability(auth.AssistantScheduleCheck), assistantH.ScheduleCheck)
				assistantGroup.GET("/graduation-audit", auth.RequireCapability(auth.AssistantGradAudit), assistantH.GradAudit)
				assistantGroup.GET("/exam-arrange", auth.RequireCapability(auth.AssistantExamArrange), assistantH.ExamArrange)
				// ── P2 教辅深度分析 ──
				assistantGroup.POST("/notification", auth.RequireCapability(auth.AssistantScheduleCheck), assistantH.Notification)
				assistantGroup.GET("/teaching-calendar", auth.RequireCapability(auth.AssistantScheduleCheck), assistantH.TeachingCalendar)
				assistantGroup.GET("/student-info", auth.RequireCapability(auth.AssistantGradAudit), assistantH.StudentInfoQuery)
			}

			// ── 学生会 AI 功能 ──
			unionGroup := secured.Group("/union")
			{
				unionGroup.GET("/event-plan", auth.RequireCapability(auth.UnionEventPlan), unionH.EventPlan)
				unionGroup.GET("/poster-gen", auth.RequireCapability(auth.UnionPosterGen), unionH.PosterGen)
				// ── P2 学生会深度分析 ──
				unionGroup.GET("/recruitment", auth.RequireCapability(auth.UnionEventPlan), unionH.Recruitment)
				unionGroup.GET("/member-manage", auth.RequireCapability(auth.UnionEventPlan), unionH.MemberManage)
				unionGroup.GET("/questionnaire", auth.RequireCapability(auth.UnionPosterGen), unionH.Questionnaire)
				unionGroup.GET("/hot-topic-track", auth.RequireCapability(auth.UnionEventPlan), unionH.HotTopicTrack)
				unionGroup.GET("/activity-analysis", auth.RequireCapability(auth.UnionEventPlan), unionH.ActivityAnalysis)
			}

			// ── 学院管理员 AI 功能 ──
			collegeGroup := secured.Group("/college")
			{
				collegeGroup.GET("/twin-screen", auth.RequireCapability(auth.CollegeTwinScreen), collegeH.TwinScreen)
				collegeGroup.GET("/data-analysis", auth.RequireCapability(auth.CollegeDataAnalysis), collegeH.DataAnalysis)
				// ── P2 学院管理员深度分析 ──
				collegeGroup.GET("/decision-advice", auth.RequireCapability(auth.CollegeDataAnalysis), collegeH.DecisionAdvice)
				collegeGroup.GET("/teacher-efficiency", auth.RequireCapability(auth.CollegeTwinScreen), collegeH.TeacherEfficiency)
				collegeGroup.GET("/course-quality", auth.RequireCapability(auth.CollegeDataAnalysis), collegeH.CourseQuality)
				collegeGroup.GET("/college-report", auth.RequireCapability(auth.CollegeTwinScreen), collegeH.CollegeReport)
			}

			// ── 学校管理员 AI 功能（P2）──
			schoolAdminGroup := secured.Group("/school-admin")
			{
				schoolAdminGroup.GET("/panorama", auth.RequireCapability(auth.CollegeTwinScreen), schoolAdminH.Panorama)
				schoolAdminGroup.POST("/policy-simulation", auth.RequireCapability(auth.CollegeDataAnalysis), schoolAdminH.PolicySimulation)
				schoolAdminGroup.GET("/college-comparison", auth.RequireCapability(auth.CollegeTwinScreen), schoolAdminH.CollegeComparison)
				schoolAdminGroup.GET("/academic-overview", auth.RequireCapability(auth.CollegeTwinScreen), schoolAdminH.AcademicOverview)
			}

			// ── 系统管理员 AI 功能（P2）──
			sysAdminGroup := secured.Group("/sys-admin")
			{
				sysAdminGroup.GET("/system-health", auth.RequireCapability(auth.CollegeTwinScreen), sysAdminH.SystemHealth)
				sysAdminGroup.GET("/knowledge-quality", auth.RequireCapability(auth.CollegeDataAnalysis), sysAdminH.KnowledgeQuality)
				sysAdminGroup.GET("/user-behavior", auth.RequireCapability(auth.CollegeTwinScreen), sysAdminH.UserBehavior)
			}

			// ── 就业指导模块（全员可见）──
			career := secured.Group("/career")
			{
				career.GET("/policies", auth.RequireCapability(auth.SelfCareerRead), educationH.ListCareerPolicies)
				career.GET("/policies/:id", auth.RequireCapability(auth.SelfCareerRead), educationH.GetCareerPolicy)
				career.GET("/jobs", auth.RequireCapability(auth.SelfCareerRead), educationH.ListJobPostings)
				career.GET("/jobs/:id", auth.RequireCapability(auth.SelfCareerRead), educationH.GetJobPosting)
				career.GET("/sessions", auth.RequireCapability(auth.SelfCareerRead), educationH.ListInfoSessions)
				career.GET("/interview/questions", auth.RequireCapability(auth.SelfCareerRead), educationH.ListInterviewQuestions)
			}

			// ── 学业学习模块（全员可见）──
			study := secured.Group("/study")
			{
				study.GET("/courses", auth.RequireCapability(auth.SelfStudyRead), educationH.ListCourses)
				study.GET("/courses/:id", auth.RequireCapability(auth.SelfStudyRead), educationH.GetCourse)
				study.GET("/grades", auth.RequireCapability(auth.SelfStudyRead), educationH.ListMyGrades)
				study.GET("/grades/summary", auth.RequireCapability(auth.SelfStudyRead), educationH.GetGradeSummary)
				study.GET("/resources", auth.RequireCapability(auth.SelfStudyRead), educationH.ListLearningResources)
				study.GET("/exams", auth.RequireCapability(auth.SelfStudyRead), educationH.ListExamSchedules)

				// ── 校历 / 课表 / 学习计划（study_plan_handler）──
				// 校历
				study.GET("/calendar/current", auth.RequireCapability(auth.SelfStudyRead), studyPlanH.GetCurrentCalendar)
				study.GET("/calendar/:semester_code", auth.RequireCapability(auth.SelfStudyRead), studyPlanH.GetCalendarBySemester)
				// 课表
				study.GET("/timetable", auth.RequireCapability(auth.SelfStudyRead), studyPlanH.GetMyTimetable)
				// 学习计划概览（用于多 Tab 首页）
				study.GET("/plans/overview", auth.RequireCapability(auth.SelfStudyRead), studyPlanH.GetPlansOverview)
				// AI 生成学习计划
				study.POST("/plans/ai-generate", auth.RequireCapability(auth.SelfStudyWrite), studyPlanH.AIGeneratePlan)
				// 学习计划 CRUD
				study.GET("/plans", auth.RequireCapability(auth.SelfStudyRead), studyPlanH.ListMyPlans)
				study.POST("/plans", auth.RequireCapability(auth.SelfStudyWrite), studyPlanH.CreatePlan)
				study.GET("/plans/:id", auth.RequireCapability(auth.SelfStudyRead), studyPlanH.GetPlan)
				study.PUT("/plans/:id", auth.RequireCapability(auth.SelfStudyWrite), studyPlanH.UpdatePlan)
				study.DELETE("/plans/:id", auth.RequireCapability(auth.SelfStudyWrite), studyPlanH.DeletePlan)
				// 计划任务
				study.POST("/plans/:id/tasks", auth.RequireCapability(auth.SelfStudyWrite), studyPlanH.AddTask)
				study.PUT("/plans/:id/tasks/:task_id", auth.RequireCapability(auth.SelfStudyWrite), studyPlanH.UpdateTask)
			}

			// ── 心理健康模块（全员可见）──
			mental := secured.Group("/mental")
			{
				mental.GET("/scales", auth.RequireCapability(auth.SelfMentalRead), educationH.ListPsychScales)
				mental.GET("/scales/:id", auth.RequireCapability(auth.SelfMentalRead), educationH.GetPsychScale)
				mental.POST("/assessments", auth.RequireCapability(auth.SelfMentalWrite), educationH.SubmitAssessment)
				mental.GET("/assessments", auth.RequireCapability(auth.SelfMentalRead), educationH.ListMyAssessments)
				mental.GET("/counselors", auth.RequireCapability(auth.SelfMentalRead), educationH.ListCounselors)
				mental.GET("/appointments", auth.RequireCapability(auth.SelfMentalRead), educationH.ListMyAppointments)
				mental.POST("/appointments", auth.RequireCapability(auth.SelfMentalWrite), educationH.CreateAppointment)
				mental.GET("/articles", auth.RequireCapability(auth.SelfMentalRead), educationH.ListPsychArticles)
				mental.GET("/articles/:id", auth.RequireCapability(auth.SelfMentalRead), educationH.GetPsychArticle)
				mental.GET("/hotlines", auth.RequireCapability(auth.SelfMentalRead), educationH.ListCrisisHotlines)
				mental.GET("/mood", auth.RequireCapability(auth.SelfMentalRead), educationH.ListMyMoodDiary)
				mental.POST("/mood", auth.RequireCapability(auth.SelfMentalWrite), educationH.CreateMoodDiary)
			}

			// ── 校园文化智能体（全员可见）──
			cultureGroup := secured.Group("/culture")
			{
				cultureGroup.GET("/anthems", auth.RequireCapability(auth.SelfCultureAnthem), cultureH.Anthems)
				cultureGroup.GET("/radio", auth.RequireCapability(auth.SelfCultureRadio), cultureH.Radio)
				cultureGroup.GET("/lectures", auth.RequireCapability(auth.SelfCultureLectures), cultureH.Lectures)
				cultureGroup.GET("/events", auth.RequireCapability(auth.SelfCultureEvents), cultureH.Events)
				cultureGroup.GET("/volunteer", auth.RequireCapability(auth.SelfCultureVolunteer), cultureH.Volunteer)
				// ── 第三方应用接入（全员可见）──
				appsGroup := secured.Group("/apps")
				{
					appsGroup.GET("", handler.NewExternalAppHandler().ListApps)
					appsGroup.GET("/:key", handler.NewExternalAppHandler().GetApp)
				}
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
			"version": "0.0.1",
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
