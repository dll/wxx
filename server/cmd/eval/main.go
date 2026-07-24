// 蔚小芯 AI 问答评测工具
// 读取 NDJSON 格式的标注评测集，逐条调用问答接口并计算核心指标
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// EvalEntry 单条评测用例（NDJSON 格式）
type EvalEntry struct {
	Question           string  `json:"question"`
	ExpectedIntent     string  `json:"expected_intent"`
	ExpectedSourcesMin int     `json:"expected_sources_min"`
	AcceptableFallback bool    `json:"acceptable_fallback"`
	MinConfidence      float64 `json:"min_confidence"`
	Category           string  `json:"category"`
}

// ChatResponse 问答接口响应格式
type ChatResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Conclusion string `json:"conclusion"`
		Sources    []struct {
			ResourceID string `json:"resource_id"`
			Title      string `json:"title"`
		} `json:"sources"`
		TraceID    string  `json:"trace_id"`
		Confidence float64 `json:"confidence"`
		Fallback   bool    `json:"fallback"`
	} `json:"data"`
	SessionID string `json:"session_id"`
}

// EvalResult 单条评测结果
type EvalResult struct {
	Index              int     `json:"index"`
	Question           string  `json:"question"`
	Category           string  `json:"category"`
	ExpectedIntent     string  `json:"expected_intent"`
	Passed             bool    `json:"passed"`
	SourcesOK          bool    `json:"sources_ok"`
	SourcesGot         int     `json:"sources_got"`
	SourcesExpected    int     `json:"sources_expected"`
	ConfidenceOK       bool    `json:"confidence_ok"`
	ConfidenceGot      float64 `json:"confidence_got"`
	MinConfidence      float64 `json:"min_confidence"`
	FallbackOK         bool    `json:"fallback_ok"`
	FallbackGot        bool    `json:"fallback_got"`
	AcceptableFallback bool    `json:"acceptable_fallback"`
	ResponseEmpty      bool    `json:"response_empty"`
	Error              string  `json:"error,omitempty"`
	DurationMs         int64   `json:"duration_ms"`
}

// EvalReport 最终评测报告
type EvalReport struct {
	Total          int                        `json:"total"`
	Passed         int                        `json:"passed"`
	PassRate       float64                    `json:"pass_rate"`
	AvgConfidence  float64                    `json:"avg_confidence"`
	AvgSources     float64                    `json:"avg_sources"`
	AvgDurationMs  int64                      `json:"avg_duration_ms"`
	SourcePassRate float64                    `json:"source_pass_rate"`
	ConfPassRate   float64                    `json:"conf_pass_rate"`
	FallbackRate   float64                    `json:"fallback_rate"`
	ByCategory     map[string]*CategoryReport `json:"by_category"`
}

// CategoryReport 按类别统计
type CategoryReport struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	PassRate float64 `json:"pass_rate"`
}

func main() {
	baseURL := flag.String("url", "https://api.pydaydayup.xyz", "后端服务地址")
	token := flag.String("token", "", "认证 Token（Bearer）")
	baselineFile := flag.String("baseline", "../../specs/eval-baseline.ndjson", "评测基线文件路径")
	concurrency := flag.Int("c", 1, "并发数")
	outputFile := flag.String("output", "", "结果输出文件（可选，默认 stdout）")
	flag.Parse()

	if *token == "" {
		log.Fatal("请通过 -token 参数提供认证 Token")
	}

	// 读取评测基线
	entries, err := loadBaseline(*baselineFile)
	if err != nil {
		log.Fatalf("读取基线文件失败: %v", err)
	}
	log.Printf("已加载 %d 条评测用例", len(entries))

	// 执行评测
	results := runEval(*baseURL, *token, entries, *concurrency)

	// 生成报告
	report := buildReport(entries, results)

	// 输出
	var out io.Writer = os.Stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			log.Fatalf("创建输出文件失败: %v", err)
		}
		defer f.Close()
		out = f
	}
	printReport(out, report, results)
}

func loadBaseline(path string) ([]EvalEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []EvalEntry
	scanner := bufio.NewScanner(f)
	// 增大缓冲区以处理长行
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var entry EvalEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			log.Printf("跳过无效行: %v", err)
			continue
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

func runEval(baseURL, token string, entries []EvalEntry, concurrency int) []EvalResult {
	results := make([]EvalResult, len(entries))
	sem := make(chan struct{}, concurrency)
	done := make(chan int, len(entries))

	for i, entry := range entries {
		sem <- struct{}{}
		go func(idx int, e EvalEntry) {
			defer func() { <-sem; done <- idx }()
			results[idx] = evalSingle(baseURL, token, idx, e)
		}(i, entry)
	}

	// 等待全部完成
	for range entries {
		<-done
	}
	return results
}

func evalSingle(baseURL, token string, index int, entry EvalEntry) EvalResult {
	result := EvalResult{
		Index:              index,
		Question:           entry.Question,
		Category:           entry.Category,
		ExpectedIntent:     entry.ExpectedIntent,
		SourcesOK:          true,
		ConfidenceOK:       true,
		FallbackOK:         true,
		SourcesExpected:    entry.ExpectedSourcesMin,
		MinConfidence:      entry.MinConfidence,
		AcceptableFallback: entry.AcceptableFallback,
	}

	start := time.Now()

	// 构造请求
	reqBody := map[string]string{
		"question": entry.Question,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", baseURL+"/api/v1/chat", bytes.NewReader(bodyBytes))
	if err != nil {
		result.Error = err.Error()
		result.Passed = false
		return result
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		result.Error = err.Error()
		result.Passed = false
		return result
	}
	defer resp.Body.Close()

	result.DurationMs = time.Since(start).Milliseconds()

	// 解析响应
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		result.Error = err.Error()
		result.Passed = false
		return result
	}

	// 检查空响应
	if chatResp.Data.Conclusion == "" {
		result.ResponseEmpty = true
		result.Passed = false
		result.SourcesOK = false
		result.ConfidenceOK = false
		return result
	}

	// 来源数量检查
	result.SourcesGot = len(chatResp.Data.Sources)
	if result.SourcesGot < entry.ExpectedSourcesMin {
		result.SourcesOK = false
	}

	// 置信度检查
	result.ConfidenceGot = chatResp.Data.Confidence
	if chatResp.Data.Confidence < entry.MinConfidence {
		result.ConfidenceOK = false
	}

	// 兜底检查
	result.FallbackGot = chatResp.Data.Fallback
	if chatResp.Data.Fallback && !entry.AcceptableFallback {
		result.FallbackOK = false
	}

	// 综合判定
	result.Passed = result.SourcesOK && result.ConfidenceOK && result.FallbackOK && !result.ResponseEmpty
	return result
}

func buildReport(entries []EvalEntry, results []EvalResult) EvalReport {
	report := EvalReport{
		Total:      len(results),
		ByCategory: make(map[string]*CategoryReport),
	}

	var totalConf float64
	var totalSources float64
	var totalDur int64
	var sourcesPassed int
	var confPassed int
	var fallbackCount int

	for _, r := range results {
		if r.Passed {
			report.Passed++
		}
		if r.SourcesOK {
			sourcesPassed++
		}
		if r.ConfidenceOK {
			confPassed++
		}
		if r.FallbackGot {
			fallbackCount++
		}
		totalConf += r.ConfidenceGot
		totalSources += float64(r.SourcesGot)
		totalDur += r.DurationMs

		// 按类别统计
		if _, ok := report.ByCategory[r.Category]; !ok {
			report.ByCategory[r.Category] = &CategoryReport{}
		}
		cr := report.ByCategory[r.Category]
		cr.Total++
		if r.Passed {
			cr.Passed++
		}
	}

	n := float64(len(results))
	report.PassRate = float64(report.Passed) / n * 100
	report.AvgConfidence = totalConf / n
	report.AvgSources = totalSources / n
	report.AvgDurationMs = totalDur / int64(len(results))
	report.SourcePassRate = float64(sourcesPassed) / n * 100
	report.ConfPassRate = float64(confPassed) / n * 100
	report.FallbackRate = float64(fallbackCount) / n * 100

	for _, cr := range report.ByCategory {
		cr.PassRate = float64(cr.Passed) / float64(cr.Total) * 100
	}

	return report
}

func printReport(w io.Writer, report EvalReport, results []EvalResult) {
	fmt.Fprintln(w, "========================================")
	fmt.Fprintln(w, "  蔚小芯 问答质量评测报告")
	fmt.Fprintln(w, "========================================")
	fmt.Fprintf(w, "评测时间：%s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "评测总数：%d\n", report.Total)
	fmt.Fprintf(w, "通过数量：%d\n", report.Passed)
	fmt.Fprintf(w, "通过率  ：%.1f%%\n\n", report.PassRate)

	fmt.Fprintln(w, "── 核心指标 ──")
	fmt.Fprintf(w, "平均置信度  ：%.3f\n", report.AvgConfidence)
	fmt.Fprintf(w, "来源覆盖率  ：%.1f%% (%d/%d)\n", report.SourcePassRate, report.Passed, report.Total)
	fmt.Fprintf(w, "置信度达标率：%.1f%%\n", report.ConfPassRate)
	fmt.Fprintf(w, "兜底率      ：%.1f%%\n", report.FallbackRate)
	fmt.Fprintf(w, "平均响应时间：%dms\n\n", report.AvgDurationMs)

	fmt.Fprintln(w, "── 按类别统计 ──")
	fmt.Fprintf(w, "%-12s %6s %6s %8s\n", "类别", "总数", "通过", "通过率")
	for cat, cr := range report.ByCategory {
		fmt.Fprintf(w, "%-12s %6d %6d %7.1f%%\n", cat, cr.Total, cr.Passed, cr.PassRate)
	}

	// 失败用例摘要
	failed := 0
	for _, r := range results {
		if !r.Passed {
			failed++
		}
	}
	if failed > 0 {
		fmt.Fprintf(w, "\n── 失败用例（%d 条）──\n", failed)
		for _, r := range results {
			if !r.Passed {
				reasons := []string{}
				if r.Error != "" {
					reasons = append(reasons, "错误: "+r.Error)
				}
				if r.ResponseEmpty {
					reasons = append(reasons, "响应为空")
				}
				if !r.SourcesOK {
					reasons = append(reasons, fmt.Sprintf("来源不足(%d<%d)", r.SourcesGot, r.SourcesExpected))
				}
				if !r.ConfidenceOK {
					reasons = append(reasons, fmt.Sprintf("置信度低(%.2f<%.2f)", r.ConfidenceGot, r.MinConfidence))
				}
				if !r.FallbackOK {
					reasons = append(reasons, "不当兜底")
				}
				fmt.Fprintf(w, "  #%d [%s] %s\n", r.Index, r.Category, r.Question)
				fmt.Fprintf(w, "      → %s\n", strings.Join(reasons, "; "))
			}
		}
	}
	fmt.Fprintln(w, "\n========================================")
}
