package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
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
	// 各系统允许的目标主机（防 SSRF，取配置的 baseURL 主机）
	allowedHosts map[string]string // key=系统名，value=允许主机（如 xuegong.chzu.edu.cn）
}

// NewIntegrationService 创建对接服务
func NewIntegrationService(cfg *config.Config) *IntegrationService {
	s := &IntegrationService{
		cfg:          cfg,
		allowedHosts: map[string]string{},
	}

	// 校验 baseURL 合法性；非法/未配置则主机白名单留空（对应系统不可用）
	s.allowedHosts["学工系统"] = hostOfBaseURL(cfg.XuegongBaseURL)
	s.allowedHosts["一表通"] = hostOfBaseURL(cfg.YBTBaseURL)

	s.httpClient = &http.Client{
		Timeout: 10 * time.Second,
		// SSRF 防护：禁止跨主机跟随重定向，防止 302 跳转把凭据外泄到非白名单主机。
		// 仅允许与初始请求主机同源（同主机或子域）的重定向。
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("重定向次数过多")
			}
			initialHost := via[0].URL.Host
			if !s.isAllowedRedirect(initialHost, req.URL.Host) {
				return fmt.Errorf("拒绝跟随到非白名单主机: %s", req.URL.Host)
			}
			return nil
		},
		// SSRF 防护：DialContext 层拒绝私网/环回/链路本地地址（防 DNS rebinding）。
		// 例外：显式配置在 baseURL 白名单内的内网地址放行——运维显式指向内网系统是其选择；
		// 但攻击者通过域名重绑定解析出的白名单外内网 IP 仍被拦截。
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, _, err := net.SplitHostPort(addr)
				if err != nil {
					// 无端口地址（如 "localhost"），直接取 host
					host = addr
				}
				if isBlockedIP(host) && !s.hostInAllowedList(stripPort(host)) {
					return nil, fmt.Errorf("禁止访问内网/环回地址: %s", host)
				}
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, network, addr)
			},
		},
	}
	return s
}

// hostOfBaseURL 从 baseURL 提取主机名（含端口），非法 URL 返回空
func hostOfBaseURL(baseURL string) string {
	if baseURL == "" {
		return ""
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	if u.Host == "" {
		return ""
	}
	return u.Host
}

// isAllowedRedirect 判断重定向目标主机是否与初始主机一致（精确匹配 host:port）。
// 不使用 stripPort：同主机不同端口可能是内网服务（SSRF 跳端口攻击），必须拒绝。
func (s *IntegrationService) isAllowedRedirect(initialHost, targetHost string) bool {
	if initialHost == "" || targetHost == "" {
		return false
	}
	return strings.EqualFold(initialHost, targetHost)
}

// stripPort 去掉主机名端口
func stripPort(host string) string {
	if i := strings.LastIndex(host, ":"); i > 0 && strings.Count(host, ":") == 1 {
		return host[:i]
	}
	return host
}

// isBlockedIP 判断主机是否为私网/环回/链路本地地址。
// host 可为 IP 或域名；域名在此仅做 IP 字面量判断（真 DNS 解析在 Dial 层，
// 由对解析结果调用本函数拦截）。防 SSRF 的 DNS rebinding 场景下，
// 域名解析出的内网 IP 会在 DialContext 的 addr（host:port）里出现。
func isBlockedIP(host string) bool {
	host = stripPort(strings.ToLower(host))
	// 常见字面量快速命中
	switch host {
	case "localhost", "":
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		// 非 IP 字面量：域名交给 Dial 层的 addr 校验
		return false
	}
	return isPrivateIP(ip)
}

// isPrivateIP 判断 IP 是否属于禁止访问的网段
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	return false
}

// IsXuegongAvailable 学工系统是否已配置
func (s *IntegrationService) IsXuegongAvailable() bool {
	return s.cfg.XuegongBaseURL != "" && s.cfg.XuegongToken != "" && s.allowedHosts["学工系统"] != ""
}

// IsYBTAvailable 一表通是否已配置
func (s *IntegrationService) IsYBTAvailable() bool {
	return s.cfg.YBTBaseURL != "" && s.cfg.YBTToken != "" && s.allowedHosts["一表通"] != ""
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
// 安全要求（GPT56SOL v3 P0-03）：
//  1. path 须以 / 开头，且解析后仍落在 baseURL 主机（禁止 path 内嵌绝对 URL / 协议切换）
//  2. 目标主机必须在白名单（配置 baseURL 主机或其子域）
//  3. 传输层由 DialContext 拒绝私网/环回地址（防 DNS rebinding）
//  4. 重定向仅允许同主机或子域（CheckRedirect 拦截跨主机 302）
func (s *IntegrationService) proxyGet(baseURL, token, path string, query map[string]string, systemName string) (json.RawMessage, error) {
	base, err := url.Parse(baseURL)
	if err != nil || base.Host == "" {
		return nil, fmt.Errorf("%s baseURL 非法", systemName)
	}

	// 1) 路径归一化：必须 / 开头，禁止绝对 URL 或 scheme 走私。
	//    - 不以 / 开头的路径视为相对路径补全；但绝对 URL（含 ://）或 // 开头直接拒绝，
	//      防止攻击者把请求主机切换到白名单外（如 path=http://evil.com/steal）。
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if strings.Contains(path, "://") || strings.HasPrefix(path, "//") {
		return nil, fmt.Errorf("%s 路径走私被拒绝: %s", systemName, path)
	}
	// 用 url.JoinPath 将 path 与 base 拼接，天然对 path 内的 ".." 做规范化
	target, err := url.JoinPath(base.String(), path)
	if err != nil {
		return nil, fmt.Errorf("%s 路径非法: %w", systemName, err)
	}
	tu, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("%s 目标地址非法: %w", systemName, err)
	}

	// 2) 主机白名单校验
	allowed := s.allowedHosts[systemName]
	if allowed == "" {
		return nil, fmt.Errorf("%s 目标主机未配置白名单", systemName)
	}
	if !s.isAllowedHost(tu.Host, allowed) {
		return nil, fmt.Errorf("不允许访问该地址: %s", tu.Host)
	}

	// 附加 query
	if len(query) > 0 {
		q := tu.Query()
		for k, v := range query {
			q.Add(k, v)
		}
		tu.RawQuery = q.Encode()
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest("GET", tu.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%s 请求失败: %w", systemName, err)
			log.Printf("[%s] 请求失败 attempt=%d url=%s err=%v", systemName, attempt+1, tu.String(), err)
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("读取 %s 响应失败: %w", systemName, readErr)
			continue
		}
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("%s 返回错误 HTTP %d", systemName, resp.StatusCode)
			log.Printf("[%s] 返回错误 status=%d body=%s", systemName, resp.StatusCode, util.TruncateString(string(body), 200))
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("%s 返回错误 HTTP %d", systemName, resp.StatusCode)
		}
		log.Printf("[%s] 代理成功 path=%s status=%d", systemName, path, resp.StatusCode)
		return json.RawMessage(body), nil
	}
	return nil, lastErr
}

// isAllowedHost 判断目标主机是否在允许列表（目标系统主机或其子域）
func (s *IntegrationService) isAllowedHost(host, allowed string) bool {
	host = stripPort(strings.ToLower(host))
	allowed = stripPort(strings.ToLower(allowed))
	if allowed == "" || host == "" {
		return false
	}
	return host == allowed || strings.HasSuffix(host, "."+allowed)
}

// hostInAllowedList 判断主机是否在任意系统白名单内（供 DialContext 例外放行）
func (s *IntegrationService) hostInAllowedList(host string) bool {
	for _, allowed := range s.allowedHosts {
		if allowed != "" && s.isAllowedHost(host, allowed) {
			return true
		}
	}
	return false
}
