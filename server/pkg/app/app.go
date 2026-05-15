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
	// 注意：serverless 环境无法 defer db.Close()，依赖进程退出时 OS 清理

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
		// 实际启动在 cmd/server/main.go 中处理，此处仅占位
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
	}

	var emotionSvc *service.EmotionService
	if llmClient != nil {
		emotionSvc = service.NewEmotionService(emotionRepo, llmClient)
		log.Println("情感预警服务已启用")
	}

	agentSvc := service.NewAgentService(agentRepo)
	integrationSvc := service.NewIntegrationService(cfg)
	adminSvc := service.NewAdminService(userRepo, auditRepo, settingsRepo)
	feedbackSvc := service.NewFeedbackService(feedbackRepo)

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
	adminHandler := handler.NewAdminHandler(adminSvc)
	feedbackHandler := handler.NewFeedbackHandler(feedbackSvc)
	studentHandler := handler.NewStudentHandler()
	counselorHandler := handler.NewCounselorHandler()
	teacherHandler := handler.NewTeacherHandler()
	assistantHandler := handler.NewAssistantHandler()
	unionHandler := handler.NewUnionHandler()
	collegeHandler := handler.NewCollegeHandler()

	// ── 5. 构建路由 ──
	router := setupRouter(cfg, db, authHandler, sessionHandler, chatHandler, kbHandler,
		voiceHandler, emotionHandler, agentHandler, exportHandler, integrationHandler, recHandler,
		adminHandler, feedbackHandler, studentHandler, counselorHandler, teacherHandler, assistantHandler, unionHandler, collegeHandler)

	return router, nil
}

// initDB 初始化 SQLite 连接
func initDB(dbPath string) (*sql.DB, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	// modernc.org/sqlite 使用 _pragma 参数设置连接选项
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
	// 创建迁移记录表
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL UNIQUE,
		executed_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}

	// 读取嵌入的迁移文件
	entries, err := server.Migrations.ReadDir("migrations")
	if err != nil {
		return err
	}

	// 按文件名排序
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	executed := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// 检查是否已执行
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM _migrations WHERE filename = ?", entry.Name()).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		// 读取并执行
		content, err := server.Migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}

		if err := execSQL(db, string(content), entry.Name()); err != nil {
			return err
		}

		// 记录迁移
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
	studentH *handler.StudentHandler,
	counselorH *handler.CounselorHandler,
	teacherH *handler.TeacherHandler,
	assistantH *handler.AssistantHandler,
	unionH *handler.UnionHandler,
	collegeH *handler.CollegeHandler,
) *gin.Engine {
	router := gin.New()

	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.TraceID())
	router.Use(gin.Logger())
	router.Use(middleware.AuditLog(db))

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
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authH.Login)
		}

		// 需要 JWT 认证
		secured := v1.Group("/")
		secured.Use(middleware.JWTAuth(cfg))
		{
			if chatH != nil {
				secured.POST("/chat", chatH.Ask)
			} else {
				secured.POST("/chat", placeholderHandler("对话接口（LLM 未配置）"))
			}

			secured.GET("/sessions", sessionH.ListSessions)
			secured.GET("/sessions/:id/messages", sessionH.GetMessages)
			secured.DELETE("/sessions/:id", sessionH.DeleteSession)
			secured.GET("/knowledge", kbH.BrowseKnowledge)
			secured.GET("/recommendations", recH.GetRecommendations)

			if emotionH != nil {
				secured.GET("/emotion/stats", emotionH.GetStats)
			}

			if emotionH != nil {
				emotion := secured.Group("/emotion")
				emotion.Use(middleware.RequireRole("counselor"))
				{
					emotion.POST("/analyze", emotionH.Analyze)
					emotion.GET("/alerts", emotionH.ListAlerts)
					emotion.PUT("/alerts/:id", emotionH.UpdateAlert)
					emotion.GET("/trends", emotionH.Trends)
				}
			}

				kb := secured.Group("/kb")
				kb.Use(middleware.RequireRole("counselor"))
				{
					kb.GET("/resources", kbH.ListResources)
					kb.POST("/resources", kbH.CreateResource)
					kb.PUT("/resources/:id", kbH.UpdateResource)
					kb.GET("/resources/:id", kbH.GetResource)
					kb.POST("/import", kbH.Import)
					kb.POST("/validate", kbH.Validate)

					// 审核操作（counselor 及以上）
					kb.POST("/resources/:id/approve", kbH.ApproveResource)
					kb.POST("/resources/:id/reject", kbH.RejectResource)
					kb.POST("/resources/:id/retire", kbH.RetireResource)
				}

				// KB 提交审核（student_union 及以上）
				kbSubmit := secured.Group("/kb")
				kbSubmit.Use(middleware.RequireRole("student_union"))
				{
					kbSubmit.POST("/resources/:id/submit", kbH.SubmitForReview)
				}

			// 知识导出（所有认证用户可访问）
			secured.GET("/kb/export", exportH.Export)

			agents := secured.Group("/agents")
			agents.Use(middleware.RequireRole("school_admin"))
			{
				agents.GET("", agentH.List)
				agents.POST("", agentH.Create)
				agents.GET("/:id", agentH.Get)
				agents.PUT("/:id", agentH.Update)
				agents.DELETE("/:id", agentH.Delete)
			}

			if voiceH != nil {
				secured.POST("/voice/asr", voiceH.ASR)
				secured.POST("/voice/tts", voiceH.TTS)
			}

			secured.GET("/export", exportH.Export)
				secured.POST("/export/answer", exportH.ExportAnswer)

			integration := secured.Group("/integration")
			integration.Use(middleware.RequireRole("counselor"))
			{
				integration.GET("/status", integrationH.Status)
				integration.GET("/xuegong/*path", integrationH.ProxyXuegong)
				integration.GET("/ybt/*path", integrationH.ProxyYBT)
			}

			secured.GET("/user/profile", authH.Profile)
				secured.POST("/user/consent", authH.Consent)

				// ── 管理端（college_admin 及以上）──
				admin := secured.Group("/admin")
				admin.Use(middleware.RequireRole("college_admin"))
				{
					admin.GET("/metrics", adminH.GetMetrics)
					admin.GET("/users", adminH.ListUsers)
					admin.GET("/audit", adminH.ListAudit)

					// 学校管理员及以上可修改用户
					adminUsers := admin.Group("")
					adminUsers.Use(middleware.RequireRole("school_admin"))
					{
						adminUsers.PUT("/users/:id", adminH.UpdateUser)
					}

					// 系统管理员独占配置管理
					adminSettings := admin.Group("")
					adminSettings.Use(middleware.RequireRole("sys_admin"))
					{
						adminSettings.GET("/settings", adminH.GetSettings)
						adminSettings.PUT("/settings", adminH.UpdateSettings)
					}
				}

				// ── 知识审核（counselor 及以上）──
				review := secured.Group("/review")
				review.Use(middleware.RequireRole("counselor"))
				{
					review.GET("/pending", kbH.ListPendingReviews)
				}

				// ── 反馈（student 提交，student_union 查看/处理）──
				secured.POST("/feedback", feedbackH.Submit)

				feedback := secured.Group("/feedback")
				feedback.Use(middleware.RequireRole("student_union"))
				{
					feedback.GET("", feedbackH.List)
					feedback.PUT("/:id", feedbackH.Resolve)
				}

				// ── 学生 AI 功能（所有认证用户可访问）──
				student := secured.Group("/student")
				{
					student.GET("/daily-briefing", studentH.DailyBriefing)
					student.GET("/learning-diary", studentH.LearningDiary)
					student.POST("/checkin", studentH.Checkin)
					student.GET("/checkin/history", studentH.CheckinHistory)
					student.GET("/digital-twin", studentH.DigitalTwin)
					student.GET("/personality", studentH.Personality)
					student.GET("/achievements", studentH.Achievements)
					student.GET("/course-map", studentH.CourseMap)
					student.GET("/course-analytics", studentH.CourseAnalytics)
					student.GET("/weekly-report", studentH.WeeklyReport)
					student.GET("/freshman-plan", studentH.GenericAI("freshman-plan"))
					student.GET("/growth-path", studentH.GenericAI("growth-path"))
					student.GET("/political-study", studentH.GenericAI("political-study"))
					student.GET("/ideological-record", studentH.GenericAI("ideological-record"))
					student.GET("/party-progress", studentH.GenericAI("party-progress"))
					student.GET("/campus-life", studentH.GenericAI("campus-life"))
					student.GET("/schedule", studentH.GenericAI("schedule"))
					student.GET("/competition-match", studentH.GenericAI("competition-match"))
					student.GET("/study-buddy", studentH.GenericAI("study-buddy"))
					student.GET("/mental-health", studentH.GenericAI("mental-health"))
					student.GET("/digital-mentor", studentH.GenericAI("digital-mentor"))
				student.GET("/qa-plaza", studentH.QAPlaza)
				student.GET("/hot-topics", studentH.HotTopics)
				student.GET("/qa-leaderboard", studentH.QALeaderboard)
				student.GET("/private-chat", studentH.PrivateChat)
				student.GET("/process-enhanced", studentH.ProcessEnhanced)
				}

				// ── 辅导员 AI 功能（counselor 及以上）──
				counselor := secured.Group("/counselor")
				counselor.Use(middleware.RequireRole("counselor"))
				{
					counselor.GET("/daily-focus", counselorH.DailyFocus)
					counselor.GET("/class-report", counselorH.ClassReport)
					counselor.GET("/twin-board", counselorH.TwinBoard)
					counselor.GET("/prediction", counselorH.Prediction)
					counselor.POST("/intervention", counselorH.Intervention)
					counselor.GET("/talk-record", counselorH.TalkRecord)
					counselor.POST("/talk-record", counselorH.TalkRecord)
					counselor.GET("/talk-tips", counselorH.TalkTips)
					counselor.GET("/ideological", counselorH.Ideological)
					counselor.GET("/class-profile", counselorH.ClassProfile)
				counselor.GET("/community-manage", counselorH.CommunityManage)
				counselor.GET("/hot-topic-sense", counselorH.HotTopicSense)
				counselor.GET("/process-edit", counselorH.ProcessEdit)
				}

				// ── 教师 AI 功能（counselor 及以上，含 teacher 角色）──
				teacher := secured.Group("/teacher")
				teacher.Use(middleware.RequireRole("counselor"))
				{
					teacher.GET("/daily-overview", teacherH.DailyOverview)
					teacher.POST("/lesson-prep", teacherH.LessonPrep)
					teacher.POST("/exam-gen", teacherH.ExamGen)
					teacher.POST("/class-interact", teacherH.ClassInteract)
					teacher.POST("/grading", teacherH.Grading)
					teacher.GET("/heatmap", teacherH.Heatmap)
					teacher.GET("/reflection", teacherH.Reflection)
					teacher.GET("/style-distribution", teacherH.StyleDist)
				teacher.GET("/community-qa", teacherH.CommunityQA)
				}

			// ── 教辅 AI 功能（counselor 及以上）──
			assistantGroup := secured.Group("/assistant")
			assistantGroup.Use(middleware.RequireRole("counselor"))
				{
					assistantGroup.GET("/schedule-check", assistantH.ScheduleCheck)
					assistantGroup.GET("/graduation-audit", assistantH.GradAudit)
					assistantGroup.GET("/exam-arrange", assistantH.ExamArrange)
				}

			// ── 学生会 AI 功能（student_union 及以上）──
			unionGroup := secured.Group("/union")
			unionGroup.Use(middleware.RequireRole("student_union"))
				{
					unionGroup.GET("/event-plan", unionH.EventPlan)
					unionGroup.GET("/poster-gen", unionH.PosterGen)
				}

			// ── 学院管理员 AI 功能（college_admin 及以上）──
			collegeGroup := secured.Group("/college")
			collegeGroup.Use(middleware.RequireRole("college_admin"))
				{
					collegeGroup.GET("/twin-screen", collegeH.TwinScreen)
					collegeGroup.GET("/data-analysis", collegeH.DataAnalysis)
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
