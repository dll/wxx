package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dll/wxx/server/internal/agent"
	"github.com/dll/wxx/server/internal/config"
	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/handler"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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
	if err := runMigrations(db, driver, nil); err != nil {
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
	// A1 AI 运行基座：LLM 调用审计日志（trace_id 贯穿、路由结果、延迟、用量）
	chatSvc.SetLLMCallLogRepo(repository.NewLLMCallLogRepo(db))
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

	kgSvc := service.NewKnowledgeGovernanceService(kbRepo)
	kgSvc.SetLLMClient(llmClient)
	kgHandler := handler.NewKnowledgeGovernanceHandler(kgSvc)

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
	// 反馈修复任务（闭环 MVP）：审核创建 → 执行端领取/验证 → 管理员验收 → 部署确认/完成。
	// 服务器仅做状态机与审计，绝不执行改码/构建/部署；内部执行端走 WXX_REPAIR_AGENT_TOKEN 专用鉴权。
	feedbackRepairTaskSvc := service.NewFeedbackRepairTaskService(
		repository.NewFeedbackRepairTaskRepo(db), feedbackSvc)
	feedbackRepairTaskHandler := handler.NewFeedbackRepairTaskHandler(feedbackRepairTaskSvc)
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
	vopcHandler := handler.NewVOPCHandler(db, cfg.VOPCCollegeID)
	if dir := strings.TrimSpace(cfg.VOPCUploadDir); dir != "" {
		vopcHandler.SetUploadDir(dir)
	}
	// v2.0 虚拟向导：不再注入真实 LLM（模板化草稿生成，零外部模型依赖）。
	// L4 远期若接入真实执行，可在此处显式 SetLLMClient（当前为兼容空实现）。
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
	// R3 成绩强校验接线：写库前查 teacher_courses 授课关系是否已 approved（仅 approved 放行）
	phase3Svc.SetTeacherCourseRepo(repository.NewTeacherCourseRepo(db))
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

	// 督办工单（2026-08-16，D5-3「洞察→工单」治理回环）：
	// 书记从治理洞察/KPI 生成督办工单分派给辅导员/教辅/党群，最小实现复用 feedback 流转心智。
	// 复用 secretaryRepo 读取育人 KPI（D5-1 联动：not_available 补料指标一键生成补料工单）。
	govTicketRepo := repository.NewGovTicketRepo(db)
	govTicketSvc := service.NewGovTicketService(govTicketRepo, secretaryRepo)
	govTicketHandler := handler.NewGovTicketHandler(govTicketSvc)

	// 教师授课关系申报+教辅审核（2026-08-17，R3 越权边界升级）：
	// approved 唯一来源为教辅真实审核操作，不脚本批量置位。
	teacherCourseRepo := repository.NewTeacherCourseRepo(db)
	teacherCourseSvc := service.NewTeacherCourseService(teacherCourseRepo)
	teacherCourseHandler := handler.NewTeacherCourseHandler(teacherCourseSvc)

	// 教师作业信息发布+成绩统计（2026-08-17，P2 轻量版）：
	// 复用 TeacherGradeWrite 门控；发布/编辑前经 GetTeacherCourseStatus 强校验 approved 授课关系；
	// 成绩统计基于真实 student_grades 只读聚合，不做新导入、不造数据。
	homeworkRepo := repository.NewHomeworkRepo(db)
	homeworkSvc := service.NewHomeworkService(homeworkRepo, teacherCourseSvc)
	homeworkHandler := handler.NewHomeworkHandler(homeworkSvc)

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
	userNotificationHandler := handler.NewUserNotificationHandler(repository.NewUserNotificationRepo(db))
	statsHandler := handler.NewStatsHandler(repository.NewStatsRepo(db))
	appVersionRepo := repository.NewAppVersionRepo(db)
	appVersionService := service.NewAppVersionService(appVersionRepo)
	appVersionHandler := handler.NewAppVersionHandler(appVersionService)
	// 校园报到步骤（直接注入 db，无独立 Service 层）
	campusHandler := handler.NewCampusHandler(repository.NewCampusRepository(db))

	// ── 5. 构建路由 ──
	// 依赖已收敛为 deps 结构体：cfg/db/userRepo 与 45 个 handler 全部打包传入 setupRouter。
	// 与改造前完全同源：handler 实例仍是上方构造的那同一批对象、同一构造方式。
	router := setupRouter(&deps{
		cfg:      cfg,
		db:       db,
		userRepo: userRepo,

		authH:               authHandler,
		sessionH:            sessionHandler,
		chatH:               chatHandler,
		kbH:                 kbHandler,
		kgH:                 kgHandler,
		voiceH:              voiceHandler,
		emotionH:            emotionHandler,
		agentH:              agentHandler,
		exportH:             exportHandler,
		integrationH:        integrationHandler,
		recH:                recHandler,
		adminH:              adminHandler,
		feedbackH:           feedbackHandler,
		modelConfigH:        modelConfigHandler,
		tokenStatsH:         tokenStatsHandler,
		studentH:            studentHandler,
		counselorH:          counselorHandler,
		teacherH:            teacherHandler,
		assistantH:          assistantHandler,
		facilityH:           facilityHandler,
		secretaryH:          secretaryHandler,
		unionH:              unionHandler,
		collegeH:            collegeHandler,
		cultureH:            cultureHandler,
		schoolAdminH:        schoolAdminHandler,
		sysAdminH:           sysAdminHandler,
		govTicketH:          govTicketHandler,
		teacherCourseH:      teacherCourseHandler,
		homeworkH:           homeworkHandler,
		processRecordH:      processRecordHandler,
		processH:            processHandler,
		forecastH:           forecastHandler,
		graduationH:         graduationHandler,
		studentFeaturesH:    studentFeaturesHandler,
		notificationH:       notificationHandler,
		uploadH:             uploadHandler,
		documentH:           documentHandler,
		educationH:          educationHandler,
		studyPlanH:          studyPlanHandler,
		statsH:              statsHandler,
		userNotificationH:   userNotificationHandler,
		appVersionH:         appVersionHandler,
		campusH:             campusHandler,
		dataImportH:         dataImportH,
		externalAppH:        externalAppHandler,
		aiBriefingH:         aiBriefingHandler,
		twinPortraitH:       twinPortraitHandler,
		portalCredH:         portalCredHandler,
		portalProxyH:        portalProxyHandler,
		vopcH:               vopcHandler,
		feedbackRepairTaskH: feedbackRepairTaskHandler,
	})

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
