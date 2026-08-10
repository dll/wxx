package service

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/model"
)

func briefingCategoryLabel(c string) string {
	switch c {
	case "ai_teaching":
		return "AI 辅助教学"
	case "ai_tool":
		return "AI 工具"
	case "ai_version":
		return "AI 版本"
	case "ai_industry":
		return "AI 行业热点"
	default:
		return c
	}
}

// ExportBriefingsMarkdown 生成 AI 简讯 Markdown 文档
func (s *AIBriefingService) ExportBriefingsMarkdown(items []*model.AIBriefing) ([]byte, error) {
	if len(items) == 0 {
		return []byte("# AI 简讯\n\n（无数据）\n"), nil
	}
	var b strings.Builder
	b.WriteString("# AI 简讯\n\n")
	b.WriteString(fmt.Sprintf("> 导出时间：%s | 共 %d 条\n\n", time.Now().Format("2006-01-02 15:04:05"), len(items)))

	for i, it := range items {
		b.WriteString(fmt.Sprintf("## %d. %s\n\n", i+1, it.Topic))
		b.WriteString(fmt.Sprintf("- **分类**：%s\n", briefingCategoryLabel(it.Category)))
		b.WriteString(fmt.Sprintf("- **来源**：%s\n", it.Source))
		b.WriteString(fmt.Sprintf("- **时间**：%s\n", it.PublishedAt))
		if it.Summary != "" {
			b.WriteString(fmt.Sprintf("\n%s\n", it.Summary))
		}
		if it.Link != "" {
			b.WriteString(fmt.Sprintf("\n[详情链接](%s)\n", it.Link))
		}
		b.WriteString("\n---\n\n")
	}
	b.WriteString("*此文档由蔚小芯 AI 简讯生成*\n")
	return []byte(b.String()), nil
}

// ExportBriefingsPDF 生成 AI 简讯 PDF（ASCII 渲染，中文以 ? 替代）
func (s *AIBriefingService) ExportBriefingsPDF(items []*model.AIBriefing) ([]byte, error) {
	if len(items) == 0 {
		items = []*model.AIBriefing{}
	}
	var content strings.Builder
	content.WriteString("BT\n/F1 16 Tf\n50 800 Td\n(AI Jian Xun - AI Briefings) Tj\n0 -24 Td\n/F1 10 Tf\n")
	content.WriteString(fmt.Sprintf("(Exported: %s) Tj\n0 -20 Td\n", time.Now().Format("2006-01-02 15:04:05")))

	y := 0
	for i, it := range items {
		if y > 36 {
			// 简单分页：跳回顶部（多页不处理换页）
			y = 0
		}
		content.WriteString("0 -18 Td\n/F1 12 Tf\n")
		content.WriteString(fmt.Sprintf("(%d. %s) Tj\n0 -14 Td\n/F1 9 Tf\n", i+1, pdfEscape(it.Topic)))
		content.WriteString(fmt.Sprintf("(%s | %s | %s) Tj\n", pdfEscape(it.Source), pdfEscape(briefingCategoryLabel(it.Category)), pdfEscape(it.PublishedAt)))
		if it.Summary != "" {
			for _, line := range splitLines(it.Summary, 80) {
				content.WriteString("0 -13 Td\n")
				content.WriteString(fmt.Sprintf("(%s) Tj\n", pdfEscape(line)))
			}
		}
		if it.Link != "" {
			content.WriteString("0 -13 Td\n")
			content.WriteString(fmt.Sprintf("(Link: %s) Tj\n", pdfEscape(it.Link)))
		}
		content.WriteString("0 -18 Td\n")
		y++
	}
	content.WriteString("ET")

	objects := []string{}
	obj1 := "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj"
	obj2 := "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj"
	obj3 := "3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj"
	obj4 := fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj", len(content.String()), content.String())
	obj5 := "5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj"
	objects = append(objects, obj1, obj2, obj3, obj4, obj5)

	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := []int{}
	offset := 0
	for _, obj := range objects {
		offsets = append(offsets, offset)
		offset += len(obj) + 1
		b.WriteString(obj)
		b.WriteString("\n")
	}
	xrefOffset := b.Len()
	b.WriteString("xref\n")
	b.WriteString(fmt.Sprintf("0 %d\n", len(objects)+1))
	b.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}
	b.WriteString("trailer\n")
	b.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	b.WriteString("startxref\n")
	b.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	b.WriteString("%%EOF\n")
	return b.Bytes(), nil
}
