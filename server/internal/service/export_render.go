package service

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/model"
	"github.com/xuri/excelize/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var defaultCJKFontCandidates = []string{
	`C:\Windows\Fonts\simhei.ttf`,
	`C:\Windows\Fonts\msyh.ttc`,
	`/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc`,
	`/usr/share/fonts/truetype/wqy/wqy-microhei.ttc`,
	`/usr/share/fonts/truetype/arphic/uming.ttc`,
}

func (s *ExportService) resolveCJKFontPath() string {
	if s.cjkFontPath != "" {
		return s.cjkFontPath
	}
	for _, p := range defaultCJKFontCandidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (s *ExportService) exportDOCX(card *model.AnswerCard, watermark bool) ([]byte, error) {
	var doc strings.Builder
	doc.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	doc.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	appendDocxParagraph(&doc, "蔚小芯 AI 回答", true)
	appendDocxParagraph(&doc, "生成时间："+time.Now().Format("2006-01-02 15:04:05"), false)
	appendDocxParagraph(&doc, "", false)
	for _, line := range wrapText(card.Conclusion, 60) {
		appendDocxParagraph(&doc, line, false)
	}
	if len(card.Sources) > 0 {
		appendDocxParagraph(&doc, "", false)
		appendDocxParagraph(&doc, "参考来源", true)
		for _, src := range card.Sources {
			appendDocxParagraph(&doc, "- "+src.Title+"（版本："+src.Version+"）", false)
		}
	}
	if watermark {
		appendDocxParagraph(&doc, "", false)
		appendDocxParagraph(&doc, "此文档由蔚小芯生成，仅供参考。", false)
	}
	doc.WriteString(`</w:body></w:document>`)

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml":   doc.String(),
	}
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func appendDocxParagraph(b *strings.Builder, text string, bold bool) {
	b.WriteString(`<w:p><w:r>`)
	if bold {
		b.WriteString(`<w:rPr><w:b/></w:rPr>`)
	}
	b.WriteString(`<w:t xml:space="preserve">` + xmlEscape(text) + `</w:t></w:r></w:p>`)
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func (s *ExportService) exportXLSX(card *model.AnswerCard) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "回答"
	idx, err := f.NewSheet(sheet)
	if err != nil {
		return nil, err
	}
	_ = f.SetCellValue(sheet, "A1", "蔚小芯 AI 回答")
	_ = f.SetCellValue(sheet, "A2", "生成时间："+time.Now().Format("2006-01-02 15:04:05"))
	_ = f.SetCellValue(sheet, "A3", card.Conclusion)
	row := 5
	if len(card.Sources) > 0 {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "参考来源")
		row++
		for _, src := range card.Sources {
			_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), src.Title)
			_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), src.Version)
			_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), src.SourceLink)
			row++
		}
	}
	f.SetActiveSheet(idx)
	if err := f.SetColWidth(sheet, "A", "C", 40); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ExportService) exportICS(card *model.AnswerCard) []byte {
	now := time.Now()
	uid := fmt.Sprintf("wxx-%d@weixiaoxin", now.Unix())
	summary := strings.ReplaceAll(card.Conclusion, "\n", " ")
	if len(summary) > 80 {
		summary = summary[:80]
	}
	summary = icsEscape(summary)
	return []byte(fmt.Sprintf(
		"BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//weixiaoxin//Export//CN\r\nBEGIN:VEVENT\r\nUID:%s\r\nDTSTAMP:%s\r\nDTSTART;VALUE=DATE:%s\r\nSUMMARY:%s\r\nDESCRIPTION:%s\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n",
		uid,
		now.UTC().Format("20060102T150405Z"),
		now.Format("20060102"),
		summary,
		icsEscape(card.TraceID),
	))
}

func icsEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func (s *ExportService) exportPNG(card *model.AnswerCard, watermark bool) ([]byte, error) {
	img, err := s.renderCardImage(card, watermark)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ExportService) renderCardImage(card *model.AnswerCard, watermark bool) (*image.RGBA, error) {
	fontPath := s.resolveCJKFontPath()
	if fontPath == "" {
		return nil, errors.New("未找到可用于中文导出的字体，请配置 EXPORT_FONT_PATH")
	}
	data, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("读取导出字体失败: %w", err)
	}
	ft, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("解析导出字体失败: %w", err)
	}
	face, err := opentype.NewFace(ft, &opentype.FaceOptions{Size: 20, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}

	width := 1000
	height := 1400
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)

	textColor := color.RGBA{R: 30, G: 30, B: 30, A: 255}
	gray := color.RGBA{R: 130, G: 130, B: 130, A: 255}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: face,
		Dot:  fixed.P(40, 70),
	}
	d.DrawString("蔚小芯 AI 回答")
	d.Dot.Y += fixed.I(40)
	d.Src = image.NewUniform(gray)
	d.DrawString("生成时间：" + time.Now().Format("2006-01-02 15:04:05"))
	d.Dot.Y += fixed.I(50)
	d.Src = image.NewUniform(textColor)
	for _, line := range wrapText(card.Conclusion, 45) {
		d.DrawString(line)
		d.Dot.Y += fixed.I(36)
	}
	if len(card.Sources) > 0 {
		d.Dot.Y += fixed.I(30)
		d.DrawString("参考来源")
		d.Dot.Y += fixed.I(36)
		for _, src := range card.Sources {
			for _, line := range wrapText("- "+src.Title+"（版本："+src.Version+"）", 45) {
				d.DrawString(line)
				d.Dot.Y += fixed.I(36)
			}
		}
	}
	if watermark {
		d.Dot.Y += fixed.I(40)
		d.Src = image.NewUniform(gray)
		d.DrawString("此文档由蔚小芯生成，仅供参考。")
	}
	return img, nil
}

func wrapText(text string, maxRunes int) []string {
	runes := []rune(text)
	if len(runes) == 0 {
		return []string{""}
	}
	var lines []string
	for len(runes) > 0 {
		lineLen := maxRunes
		if lineLen > len(runes) {
			lineLen = len(runes)
		}
		lines = append(lines, string(runes[:lineLen]))
		runes = runes[lineLen:]
	}
	return lines
}

// exportPDF 优先使用中文渲染图片生成 PDF；无法加载字体时回退到原 ASCII 实现。
func (s *ExportService) exportPDF(card *model.AnswerCard, watermark bool) []byte {
	if data, err := s.exportPDFWithImage(card, watermark); err == nil {
		return data
	}
	return s.exportPDFASCII(card, watermark)
}

func (s *ExportService) exportPDFWithImage(card *model.AnswerCard, watermark bool) ([]byte, error) {
	img, err := s.renderCardImage(card, watermark)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	var jpegBuf bytes.Buffer
	if err := jpeg.Encode(&jpegBuf, img, &jpeg.Options{Quality: 92}); err != nil {
		return nil, err
	}
	jpegData := jpegBuf.Bytes()

	iw := bounds.Dx()
	ih := bounds.Dy()
	pageW := 595.0
	pageH := 842.0
	scale := pageW / float64(iw)
	if float64(ih)*scale > pageH {
		scale = pageH / float64(ih)
	}
	drawW := float64(iw) * scale
	drawH := float64(ih) * scale
	x := (pageW - drawW) / 2
	y := (pageH - drawH) / 2

	content := fmt.Sprintf(
		"q\n%s %s %s %s re\nW n\n%s %s %s %s cm\n/Im1 Do\nQ",
		formatPDFNumber(x), formatPDFNumber(y), formatPDFNumber(drawW), formatPDFNumber(drawH),
		formatPDFNumber(drawW), formatPDFNumber(drawH), formatPDFNumber(x), formatPDFNumber(y),
	)
	obj1 := "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj"
	obj2 := "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj"
	obj3 := fmt.Sprintf("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Contents 4 0 R /Resources << /XObject << /Im1 5 0 R >> >> >>\nendobj", pageW, pageH)
	obj4 := fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj", len(content), content)
	obj5 := fmt.Sprintf("5 0 obj\n<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>\nstream\n", iw, ih, len(jpegData))
	obj6 := "endstream\nendobj"

	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := []int{b.Len()}
	b.WriteString(obj1 + "\n")
	offsets = append(offsets, b.Len())
	b.WriteString(obj2 + "\n")
	offsets = append(offsets, b.Len())
	b.WriteString(obj3 + "\n")
	offsets = append(offsets, b.Len())
	b.WriteString(obj4 + "\n")
	offsets = append(offsets, b.Len())
	b.WriteString(obj5)
	offsets = append(offsets, b.Len())
	b.Write(jpegData)
	b.WriteString("\n")
	offsets = append(offsets, b.Len())
	b.WriteString(obj6 + "\n")
	xrefOffset := b.Len()
	b.WriteString("xref\n")
	b.WriteString(fmt.Sprintf("0 %d\n", len(offsets)+1))
	b.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}
	b.WriteString("trailer\n")
	b.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", len(offsets)+1))
	b.WriteString("startxref\n")
	b.WriteString(fmt.Sprintf("%d\n%%EOF\n", xrefOffset))
	return b.Bytes(), nil
}

func formatPDFNumber(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

// exportPDFASCII 保留原实现作为无中文字体环境下的降级。
func (s *ExportService) exportPDFASCII(card *model.AnswerCard, watermark bool) []byte {
	var b strings.Builder
	content := s.buildPDFContent(card, watermark)
	objects := []string{}
	obj1 := fmt.Sprintf("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj")
	obj2 := fmt.Sprintf("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj")
	obj3 := fmt.Sprintf("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj")
	obj4 := fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj", len(content), content)
	obj5 := "5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj"
	objects = append(objects, obj1, obj2, obj3, obj4, obj5)
	offsets := []int{}
	offset := 0
	for _, obj := range objects {
		offsets = append(offsets, offset)
		offset += len(obj) + 1
	}
	b.WriteString("%PDF-1.4\n")
	for _, obj := range objects {
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
	return []byte(b.String())
}
