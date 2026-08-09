package config

import (
	"os"
	"testing"
)

// ── envOr 测试 ──

func TestEnvOr_ReturnsValue(t *testing.T) {
	os.Setenv("TEST_ENV_OR_KEY", "custom-value")
	defer os.Unsetenv("TEST_ENV_OR_KEY")

	result := envOr("TEST_ENV_OR_KEY", "default")
	if result != "custom-value" {
		t.Errorf("期望 custom-value，得到 %s", result)
	}
}

func TestEnvOr_ReturnsFallback(t *testing.T) {
	os.Unsetenv("TEST_ENV_OR_MISSING")
	result := envOr("TEST_ENV_OR_MISSING", "default")
	if result != "default" {
		t.Errorf("期望 default，得到 %s", result)
	}
}

func TestEnvOr_EmptyValueGoesToFallback(t *testing.T) {
	os.Setenv("TEST_ENV_OR_EMPTY", "")
	defer os.Unsetenv("TEST_ENV_OR_EMPTY")

	result := envOr("TEST_ENV_OR_EMPTY", "fallback")
	if result != "fallback" {
		t.Errorf("期望 fallback，得到 %s", result)
	}
}

// ── envIntOr 测试 ──

func TestEnvIntOr_ReturnsValue(t *testing.T) {
	os.Setenv("TEST_ENV_INT_KEY", "42")
	defer os.Unsetenv("TEST_ENV_INT_KEY")

	result := envIntOr("TEST_ENV_INT_KEY", 10)
	if result != 42 {
		t.Errorf("期望 42，得到 %d", result)
	}
}

func TestEnvIntOr_ReturnsFallback(t *testing.T) {
	os.Unsetenv("TEST_ENV_INT_MISSING")
	result := envIntOr("TEST_ENV_INT_MISSING", 10)
	if result != 10 {
		t.Errorf("期望 10，得到 %d", result)
	}
}

func TestEnvIntOr_InvalidNumberFallback(t *testing.T) {
	os.Setenv("TEST_ENV_INT_INVALID", "not-a-number")
	defer os.Unsetenv("TEST_ENV_INT_INVALID")

	result := envIntOr("TEST_ENV_INT_INVALID", 99)
	if result != 99 {
		t.Errorf("期望 99，得到 %d", result)
	}
}

func TestEnvIntOr_EmptyStringFallback(t *testing.T) {
	os.Setenv("TEST_ENV_INT_EMPTY", "")
	defer os.Unsetenv("TEST_ENV_INT_EMPTY")

	result := envIntOr("TEST_ENV_INT_EMPTY", 5)
	if result != 5 {
		t.Errorf("期望 5，得到 %d", result)
	}
}

func TestEnvIntOr_NegativeValue(t *testing.T) {
	os.Setenv("TEST_ENV_INT_NEG", "-1")
	defer os.Unsetenv("TEST_ENV_INT_NEG")

	result := envIntOr("TEST_ENV_INT_NEG", 10)
	if result != -1 {
		t.Errorf("期望 -1，得到 %d", result)
	}
}

// ── Load 测试 ──

func TestLoad_Defaults(t *testing.T) {
	// 清理可能影响测试的环境变量
	keysToClean := []string{
		"APP_PORT", "APP_MODE", "LOG_LEVEL",
		"JWT_SECRET", "JWT_EXPIRE_HOURS",
		"SQLITE_PATH",
		"ZHIPU_API_KEY", "ZHIPU_BASE_URL", "ZHIPU_MODEL",
		"DEEPSEEK_API_KEY", "DEEPSEEK_BASE_URL", "DEEPSEEK_MODEL",
		"XFYUN_APP_ID", "XFYUN_API_KEY", "XFYUN_API_SECRET", "XFYUN_SPEECH_URL",
		"XUEGONG_BASE_URL", "XUEGONG_TOKEN",
		"YBT_BASE_URL", "YBT_TOKEN",
		"SSO_BASE_URL", "SSO_CALLBACK_URL",
		"WEIYUAN_SYNC_SECRET", "WEIYUAN_IMPORT_URL",
	}
	for _, k := range keysToClean {
		os.Unsetenv(k)
	}

	cfg := Load()

	if cfg.AppPort != "8080" {
		t.Errorf("期望 AppPort=8080，得到 %s", cfg.AppPort)
	}
	if cfg.AppMode != "debug" {
		t.Errorf("期望 AppMode=debug，得到 %s", cfg.AppMode)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("期望 LogLevel=info，得到 %s", cfg.LogLevel)
	}
	if cfg.JWTExpireHours != 2 {
		t.Errorf("期望 JWTExpireHours=2，得到 %d", cfg.JWTExpireHours)
	}
	if cfg.ZhipuBaseURL != "https://open.bigmodel.cn/api/paas/v4/chat/completions" {
		t.Errorf("ZhipuBaseURL 默认值错误: %s", cfg.ZhipuBaseURL)
	}
	if cfg.ZhipuModel != "glm-4" {
		t.Errorf("ZhipuModel 默认值错误: %s", cfg.ZhipuModel)
	}
	if cfg.DeepSeekBaseURL != "https://api.deepseek.com/chat/completions" {
		t.Errorf("DeepSeekBaseURL 默认值错误: %s", cfg.DeepSeekBaseURL)
	}
	if cfg.DeepSeekModel != "deepseek-v4-flash" {
		t.Errorf("期望 DeepSeekModel=deepseek-v4-flash，得到 %s", cfg.DeepSeekModel)
	}
	if cfg.XfyunSpeechURL != "https://iat-api.xfyun.cn/v2/iat" {
		t.Errorf("XfyunSpeechURL 默认值错误: %s", cfg.XfyunSpeechURL)
	}
	if cfg.SSOCallbackURL != "http://localhost:8080/api/v1/auth/sso/callback" {
		t.Errorf("SSOCallbackURL 默认值错误: %s", cfg.SSOCallbackURL)
	}
	if cfg.SQLitePath != "./data/wxx.db" {
		t.Errorf("期望 SQLitePath=./data/wxx.db，得到 %s", cfg.SQLitePath)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("APP_PORT", "9090")
	os.Setenv("JWT_SECRET", "test-secret-123")
	os.Setenv("JWT_EXPIRE_HOURS", "24")
	os.Setenv("DEEPSEEK_API_KEY", "sk-test-key")
	defer func() {
		os.Unsetenv("APP_PORT")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("JWT_EXPIRE_HOURS")
		os.Unsetenv("DEEPSEEK_API_KEY")
	}()

	cfg := Load()

	if cfg.AppPort != "9090" {
		t.Errorf("期望 AppPort=9090，得到 %s", cfg.AppPort)
	}
	if cfg.JWTSecret != "test-secret-123" {
		t.Errorf("期望 JWTSecret=test-secret-123，得到 %s", cfg.JWTSecret)
	}
	if cfg.JWTExpireHours != 24 {
		t.Errorf("期望 JWTExpireHours=24，得到 %d", cfg.JWTExpireHours)
	}
	if cfg.DeepSeekAPIKey != "sk-test-key" {
		t.Errorf("期望 DeepSeekAPIKey=sk-test-key，得到 %s", cfg.DeepSeekAPIKey)
	}
}
