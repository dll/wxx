package service

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dll/wxx/server/internal/repository"
)

// PortalSession 单个用户的门户登录会话
type PortalSession struct {
	account   string
	password  string
	portalURL string
	client    *http.Client
	jar       *cookiejar.Jar
	expireAt  time.Time
	mu        sync.Mutex
}

// PortalProxyService 学校门户会话代理服务
// 以用户保存的门户凭证登录 my0.chzu.edu.cn，获得会话 cookie，
// 代理访问校内各级网站页面。凭证仅用于登录，不落日志、不回显。
type PortalProxyService struct {
	credRepo    *repository.PortalCredentialRepo
	loginPath   string
	accountKey  string
	passwordKey string
	// 允许代理的目标域（防 SSRF，默认仅门户本身及其子域）
	allowedHosts []string
	ttl          time.Duration
	mu           sync.Mutex
	sessions     map[int64]*PortalSession
}

// NewPortalProxyService 创建门户代理服务。
// 门户登录为统一身份认证，表单字段可在部署期通过环境变量配置；
// 默认适配多数高校门户的 username/password 表单。
func NewPortalProxyService(credRepo *repository.PortalCredentialRepo) *PortalProxyService {
	return &PortalProxyService{
		credRepo:     credRepo,
		loginPath:    "/login",
		accountKey:   "username",
		passwordKey:  "password",
		allowedHosts: []string{"my0.chzu.edu.cn", "chzu.edu.cn"},
		ttl:          2 * time.Hour,
		sessions:     map[int64]*PortalSession{},
	}
}

// getSession 获取（或创建）用户门户会话，未登录则先用凭证登录
func (s *PortalProxyService) getSession(userID int64) (*PortalSession, error) {
	s.mu.Lock()
	ss := s.sessions[userID]
	s.mu.Unlock()
	if ss != nil && time.Now().Before(ss.expireAt) {
		return ss, nil
	}

	// 读取凭证（密码为密文，仅在此解密用于登录）
	cred, err := s.credRepo.Get(userID)
	if err != nil {
		return nil, err
	}
	if cred == nil || cred.PortalAccount == "" || cred.PortalPasswordEnc == "" {
		return nil, fmt.Errorf("未绑定学校门户登录信息，请先在个人中心保存")
	}

	jar, _ := cookiejar.New(nil)
	decPwd, derr := repository.DecryptPortalPassword(cred.PortalPasswordEnc)
	if derr != nil || decPwd == "" {
		return nil, fmt.Errorf("门户凭证解密失败，请重新保存")
	}
	ss = &PortalSession{
		account:   cred.PortalAccount,
		password:  decPwd,
		portalURL: cred.PortalURL,
		jar:       jar,
		client:    &http.Client{Jar: jar, Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		}},
	}
	if err := s.login(ss); err != nil {
		return nil, err
	}
	ss.expireAt = time.Now().Add(s.ttl)

	s.mu.Lock()
	s.sessions[userID] = ss
	s.mu.Unlock()
	return ss, nil
}

// login 用表单登录门户（统一身份认证）。
// 多数高校门户支持 username/password POST 表单；CAS 场景由 loginPath 与字段配置适配。
func (s *PortalProxyService) login(ss *PortalSession) error {
	base := strings.TrimRight(ss.portalURL, "/")
	form := url.Values{}
	form.Set(s.accountKey, ss.account)
	form.Set(s.passwordKey, ss.password)
	req, err := http.NewRequest(http.MethodPost, base+s.loginPath, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("创建登录请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (WeiXiaoXin)")
	resp, err := ss.client.Do(req)
	if err != nil {
		return fmt.Errorf("门户登录失败: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	return nil
}

// Proxy 代理请求校内页面。path 为门户站点路径，如 /home、/edu/course 等。
func (s *PortalProxyService) Proxy(userID int64, path string, query url.Values, header http.Header) (int, http.Header, []byte, error) {
	ss, err := s.getSession(userID)
	if err != nil {
		return 0, nil, nil, err
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	target, err := url.Parse(strings.TrimRight(ss.portalURL, "/") + path)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("目标地址非法: %w", err)
	}
	// SSRF 防护：仅允许门户本身及其子域
	if !s.hostAllowed(target.Host) {
		return 0, nil, nil, fmt.Errorf("不允许访问该地址: %s", target.Host)
	}
	target.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, target.String(), nil)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("创建代理请求失败: %w", err)
	}
	// 透传必要请求头
	if ua := header.Get("User-Agent"); ua != "" {
		req.Header.Set("User-Agent", ua)
	} else {
		req.Header.Set("User-Agent", "Mozilla/5.0 (WeiXiaoXin)")
	}
	if accept := header.Get("Accept"); accept != "" {
		req.Header.Set("Accept", accept)
	}
	if referer := header.Get("Referer"); referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := ss.client.Do(req)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("门户代理请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return 0, nil, nil, fmt.Errorf("读取代理响应失败: %w", err)
	}

	out := make(http.Header)
	for _, k := range []string{"Content-Type", "Content-Language", "Cache-Control", "Expires"} {
		if v := resp.Header.Get(k); v != "" {
			out.Set(k, v)
		}
	}
	// HTML 页面改写内部链接，使其通过代理访问
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/html") {
		body = s.rewriteHTML(body, ss.portalURL)
	}
	return resp.StatusCode, out, body, nil
}

// rewriteHTML 改写 HTML 中的内部链接与表单 action 为代理路径。
// 目标：门户页内的相对链接与门户域绝对链接都能经 /api/v1/user/portal/* 继续访问。
func (s *PortalProxyService) rewriteHTML(body []byte, portalURL string) []byte {
	text := string(body)
	base := strings.TrimRight(portalURL, "/")
	proxyPrefix := "/api/v1/user/portal"

	// 1) 门户域绝对链接：https://my0.chzu.edu.cn/xxx → /api/v1/user/portal/xxx
	text = strings.ReplaceAll(text, base+"/", proxyPrefix+"/")
	// 裸域名（无协议前缀，如 //my0.chzu.edu.cn/xxx）
	host := strings.TrimPrefix(strings.TrimPrefix(portalURL, "https://"), "http://")
	host = strings.Split(host, "/")[0]
	text = strings.ReplaceAll(text, "//"+host+"/", proxyPrefix+"/")

	// 2) 根路径相对链接：href="/xxx" / action="/xxx" / src="/xxx" → proxy/xxx
	//    用正则处理避免误伤 CSS/JS 外部资源
	re := regexp.MustCompile(`(href|action|src)\s*=\s*"/([^"]*)"`)
	text = re.ReplaceAllStringFunc(text, func(m string) string {
		// 已带代理前缀的跳过
		if strings.Contains(m, proxyPrefix) {
			return m
		}
		sub := re.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		attr := sub[1]
		path := sub[2]
		// 静态资源(js/css/img)保持相对门户域访问，避免代理会话开销
		if isStaticResourcePath(path) {
			return m
		}
		return fmt.Sprintf(`%s="%s/%s"`, attr, proxyPrefix, path)
	})
	return []byte(text)
}

// isStaticResourcePath 判断是否为门户静态资源路径
func isStaticResourcePath(p string) bool {
	if strings.HasPrefix(p, "assets/") || strings.HasPrefix(p, "static/") ||
		strings.HasPrefix(p, "js/") || strings.HasPrefix(p, "css/") ||
		strings.HasPrefix(p, "img/") || strings.HasPrefix(p, "images/") ||
		strings.HasPrefix(p, "fonts/") || strings.HasPrefix(p, "favicon") {
		return true
	}
	return false
}

// hostAllowed 判断目标主机是否在允许列表（门户及其子域）
func (s *PortalProxyService) hostAllowed(host string) bool {
	host = strings.ToLower(host)
	// 去掉端口
	if i := strings.LastIndex(host, ":"); i > 0 && strings.Count(host, ":") == 1 {
		host = host[:i]
	}
	for _, a := range s.allowedHosts {
		a = strings.ToLower(a)
		if host == a || strings.HasSuffix(host, "."+a) {
			return true
		}
	}
	return false
}

// Invalidate 主动失效某用户会话（如解绑凭证后）
func (s *PortalProxyService) Invalidate(userID int64) {
	s.mu.Lock()
	delete(s.sessions, userID)
	s.mu.Unlock()
}
