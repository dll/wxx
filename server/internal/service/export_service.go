package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/model"
)

// ExportFormat 导出格式
type ExportFormat string

const (
	ExportPDF  ExportFormat = "pdf"
	ExportJSON ExportFormat = "json"
	ExportMD   ExportFormat = "md"
)

// ExportService 多格式导出服务
type ExportService struct{}

// NewExportService 创建导出服务
func NewExportService() *ExportService {
	return &ExportService{}
}

// ExportAnswer 将 AnswerCard 导出为指定格式
func (s *ExportService) ExportAnswer(card *model.AnswerCard, format ExportFormat, watermark bool) ([]byte, string, error) {
	switch format {
	case ExportJSON:
		data, err := s.exportJSON(card)
		return data, "application/json", err
	case ExportMD:
		data := s.exportMarkdown(card)
		return []byte(data), "text/markdown; charset=utf-8", nil
	case ExportPDF:
		data := s.exportPDF(card, watermark)
		return data, "application/pdf", nil
	default:
		return nil, "", fmt.Errorf("不支持的导出格式: %s", format)
	}
}

func (s *ExportService) exportJSON(card *model.AnswerCard) ([]byte, error) {
	// 使用已有的序列化方法
	jsonStr := MarshalAnswerCard(card)
	return []byte(jsonStr), nil
}

func (s *ExportService) exportMarkdown(card *model.AnswerCard) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s\n\n", "蔚小芯 AI 回答"))
	b.WriteString(fmt.Sprintf("> 生成时间：%s | 追踪ID：%s\n\n", time.Now().Format("2006-01-02 15:04:05"), card.TraceID))

	b.WriteString("## 回答\n\n")
	b.WriteString(card.Conclusion)
	b.WriteString("\n\n")

	if len(card.Sources) > 0 {
		b.WriteString("## 参考来源\n\n")
		for i, s := range card.Sources {
			b.WriteString(fmt.Sprintf("%d. **%s**（版本：%s）\n", i+1, s.Title, s.Version))
			if s.SourceLink != "" {
				b.WriteString(fmt.Sprintf("   - 链接：%s\n", s.SourceLink))
			}
		}
		b.WriteString("\n")
	}

	if len(card.FollowUps) > 0 {
		b.WriteString("## 相关追问\n\n")
		for _, f := range card.FollowUps {
			b.WriteString(fmt.Sprintf("- %s\n", f))
		}
		b.WriteString("\n")
	}

	if card.Fallback {
		b.WriteString("> 注意：此回答为兜底回复，建议进一步咨询辅导员。\n")
	}

	b.WriteString("---\n*此文档由蔚小芯 AI 学工助手生成*\n")
	return b.String()
}

func (s *ExportService) exportPDF(card *model.AnswerCard, watermark bool) []byte {
	var b strings.Builder

	// 最小可用 PDF 生成
	// 使用 PDF 文本格式（跨平台，无字体依赖）
	objects := []string{}
	content := s.buildPDFContent(card, watermark)

	// PDF 对象编号
	obj1 := fmt.Sprintf("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj")
	obj2 := fmt.Sprintf("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj")
	obj3 := fmt.Sprintf("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj")
	obj4 := fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj", len(content), content)
	obj5 := "5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj"

	objects = append(objects, obj1, obj2, obj3, obj4, obj5)

	// 计算 xref 偏移
	offsets := []int{}
	offset := 0
	for _, obj := range objects {
		offsets = append(offsets, offset)
		offset += len(obj) + 1 // +1 for newline
	}

	// 组装 PDF
	b.WriteString("%PDF-1.4\n")
	for _, obj := range objects {
		b.WriteString(obj)
		b.WriteString("\n")
	}

	// xref 表
	xrefOffset := b.Len()
	b.WriteString("xref\n")
	b.WriteString(fmt.Sprintf("0 %d\n", len(objects)+1))
	b.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}

	// trailer
	b.WriteString("trailer\n")
	b.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	b.WriteString("startxref\n")
	b.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	b.WriteString("%%EOF\n")

	return []byte(b.String())
}

func (s *ExportService) buildPDFContent(card *model.AnswerCard, watermark bool) string {
	var lines []string

	// PDF 文本操作符
	lines = append(lines, "BT")
	lines = append(lines, "/F1 12 Tf")
	lines = append(lines, "50 800 Td")
	lines = append(lines, fmt.Sprintf("(生成时间: %s) Tj", time.Now().Format("2006-01-02 15:04:05")))
	lines = append(lines, "0 -20 Td")
	lines = append(lines, "/F1 16 Tf")
	lines = append(lines, "(蔚小芯 AI 回答) Tj")
	lines = append(lines, "0 -30 Td")
	lines = append(lines, "/F1 12 Tf")

	// 回答正文（按行输出，处理中文）
	for _, line := range splitLines(card.Conclusion, 80) {
		escaped := pdfEscape(line)
		lines = append(lines, fmt.Sprintf("(%s) Tj", escaped))
		lines = append(lines, "0 -18 Td")
	}

	// 来源
	if len(card.Sources) > 0 {
		lines = append(lines, "0 -20 Td")
		lines = append(lines, "/F1 10 Tf")
		lines = append(lines, "(参考来源:) Tj")
		for _, src := range card.Sources {
			lines = append(lines, "0 -14 Td")
			escaped := pdfEscape(fmt.Sprintf("- %s (版本: %s)", src.Title, src.Version))
			lines = append(lines, fmt.Sprintf("(%s) Tj", escaped))
		}
	}

	// 水印
	if watermark {
		lines = append(lines, "0 -40 Td")
		lines = append(lines, "/F1 8 Tf")
		lines = append(lines, "0.5 0.5 0.5 rg")
		lines = append(lines, "(此文档由蔚小芯生成，仅供参考。) Tj")
	}

	lines = append(lines, "ET")
	return strings.Join(lines, "\n")
}

// splitLines 按最大宽度拆分文本
func splitLines(text string, maxLen int) []string {
	var lines []string
	runes := []rune(text)

	for i := 0; i < len(runes); {
		end := i + maxLen
		if end > len(runes) {
			end = len(runes)
		}
		// 尽量在换行符处断开
		chunk := string(runes[i:end])
		if idx := strings.IndexByte(chunk, '\n'); idx >= 0 {
			end = i + idx
			chunk = string(runes[i:end])
			i = end + 1 // 跳过换行符
		} else {
			i = end
		}
		if strings.TrimSpace(chunk) != "" {
			lines = append(lines, chunk)
		}
	}
	if len(lines) == 0 {
		lines = append(lines, text)
	}
	return lines
}

// pdfEscape 转义 PDF 字符串中的特殊字符
func pdfEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	// 过滤或替换非 ASCII 字符（PDF 标准字体不支持中文，用 ? 替代）
	var result strings.Builder
	for _, r := range s {
		if r < 128 {
			result.WriteRune(r)
		} else {
			result.WriteString("?")
		}
	}
	return result.String()
}
