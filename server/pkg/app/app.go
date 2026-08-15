package app

import (
	"context"
	"database/sql"
	"fmt"
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
	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/handler"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	_ "github.com/go-sql-driver/mysql"                   // MySQL 驱动（DB_DRIVER=mysql）
	_ "github.com/tursodatabase/libsql-client-go/libsql" // Turso 云数据库驱动（libsql:// 协议）
	_ "modernc.org/sqlite"                               // 纯 Go SQLite 驱动（本地文件 + FTS5）
)

var (
	instance http.Handler
	initOnce sync.Once
	initErr  error

	// appRedis 进程内 Redis 客户端（REDIS_ADDR 配置时启用；用于缓存等）
	appRedis *redis.Client
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

	// ── 2. 初始化数据库 ──
	// 方言优先级：DB_DRIVER=mysql → MySQL；DB_PATH 以 libsql:// 开头 → Turso；其他 → 本地 SQLite
	driver := dbutil.DriverSQLite
	dbPath := cfg.SQLitePath
	if cfg.DBDriver == "mysql" {
		driver = dbutil.DriverMySQL
	}
	isTurso := strings.HasPrefix(dbPath, "libsql://")
	if isTurso {
		driver = dbutil.DriverTurso
		// 必须显式失败：TursoDSN 缺令牌时返回空串，若继续走下去会退化成
		// 本地 SQLite 文件，在无服务器环境表现为「数据静默丢失」。
		dsn := cfg.TursoDSN()
		if dsn == "" {
			return nil, fmt.Errorf("检测到 Turso 路径 %s，但 TURSO_DB_URL / TURSO_DB_TOKEN 未配置完整，拒绝回退到本地 SQLite", dbPath)
		}
		dbPath = dsn
		log.Printf("使用 Turso 云数据库: %s", cfg.TursoDBUrl)
	}
	if os.Getenv("VERCEL") != "" {
		log.Printf("Vercel 环境：使用配置的数据库路径 %s", dbPath)
	}

	db, err := initDB(cfg, dbPath, driver)
	if err != nil {
		return nil, err
	}

	// ── 3. 自动迁移 ──
	if err := runMigrations(db, driver); err != nil {
		return nil, err
	}

	// ── 3.5 种子账号密码修复 ──
	// 保证内置账号（admin / collegeadmin / 各角色种子）都能用统一密码登录。
	// 若缺失或哈希损坏，重置为 wxx123456；已存在的有效哈希保持不变。
	if err := fixPasswordHashes(db); err != nil {
		return nil, fmt.Errorf("修复种子账号密码失败: %w", err)
	}

	// ── 3.6 Redis 缓存（可选）──
	// REDIS_ADDR 配置时启用；连接失败不阻断启动，仅记录警告（缓存降级为直查 DB）
	if cfg.RedisAddr != "" {
		appRedis = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPass,
			DB:       cfg.RedisDB,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := appRedis.Ping(ctx).Err(); err != nil {
			log.Printf("警告: Redis 连接失败（缓存降级）: %v", err)
			appRedis = nil
		} else {
			log.Printf("Redis 已连接: %s", cfg.RedisAddr)
		}
		cancel()
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
	twinRepo := repository.NewTwinRepo(db)
	chatMetricsRepo := repository.NewChatMetricsRepo(db)
	chatMetricsSvc := service.NewChatMetricsService(chatMetricsRepo)

	// ── 服务层 ──
	graduationService := service.NewGraduationService(graduationRepo)
	studentFeaturesService := service.NewStudentFeaturesService(studentFeaturesRepo)

	// LLM 客户端（主模型 DeepSeek / 智谱，带 8s 超时 + 双模型失败切换）
	var llmClient llm.ChatClient
	var deepSeekClient llm.ChatClient
	var zhipuClient llm.ChatClient
	if cfg.DeepSeekAPIKey != "" {
		deepSeekClient = llm.NewDeepSeekClient(cfg)
		log.Println("LLM 主模型: DeepSeek")
	}
	if cfg.ZhipuAPIKey != "" {
		zhipuClient = llm.NewZhipuClient(cfg)
		log.Println("LLM 备选模型: 智谱清言")
	}
	switch {
	case deepSeekClient != nil && zhipuClient != nil:
		llmClient = llm.NewFailoverClient(deepSeekClient, zhipuClient, 90*time.Second)
	case deepSeekClient != nil:
		llmClient = deepSeekClient
	case zhipuClient != nil:
		llmClient = zhipuClient
	default:
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
	kbSvc := service.NewKBService(kbRepo, db)
	forecastSvc := service.NewForecastService(db, forecastRepo, emotionRepo, feedbackRepo, llmClient)

	chatSvc := service.NewChatService(sessionRepo, messageRepo, kbRepo, agentRepo, llmClient)
	if llmClient != nil {
		if einoOrch, err := agent.NewEinoOrchestrator(kbRepo, llmClient); err == nil {
			chatSvc.SetOrchestrator(einoOrch)
			log.Println("多智能体编排: Eino Graph")
		} else {
			log.Printf("Eino 编排初始化失败，回退自研编排: %v", err)
			chatSvc.SetOrchestrator(agent.NewOrchestrator(kbRepo, llmClient))
		}
	}

	var emotionSvc *service.EmotionService
	if llmClient != nil {
		emotionSvc = service.NewEmotionService(emotionRepo, llmClient)
		log.Println("情感预警服务已启用")
	}

	agentSvc := service.NewAgentService(agentRepo)
	studentSvc := service.NewStudentService(userRepo, sessionRepo, messageRepo, emotionRepo, kbRepo, twinRepo, llmClient)
	counselorSvc := service.NewCounselorService(userRepo, emotionRepo, twinRepo, llmClient)
	integrationSvc := service.NewIntegrationService(cfg)
	adminSvc := service.NewAdminService(userRepo, auditRepo, settingsRepo)
	adminSvc.SetChatMetricsRepo(chatMetricsRepo)
	feedbackSvc := service.NewFeedbackService(feedbackRepo, userRepo, feedbackScreenshotRepo)
	feedbackSvc.SetDB(db)
	feedbackSvc.SetRepairRepo(repository.NewFeedbackRepairRepo(db))
	modelConfigSvc := service.NewModelConfigService(modelConfigRepo)
	// 用户模型配置（default_provider + Key + 模型名）参与对话：覆盖服务器默认
	chatSvc.SetModelConfigService(modelConfigSvc)
	tokenStatsSvc := service.NewTokenStatsService(tokenUsageRepo, userRepo, cfg.DailyChatQuotaPerUser, cfg.MonthlyChatQuotaPerUser, cfg.MonthlyTokenQuotaPerUser)
	// 绑定自备 Key 的用户豁免系统 token 额度
	tokenStatsSvc.SetModelConfigService(modelConfigSvc)
	// 管理员可在 /admin/settings 配置 monthly_token_quota（运行时生效，覆盖默认 10 万）
	tokenStatsSvc.SetQuotaSettingsFunc(func(key string) string {
		v, _ := settingsRepo.Get(key)
		return v
	})
	processRecordSvc := service.NewProcessRecordService(processRecordRepo, kbRepo)
	processSvc := service.NewProcessService(kbRepo, kbSvc, db)
	notificationSvc := service.NewNotificationService(db, cfg.QQWebhookURL, cfg.WechatWebhookURL)
	uploadDir := "./data/uploads"
	if os.Getenv("VERCEL") != "" {
		uploadDir = "/tmp/uploads"
		log.Printf("Vercel 环境：上传目录 %s", uploadDir)
	}
	docSvc := service.NewDocumentService(uploadDir, 100)
	docParseSvc := service.NewDocumentService(uploadDir, 100)
	// 注入 LLM 客户端，启用文档元数据精修（标题/摘要/关键词）
	docSvc.SetLLMClient(llmClient)
	docParseSvc.SetLLMClient(llmClient)
	// 注入视觉（OCR）客户端（智谱 GLM-4V），启用扫描件/图片型 PDF/DOCX 的 OCR 识别
	if ocrKey := cfg.Zhipu4VAPIKEY; ocrKey != "" || cfg.ZhipuAPIKey != "" {
		ocrClient := llm.NewZhipu4VClient(cfg)
		docSvc.SetOCRClient(ocrClient)
		docParseSvc.SetOCRClient(ocrClient)
		// 反馈「在线修复」：视觉解析截图 + 文本模型诊断
		feedbackSvc.SetAIRepairClients(ocrClient, llmClient)
		log.Printf("文档 OCR 已启用：%s（%s）", ocrClient.Name(), cfg.Zhipu4VModel)
	}
	// 启用知识库批量精修（复用精修器，逐条 LLM 精修存量资源元数据）
	kbSvc.SetRefiner(docSvc)
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
	chatHandler.SetMetricsService(chatMetricsSvc)
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
	exportSvc.SetCJKFontPath(cfg.ExportFontPath)
	exportHandler := handler.NewExportHandler(kbSvc, exportSvc)
	exportLogRepo := repository.NewExportLogRepo(db)
	exportLogSvc := service.NewExportLogService(exportLogRepo)
	exportHandler.SetExportLogService(exportLogSvc)
	if cfg.HMACSecret != "" {
		exportHandler.SetHMACSecret(cfg.HMACSecret)
		log.Println("知识导出包 HMAC-SHA256 签名已启用")
	}
	pkgSvc := service.NewKnowledgePackageService(kbSvc, kbRepo)
	pkgSvc.SetHMACSecret(cfg.HMACSecret)
	exportHandler.SetPackageService(pkgSvc)
	integrationHandler := handler.NewIntegrationHandler(integrationSvc)
	adminHandler := handler.NewAdminHandler(adminSvc, authSvc)
	feedbackHandler := handler.NewFeedbackHandler(feedbackSvc)
	modelConfigHandler := handler.NewModelConfigHandler(modelConfigSvc)
	tokenStatsHandler := handler.NewTokenStatsHandler(tokenStatsSvc)
	processRecordHandler := handler.NewProcessRecordHandler(processRecordSvc)
	processHandler := handler.NewProcessHandler(processSvc)
	// 第三方应用中心
	externalAppHandler := handler.NewExternalAppHandler(
		service.NewExternalAppService(repository.NewExternalAppRepo(db)))
	// AI 简讯（首页资讯 + 管理 CRUD + RSS 自动抓取）
	aiBriefingSvc := service.NewAIBriefingService(repository.NewAIBriefingRepo(db))
	aiBriefingHandler := handler.NewAIBriefingHandler(aiBriefingSvc)
	studentHandler := handler.NewStudentHandler(studentSvc, db)
	// 数字孪生五维聚合服务（S1.1）：注入现有 StudentHandler，/student/digital-twin 走真实数据，失败兜底 mock
	twinSvc := service.NewTwinService(twinRepo, userRepo, llmClient)
	studentHandler.SetTwinService(twinSvc)

	// 数字孪生画像（CogView 文生图/图生图；未配置 key 时服务返回提示）
	twinPortraitRepo := repository.NewTwinPortraitRepo(db)
	twinPortraitSvc := service.NewTwinPortraitService(twinPortraitRepo, llm.NewZhipuCogViewClient(cfg))
	twinPortraitHandler := handler.NewTwinPortraitHandler(twinPortraitSvc)

	// 学校门户凭证（AES-GCM 加密存储）
	portalCredRepo := repository.NewPortalCredentialRepo(db)
	portalCredSvc := service.NewPortalCredentialService(portalCredRepo)
	portalCredHandler := handler.NewPortalCredentialHandler(portalCredSvc)

	// 学校门户会话代理（持凭证登录后代理校内页面；凭证仅内存态解密）
	portalProxySvc := service.NewPortalProxyService(portalCredRepo)
	portalProxyHandler := handler.NewPortalProxyHandler(portalProxySvc)
	portalCredHandler.SetProxyService(portalProxySvc)

	// 打卡服务（S1 学生核心功能）
	checkinRepo := repository.NewCheckinRepo(db)
	checkinSvc := service.NewCheckinService(checkinRepo)
	studentHandler.SetCheckinService(checkinSvc)

	// 阶段二真实数据服务（积分成就 / 问答广场 / 谈心记录）
	phase2Repo := repository.NewPhase2Repo(db)
	phase2Svc := service.NewPhase2Service(phase2Repo, checkinRepo)
	studentHandler.SetPhase2Service(phase2Svc)
	counselorSvc.SetPhase2Service(phase2Svc)

	// 阶段三数据底座服务（成绩/课表导入 + 教辅真实数据）
	dataImportRepo := repository.NewDataImportRepo(db)
	phase3Svc := service.NewPhase3Service(dataImportRepo)
	dataImportH := handler.NewDataImportHandler(phase3Svc)

	// 性格洞察服务（S1 学生核心功能）
	personalityRepo := repository.NewPersonalityRepo(db)
	personalitySvc := service.NewPersonalityService(personalityRepo, userRepo, twinRepo, llmClient)
	studentHandler.SetPersonalityService(personalitySvc)

	counselorHandler := handler.NewCounselorHandler(counselorSvc)
	counselorHandler.SetPhase2Service(phase2Svc)

	var teacherSvc *service.TeacherService
	if llmClient != nil {
		teacherSvc = service.NewTeacherService(llmClient)
		log.Println("教师 AI 服务已启用")
	}
	teacherHandler := handler.NewTeacherHandler(teacherSvc)

	var assistantSvc *service.AssistantService
	if llmClient != nil {
		assistantSvc = service.NewAssistantService(llmClient)
		assistantSvc.SetPhase3Service(phase3Svc)
		assistantSvc.SetUserRepo(userRepo)
	}
	assistantHandler := handler.NewAssistantHandler(assistantSvc)

	// 后勤服务台（并入教辅角色）：纯真实数据记录，不依赖 LLM，始终构建。
	facilityRepo := repository.NewFacilityRepo(db)
	facilitySvc := service.NewFacilityService(facilityRepo)
	facilityHandler := handler.NewFacilityHandler(facilitySvc)

	// 第二课堂看板（辅导员侧）：真实聚合名下学生活动参与/积分，不依赖 LLM。
	secondClassRepo := repository.NewSecondClassRepo(db)
	counselorSvc.SetSecondClassRepo(secondClassRepo)

	// 书记教育成果（毕业去向登记+审核+教育成果大屏）：真实数据聚合，不依赖 LLM，始终构建。
	secretaryRepo := repository.NewSecretaryOutcomeRepo(db)
	secretarySvc := service.NewSecretaryOutcomeService(secretaryRepo)
	secretaryHandler := handler.NewSecretaryOutcomeHandler(secretarySvc)

	// 学生会服务始终构建：真实数据统计（成员/活动分析）不依赖 LLM，LLM 仅用于增强解读。
	unionSvc := service.NewUnionService(db, llmClient)
	unionHandler := handler.NewUnionHandler(unionSvc)

	// 学院/学校服务始终构建：即便无 LLM，也能返回真实聚合数据（LLM 仅用于解读增强）
	collegeSvc := service.NewCollegeService(userRepo, emotionRepo, twinRepo, llmClient)
	collegeHandler := handler.NewCollegeHandler(collegeSvc)
	cultureHandler := handler.NewCultureHandler()

	schoolAdminSvc := service.NewSchoolAdminService(userRepo, emotionRepo, twinRepo, llmClient)
	schoolAdminHandler := handler.NewSchoolAdminHandler(schoolAdminSvc)

	// 系统管理员服务始终构建：真实聚合（用户数/知识库统计/运行时指标）不依赖 LLM
	sysAdminSvc := service.NewSysAdminService(llmClient, userRepo, kbRepo, auditRepo)
	sysAdminHandler := handler.NewSysAdminHandler(sysAdminSvc)
	forecastHandler := handler.NewForecastHandler(forecastSvc)
	notificationHandler := handler.NewNotificationHandler(notificationSvc)
	uploadHandler := handler.NewUploadHandler(docSvc, kbSvc)
	documentHandler := handler.NewDocumentHandler(docParseSvc)
	graduationHandler := handler.NewGraduationHandler(graduationService)
	studentFeaturesHandler := handler.NewStudentFeaturesHandler(studentFeaturesService)
	educationHandler := handler.NewEducationHandler(db)
	studyPlanSvc := service.NewStudyPlanService(db, llmClient)
	studyPlanHandler := handler.NewStudyPlanHandler(db, studyPlanSvc)
	userNotificationHandler := handler.NewUserNotificationHandler(db)
	statsHandler := handler.NewStatsHandler(db)
	appVersionRepo := repository.NewAppVersionRepo(db)
	appVersionService := service.NewAppVersionService(appVersionRepo)
	appVersionHandler := handler.NewAppVersionHandler(appVersionService)
	// 校园报到步骤（直接注入 db，无独立 Service 层）
	campusHandler := handler.NewCampusHandler(repository.NewCampusRepository(db))

	// ── 5. 构建路由 ──
	router := setupRouter(cfg, db, userRepo, authHandler, sessionHandler, chatHandler, kbHandler,
		voiceHandler, emotionHandler, agentHandler, exportHandler, integrationHandler, recHandler,
		adminHandler, feedbackHandler, modelConfigHandler, tokenStatsHandler,
		studentHandler, counselorHandler, teacherHandler, assistantHandler, facilityHandler, secretaryHandler, unionHandler, collegeHandler,
		cultureHandler, schoolAdminHandler, sysAdminHandler, processRecordHandler, processHandler, forecastHandler, graduationHandler, studentFeaturesHandler, notificationHandler, uploadHandler, documentHandler, educationHandler, studyPlanHandler, statsHandler, userNotificationHandler, appVersionHandler, campusHandler, dataImportH, externalAppHandler, aiBriefingHandler, twinPortraitHandler, portalCredHandler, portalProxyHandler)

	// ── 6. 数据保留清理（9.2 合规基线）──
	retentionSvc := service.NewRetentionService(db)
	retentionInterval := time.Duration(cfg.RetentionIntervalHours) * time.Hour
	if retentionInterval <= 0 {
		retentionInterval = 24 * time.Hour
	}
	go retentionSvc.RunLoop(
		context.Background(),
		retentionInterval,
		cfg.RetentionAuditDays,
		cfg.RetentionSessionDays,
		cfg.RetentionEmotionDays,
		cfg.RetentionExportDays,
	)

	// AI 简讯定时抓取调度（每分钟检查来源抓取时刻）
	go aiBriefingSvc.RunLoop(context.Background())

	return router, nil
}

// initDB 初始化数据库连接
// 自动识别 DSN 协议选择驱动：
//   - DB_DRIVER=mysql → MySQL（go-sql-driver/mysql）
//   - libsql:// 开头 → Turso 云数据库（libsql 驱动）
//   - 其他 → 本地 SQLite 文件（modernc.org/sqlite 驱动）
func initDB(cfg *config.Config, dbPath string, driver dbutil.Driver) (*sql.DB, error) {
	var driverName, dsn string

	switch driver {
	case dbutil.DriverMySQL:
		driverName = "mysql"
		dsn = cfg.MySQLDSN()
		log.Printf("使用 MySQL 数据库: %s@%s:%s/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	case dbutil.DriverTurso:
		driverName = "libsql"
		dsn = dbPath
		log.Printf("使用 Turso 云数据库: %s", dbPath)
	default:
		// 本地 SQLite 文件：确保目录存在，附加 pragma 参数
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return nil, err
		}
		driverName = "sqlite"
		dsn = dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	// 连接池配置：Turso/MySQL 支持并发连接，本地 SQLite 限制单连接
	switch driver {
	case dbutil.DriverTurso, dbutil.DriverMySQL:
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(time.Minute * 5)
	default:
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(2)
		db.SetConnMaxLifetime(0)
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	switch driver {
	case dbutil.DriverMySQL:
		log.Printf("MySQL 数据库已连接: %s@%s:%s/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	case dbutil.DriverTurso:
		log.Printf("Turso 云数据库已连接: %s", dbPath)
	default:
		log.Printf("SQLite 数据库已连接: %s", dbPath)
	}
	return db, nil
}

// runMigrations 从嵌入的迁移文件执行数据库迁移
// MySQL 模式：对迁移 SQL 做 SQLite→MySQL 方言转换；FTS5 语句被跳过
func runMigrations(db *sql.DB, driver dbutil.Driver) error {
	if driver == dbutil.DriverMySQL {
		// MySQL 方言的迁移记录表
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			filename VARCHAR(255) NOT NULL UNIQUE,
			executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
			return err
		}
	} else {
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL UNIQUE,
			executed_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
		)`); err != nil {
			return err
		}
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

		if err := execSQL(db, string(content), entry.Name(), driver); err != nil {
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
// MySQL 模式：逐条做 SQLite→MySQL 方言转换，跳过 FTS5 语句（MySQL 无 FTS5 虚拟表）
func execSQL(db *sql.DB, content, filename string, driver dbutil.Driver) error {
	statements := splitSQL(content)
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		// MySQL/Turso 不支持 FTS5 虚拟表及其触发器，跳过
		if driver != dbutil.DriverSQLite && (strings.Contains(strings.ToUpper(stmt), "FTS5") || strings.Contains(strings.ToUpper(stmt), "KB_FTS")) {
			log.Printf("迁移 %s 第 %d 条语句跳过（%s 不支持 FTS5）: %.60s...", filename, i+1, driver, stmt)
			continue
		}

		// LONGTEXT 仅 MySQL 需要（SQLite TEXT 无长度限制），非 MySQL 驱动跳过
		if driver != dbutil.DriverMySQL && strings.Contains(strings.ToUpper(stmt), "LONGTEXT") {
			log.Printf("迁移 %s 第 %d 条语句跳过（%s 无需 LONGTEXT）: %.60s...", filename, i+1, driver, stmt)
			continue
		}

		// SQLite → MySQL 方言转换
		if driver == dbutil.DriverMySQL {
			stmt = dbutil.ToMySQL(stmt)
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
		}

		if _, err := db.Exec(stmt); err != nil {
			// ALTER TABLE ADD COLUMN 重复列名视为非致命错误（列已存在 = 目标状态）
			if isDuplicateColumnError(err) && strings.Contains(strings.ToUpper(stmt), "ALTER TABLE") {
				log.Printf("迁移 %s 第 %d 条语句跳过（列已存在）: %v", filename, i+1, err)
				continue
			}
			// 重复索引（MySQL 1061 Duplicate key name / SQLite already exists）
			if isDuplicateIndexError(err) && strings.Contains(strings.ToUpper(stmt), "CREATE INDEX") {
				log.Printf("迁移 %s 第 %d 条语句跳过（索引已存在）: %v", filename, i+1, err)
				continue
			}
			log.Printf("迁移 %s 第 %d 条语句失败: %v", filename, i+1, err)
			return err
		}
	}
	return nil
}

// isDuplicateColumnError 检测 "duplicate column name" 错误（SQLite 小写 / MySQL 大写）
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column name") ||
		strings.Contains(msg, "duplicate column")
}

// isDuplicateIndexError 检测重复索引错误（MySQL 1061 Duplicate key name / SQLite）
func isDuplicateIndexError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "duplicate key name") ||
		strings.Contains(lower, "already exists")
}

// splitSQL 按分号分割 SQL 语句，正确处理触发器复合语句与行尾注释（`; -- 注释`）
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

		// 语句结束判定基于"去掉行尾注释后的内容"：
		// 例如 `ALTER TABLE ... DEFAULT '[]';  -- JSON 数组` 应以 `;` 结束语句。
		base := trimmed
		if idx := strings.LastIndex(base, ";"); idx >= 0 {
			after := strings.TrimSpace(base[idx+1:])
			if after == "" || strings.HasPrefix(after, "--") {
				base = base[:idx+1]
			}
		}

		current.WriteString(line)
		current.WriteString("\n")

		if inTrigger && strings.HasSuffix(trimmed, "END;") {
			statements = append(statements, current.String())
			current.Reset()
			inTrigger = false
			continue
		}

		if !inTrigger && strings.HasSuffix(base, ";") {
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
	facilityH *handler.FacilityHandler,
	secretaryH *handler.SecretaryOutcomeHandler,
	unionH *handler.UnionHandler,
	collegeH *handler.CollegeHandler,
	cultureH *handler.CultureHandler,
	schoolAdminH *handler.SchoolAdminHandler,
	sysAdminH *handler.SysAdminHandler,
	processRecordH *handler.ProcessRecordHandler,
	processH *handler.ProcessHandler,
	forecastH *handler.ForecastHandler,
	graduationH *handler.GraduationHandler,
	studentFeaturesH *handler.StudentFeaturesHandler,
	notificationH *handler.NotificationHandler,
	uploadH *handler.UploadHandler,
	documentH *handler.DocumentHandler,
	educationH *handler.EducationHandler,
	studyPlanH *handler.StudyPlanHandler,
	statsH *handler.StatsHandler,
	userNotificationH *handler.UserNotificationHandler,
	appVersionH *handler.AppVersionHandler,
	campusH *handler.CampusHandler,
	dataImportH *handler.DataImportHandler,
	externalAppH *handler.ExternalAppHandler,
	aiBriefingH *handler.AIBriefingHandler,
	twinPortraitH *handler.TwinPortraitHandler,
	portalCredH *handler.PortalCredentialHandler,
	portalProxyH *handler.PortalProxyHandler,
) *gin.Engine {
	router := gin.New()

	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(middleware.CORSWithConfig(cfg.CORSAllowedOrigins, cfg.IsRelease()))
	router.Use(middleware.GlobalIPRateLimiter()) // 全局限流（IP 维度，100 req/min/IP）
	router.Use(middleware.TraceID())
	router.Use(middleware.PIIMask()) // PII 检测与脱敏（在请求进入 handler 前检测并脱敏）
	router.Use(gin.Logger())
	router.Use(middleware.AuditLog(db))

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

	// ── 前端静态文件服务（临时 8080 直连方案）──
	// Flutter Web 构建产物位于 FRONTEND_STATIC_DIR（默认 /opt/wxx/frontend/web）
	// 静态文件目录存在时才挂载；API 路由优先，SPA 路由回退到 index.html
	staticDir := cfg.FrontendStaticDir
	if staticDir != "" {
		if _, err := os.Stat(staticDir); err == nil {
			// 缓存控制中间件：入口文件（main.dart.js / index.html /
			// flutter_bootstrap.js / flutter_service_worker.js）禁止缓存，
			// 确保浏览器每次都拉取最新版本；/assets/ 下带哈希的资源可长缓存。
			noCachePaths := map[string]bool{
				"/main.dart.js":              true,
				"/index.html":                true,
				"/flutter_bootstrap.js":      true,
				"/flutter_service_worker.js": true,
			}
			router.Use(func(c *gin.Context) {
				if noCachePaths[c.Request.URL.Path] {
					c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
					c.Header("Pragma", "no-cache")
					c.Header("Expires", "0")
				}
				c.Next()
			})
			// 让 SPA 应用内路由（/campus 等）回退到 index.html
			router.NoRoute(func(c *gin.Context) {
				if !strings.HasPrefix(c.Request.URL.Path, "/api/") &&
					!strings.HasPrefix(c.Request.URL.Path, "/health") {
					indexPath := filepath.Join(staticDir, "index.html")
					if _, err := os.Stat(indexPath); err == nil {
						// SPA 回退的 index.html 也禁止缓存
						c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
						c.File(indexPath)
						return
					}
				}
				c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "接口不存在"})
			})
			// 静态资源（JS/CSS/图片等）
			router.Static("/assets", filepath.Join(staticDir, "assets"))
			// CanvasKit 本地化：flutter_bootstrap.js 通过 config.canvasKitBaseUrl
			// 指向 /canvaskit/，需显式注册 Static 目录，否则请求会被 SPA 回退到 index.html。
			router.Static("/canvaskit", filepath.Join(staticDir, "canvaskit"))
			router.StaticFile("/index.html", filepath.Join(staticDir, "index.html"))
			router.StaticFile("/main.dart.js", filepath.Join(staticDir, "main.dart.js"))
			router.StaticFile("/favicon.png", filepath.Join(staticDir, "favicon.png"))
			router.StaticFile("/favicon.ico", filepath.Join(staticDir, "favicon.ico"))
			router.StaticFile("/flutter_bootstrap.js", filepath.Join(staticDir, "flutter_bootstrap.js"))
			router.StaticFile("/flutter_service_worker.js", filepath.Join(staticDir, "flutter_service_worker.js"))
			router.StaticFile("/manifest.json", filepath.Join(staticDir, "manifest.json"))
			router.StaticFile("/version.json", filepath.Join(staticDir, "version.json"))
			log.Printf("前端静态文件已挂载: %s", staticDir)
		} else {
			log.Printf("警告: 前端静态目录不存在，跳过静态文件服务: %s", staticDir)
		}
	}

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// 认证（公开）
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", middleware.LoginIPRateLimiter(), authH.Login)
			authGroup.POST("/sso/callback", middleware.LoginIPRateLimiter(), authH.SSOCallback)
			authGroup.POST("/qr-login", handler.CreateQRSession)
			authGroup.GET("/qr-status", handler.GetQRSessionStatus)
			authGroup.PUT("/qr-scan", handler.ScanQRSession)
			authGroup.POST("/send-code", middleware.LoginIPRateLimiter(), authH.SendCode)
			authGroup.POST("/guest-register", middleware.LoginIPRateLimiter(), authH.GuestRegister)
		}

		// 版本更新（公开）
		v1.GET("/version/check", appVersionH.CheckUpdate)
		v1.GET("/version/latest", appVersionH.GetLatestVersion)

		// 校园报到步骤（公开，无需登录）
		v1.GET("/campus/steps", campusH.ListPublicSteps)

		// 知识大厅（公开，仅返回全校公开已发布资源）
		v1.GET("/knowledge/public", kbH.BrowseKnowledgePublic)

		// 需要 JWT 认证
		secured := v1.Group("/")
		secured.Use(middleware.JWTAuth(cfg))
		secured.Use(middleware.EnsureUserExists(userRepo))
		{
			// ── AI 对话（self.chat）──
			// 安全修复 SEC-02：对话为主要 PII 输入入口，要求已同意隐私政策/用户协议方可访问
			secured.POST("/chat", middleware.RequireConsent(), auth.RequireCapability(auth.SelfChat), middleware.ChatUserRateLimiter(), chatH.Ask)
			secured.POST("/chat/stream", middleware.RequireConsent(), auth.RequireCapability(auth.SelfChat), middleware.ChatUserRateLimiter(), chatH.Stream)

			// ── 会话/知识/推荐（self.* 能力）──
			secured.GET("/sessions", auth.RequireCapability(auth.SelfSessionRead), sessionH.ListSessions)
			secured.GET("/sessions/:id/messages", auth.RequireCapability(auth.SelfSessionRead), sessionH.GetMessages)
			secured.DELETE("/sessions/:id", auth.RequireCapability(auth.SelfSessionDelete), sessionH.DeleteSession)
			secured.PATCH("/sessions/:id", auth.RequireCapability(auth.SelfSessionRead), sessionH.RenameSession)
			secured.GET("/knowledge", auth.RequireCapability(auth.SelfKnowledgeRead), kbH.BrowseKnowledge)
			secured.GET("/recommendations", auth.RequireCapability(auth.SelfRecommendRead), recH.GetRecommendations)

			// ── 情感数据 ──
			if emotionH != nil {
				// 自身情感统计：所有用户都可看自己。
				// 独立授权语义：需同时拥有 self.emotion.consent（独立于通用隐私 consent）
				secured.GET("/emotion/stats", auth.RequireAnyCapability(auth.SelfEmotionStats, auth.SelfEmotionConsent), emotionH.GetStats)
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
				// ── 毕设选题管理（学院管理员）──
				graduation.POST("/admin/topics", auth.RequireCapability(auth.CollegeGraduationWrite), graduationH.CreateTopic)
				graduation.PUT("/admin/topics/:id", auth.RequireCapability(auth.CollegeGraduationWrite), graduationH.UpdateTopic)
				graduation.DELETE("/admin/topics/:id", auth.RequireCapability(auth.CollegeGraduationWrite), graduationH.DeleteTopic)
			}

			// ── 学科竞赛 ──
			competition := secured.Group("/competition")
			{
				competition.GET("/list", auth.RequireCapability(auth.SelfCompetitionRead), studentFeaturesH.ListCompetitions)
				competition.GET("/match", auth.RequireCapability(auth.SelfCompetitionRead), studentFeaturesH.CompetitionMatch)
				competition.GET("/:id", auth.RequireCapability(auth.SelfCompetitionRead), studentFeaturesH.GetCompetition)
				competition.POST("/register", auth.RequireCapability(auth.SelfCompetitionWrite), studentFeaturesH.RegisterCompetition)
				competition.GET("/my-registrations", auth.RequireCapability(auth.SelfCompetitionRead), studentFeaturesH.GetMyCompetitionRegistrations)
				competition.POST("/submit-work", auth.RequireCapability(auth.SelfCompetitionWrite), studentFeaturesH.SubmitWork)
				competition.GET("/stats", auth.RequireCapability(auth.SelfCompetitionRead), studentFeaturesH.GetCompetitionStats)
				// ── 竞赛管理（学校/学院管理员）──
				competition.GET("/admin/list", auth.RequireCapability(auth.SystemSettingsWrite), studentFeaturesH.AdminListCompetitions)
				competition.POST("/admin", auth.RequireCapability(auth.SystemSettingsWrite), studentFeaturesH.AdminCreateCompetition)
				competition.PUT("/admin/:id", auth.RequireCapability(auth.SystemSettingsWrite), studentFeaturesH.AdminUpdateCompetition)
				competition.DELETE("/admin/:id", auth.RequireCapability(auth.SystemSettingsWrite), studentFeaturesH.AdminDeleteCompetition)
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
				// 高级查询与字典（必须在 /resources/:id 之前注册，避免 "advanced" 被匹配为 :id）
				kb.GET("/resources/advanced", auth.RequireCapability(auth.CounselorKBWrite), kbH.ListResourcesAdvanced)
				kb.GET("/dict", auth.RequireCapability(auth.CounselorKBWrite), kbH.GetDictValues)
				kb.GET("/stats", auth.RequireCapability(auth.CounselorKBWrite), kbH.GetStats)

				kb.GET("/resources", auth.RequireCapability(auth.CounselorKBWrite), kbH.ListResources)
				kb.POST("/resources", auth.RequireCapability(auth.CounselorKBWrite), kbH.CreateResource)
				kb.PUT("/resources/:id", auth.RequireCapability(auth.CounselorKBWrite), kbH.UpdateResource)
				kb.GET("/resources/:id", auth.RequireCapability(auth.CounselorKBWrite), kbH.GetResource)
				kb.POST("/import", auth.RequireCapability(auth.CounselorKBWrite), kbH.Import)
				kb.POST("/validate", auth.RequireCapability(auth.CounselorKBWrite), kbH.Validate)

				// 批量操作（counselor.kb.review）
				kb.POST("/batch/approve", auth.RequireCapability(auth.CounselorKBReview), kbH.BatchApprove)
				kb.POST("/batch/reject", auth.RequireCapability(auth.CounselorKBReview), kbH.BatchReject)
				kb.POST("/batch/retire", auth.RequireCapability(auth.CounselorKBReview), kbH.BatchRetire)
				kb.POST("/batch/delete", auth.RequireCapability(auth.CounselorKBWrite), kbH.BatchDelete)
				kb.POST("/batch/refine", auth.RequireCapability(auth.CounselorKBWrite), kbH.BatchRefine)

				// 知识审核（counselor.kb.review）
				kb.POST("/resources/:id/approve", auth.RequireCapability(auth.CounselorKBReview), kbH.ApproveResource)
				kb.POST("/resources/:id/reject", auth.RequireCapability(auth.CounselorKBReview), kbH.RejectResource)
				kb.POST("/resources/:id/retire", auth.RequireCapability(auth.CounselorKBReview), kbH.RetireResource)

				// 知识提交（union.kb.submit，student_union 起）
				kb.POST("/resources/:id/submit", auth.RequireCapability(auth.UnionKBSubmit), kbH.SubmitForReview)
			}

			// ── 知识同步导出（school.kb.sync.export，学校级运维）──
			// 安全修复 RB-01：知识全量导出不再对所有登录用户开放，仅学校级同步能力可用，并在服务层按 scope 过滤
			secured.GET("/kb/export", auth.RequireCapability(auth.SchoolKBSyncExport), exportH.Export)
			secured.GET("/kb/export/package", auth.RequireCapability(auth.SchoolKBSyncExport), exportH.ExportPackage)
			secured.POST("/kb/import/package", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), exportH.ImportPackage)
			secured.POST("/kb/import/package/chunk/init", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), exportH.InitChunkUpload)
			secured.PUT("/kb/import/package/chunk/:upload_id/:chunk_index", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), exportH.UploadChunk)
			secured.GET("/kb/import/package/chunk/status/:upload_id", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), exportH.ChunkUploadStatus)
			secured.POST("/kb/import/package/chunk/complete/:upload_id", auth.RequireAnyCapability(auth.CounselorKBWrite, auth.SchoolKBSyncExport), exportH.CompleteChunkUpload)

			// ── 智能体管理（school.agent.write）──
			agents := secured.Group("/agents")
			{
				agents.GET("", auth.RequireCapability(auth.SchoolAgentWrite), agentH.List)
				agents.POST("", auth.RequireCapability(auth.SchoolAgentWrite), agentH.Create)
				agents.GET("/:id", auth.RequireCapability(auth.SchoolAgentWrite), agentH.Get)
				agents.PUT("/:id", auth.RequireCapability(auth.SchoolAgentWrite), agentH.Update)
				agents.DELETE("/:id", auth.RequireCapability(auth.SchoolAgentWrite), agentH.Delete)
			}

			// 已启用智能体列表（对话页选择器用，普通登录用户即可访问）
			secured.GET("/agents/active", agentH.ListActive)

			// ── 语音 ASR/TTS（self.voice）──
			if voiceH != nil {
				secured.POST("/voice/asr", auth.RequireCapability(auth.SelfVoice), voiceH.ASR)
				secured.POST("/voice/tts", auth.RequireCapability(auth.SelfVoice), voiceH.TTS)
			}

			// ── 知识资源导出（school.kb.sync.export，同 /kb/export，学校级同步）──
			// 安全修复 RB-01：与 /kb/export 同一 handler，统一收敛到学校级同步能力
			secured.GET("/export", auth.RequireCapability(auth.SchoolKBSyncExport), exportH.Export)
			// 导出本人回答卡片（self.export.self）——仅处理调用者自行提交的卡片数据，无越权风险
			secured.POST("/export/answer", auth.RequireCapability(auth.SelfExportSelf), exportH.ExportAnswer)

			// ── 校外系统对接（counselor.integration.read）──
			integration := secured.Group("/integration")
			{
				integration.GET("/status", auth.RequireCapability(auth.CounselorIntegrationRead), integrationH.Status)
				integration.GET("/xuegong/*path", auth.RequireCapability(auth.CounselorIntegrationRead), integrationH.ProxyXuegong)
				integration.GET("/ybt/*path", auth.RequireCapability(auth.CounselorIntegrationRead), integrationH.ProxyYBT)
			}

			secured.GET("/user/profile", authH.Profile)
			secured.GET("/user/profile/detail", authH.ProfileDetail)
			secured.GET("/user/ai-key", authH.GetAIKey)
			secured.PUT("/user/ai-key", authH.SaveAIKey)
			secured.DELETE("/user/ai-key", authH.ClearAIKey)
			// 我的操作日志（普通用户查看自己的日志）
			secured.GET("/user/logs", adminH.MyLogs)
			secured.DELETE("/user/logs/:id", adminH.DeleteMyLog)
			// 学校门户凭证（加密存储，密码不回显）
			secured.GET("/user/portal-credential", portalCredH.Get)
			secured.PUT("/user/portal-credential", portalCredH.Save)
			secured.DELETE("/user/portal-credential", portalCredH.Delete)
			// 学校门户页面代理（持用户门户凭证登录后访问校内各级网站）
			secured.GET("/user/portal/*path", portalProxyH.Proxy)
			secured.GET("/user/portal", portalProxyH.Proxy)
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
				admin.GET("/stats/dashboard", auth.RequireCapability(auth.CollegeMetricsRead), statsH.GetDashboardStats)
				admin.GET("/metrics", auth.RequireCapability(auth.CollegeMetricsRead), adminH.GetMetrics)
				admin.GET("/metrics/fallback-questions", auth.RequireCapability(auth.CollegeMetricsRead), adminH.TopFallbackQuestions)
				admin.GET("/users", auth.RequireCapability(auth.CollegeUserRead), adminH.ListUsers)
				admin.GET("/audit", auth.RequireCapability(auth.CollegeAuditRead), adminH.ListAudit)
				admin.DELETE("/audit", auth.RequireCapability(auth.CollegeAuditRead), adminH.DeleteAudit)
				// 审计恢复快照（college_admin+ 可查看，sys_admin 可恢复）
				admin.GET("/audit/snapshots", auth.RequireCapability(auth.CollegeAuditRead), adminH.ListSnapshots)
				admin.POST("/audit/snapshots/:id/restore", auth.RequireCapability(auth.SystemAuditAll), adminH.RestoreSnapshot)

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
				// 公开功能开关：登录用户可读（管理员在 /admin/settings 里改 feature.* 键）
				secured.GET("/public/feature-switches", adminH.GetPublicFeatureSwitches)

				// 应用版本管理（sys_admin）
				admin.GET("/app-versions", auth.RequireCapability(auth.SystemSettingsWrite), appVersionH.ListVersions)
				admin.POST("/app-versions", auth.RequireCapability(auth.SystemSettingsWrite), appVersionH.CreateVersion)
				admin.PUT("/app-versions", auth.RequireCapability(auth.SystemSettingsWrite), appVersionH.UpdateVersion)
				admin.DELETE("/app-versions/:id", auth.RequireCapability(auth.SystemSettingsWrite), appVersionH.DeleteVersion)

				// 第三方应用管理（sys_admin）
				admin.GET("/apps", auth.RequireCapability(auth.SystemSettingsWrite), externalAppH.ListAdmin)
				admin.POST("/apps", auth.RequireCapability(auth.SystemSettingsWrite), externalAppH.Create)
				admin.PUT("/apps/:id", auth.RequireCapability(auth.SystemSettingsWrite), externalAppH.Update)
				admin.DELETE("/apps/:id", auth.RequireCapability(auth.SystemSettingsWrite), externalAppH.Delete)
				// 应用中心（登录用户可见，按角色过滤）
				secured.GET("/apps", externalAppH.ListVisible)

				// ── AI 简讯（sys_admin 管理）──
				admin.GET("/ai-briefings", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.List)
				admin.POST("/ai-briefings", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.Create)
				admin.PUT("/ai-briefings/:id", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.Update)
				admin.PUT("/ai-briefings/:id/status", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.UpdateStatus)
				admin.DELETE("/ai-briefings/:id", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.Delete)
				admin.POST("/ai-briefings/batch-delete", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.DeleteMany)
				admin.DELETE("/ai-briefings/clear", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.ClearAll)
				admin.GET("/ai-briefings/stats", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.Stats)
				admin.POST("/ai-briefings/export", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.Export)
				admin.POST("/ai-briefings/fetch", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.FetchNow)
				admin.GET("/ai-briefings/sources", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.ListSources)
				admin.POST("/ai-briefings/sources", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.CreateSource)
				admin.PUT("/ai-briefings/sources/:id", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.UpdateSource)
				admin.DELETE("/ai-briefings/sources/:id", auth.RequireCapability(auth.SystemAIBriefing), aiBriefingH.DeleteSource)
				// AI 简讯（登录用户可见）
				secured.GET("/ai-briefings", aiBriefingH.ListUser)
				secured.GET("/ai-briefings/hot", aiBriefingH.ListUserHot)
				secured.GET("/ai-briefings/favorites", aiBriefingH.ListFavorites)
				secured.POST("/ai-briefings/:id/favorite", aiBriefingH.Favorite)
				secured.DELETE("/ai-briefings/:id/favorite", aiBriefingH.Unfavorite)

				// 数字孪生画像（登录用户可见，文生图/图生图）
				secured.GET("/twin-portraits", auth.RequireCapability(auth.SelfTwinRead), twinPortraitH.List)
				secured.GET("/twin-portraits/:type", auth.RequireCapability(auth.SelfTwinRead), twinPortraitH.Get)
				secured.POST("/twin-portraits/generate", auth.RequireCapability(auth.SelfTwinWrite), twinPortraitH.Generate)

				// 游客管理（college_admin+）
				admin.GET("/guests/pending", auth.RequireCapability(auth.CollegeUserRead), adminH.ListPendingGuests)
				admin.PUT("/guests/:id/approve", auth.RequireCapability(auth.CollegeUserRead), adminH.ApproveGuest)
				admin.PUT("/guests/:id/reject", auth.RequireCapability(auth.CollegeUserRead), adminH.RejectGuest)
				// 学生导入（除学生和游客外的组织角色均可用）
				admin.POST("/users/import", auth.RequireCapability(auth.CounselorImportStudent), adminH.ImportStudents)
				// ── 数据底座导入（成绩/课表，college_admin+）──
				admin.POST("/grades/import", auth.RequireCapability(auth.CollegeUserRead), dataImportH.ImportGrades)
				admin.POST("/schedules/import", auth.RequireCapability(auth.BatchScheduleImport), dataImportH.ImportSchedules)

				// ── 校园报到步骤管理（college_admin+）──
				campusAdmin := admin.Group("/campus")
				{
					campusAdmin.GET("/steps", auth.RequireCapability(auth.CollegeUserRead), campusH.ListAdminSteps)
					campusAdmin.POST("/steps", auth.RequireCapability(auth.CollegeUserRead), campusH.CreateStep)
					campusAdmin.PUT("/steps/:id", auth.RequireCapability(auth.CollegeUserRead), campusH.UpdateStep)
					campusAdmin.POST("/steps/:id/submit", auth.RequireCapability(auth.CollegeUserRead), campusH.SubmitStep)
					campusAdmin.POST("/steps/:id/publish", auth.RequireCapability(auth.CollegeDataAnalysis), campusH.PublishStep)
					// 管理员拖拽校正坐标（college_admin+，不走审核流程，已发布步骤也可调整）
					campusAdmin.PATCH("/steps/:id/coords", auth.RequireCapability(auth.CollegeUserRead), campusH.UpdateStepCoords)
					campusAdmin.DELETE("/steps/:id", auth.RequireCapability(auth.CollegeUserRead), campusH.DeleteStep)
					// 管理员强制更新/删除（不限状态，已发布步骤也可直接修改内容或删除）
					campusAdmin.PUT("/steps/:id/force", auth.RequireCapability(auth.CollegeUserRead), campusH.UpdateStepForce)
					campusAdmin.DELETE("/steps/:id/force", auth.RequireCapability(auth.CollegeUserRead), campusH.DeleteStepForce)
				}
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
			// 反馈详情（所有登录用户可查自己的，管理端可查所有；权限由 handler 内 user_id 校验）
			secured.GET("/feedback/:id", feedbackH.Get)
			// 反馈满意度评价（用户对已解决的反馈打分）
			secured.PUT("/feedback/:id/rate", auth.RequireCapability(auth.SelfFeedbackSubmit), feedbackH.Rate)
			// 反馈处理记录
			secured.GET("/feedback/:id/logs", feedbackH.GetLogs)
			// 管理员反馈列表和处理（注意：直接注册避免 Group 产生的尾部斜杠重定向丢 Authorization 头）
			secured.GET("/feedback", auth.RequireCapability(auth.UnionFeedbackList), feedbackH.List)
			secured.PUT("/feedback/:id", auth.RequireCapability(auth.UnionFeedbackList), feedbackH.Resolve)
			secured.POST("/feedback/:id/ai-repair", auth.RequireCapability(auth.UnionFeedbackWrite), feedbackH.AIRepair)
			// 修复工单轮询/审计（管理端）
			secured.GET("/feedback/:id/ai-repair/job", auth.RequireCapability(auth.UnionFeedbackWrite), feedbackH.LatestRepairJob)
			// 管理端反馈统计
			secured.GET("/admin/feedback/stats", auth.RequireCapability(auth.UnionFeedbackRead), feedbackH.Stats)
			// 管理端关联知识资源
			secured.PUT("/admin/feedback/:id/link-resource", auth.RequireCapability(auth.UnionFeedbackWrite), feedbackH.LinkResource)
			// 反馈截图：从 SQLite blob 流式输出（需认证，原公开路由已移入 secured 组修复越权）
			secured.GET("/uploads/feedback/:filename", feedbackH.ServeScreenshot)

			// ── 办事流程办理记录 ──
			process := secured.Group("/process/records")
			{
				process.GET("", auth.RequireCapability(auth.SelfProcessRead), processRecordH.ListMine)
				process.POST("/:flow/start", auth.RequireCapability(auth.SelfProcessRead), processRecordH.StartOrResume)
				process.POST("/:flow/progress", auth.RequireCapability(auth.SelfProcessRead), processRecordH.UpdateProgress)
			}

			// ── 办事流程定义（学生端动态列表）──
			processDef := secured.Group("/process/definitions")
			processDef.Use(auth.RequireCapability(auth.SelfProcessRead))
			{
				processDef.GET("", processH.ListDefinitions)
				processDef.GET("/:id", processH.GetDefinition)
			}

			// ── 办事流程管理/审核（counselor+，学校/学院管理员继承）──
			processAdmin := secured.Group("/process/admin")
			processAdmin.Use(auth.RequireAnyCapability(auth.CounselorKBWrite, auth.CounselorKBReview))
			{
				processAdmin.GET("", auth.RequireCapability(auth.CounselorKBWrite), processH.ListAdmin)
				processAdmin.GET("/pending", auth.RequireCapability(auth.CounselorKBReview), processH.ListPending)
				processAdmin.POST("", auth.RequireCapability(auth.CounselorKBWrite), processH.Create)
				processAdmin.GET("/:id", auth.RequireCapability(auth.CounselorKBWrite), processH.GetAdmin)
				processAdmin.PUT("/:id", auth.RequireCapability(auth.CounselorKBWrite), processH.Update)
				processAdmin.DELETE("/:id", auth.RequireCapability(auth.CounselorKBWrite), processH.Delete)
				processAdmin.POST("/:id/submit", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), processH.Submit)
				processAdmin.POST("/:id/approve", auth.RequireCapability(auth.CounselorKBReview), processH.Approve)
				processAdmin.POST("/:id/reject", auth.RequireCapability(auth.CounselorKBReview), processH.Reject)
				processAdmin.POST("/:id/retire", auth.RequireCapability(auth.CounselorKBReview), processH.Retire)
			}

			// ── 学生 AI 功能（个人能力，所有角色继承自 student 都可用）──
			student := secured.Group("/student")
			{
				student.GET("/home", auth.RequireCapability(auth.SelfStudyRead), studentH.Home)
				student.GET("/profile", auth.RequireCapability(auth.SelfTwinRead), studentH.PersonalProfile)
				student.GET("/twin-profile", auth.RequireCapability(auth.SelfTwinRead), studentH.TwinProfile)
				student.GET("/personality-profile", auth.RequireCapability(auth.SelfPersonalityRead), studentH.PersonalityProfile)
				// 头像上传与服务（GET 供前端 <img> 直接加载，认证保护）
				student.POST("/avatar", auth.RequireCapability(auth.SelfProfileWrite), studentH.UploadAvatar)
				student.GET("/avatar/:user_id", auth.RequireCapability(auth.SelfTwinRead), studentH.ServeAvatar)
				student.GET("/daily-briefing", auth.RequireCapability(auth.SelfBriefingRead), studentH.DailyBriefing)
				student.GET("/learning-diary", auth.RequireCapability(auth.SelfDiaryRead), studentH.LearningDiary)
				student.POST("/checkin", auth.RequireCapability(auth.SelfCheckinWrite), studentH.Checkin)
				student.GET("/checkin/history", auth.RequireCapability(auth.SelfCheckinWrite), studentH.CheckinHistory)
				student.POST("/checkin/makeup", auth.RequireCapability(auth.SelfCheckinWrite), studentH.CheckinMakeup)
				// 毕业去向自报（学生，待教辅审核；2026-08-15）
				student.POST("/outcome/self-report", auth.RequireCapability(auth.OutcomeRecordWrite), secretaryH.SubmitOutcome)
				student.GET("/outcome/my", auth.RequireCapability(auth.OutcomeRecordRead), secretaryH.ListOutcomes)
				student.POST("/schedule/import", auth.RequireCapability(auth.SelfProfileWrite), dataImportH.ImportMySchedule)
				student.GET("/digital-twin", auth.RequireCapability(auth.SelfTwinRead), studentH.DigitalTwin)
				student.GET("/personality", auth.RequireCapability(auth.SelfPersonalityRead), studentH.Personality)
				student.GET("/achievements", auth.RequireCapability(auth.SelfAchievements), studentH.Achievements)
				student.GET("/course-map", auth.RequireCapability(auth.SelfCourseMapRead), studentH.CourseMap)
				student.GET("/course-analytics", auth.RequireCapability(auth.SelfCourseAnalytics), studentH.CourseAnalytics)
				student.GET("/weekly-report", auth.RequireCapability(auth.SelfWeeklyReport), studentH.WeeklyReport)
				student.GET("/freshman-plan", auth.RequireCapability(auth.SelfGenericAI), studentH.GenericAI("freshman-plan"))
				student.GET("/growth-path", auth.RequireCapability(auth.SelfGenericAI), studentH.GrowthPath)
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
				student.GET("/qa/posts", auth.RequireCapability(auth.SelfCommunityRead), studentH.ListQAPosts)
				student.POST("/qa/posts", auth.RequireCapability(auth.SelfCommunityRead), studentH.CreateQAPost)
				student.GET("/qa/posts/:id", auth.RequireCapability(auth.SelfCommunityRead), studentH.GetQAPostDetail)
				student.POST("/qa/posts/:id/answer", auth.RequireCapability(auth.SelfCommunityRead), studentH.AnswerQAPost)
				student.GET("/hot-topics", auth.RequireCapability(auth.SelfCommunityRead), studentH.HotTopics)
				student.GET("/qa-leaderboard", auth.RequireCapability(auth.SelfCommunityRead), studentH.QALeaderboard)
				student.GET("/private-chat", auth.RequireCapability(auth.SelfPrivateChat), studentH.PrivateChat)
				student.GET("/process-enhanced", auth.RequireCapability(auth.SelfProcessRead), studentH.ProcessEnhanced)
				student.GET("/freshmen-guide", auth.RequireCapability(auth.SelfKnowledgeRead), studentH.FreshmenGuide)
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
				me.POST("/checkin/makeup", auth.RequireCapability(auth.SelfCheckinWrite), studentH.CheckinMakeup)
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
				// 第二课堂班级看板（辅导员）：真实聚合名下学生活动参与/积分
				counselor.GET("/second-class-board", auth.RequireCapability(auth.CounselorSecondClassBoard), counselorH.SecondClassBoard)
				// ── P2 辅导员深度分析 ──
				counselor.GET("/follow-up-reminders", auth.RequireCapability(auth.CounselorTalkRecord), counselorH.FollowUpReminders)
				counselor.GET("/checkin-stats", auth.RequireCapability(auth.CounselorClassReport), counselorH.CheckinStats)
				counselor.POST("/smart-notify", auth.RequireCapability(auth.CounselorInterventionWrite), counselorH.SmartNotify)
				counselor.GET("/monthly-brief", auth.RequireCapability(auth.CounselorClassReport), counselorH.MonthlyBrief)
				counselor.POST("/session-insight", auth.RequireCapability(auth.CounselorTalkRecord), counselorH.SessionInsight)
			}

			// ── 通知推送（辅导员及以上角色） ──
			// 移动到 /admin/notifications/push 路径下，避免与用户站内通知冲突
			notificationPush := secured.Group("/admin/notifications/push")
			notificationPush.Use(auth.RequireCapability(auth.CounselorNotify))
			{
				notificationPush.POST("", notificationH.Create)
				notificationPush.GET("", notificationH.List)
				notificationPush.POST("/:id/publish", notificationH.Publish)
				notificationPush.DELETE("/:id", notificationH.Delete)
				notificationPush.GET("/webhook-status", notificationH.WebhookStatus)
			}

			// ── 用户站内通知（所有登录用户） ──
			secured.GET("/notifications", userNotificationH.ListNotifications)
			secured.GET("/notifications/unread-count", userNotificationH.GetUnreadCount)
			secured.PUT("/notifications/:id/read", userNotificationH.MarkAsRead)
			secured.PUT("/notifications/read-all", userNotificationH.MarkAllAsRead)

			// ── 管理员发送系统通知 ──
			secured.POST("/admin/notifications/send", auth.RequireCapability(auth.SystemSettingsWrite), userNotificationH.SendSystemNotification)

			// ── 管理员站内通知管理（查看/删除/清空） ──
			secured.GET("/admin/notifications/list", auth.RequireCapability(auth.SystemSettingsWrite), userNotificationH.AdminListNotifications)
			secured.DELETE("/admin/notifications/:id", auth.RequireCapability(auth.SystemSettingsWrite), userNotificationH.AdminDeleteNotification)
			secured.DELETE("/admin/notifications", auth.RequireCapability(auth.SystemSettingsWrite), userNotificationH.AdminClearNotifications)

			// ── 文档解析 ──
			secured.POST("/documents/parse", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), documentH.ParseDocument)
			secured.POST("/documents/refine", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), documentH.RefineDocument)
			secured.GET("/documents/formats", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), documentH.SupportedFormats)

			// ── 文档上传与知识入库 ──
			secured.POST("/kb/upload", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), uploadH.Upload)
			secured.GET("/kb/formats", auth.RequireAnyCapability(auth.UnionKBSubmit, auth.CounselorKBWrite), uploadH.SupportedFormats)

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
				// ── P2 补充功能 ──
				assistantGroup.GET("/material-templates", auth.RequireCapability(auth.AssistantScheduleCheck), assistantH.MaterialTemplates)
				assistantGroup.POST("/doc-process", auth.RequireCapability(auth.AssistantScheduleCheck), assistantH.DocProcess)
				assistantGroup.GET("/workflow-automation", auth.RequireCapability(auth.AssistantScheduleCheck), assistantH.WorkflowAutomation)
				assistantGroup.GET("/process-steps-manage", auth.RequireCapability(auth.AssistantGradAudit), assistantH.ProcessStepsManage)
				assistantGroup.GET("/music-radio", auth.RequireCapability(auth.AssistantScheduleCheck), assistantH.MusicRadio)
				assistantGroup.GET("/activity-register", auth.RequireCapability(auth.AssistantScheduleCheck), assistantH.ActivityRegister)
				// ── 后勤服务台（并入教辅，2026-08-15）──
				assistantGroup.GET("/facility/roles", auth.RequireCapability(auth.FacilityRecordRead), facilityH.RoleMeta)
				assistantGroup.POST("/facility/record", auth.RequireCapability(auth.FacilityRecordWrite), facilityH.CreateRecord)
				assistantGroup.GET("/facility/records", auth.RequireCapability(auth.FacilityRecordRead), facilityH.ListRecords)
				assistantGroup.GET("/facility/dashboard", auth.RequireCapability(auth.FacilityDashboard), facilityH.Dashboard)

				// ── 毕业去向登记与审核（教辅，2026-08-15 书记教育成果闭环）──
				assistantGroup.GET("/outcome/meta", auth.RequireCapability(auth.OutcomeRecordRead), secretaryH.OutcomeMeta)
				assistantGroup.POST("/outcome/record", auth.RequireCapability(auth.OutcomeRecordWrite), secretaryH.SubmitOutcome)
				assistantGroup.GET("/outcome/records", auth.RequireCapability(auth.OutcomeRecordRead), secretaryH.ListOutcomes)
				assistantGroup.GET("/outcome/pending", auth.RequireCapability(auth.OutcomeReview), secretaryH.CountPending)
				assistantGroup.PUT("/outcome/review/:id", auth.RequireCapability(auth.OutcomeReview), secretaryH.ReviewOutcome)
			}

			// ── 学生会 AI 功能 ──
			unionGroup := secured.Group("/union")
			{
				unionGroup.POST("/event-plan", auth.RequireCapability(auth.UnionEventPlan), unionH.EventPlan)
				unionGroup.POST("/poster-gen", auth.RequireCapability(auth.UnionPosterGen), unionH.PosterGen)
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
				collegeGroup.GET("/process-step-edit", auth.RequireCapability(auth.CollegeDataAnalysis), collegeH.ProcessStepEdit)
				// ── 书记教育成果大屏（2026-08-15）：school_admin 全校/college_admin 本院 ──
				// college 参数空=全校（学校书记），传学院=本院（学院书记）
				collegeGroup.GET("/education-outcome", auth.RequireCapability(auth.OutcomeDashboard), secretaryH.OutcomeDashboard)
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
				// ── 就业指导管理（学校/学院管理员）──
				career.GET("/admin/policies", auth.RequireCapability(auth.SystemSettingsWrite), educationH.AdminListCareerPolicies)
				career.POST("/admin/policies", auth.RequireCapability(auth.SystemSettingsWrite), educationH.AdminCreateCareerPolicy)
				career.DELETE("/admin/policies/:id", auth.RequireCapability(auth.SystemSettingsWrite), educationH.AdminDeleteCareerPolicy)
				career.GET("/admin/jobs", auth.RequireCapability(auth.SystemSettingsWrite), educationH.AdminListJobPostings)
				career.POST("/admin/jobs", auth.RequireCapability(auth.SystemSettingsWrite), educationH.AdminCreateJobPosting)
				career.DELETE("/admin/jobs/:id", auth.RequireCapability(auth.SystemSettingsWrite), educationH.AdminDeleteJobPosting)
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

			// ── 身体健康模块（学生本人）──
			health := secured.Group("/health")
			{
				health.GET("/basic", auth.RequireCapability(auth.SelfHealthRead), educationH.GetHealthBasicInfo)
				health.PUT("/basic", auth.RequireCapability(auth.SelfHealthWrite), educationH.UpsertHealthBasicInfo)
				health.GET("/checkups", auth.RequireCapability(auth.SelfHealthRead), educationH.ListHealthCheckups)
				health.POST("/checkups", auth.RequireCapability(auth.SelfHealthWrite), educationH.CreateHealthCheckup)
				health.PUT("/checkups/:id", auth.RequireCapability(auth.SelfHealthWrite), educationH.UpdateHealthCheckup)
				health.DELETE("/checkups/:id", auth.RequireCapability(auth.SelfHealthWrite), educationH.DeleteHealthCheckup)
				health.GET("/records", auth.RequireCapability(auth.SelfHealthRead), educationH.ListHealthRecords)
				health.POST("/records", auth.RequireCapability(auth.SelfHealthWrite), educationH.CreateHealthRecord)
				health.PUT("/records/:id", auth.RequireCapability(auth.SelfHealthWrite), educationH.UpdateHealthRecord)
				health.DELETE("/records/:id", auth.RequireCapability(auth.SelfHealthWrite), educationH.DeleteHealthRecord)
				health.GET("/daily", auth.RequireCapability(auth.SelfHealthRead), educationH.ListHealthDaily)
				health.PUT("/daily", auth.RequireCapability(auth.SelfHealthWrite), educationH.UpsertHealthDaily)
				health.DELETE("/daily/:date", auth.RequireCapability(auth.SelfHealthWrite), educationH.DeleteHealthDaily)
				health.GET("/activities", auth.RequireCapability(auth.SelfHealthRead), educationH.ListHealthActivities)
				health.POST("/activities", auth.RequireCapability(auth.UnionEventPlan), educationH.CreateHealthActivity)
				health.POST("/activities/:id/favorite", auth.RequireCapability(auth.SelfHealthWrite), educationH.ToggleActivityFavorite)
				health.POST("/activities/:id/signup", auth.RequireCapability(auth.SelfHealthWrite), educationH.ToggleActivitySignup)
				health.POST("/activities/:id/status", auth.RequireCapability(auth.UnionEventPlan), educationH.UpdateHealthActivityStatus)
				health.POST("/activities/:id/attend/:uid", auth.RequireCapability(auth.UnionEventPlan), educationH.AttendActivitySignup)
				health.GET("/activities/review-stats", auth.RequireCapability(auth.UnionEventPlan), educationH.ActivityReviewStats)
				health.GET("/activities/:id/signups", auth.RequireCapability(auth.UnionEventPlan), educationH.ListActivitySignups)
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
	startTime := time.Now()
	return func(c *gin.Context) {
		// 数据库连通性
		dbStatus := "ok"
		dbLatency := ""
		t0 := time.Now()
		if err := db.Ping(); err != nil {
			dbStatus = "error: " + err.Error()
		} else {
			dbLatency = time.Since(t0).String()
		}

		// FTS5 可用性（MySQL 无 FTS5 虚拟表，报 unavailable）
		ftsStatus := "ok"
		if !dbutil.IsMySQL(db) {
			var ftsCheck int
			if err := db.QueryRow("SELECT 1 FROM kb_fts LIMIT 1").Scan(&ftsCheck); err != nil {
				ftsStatus = "unavailable"
			}
		} else {
			ftsStatus = "unavailable (mysql)"
		}

		// LLM API 配置（仅检查 key 是否配置，不实际调用）
		llmStatus := "configured"
		if os.Getenv("ZHIPU_API_KEY") == "" && os.Getenv("DEEPSEEK_API_KEY") == "" && os.Getenv("SPARK_API_KEY") == "" {
			llmStatus = "no_api_key"
		}

		// Redis 状态（可选组件）
		redisStatus := "disabled"
		if appRedis != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			if err := appRedis.Ping(ctx).Err(); err != nil {
				redisStatus = "error: " + err.Error()
			} else {
				redisStatus = "ok"
			}
		}

		// 总体状态
		overall := "healthy"
		if dbStatus != "ok" {
			overall = "degraded"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  overall,
			"service": "蔚小芯",
			"version": "0.0.1",
			"uptime":  time.Since(startTime).String(),
			"dependencies": gin.H{
				"database": gin.H{"status": dbStatus, "latency": dbLatency, "driver": string(dbutil.DriverOf(db))},
				"redis":    gin.H{"status": redisStatus},
				"fts5":     gin.H{"status": ftsStatus},
				"llm_api":  gin.H{"status": llmStatus},
			},
			"time": time.Now().Format(time.RFC3339),
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
