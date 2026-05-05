package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/config"
)

// IntegrationService 校外系统代理服务（只读）
type IntegrationService struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewIntegrationService 创建对接服务
func NewIntegrationService(cfg *config.Config) *IntegrationService {
	return &IntegrationService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IsXuegongAvailable 学工系统是否已配置
func (s *IntegrationService) IsXuegongAvailable() bool {
	return s.cfg.XuegongBaseURL != "" && s.cfg.XuegongToken != ""
}

// IsYBTAvailable 一表通是否已配置
func (s *IntegrationService) IsYBTAvailable() bool {
	return s.cfg.YBTBaseURL != "" && s.cfg.YBTToken != ""
}

// ProxyXuegong 代理转发 GET 请求到学工系统
func (s *IntegrationService) ProxyXuegong(path string, query map[string]string) (json.RawMessage, error) {
	if !s.IsXuegongAvailable() {
		return nil, fmt.Errorf("学工系统未配置（请设置 XUEGONG_BASE_URL 和 XUEGONG_TOKEN）")
	}
	return s.proxyGet(s.cfg.XuegongBaseURL, s.cfg.XuegongToken, path, query, "学工系统")
}

// ProxyYBT 代理转发 GET 请求到一表通
func (s *IntegrationService) ProxyYBT(path string, query map[string]string) (json.RawMessage, error) {
	if !s.IsYBTAvailable() {
		return nil, fmt.Errorf("一表通未配置（请设置 YBT_BASE_URL 和 YBT_TOKEN）")
	}
	return s.proxyGet(s.cfg.YBTBaseURL, s.cfg.YBTToken, path, query, "一表通")
}

// proxyGet 通用 GET 代理
func (s *IntegrationService) proxyGet(baseURL, token, path string, query map[string]string, systemName string) (json.RawMessage, error) {
	// 拼接完整 URL
	url := baseURL + path

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置认证头
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	// 拼接查询参数
	if len(query) > 0 {
		q := req.URL.Query()
		for k, v := range query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[%s] 请求失败 url=%s err=%v", systemName, url, err)
		return nil, fmt.Errorf("%s 请求超时或不可达", systemName)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 限制 1MB
	if err != nil {
		return nil, fmt.Errorf("读取 %s 响应失败: %w", systemName, err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode >= 400 {
		log.Printf("[%s] 返回错误 status=%d body=%s", systemName, resp.StatusCode, truncateStr(string(body), 200))
		return nil, fmt.Errorf("%s 返回错误 HTTP %d", systemName, resp.StatusCode)
	}

	log.Printf("[%s] 代理成功 path=%s status=%d", systemName, path, resp.StatusCode)
	return json.RawMessage(body), nil
}

// truncateStr 截断字符串用于日志
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
