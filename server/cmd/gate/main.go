// 蔚小芯 质量门禁工具
// 读取评测报告 JSON，检查各项指标是否达到验收标准
// 用法: go run server/cmd/gate/main.go -report eval-result.json
// 退出码: 0=通过, 1=未达标
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// EvalReport 评测报告结构（与 cmd/eval 输出一致）
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

// 质量门禁阈值（与 CLAUDE.md 验收指标对齐）
var gates = []struct {
	Name      string
	Check     func(r *EvalReport) bool
	Threshold string
}{
	{
		Name:      "整体命中率",
		Threshold: "≥ 85%",
		Check:     func(r *EvalReport) bool { return r.PassRate >= 85.0 },
	},
	{
		Name:      "引用覆盖率",
		Threshold: "≥ 92%",
		Check:     func(r *EvalReport) bool { return r.SourcePassRate >= 92.0 },
	},
	{
		Name:      "兜底率",
		Threshold: "≤ 10%",
		Check:     func(r *EvalReport) bool { return r.FallbackRate <= 10.0 },
	},
	{
		Name:      "平均响应时间",
		Threshold: "≤ 2500ms",
		Check:     func(r *EvalReport) bool { return r.AvgDurationMs <= 2500 },
	},
	{
		Name:      "平均置信度",
		Threshold: "≥ 0.6",
		Check:     func(r *EvalReport) bool { return r.AvgConfidence >= 0.6 },
	},
}

func main() {
	reportFile := flag.String("report", "", "评测报告 JSON 文件路径")
	flag.Parse()

	if *reportFile == "" {
		fmt.Fprintln(os.Stderr, "用法: go run server/cmd/gate/main.go -report <eval-result.json>")
		os.Exit(1)
	}

	data, err := os.ReadFile(*reportFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取报告文件失败: %v\n", err)
		os.Exit(1)
	}

	var report EvalReport
	if err := json.Unmarshal(data, &report); err != nil {
		fmt.Fprintf(os.Stderr, "解析报告 JSON 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("========================================")
	fmt.Println("  蔚小芯 质量门禁检查")
	fmt.Println("========================================")
	fmt.Printf("评测样本数: %d\n\n", report.Total)

	allPassed := true
	for _, g := range gates {
		passed := g.Check(&report)
		status := "✓ 通过"
		if !passed {
			status = "✗ 未达标"
			allPassed = false
		}
		fmt.Printf("  [%s] %s (阈值: %s)\n", status, g.Name, g.Threshold)
	}

	fmt.Println()
	fmt.Printf("  通过率: %.1f%% | 引用覆盖: %.1f%% | 兜底率: %.1f%% | 平均延迟: %dms | 平均置信度: %.3f\n",
		report.PassRate, report.SourcePassRate, report.FallbackRate, report.AvgDurationMs, report.AvgConfidence)
	fmt.Println()

	if allPassed {
		fmt.Println("  ══ 质量门禁: 全部通过 ══")
		os.Exit(0)
	} else {
		fmt.Println("  ══ 质量门禁: 未通过，请修复后重新评测 ══")
		os.Exit(1)
	}
}
