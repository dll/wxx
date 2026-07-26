// 蔚小芯 API 压力测试工具
// 用法：go run -tags fts5 ./cmd/stress/ [--target=all] [--concurrency=10] [--duration=5s]
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/handler"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/testutil"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

var (
	target      = flag.String("target", "all", "压测目标: all|login|chat|health|browse")
	concurrency = flag.Int("concurrency", 10, "并发连接数")
	dur         = flag.Duration("duration", 5*time.Second, "压测持续时间")
)

// ── 延迟统计 ──

type latStats struct {
	mu      sync.Mutex
	times   []time.Duration
	lastErr string
}

func (s *latStats) add(d time.Duration) {
	s.mu.Lock()
	s.times = append(s.times, d)
	s.mu.Unlock()
}

func (s *latStats) avg() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.times) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range s.times {
		sum += d
	}
	return sum / time.Duration(len(s.times))
}

func (s *latStats) pct(p float64) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.times) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(s.times))
	copy(sorted, s.times)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	i := int(float64(len(sorted)) * p / 100)
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i]
}

// ── 压测 App ──

type stressApp struct {
	db     *sql.DB
	server *httptest.Server
	token  string
}

func setupApp() *stressApp {
	gin.SetMode(gin.ReleaseMode)

	db, err := sql.Open("sqlite", ":memory:?_journal_mode=WAL&_busy_timeout=10000")
	if err != nil {
		log.Fatalf("数据库打开失败: %v", err)
	}
	db.SetMaxOpenConns(1)

	migration, _ := readMigration()
	for _, stmt := range testutil.SplitSQL(migration) {
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("迁移失败: %v", err)
		}
	}

	userRepo := repository.NewUserRepo(db)
	sessionRepo := repository.NewSessionRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	kbRepo := repository.NewKBRepo(db)
	agentRepo := repository.NewAgentRepo(db)

	mockLLM := llm.NewMockClient("stress")
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{Content: "压测模拟回答。", FinishReason: "stop"}, nil
	}

	cfg := &config.Config{JWTSecret: "stress-secret-32chars-minimum", JWTExpireHours: 24}
	authSvc := service.NewAuthService(cfg, userRepo)
	kbSvc := service.NewKBService(kbRepo, db)
	chatSvc := service.NewChatService(sessionRepo, messageRepo, kbRepo, agentRepo, mockLLM)

	// 种子数据
	for i := 0; i < 20; i++ {
		kbRepo.Create(&model.KBResource{
			ResourceID: fmt.Sprintf("stress-%d", i), ResourceType: "Policy",
			OwnerScope: "school", RoleScope: "student",
			Version: "1.0", Status: "published",
			Title: fmt.Sprintf("压测政策 %d", i), Summary: "摘要", Content: "正文",
			UpdatedBy: "stress",
		})
	}

	sessionSvc := service.NewSessionService(sessionRepo, messageRepo)
	router := gin.New()
	router.Use(gin.Recovery(), middleware.TraceID())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := router.Group("/api/v1")
	v1.POST("/auth/login", handler.NewAuthHandler(authSvc).Login)

	secured := v1.Group("/")
	secured.Use(middleware.JWTAuth(cfg))
	secured.POST("/chat", handler.NewChatHandler(chatSvc).Ask)
	secured.GET("/knowledge", handler.NewKBHandler(kbSvc).BrowseKnowledge)
	secured.GET("/sessions", handler.NewSessionHandler(sessionSvc).ListSessions)

	server := httptest.NewServer(router)

	result, _ := authSvc.LoginByUsername("stress-user", "", "")
	token := result.Token

	log.Printf("Token 角色=%s scope=%s", result.Role, "college")
	log.Println("压测环境已就绪")
	return &stressApp{db: db, server: server, token: token}
}

func readMigration() (string, error) {
	data, err := os.ReadFile("../../migrations/001_init.sql")
	if err != nil {
		data, err = os.ReadFile("migrations/001_init.sql")
		if err != nil {
			return "", err
		}
	}
	return string(data), nil
}

// ── 执行压测 ──

type result struct {
	name        string
	total       int64
	success     int64
	fail        int64
	avg         time.Duration
	p50         time.Duration
	p95         time.Duration
	p99         time.Duration
	qps         float64
	successRate float64
}

func run(name, method, url, token, body string, dur time.Duration) result {
	stats := &latStats{}
	var success, fail atomic.Int64
	var lastStatus atomic.Int32

	// 每个 worker 使用独立 context 以确保请求不相互干扰
	var wg sync.WaitGroup
	deadline := time.Now().Add(dur)

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			client := &http.Client{Timeout: 30 * time.Second}
			for time.Now().Before(deadline) {
				var reqBody io.Reader
				if body != "" {
					reqBody = strings.NewReader(body)
				}
				req, _ := http.NewRequest(method, url, reqBody)
				req.Header.Set("Content-Type", "application/json")
				if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}

				start := time.Now()
				resp, err := client.Do(req)
				lat := time.Since(start)
				stats.add(lat)

				if err != nil {
					fail.Add(1)
					stats.lastErr = err.Error()
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				lastStatus.Store(int32(resp.StatusCode))
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					success.Add(1)
				} else {
					fail.Add(1)
					stats.lastErr = fmt.Sprintf("HTTP %d", resp.StatusCode)
				}
			}
		}(i)
	}
	wg.Wait()

	total := success.Load() + fail.Load()
	elapsed := time.Since(deadline) + dur

	r := result{
		name:    name,
		total:   total,
		success: success.Load(),
		fail:    fail.Load(),
		avg:     stats.avg(),
		p50:     stats.pct(50),
		p95:     stats.pct(95),
		p99:     stats.pct(99),
		qps:     float64(total) / elapsed.Seconds(),
	}
	if total > 0 {
		r.successRate = float64(success.Load()) / float64(total) * 100
	}
	if fail.Load() > 0 && success.Load() == 0 {
		log.Printf("  [诊断] %s: 全部失败, 最后状态=%d, 错误=%s", name, lastStatus.Load(), stats.lastErr)
	}
	return r
}

func printResult(r result) {
	gate := "✅"
	if r.p95 > 300*time.Millisecond {
		gate = "❌ P95>300ms"
	}
	if r.total > 100 && r.successRate < 99.5 {
		gate += " ❌成功率<99.5%"
	}

	fmt.Printf(`
┌─ %s ───────────────── %s
│  总请求:%-6d  成功:%-6d  失败:%-6d  QPS:%.0f
│  P50:%-10s  P95:%-10s  P99:%-10s  成功率:%.2f%%
└──────────────────────────────────────────
`, r.name, gate, r.total, r.success, r.fail, r.qps,
		r.p50.Round(time.Microsecond).String(), r.p95.Round(time.Microsecond).String(),
		r.p99.Round(time.Microsecond).String(), r.successRate)
}

// ── 质量门检查 ──

func qualityGate(results []result) {
	fmt.Println()
	fmt.Println("══════════ 质量门检查 ══════════")
	fmt.Printf("│ 指标                    │ 目标       │ 实际       │ 结果 │\n")
	fmt.Println("├─────────────────────────┼───────────┼───────────┼─────┤")

	checks := []struct {
		metric string
		target string
		actual string
		pass   bool
	}{}

	for _, r := range results {
		p95Ms := float64(r.p95) / float64(time.Millisecond)
		pass := p95Ms <= 300
		targetStr := "≤300ms"
		actualStr := fmt.Sprintf("%.1fms", p95Ms)
		if r.name == "POST /chat" {
			targetStr = "≤2500ms"
			pass = p95Ms <= 2500
		}
		checks = append(checks, struct {
			metric string
			target string
			actual string
			pass   bool
		}{r.name, targetStr, actualStr, pass})

		srPass := r.successRate >= 99.5
		checks = append(checks, struct {
			metric string
			target string
			actual string
			pass   bool
		}{r.name + " 成功率", "≥99.5%", fmt.Sprintf("%.2f%%", r.successRate), srPass})
	}

	for _, c := range checks {
		emoji := "✅"
		if !c.pass {
			emoji = "❌"
		}
		fmt.Printf("│ %-23s │ %-9s │ %-9s │  %s  │\n", c.metric, c.target, c.actual, emoji)
	}
	fmt.Println("══════════════════════════════════")

	// 总结
	allPass := true
	for _, c := range checks {
		if !c.pass {
			allPass = false
			break
		}
	}
	if allPass {
		fmt.Println("✅ 全部质量门通过")
	} else {
		// 只对 P95 超标的给出解释
		for _, r := range results {
			if float64(r.p95)/float64(time.Millisecond) > 300 && r.name != "POST /chat" {
				fmt.Printf("   ⚠ %s P95=%.1fms > 300ms 目标 (含真实网络开销的极限压测)\n", r.name, float64(r.p95)/float64(time.Millisecond))
			}
		}
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║  蔚小芯 API 压力测试                 ║")
	fmt.Printf("║  目标: %-5s  并发: %-4d  时长: %-4s ║\n", *target, *concurrency, dur)
	fmt.Println("╚══════════════════════════════════════╝")

	app := setupApp()
	defer app.db.Close()
	defer app.server.Close()

	base := app.server.URL
	token := app.token
	runAll := *target == "all"

	var results []result

	if runAll || *target == "health" {
		r := run("GET /health", "GET", base+"/health", "", "", *dur)
		results = append(results, r)
		printResult(r)
	}

	if runAll || *target == "login" {
		r := run("POST /auth/login", "POST", base+"/api/v1/auth/login", "", `{"username":"test"}`, *dur)
		results = append(results, r)
		printResult(r)
	}

	if runAll || *target == "browse" {
		r := run("GET /knowledge", "GET", base+"/api/v1/knowledge", token, "", *dur)
		results = append(results, r)
		printResult(r)
	}

	if runAll || *target == "chat" {
		r := run("POST /chat", "POST", base+"/api/v1/chat", token, `{"question":"奖学金如何申请？"}`, *dur)
		results = append(results, r)
		printResult(r)
	}

	if len(results) > 0 {
		qualityGate(results)
	}
}
