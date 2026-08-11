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
		{"jw.chzu.edu.cn", true},   // 子域
		{"bbs.chzu.edu.cn", true},  // 子域
		{"chzu.edu.cn", true},      // 门户根域
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
