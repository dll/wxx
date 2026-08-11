package service

import (
	"net/url"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// TestPortalProxy_HostAllowed 校验 SSRF 防护：仅允许门户及其子域
func TestPortalProxy_HostAllowed(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	s := NewPortalProxyService(repository.NewPortalCredentialRepo(db))

	cases := []struct {
		host string
		ok   bool
	}{
		{"my0.chzu.edu.cn", true},
		{"jw.chzu.edu.cn", true},  // 子域
		{"bbs.chzu.edu.cn", true}, // 子域
		{"chzu.edu.cn", true},     // 门户根域
		{"evil.com", false},
		{"my0.chzu.edu.cn.evil.com", false}, // 前缀混淆
		{"192.168.1.1", false},
	}
	for _, c := range cases {
		if got := s.hostAllowed(c.host); got != c.ok {
			t.Errorf("hostAllowed(%s) = %v, 期望 %v", c.host, got, c.ok)
		}
	}
}

// TestPortalProxy_ProxyWithoutCredential 校验未绑定凭证时返回友好错误
func TestPortalProxy_ProxyWithoutCredential(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	s := NewPortalProxyService(repository.NewPortalCredentialRepo(db))

	_, _, _, err := s.Proxy(9999, "/home", url.Values{}, nil)
	if err == nil {
		t.Fatal("未绑定凭证应返回错误")
	}
	if !strings.Contains(err.Error(), "未绑定") {
		t.Fatalf("错误信息应提示未绑定: %v", err)
	}
}

// TestRewriteHTML 校验门户 HTML 链接改写为代理路径
func TestRewriteHTML(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	s := NewPortalProxyService(repository.NewPortalCredentialRepo(db))

	html := `<html>
<a href="/home">首页</a>
<a href="https://my0.chzu.edu.cn/edu/course">选课</a>
<a href="/assets/app.js">脚本</a>
<form action="/login" method="post"></form>
<img src="/img/logo.png">
</html>`

	out := string(s.rewriteHTML([]byte(html), "https://my0.chzu.edu.cn/"))
	if !strings.Contains(out, `href="/api/v1/user/portal/home"`) {
		t.Errorf("相对链接未改写: %s", out)
	}
	if !strings.Contains(out, `href="/api/v1/user/portal/edu/course"`) {
		t.Errorf("绝对链接未改写: %s", out)
	}
	if !strings.Contains(out, `action="/api/v1/user/portal/login"`) {
		t.Errorf("表单 action 未改写: %s", out)
	}
	// 静态资源不应改写
	if !strings.Contains(out, `src="/img/logo.png"`) {
		t.Errorf("静态资源不应改写: %s", out)
	}
	if !strings.Contains(out, `href="/assets/app.js"`) {
		t.Errorf("assets 静态资源不应改写: %s", out)
	}
}
