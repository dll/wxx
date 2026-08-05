package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/util"
)

// TestParseRealHandbookEndToEnd 用真实《学生手册》走完整解析链路，
// 断言前端表单看到的 title / summary / content / keywords 均无 OOXML 污染。
// 复现的是用户报告的现象：标题栏出现 <w:p w:rsidR="00884AF8" .../>，
// 关键词出现 宋体 / val / rpr / rfonts / ascii / hansi / ppr / szcs。
func TestParseRealHandbookEndToEnd(t *testing.T) {
	matches, _ := filepath.Glob(filepath.Join("..", "..", "..", "data", "*.docx"))
	if len(matches) == 0 {
		t.Skip("data/ 下无 .docx 样本，跳过端到端回归")
	}

	svc := NewDocumentService(t.TempDir(), 10)

	// 用户报告中出现过的污染标志
	badMarkers := []string{
		"<w:", "</w:", "w:rsidR", "rsidRDefault", "txbxContent",
		"wps:", "bodyPr", "rFonts", "szCs", "rPr", "pPr",
	}
	// 关键词里曾出现的 XML 属性名
	badKeywords := []string{
		"val", "rpr", "ppr", "rfonts", "szcs", "ascii", "hansi", "宋体",
	}

	for _, path := range matches {
		name := filepath.Base(path)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("读取样本失败 %s: %v", name, err)
		}

		content, err := svc.readDocxFromBytes(data)
		if err != nil {
			// 扫描件/图片型 DOCX（无文本层，需 OCR）为合法样本，跳过标记断言
			if errors.Is(err, ErrNoTextLayer) {
				t.Logf("样本 %s 为扫描件/图片型（无文本层），跳过", name)
				continue
			}
			// 非 DOCX 文件（如纯文本/PDF 误命名为 .docx）跳过，不视为测试失败
			if strings.Contains(err.Error(), "not a valid zip file") {
				t.Logf("样本 %s 非有效 DOCX（%v），跳过", name, err)
				continue
			}
			t.Fatalf("解析样本失败 %s: %v", name, err)
		}
		content = strings.TrimSpace(content)

		title := extractDocTitle(content, name)
		summary := extractDocSummary(content, 200)
		keywords := extractDocKeywords(content, 10)
		wordCount := countDocWords(content)
		paragraphs := countDocParagraphs(content)

		t.Logf("样本 %s\n  字数=%d 段落=%d\n  标题=%q\n  摘要=%.60q\n  关键词=%v",
			name, wordCount, paragraphs, title, summary, keywords)

		// 三个回填字段都不得含 OOXML 标记
		for field, val := range map[string]string{
			"title":   title,
			"summary": summary,
			"content": content,
		} {
			for _, bad := range badMarkers {
				if strings.Contains(val, bad) {
					t.Errorf("%s 的 %s 字段含 OOXML 标记 %q", name, field, bad)
				}
			}
			if util.ContainsMarkup(val) {
				t.Errorf("%s 的 %s 字段仍被判定含标记：%.120q", name, field, val)
			}
		}

		// 关键词必须是中文实义词，不能是 XML 属性名
		if len(keywords) == 0 {
			t.Errorf("%s 未提取到关键词", name)
		}
		for _, kw := range keywords {
			for _, bad := range badKeywords {
				if strings.EqualFold(kw, bad) {
					t.Errorf("%s 关键词含 XML 属性名 %q，全部: %v", name, kw, keywords)
				}
			}
		}

		// 标题不应过长或为空（用户截图中标题被灌入整段 XML）
		titleRunes := []rune(title)
		if len(titleRunes) == 0 {
			t.Errorf("%s 标题为空", name)
		}
		if len(titleRunes) > 100 {
			t.Errorf("%s 标题过长（%d 字），疑似灌入正文", name, len(titleRunes))
		}

		if wordCount == 0 {
			t.Errorf("%s 字数统计为 0", name)
		}
	}
}
