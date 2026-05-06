package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/temporal"
	"github.com/dll/wxx/server/internal/temporal/workflows"
	"github.com/dll/wxx/server/internal/util"
	sdkclient "go.temporal.io/sdk/client"
)

// IntegrationService 校外系统代理服务（只读）
type IntegrationService struct {
	cfg            *config.Config
	httpClient     *http.Client
	temporalClient *temporal.Client // 可选：Temporal 工作流客户端
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

// SetTemporalClient 设置 Temporal 客户端（nil = 走直接调用路径）
func (s *IntegrationService) SetTemporalClient(tc *temporal.Client) {
	s.temporalClient = tc
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
// 当 Temporal 已配置时，通过工作流引擎执行（获得重试保护）
func (s *IntegrationService) ProxyXuegong(path string, query map[string]string) (json.RawMessage, error) {
	if !s.IsXuegongAvailable() {
		return nil, fmt.Errorf("学工系统未配置（请设置 XUEGONG_BASE_URL 和 XUEGONG_TOKEN）")
	}
	if s.temporalClient != nil {
		return s.proxyViaTemporal("xuegong", path, query)
	}
	return s.proxyGet(s.cfg.XuegongBaseURL, s.cfg.XuegongToken, path, query, "学工系统")
}

// ProxyYBT 代理转发 GET 请求到一表通
// 当 Temporal 已配置时，通过工作流引擎执行（获得重试保护）
func (s *IntegrationService) ProxyYBT(path string, query map[string]string) (json.RawMessage, error) {
	if !s.IsYBTAvailable() {
		return nil, fmt.Errorf("一表通未配置（请设置 YBT_BASE_URL 和 YBT_TOKEN）")
	}
	if s.temporalClient != nil {
		return s.proxyViaTemporal("ybt", path, query)
	}
	return s.proxyGet(s.cfg.YBTBaseURL, s.cfg.YBTToken, path, query, "一表通")
}

// proxyViaTemporal 通过 Temporal 工作流执行代理请求
func (s *IntegrationService) proxyViaTemporal(system, path string, query map[string]string) (json.RawMessage, error) {
	ctx := context.Background()
	workflowOpts := sdkclient.StartWorkflowOptions{
		ID:                       fmt.Sprintf("proxy-%s-%d", system, time.Now().UnixNano()),
		TaskQueue:                s.temporalClient.TaskQueue(),
		WorkflowExecutionTimeout: 60 * time.Second,
	}

	input := workflows.IntegrationProxyInput{
		System: system,
		Path:   path,
		Query:  query,
	}

	run, err := s.temporalClient.SDKClient().ExecuteWorkflow(ctx, workflowOpts, workflows.IntegrationProxyWorkflow, input)
	if err != nil {
		log.Printf("启动代理工作流失败: %v，使用直接调用", err)
		// 降级到直接调用
		if system == "xuegong" {
			return s.proxyGet(s.cfg.XuegongBaseURL, s.cfg.XuegongToken, path, query, "学工系统")
		}
		return s.proxyGet(s.cfg.YBTBaseURL, s.cfg.YBTToken, path, query, "一表通")
	}

	var output workflows.IntegrationProxyOutput
	err = run.Get(ctx, &output)
	if err != nil {
		return nil, fmt.Errorf("代理工作流执行失败: %w", err)
	}

	return json.RawMessage(output.BodyJSON), nil
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
		log.Printf("[%s] 返回错误 status=%d body=%s", systemName, resp.StatusCode, util.TruncateString(string(body), 200))
		return nil, fmt.Errorf("%s 返回错误 HTTP %d", systemName, resp.StatusCode)
	}

	log.Printf("[%s] 代理成功 path=%s status=%d", systemName, path, resp.StatusCode)
	return json.RawMessage(body), nil
}
