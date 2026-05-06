package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/util"
)

// IntegrationService 校外系统代理服务（只读）
// 注：不通过 Temporal 调度（活动通过函数字段注入，避免循环依赖），
// 直接代理 HTTP 请求，由调用方控制重试策略。
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
	url := baseURL + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	if len(query) > 0 {
		q := req.URL.Query()
		for k, v := range query {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[%s] 请求失败 url=%s err=%v", systemName, url, err)
		return nil, fmt.Errorf("%s 请求超时或不可达", systemName)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 限制 1MB
	if err != nil {
		return nil, fmt.Errorf("读取 %s 响应失败: %w", systemName, err)
	}

	if resp.StatusCode >= 400 {
		log.Printf("[%s] 返回错误 status=%d body=%s", systemName, resp.StatusCode, util.TruncateString(string(body), 200))
		return nil, fmt.Errorf("%s 返回错误 HTTP %d", systemName, resp.StatusCode)
	}

	log.Printf("[%s] 代理成功 path=%s status=%d", systemName, path, resp.StatusCode)
	return json.RawMessage(body), nil
}
