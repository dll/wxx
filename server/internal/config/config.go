package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 应用全局配置，从环境变量加载
type Config struct {
	// 服务
	AppPort  string // 监听端口
	AppMode  string // debug | release
	LogLevel string // debug | info | warn | error

	// JWT
	JWTSecret      string // 签名密钥
	JWTExpireHours int    // 过期时间（小时）

	// SQLite
	SQLitePath string // 数据库文件路径

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
	SSOBaseURL     string
	SSOCallbackURL string

	// 蔚园智答同步
	WeiyuanSyncSecret string
	WeiyuanImportURL  string

	// 通知推送 Webhook（QQ群/微信群）
	QQWebhookURL    string // QQ 群机器人 Webhook URL
	WechatWebhookURL string // 企业微信机器人 Webhook URL

	// Temporal 工作流引擎
	TemporalHostPort  string // e.g., "localhost:7233"（空 = 禁用）
	TemporalNamespace string // e.g., "wxx"
	TemporalTaskQueue string // e.g., "wxx-critical"
}

// Load 加载配置。优先从 .env 文件读取，再从系统环境变量补充。
func Load() *Config {
	// 开发环境加载 .env，生产环境忽略（依赖系统环境变量）
	// 兼容从 server/ 子目录启动的情况，依次尝试当前目录和父目录
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	return &Config{
		AppPort:  envOr("APP_PORT", "8080"),
		AppMode:  envOr("APP_MODE", "debug"),
		LogLevel: envOr("LOG_LEVEL", "info"),

		JWTSecret:      envOr("JWT_SECRET", ""),
		JWTExpireHours: envIntOr("JWT_EXPIRE_HOURS", 2),

		SQLitePath: envOr("SQLITE_PATH", "./data/wxx.db"),

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

		SSOBaseURL:     envOr("SSO_BASE_URL", ""),
		SSOCallbackURL: envOr("SSO_CALLBACK_URL", "http://localhost:8080/api/v1/auth/callback"),

		WeiyuanSyncSecret: envOr("WEIYUAN_SYNC_SECRET", ""),
		WeiyuanImportURL:  envOr("WEIYUAN_IMPORT_URL", ""),

		// 通知推送 Webhook
		QQWebhookURL:     envOr("QQ_WEBHOOK_URL", ""),
		WechatWebhookURL: envOr("WECHAT_WEBHOOK_URL", ""),

		// Temporal（空 = 禁用）
		TemporalHostPort:  envOr("TEMPORAL_HOST_PORT", ""),
		TemporalNamespace: envOr("TEMPORAL_NAMESPACE", "wxx"),
		TemporalTaskQueue: envOr("TEMPORAL_TASK_QUEUE", "wxx-critical"),
	}
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
