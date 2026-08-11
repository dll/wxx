package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// TestParseFeed_RSS 校验 RSS 2.0 解析
func TestParseFeed_RSS(t *testing.T) {
	rssBody := `<?xml version="1.0"?>
<rss version="2.0"><channel>
  <title>AI 内参</title>
  <item><title>大模型新版本发布</title><link>https://a.com/1</link>
    <description>AI 行业热点：大模型版本更新</description>
    <pubDate>Mon, 10 Aug 2026 08:00:00 GMT</pubDate></item>
  <item><title>AI 工具测评</title><link>https://a.com/2</link>
    <description>Agent 工具对比</description>
    <pubDate>Tue, 11 Aug 2026 08:00:00 +0800</pubDate></item>
</channel></rss>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(rssBody))
	}))
	defer srv.Close()

	svc := NewAIBriefingService(nil)
	items := svc.parseFeed(srv.URL)
	if len(items) != 2 {
		t.Fatalf("RSS 应解析 2 条，实际 %d", len(items))
	}
	if items[0].title != "大模型新版本发布" || items[0].link != "https://a.com/1" {
		t.Fatalf("RSS 条目解析错误: %+v", items[0])
	}
	if items[0].published != "2026-08-10 08:00:00" {
		t.Fatalf("RFC2822 时间解析错误: %s", items[0].published)
	}
	if items[1].published != "2026-08-11 08:00:00" {
		t.Fatalf("RFC2822+0800 时间解析错误: %s", items[1].published)
	}
}

// TestParseFeed_Atom 校验 Atom 1.0 解析
func TestParseFeed_Atom(t *testing.T) {
	atomBody := `<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>AI 前沿</title>
  <entry>
    <title>Agent 编排框架进展</title>
    <link href="https://b.com/1"/>
    <summary>上下文隔离与共享方案</summary>
    <published>2026-08-10T09:30:00Z</published>
  </entry>
</feed>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(atomBody))
	}))
	defer srv.Close()

	svc := NewAIBriefingService(nil)
	items := svc.parseFeed(srv.URL)
	if len(items) != 1 {
		t.Fatalf("Atom 应解析 1 条，实际 %d", len(items))
	}
	if items[0].title != "Agent 编排框架进展" || items[0].link != "https://b.com/1" {
		t.Fatalf("Atom 条目解析错误: %+v", items[0])
	}
	if items[0].published != "2026-08-10 09:30:00" {
		t.Fatalf("RFC3339 时间解析错误: %s", items[0].published)
	}
}

// TestExportBriefings 校验 md/pdf 导出
func TestExportBriefings(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	// 清空 067 seed，隔离测试数据
	if _, err := db.Exec("DELETE FROM ai_briefings"); err != nil {
		t.Fatalf("清空 ai_briefings 失败: %v", err)
	}
	svc := NewAIBriefingService(repository.NewAIBriefingRepo(db))

	// 无数据
	md, err := svc.ExportBriefingsMarkdown(nil)
	if err != nil {
		t.Fatalf("导出 md 失败: %v", err)
	}
	if !strings.Contains(string(md), "无数据") {
		t.Fatalf("空导出应含'无数据': %s", md)
	}

	// 有数据
	mk := func(cat, topic string) {
		_, _ = svc.repo.Create(&model.AIBriefing{
			Source: "AI内参", Category: cat, Topic: topic,
			Summary: "摘要", Link: "https://example.com", Keyword: "AI",
			PublishedAt: "2026-08-10 09:00:00", Status: 1,
		})
	}
	mk("ai_teaching", "大模型版本更新")
	mk("ai_tool", "AI 工具测评")
	// 清空 seed 后自增 id 不连续，按标题筛选本次条目
	all, _, _ := svc.repo.List("", "", "", 1, 100)
	items := make([]*model.AIBriefing, 0, 2)
	for _, b := range all {
		if b.Topic == "大模型版本更新" || b.Topic == "AI 工具测评" {
			items = append(items, b)
		}
	}
	if len(items) != 2 {
		t.Fatalf("应筛出 2 条测试资讯，实际 %d", len(items))
	}

	md, err = svc.ExportBriefingsMarkdown(items)
	if err != nil {
		t.Fatalf("导出 md 失败: %v", err)
	}
	if !bytes.Contains(md, []byte("大模型版本更新")) || !bytes.Contains(md, []byte("AI 工具测评")) {
		t.Fatalf("md 内容不符: %s", md)
	}

	pdf, err := svc.ExportBriefingsPDF(items)
	if err != nil {
		t.Fatalf("导出 pdf 失败: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Fatalf("pdf 头错误: %s", pdf[:8])
	}
}
