package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const defaultJWTSecret = "dev-secret-not-for-production-min-32-chars!!"

// Config 应用全局配置，从环境变量加载
type Config struct {
	// 服务
	AppPort  string // 监听端口
	AppMode  string // debug | release
	LogLevel string // debug | info | warn | error

	// JWT
	JWTSecret      string // 签名密钥，优先从环境变量 JWT_SECRET 读取
	JWTExpireHours int    // 过期时间（小时）

	// SQLite
	SQLitePath string // 数据库文件路径，优先从环境变量 DB_PATH 读取，其次 SQLITE_PATH

	// Turso 云数据库（用于 sync-db 和数据同步）
	TursoDBUrl   string // libsql://host 格式
	TursoDBToken string // 认证令牌

	// 智谱清言
	ZhipuAPIKey  string
	ZhipuBaseURL string
	ZhipuModel   string

	// 智谱 GLM-4V（多模态）
	Zhipu4VAPIKEY string
	Zhipu4VModel  string

	// DeepSeek
	DeepSeekAPIKey  string
	DeepSeekBaseURL string
	DeepSeekModel   string

	// 讯飞星火（语音）
	XfyunAppID     string
	XfyunAPIKey    string
	XfyunAPISecret string
	XfyunSpeechURL string

	// 学工系统
	XuegongBaseURL string
	XuegongToken   string

	// 一表通
	YBTBaseURL string
	YBTToken   string

	// SSO
	SSOBaseURL      string
	SSOCallbackURL  string
	SSOClientID     string
	SSOClientSecret string
	SSOUserInfoPath string
	SSOMock         bool

	// 蔚园智答同步
	WeiyuanSyncSecret string
	WeiyuanImportURL  string

	// 通知推送 Webhook（QQ群/微信群）
	QQWebhookURL     string // QQ 群机器人 Webhook URL
	WechatWebhookURL string // 企业微信机器人 Webhook URL

	// Temporal 工作流引擎
	TemporalHostPort  string // e.g., "localhost:7233"（空 = 禁用）
	TemporalNamespace string // e.g., "wxx"
	TemporalTaskQueue string // e.g., "wxx-critical"

	// LLM 配额
	DailyChatQuotaPerUser   int // 每个用户每日对话次数上限，0 表示不限
	MonthlyChatQuotaPerUser int // 每个用户每月对话次数上限，0 表示不限

	// CORS
	CORSAllowedOrigins string // 允许的跨域来源，逗号分隔，支持通配符子域名如 *.vercel.app，默认 "*"

	// 前端静态文件目录（临时 8080 直连方案）
	FrontendStaticDir string // Flutter Web 构建产物目录，如 /opt/wxx/frontend/web

	// 游客手机注册开关（预研期默认关闭：无短信通道，账号走管理员导入）
	EnableGuestRegister bool // 环境变量 ENABLE_GUEST_REGISTER，默认 false

	// 知识同步包签名（Q-05）
	HMACSecret string // 知识导出包 HMAC-SHA256 签名密钥（环境变量 HMAC_SECRET）

	// 数据保留策略（9.2 合规基线）
	RetentionAuditDays     int // 审计日志保留天数，默认 180
	RetentionSessionDays   int // 会话/消息保留天数，默认 365（1 学年）
	RetentionEmotionDays   int // 情感记录保留天数，默认 365
	RetentionExportDays    int // 导出日志保留天数，默认 180
	RetentionIntervalHours int // 清理任务间隔小时，默认 24

	// 导出字体（用于 PDF/PNG 中文渲染，空值自动探测）
	ExportFontPath string
}

// Load 加载配置。优先从 .env 文件读取，再从系统环境变量补充。
func Load() *Config {
	// 开发环境加载 .env，生产环境忽略（依赖系统环境变量）
	// 兼容从 server/ 子目录启动的情况，依次尝试当前目录和父目录
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	cfg := &Config{
		AppPort:  envOr("APP_PORT", "8080"),
		AppMode:  envOr("APP_MODE", "debug"),
		LogLevel: envOr("LOG_LEVEL", "info"),

		JWTSecret:      envOr("JWT_SECRET", defaultJWTSecret),
		JWTExpireHours: envIntOr("JWT_EXPIRE_HOURS", 2),

		SQLitePath: envOr("DB_PATH", envOr("SQLITE_PATH", "./data/wxx.db")),

		FrontendStaticDir: envOr("FRONTEND_STATIC_DIR", "/opt/wxx/frontend/web"),
		EnableGuestRegister: envBoolOr("ENABLE_GUEST_REGISTER", false),

		TursoDBUrl:   envOr("TURSO_DB_URL", ""),
		TursoDBToken: envOr("TURSO_DB_TOKEN", ""),

		ZhipuAPIKey:  envOr("ZHIPU_API_KEY", ""),
		ZhipuBaseURL: envOr("ZHIPU_BASE_URL", "https://open.bigmodel.cn/api/paas/v4/chat/completions"),
		ZhipuModel:   envOr("ZHIPU_MODEL", "glm-4"),

		Zhipu4VAPIKEY: envOr("ZHIPU_4V_API_KEY", ""),
		Zhipu4VModel:  envOr("ZHIPU_4V_MODEL", "glm-4v"),

		DeepSeekAPIKey:  envOr("DEEPSEEK_API_KEY", ""),
		DeepSeekBaseURL: envOr("DEEPSEEK_BASE_URL", "https://api.deepseek.com/chat/completions"),
		DeepSeekModel:   envOr("DEEPSEEK_MODEL", "deepseek-v4-pro"),

		XfyunAppID:     envOr("XFYUN_APP_ID", ""),
		XfyunAPIKey:    envOr("XFYUN_API_KEY", ""),
		XfyunAPISecret: envOr("XFYUN_API_SECRET", ""),
		XfyunSpeechURL: envOr("XFYUN_SPEECH_URL", "https://iat-api.xfyun.cn/v2/iat"),

		XuegongBaseURL: envOr("XUEGONG_BASE_URL", ""),
		XuegongToken:   envOr("XUEGONG_TOKEN", ""),

		YBTBaseURL: envOr("YBT_BASE_URL", ""),
		YBTToken:   envOr("YBT_TOKEN", ""),

		SSOBaseURL:      envOr("SSO_BASE_URL", ""),
		SSOCallbackURL:  envOr("SSO_CALLBACK_URL", "http://localhost:8080/api/v1/auth/sso/callback"),
		SSOClientID:     envOr("SSO_CLIENT_ID", ""),
		SSOClientSecret: envOr("SSO_CLIENT_SECRET", ""),
		SSOUserInfoPath: envOr("SSO_USERINFO_PATH", "/userinfo"),
		SSOMock:         envBoolOr("SSO_MOCK", false),

		WeiyuanSyncSecret: envOr("WEIYUAN_SYNC_SECRET", ""),
		WeiyuanImportURL:  envOr("WEIYUAN_IMPORT_URL", ""),

		// 通知推送 Webhook
		QQWebhookURL:     envOr("QQ_WEBHOOK_URL", ""),
		WechatWebhookURL: envOr("WECHAT_WEBHOOK_URL", ""),

		// Temporal（空 = 禁用）
		TemporalHostPort:  envOr("TEMPORAL_HOST_PORT", ""),
		TemporalNamespace: envOr("TEMPORAL_NAMESPACE", "wxx"),
		TemporalTaskQueue: envOr("TEMPORAL_TASK_QUEUE", "wxx-critical"),

		// LLM 配额
		DailyChatQuotaPerUser:   envIntOr("DAILY_CHAT_QUOTA_PER_USER", 200),
		MonthlyChatQuotaPerUser: envIntOr("MONTHLY_CHAT_QUOTA_PER_USER", 3000),

		// CORS
		CORSAllowedOrigins: envOr("CORS_ALLOWED_ORIGINS", "*"),

		// 知识同步包签名（Q-05）
		HMACSecret: envOr("HMAC_SECRET", ""),

		// 数据保留策略
		RetentionAuditDays:     envIntOr("RETENTION_AUDIT_DAYS", 180),
		RetentionSessionDays:   envIntOr("RETENTION_SESSION_DAYS", 365),
		RetentionEmotionDays:   envIntOr("RETENTION_EMOTION_DAYS", 365),
		RetentionExportDays:    envIntOr("RETENTION_EXPORT_DAYS", 180),
		RetentionIntervalHours: envIntOr("RETENTION_INTERVAL_HOURS", 24),

		ExportFontPath: envOr("EXPORT_FONT_PATH", ""),
	}

	if err := cfg.Validate(); err != nil {
		if cfg.IsRelease() {
			log.Fatalf("[FATAL] 配置验证失败: %v", err)
		} else {
			log.Printf("[WARN] 配置验证警告(debug模式可忽略): %v", err)
		}
	}

	return cfg
}

// IsRelease 判断当前是否为生产模式
func (c *Config) IsRelease() bool {
	return c.AppMode == "release"
}

// TursoDSN 构建包含认证令牌的 Turso 连接字符串
func (c *Config) TursoDSN() string {
	if c.TursoDBUrl == "" || c.TursoDBToken == "" {
		return ""
	}
	if strings.Contains(c.TursoDBUrl, "?") {
		return c.TursoDBUrl + "&authToken=" + c.TursoDBToken
	}
	return c.TursoDBUrl + "?authToken=" + c.TursoDBToken
}

// Validate 集中做配置校验，返回第一个遇到的错误（A-09 增强版）
func (c *Config) Validate() error {
	// ── JWT 安全 ──
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET 不能为空")
	}
	if c.IsRelease() && c.JWTSecret == defaultJWTSecret {
		return fmt.Errorf("JWT_SECRET 使用了默认值，生产环境必须配置为自定义强密钥")
	}
	if c.IsRelease() && len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET 长度不足 32 位（当前 %d 位），生产环境密钥至少需要 32 字符", len(c.JWTSecret))
	}

	// ── 端口合法性 ──
	port, err := strconv.Atoi(c.AppPort)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("APP_PORT 不合法（当前值 %q），需为 1~65535 的整数", c.AppPort)
	}

	// ── 至少配置一个 LLM API Key ──
	if c.ZhipuAPIKey == "" && c.DeepSeekAPIKey == "" && c.XfyunAPIKey == "" {
		return fmt.Errorf("至少需要配置一个 LLM API Key（ZHIPU_API_KEY / DEEPSEEK_API_KEY / XFYUN_API_KEY）")
	}

	// ── SQLite 路径 ──
	if c.SQLitePath == "" {
		return fmt.Errorf("DB_PATH / SQLITE_PATH 不能为空")
	}

	// ── AppMode 枚举 ──
	if c.AppMode != "debug" && c.AppMode != "release" && c.AppMode != "test" {
		return fmt.Errorf("APP_MODE 需为 debug / release / test（当前值 %q）", c.AppMode)
	}

	return nil
}

// envOr 读取环境变量，不存在时返回默认值
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envIntOr 读取环境变量并转为整数，失败或不存在时返回默认值
func envIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBoolOr(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return strings.EqualFold(v, "true") || v == "1"
}
