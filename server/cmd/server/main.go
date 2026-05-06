package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/handler"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.Println("蔚小芯后端启动中...")

	// ── 1. 加载配置 ──
	cfg := config.Load()

	// ── 2. 设置运行模式 ──
	if cfg.AppMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// ── 3. 初始化 SQLite 数据库 ──
	db, err := initDB(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer db.Close()

	// ── 4. 初始化各层依赖 ──

	// Repository 层
	userRepo := repository.NewUserRepo(db)
	sessionRepo := repository.NewSessionRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	kbRepo := repository.NewKBRepo(db)
	emotionRepo := repository.NewEmotionRepo(db)
	agentRepo := repository.NewAgentRepo(db)

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

	// 讯飞语音客户端（ASR + TTS）
	var xfyunClient *llm.XfyunClient
	if cfg.XfyunAPIKey != "" && cfg.XfyunAPISecret != "" {
		xfyunClient = llm.NewXfyunClient(cfg)
		log.Println("讯飞语音客户端已启用")
	} else {
		log.Println("提示：未配置讯飞语音 API，语音功能不可用")
	}

	// Service 层
	authSvc := service.NewAuthService(cfg, userRepo)
	sessionSvc := service.NewSessionService(sessionRepo, messageRepo)
	kbSvc := service.NewKBService(kbRepo)
	var chatSvc *service.ChatService
	if llmClient != nil {
		chatSvc = service.NewChatService(sessionRepo, messageRepo, kbRepo, agentRepo, llmClient)
	}

	// 情感预警服务（依赖 LLM Client）
	var emotionSvc *service.EmotionService
	if llmClient != nil {
		emotionSvc = service.NewEmotionService(emotionRepo, llmClient)
		log.Println("情感预警服务已启用")
	}

	// 智能体管理服务（独立于 LLM Client）
	agentSvc := service.NewAgentService(agentRepo)
	log.Println("智能体管理服务已启用")

	// 校外系统对接服务（学工 / 一表通）
	integrationSvc := service.NewIntegrationService(cfg)
	if integrationSvc.IsXuegongAvailable() {
		log.Println("学工系统对接已配置")
	}
	if integrationSvc.IsYBTAvailable() {
		log.Println("一表通对接已配置")
	}

	// Handler 层
	authHandler := handler.NewAuthHandler(authSvc)
	sessionHandler := handler.NewSessionHandler(sessionSvc)
	kbHandler := handler.NewKBHandler(kbSvc)
	var chatHandler *handler.ChatHandler
	if chatSvc != nil {
		chatHandler = handler.NewChatHandler(chatSvc)
		// 注入情感分析服务到聊天链路
		if emotionSvc != nil {
			chatHandler.SetEmotionService(emotionSvc)
		}
	}

	// Voice handler（语音 ASR + TTS）
	var voiceHandler *handler.VoiceHandler
	if xfyunClient != nil {
		voiceHandler = handler.NewVoiceHandler(xfyunClient)
	}

	// Emotion handler（情感预警 API）
	var emotionHandler *handler.EmotionHandler
	if emotionSvc != nil {
		emotionHandler = handler.NewEmotionHandler(emotionSvc)
	}

	// Agent handler（智能体管理 API）
	agentHandler := handler.NewAgentHandler(agentSvc)

	// Recommendation handler（个性化推荐）
	recSvc := service.NewRecommendationService(kbRepo, messageRepo)
	recHandler := handler.NewRecommendationHandler(recSvc)

	// Export handler（知识导出）
	exportHandler := handler.NewExportHandler(kbSvc)

	// Integration handler（校外系统对接）
	integrationHandler := handler.NewIntegrationHandler(integrationSvc)

	// ── 5. 构建路由 ──
	router := setupRouter(cfg, db, authHandler, sessionHandler, chatHandler, kbHandler, voiceHandler, emotionHandler, agentHandler, exportHandler, integrationHandler, recHandler)

	// ── 6. 启动 HTTP 服务（支持优雅关闭）──
	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	go func() {
		log.Printf("蔚小芯服务已启动 → http://localhost:%s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// ── 7. 等待中断信号，优雅关闭 ──
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务关闭异常: %v", err)
	}
	log.Println("蔚小芯服务已安全退出")
}

// initDB 初始化 SQLite 连接，启用 WAL 模式
func initDB(dbPath string) (*sql.DB, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	// 打开数据库连接（WAL 模式 + 5s 忙等待 + 外键）
	dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// 设置连接池参数（SQLite 单写多读，不宜过大）
	db.SetMaxOpenConns(1)    // SQLite 写操作串行
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0) // 不过期

	// 验证连接可用
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// 启用 WAL 模式
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("警告：设置 WAL 模式失败: %v", err)
	}

	log.Printf("SQLite 数据库已连接: %s", dbPath)
	return db, nil
}

// setupRouter 构建 Gin 路由树
func setupRouter(cfg *config.Config, db *sql.DB, authH *handler.AuthHandler, sessionH *handler.SessionHandler, chatH *handler.ChatHandler, kbH *handler.KBHandler, voiceHandler *handler.VoiceHandler, emotionHandler *handler.EmotionHandler, agentHandler *handler.AgentHandler, exportHandler *handler.ExportHandler, integrationHandler *handler.IntegrationHandler, recHandler *handler.RecommendationHandler) *gin.Engine {
	router := gin.New()

	// 全局中间件
	router.Use(gin.Recovery())          // panic 恢复
	router.Use(middleware.CORS())       // 跨域
	router.Use(middleware.TraceID())    // 链路追踪
	router.Use(gin.Logger())           // 请求日志
	router.Use(middleware.AuditLog(db)) // 审计日志

	// ── 公共路由（无需认证）──
	router.GET("/health", healthHandler(db))

	// ── API v1 路由组 ──
	v1 := router.Group("/api/v1")
	{
		// 认证相关（公开）
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authH.Login)
		}

		// 需要 JWT 认证的路由
		secured := v1.Group("/")
		secured.Use(middleware.JWTAuth(cfg))
		{
			// 问答
			if chatH != nil {
				secured.POST("/chat", chatH.Ask)
			} else {
				secured.POST("/chat", placeholderHandler("对话接口（LLM 未配置）"))
			}

			// 会话历史
			secured.GET("/sessions", sessionH.ListSessions)
			secured.GET("/sessions/:id/messages", sessionH.GetMessages)
			secured.DELETE("/sessions/:id", sessionH.DeleteSession)

			// 知识大厅浏览（所有已认证用户可访问）
			secured.GET("/knowledge", kbH.BrowseKnowledge)

			// 个性化推荐（所有已认证用户可访问）
			secured.GET("/recommendations", recHandler.GetRecommendations)

			// 情感预警统计（所有已认证用户可访问，角色过滤在 service 层处理）
			if emotionHandler != nil {
				secured.GET("/emotion/stats", emotionHandler.GetStats)
			}

			// 情感预警管理（需 counselor 及以上角色）
			if emotionHandler != nil {
				emotion := secured.Group("/emotion")
				emotion.Use(middleware.RequireRole("counselor"))
				{
					emotion.POST("/analyze", emotionHandler.Analyze)
					emotion.GET("/alerts", emotionHandler.ListAlerts)
					emotion.PUT("/alerts/:id", emotionHandler.UpdateAlert)
					emotion.GET("/trends", emotionHandler.Trends)
				}
			}

			// 知识库管理（需 counselor 及以上角色）
			kb := secured.Group("/kb")
			kb.Use(middleware.RequireRole("counselor"))
			{
				kb.GET("/resources", kbH.ListResources)
				kb.POST("/resources", kbH.CreateResource)
				kb.PUT("/resources/:id", kbH.UpdateResource)
				kb.GET("/resources/:id", kbH.GetResource)
				kb.POST("/import", kbH.Import)
			}

			// 智能体管理（需 school_admin 及以上角色）
			agents := secured.Group("/agents")
			agents.Use(middleware.RequireRole("school_admin"))
			{
				agents.GET("", agentHandler.List)
				agents.POST("", agentHandler.Create)
				agents.GET("/:id", agentHandler.Get)
				agents.PUT("/:id", agentHandler.Update)
				agents.DELETE("/:id", agentHandler.Delete)
			}

			// 语音接口（ASR + TTS）
			if voiceHandler != nil {
				secured.POST("/voice/asr", voiceHandler.ASR)
				secured.POST("/voice/tts", voiceHandler.TTS)
			}

			// 知识导出（需认证）
			secured.GET("/export", exportHandler.Export)

			// 校外系统对接（需 counselor 及以上角色）
			integration := secured.Group("/integration")
			integration.Use(middleware.RequireRole("counselor"))
			{
				integration.GET("/status", integrationHandler.Status)
				integration.GET("/xuegong/*path", integrationHandler.ProxyXuegong)
				integration.GET("/ybt/*path", integrationHandler.ProxyYBT)
			}

			// 用户信息
			secured.GET("/user/profile", authH.Profile)
		}
	}

	return router
}

// healthHandler 健康检查接口
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

// placeholderHandler 占位 handler
func placeholderHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"code":    501,
			"message": name + " 待实现",
		})
	}
}
