package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/llm"
)

// fakeOCR 模拟视觉 OCR 客户端：不调用真实 API，验证 OCR 集成路径（提取→OCR→回填）
type fakeOCR struct {
	text string
	err  error
}

func (f *fakeOCR) OCR(_ context.Context, imgs []llm.OCRImage) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if len(imgs) == 0 {
		return "", nil
	}
	return f.text, nil
}

func (f *fakeOCR) Name() string { return "fake" }

func samplePath(name string) (string, bool) {
	p := filepath.Join("..", "..", "..", "data", name)
	_, err := os.Stat(p)
	return p, err == nil
}

// TestOCRIntegration_ScannedDocx 扫描件 DOCX（无文本层）在有 OCR 客户端时应回填 OCR 文本
func TestOCRIntegration_ScannedDocx(t *testing.T) {
	p, ok := samplePath("滁州学院2026级普通专升本新生入学须知.docx")
	if !ok {
		t.Skip("缺少扫描 DOCX 样本")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewDocumentService(t.TempDir(), 10)
	svc.SetOCRClient(&fakeOCR{text: "这是 OCR 识别的文本"})
	text, err := svc.readDocxFromBytes(data)
	if err != nil {
		t.Fatalf("应通过 OCR 解析，实际报错: %v", err)
	}
	if strings.TrimSpace(text) != "这是 OCR 识别的文本" {
		t.Fatalf("OCR 文本未回填: %q", text)
	}
}

// TestOCRIntegration_NoOCRClient 无 OCR 客户端时扫描件仍报无文本层（行为不变）
func TestOCRIntegration_NoOCRClient(t *testing.T) {
	p, ok := samplePath("滁州学院2026级普通专升本新生入学须知.docx")
	if !ok {
		t.Skip("缺少扫描 DOCX 样本")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewDocumentService(t.TempDir(), 10)
	_, err = svc.readDocxFromBytes(data)
	if err == nil || !strings.Contains(err.Error(), "无文本层") {
		t.Fatalf("无 OCR 时应报无文本层，实际: %v", err)
	}
}

// TestOCRIntegration_ScannedPdf 扫描件 PDF（无文本层）在有 OCR 客户端时应回填 OCR 文本
func TestOCRIntegration_ScannedPdf(t *testing.T) {
	p, ok := samplePath("滁州学院2026级普通本科、对口本科新生入学指南.pdf")
	if !ok {
		t.Skip("缺少扫描 PDF 样本")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewDocumentService(t.TempDir(), 10)
	svc.SetOCRClient(&fakeOCR{text: "PDF OCR 文本"})
	text, pages, _, err := svc.readPdfFromBytes(data)
	if err != nil {
		t.Fatalf("应通过 OCR 解析，实际报错: %v", err)
	}
	if pages == 0 || strings.TrimSpace(text) != "PDF OCR 文本" {
		t.Fatalf("OCR 文本未回填: pages=%d text=%q", pages, text)
	}
}
