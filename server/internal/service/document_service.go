package service

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	"github.com/xuri/excelize/v2"
)

type DocumentService struct {
	uploadDir string
	maxSize   int64
}

type DocumentResult struct {
	FileName    string   `json:"file_name"`
	FileType    string   `json:"file_type"`
	FileSize    int64    `json:"file_size"`
	TextContent string   `json:"text_content"`
	Pages       int      `json:"pages,omitempty"`
	Images      []string `json:"images,omitempty"`
	Error       string   `json:"error,omitempty"`
}

func NewDocumentService(uploadDir string, maxSizeMB int) *DocumentService {
	os.MkdirAll(uploadDir, 0755)
	return &DocumentService{
		uploadDir: uploadDir,
		maxSize:   int64(maxSizeMB) * 1024 * 1024,
	}
}

func (s *DocumentService) ProcessUpload(file *multipart.FileHeader) (*DocumentResult, error) {
	if file.Size > s.maxSize {
		return nil, fmt.Errorf("文件大小超过限制（最大 %d MB）", s.maxSize/(1024*1024))
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))

	// 读取文件内容到内存（避免 Vercel 只读文件系统问题）
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()

	fileData, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("读取文件内容失败: %w", err)
	}

	// 尝试保存到磁盘（非关键步骤，失败不影响解析）
	savePath := ""
	timestamp := time.Now().UnixMilli()
	saveName := fmt.Sprintf("%d_%s", timestamp, file.Filename)
	savePath = filepath.Join(s.uploadDir, saveName)
	if dst, err := os.Create(savePath); err == nil {
		_, _ = dst.Write(fileData)
		_ = dst.Close()
	} else {
		// 保存失败时继续解析，不中断流程
		savePath = ""
	}

	// 从内存直接解析文本
	result := &DocumentResult{
		FileName: file.Filename,
		FileType: ext,
		FileSize: file.Size,
	}

	switch ext {
	case ".txt":
		result.TextContent = string(fileData)
	case ".csv":
		result.TextContent, err = s.readCsvFromBytes(fileData)
	case ".pdf":
		result.TextContent, result.Pages, err = s.readPdfFromBytes(fileData)
	case ".docx":
		result.TextContent, err = s.readDocxFromBytes(fileData)
	case ".xlsx":
		result.TextContent, err = s.readXlsxFromBytes(fileData)
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp":
		result.TextContent = fmt.Sprintf("[图片文件] %s (%d KB, %s)", file.Filename, file.Size/1024, ext)
		if savePath != "" {
			result.Images = []string{savePath}
		}
	case ".mp4", ".avi", ".mov", ".mkv", ".flv", ".wmv":
		result.TextContent = fmt.Sprintf("[视频文件] %s (%d KB, %s)", file.Filename, file.Size/1024, ext)
	default:
		result.TextContent = fmt.Sprintf("[未识别格式文件] %s (%d KB, %s)", file.Filename, file.Size/1024, ext)
	}

	if err != nil {
		result.Error = err.Error()
		result.TextContent = fmt.Sprintf("文本提取失败: %v", err)
	}

	return result, nil
}

func (s *DocumentService) readTxt(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *DocumentService) readCsv(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for i, row := range records {
		b.WriteString(fmt.Sprintf("第%d行: %s\n", i+1, strings.Join(row, ", ")))
	}
	return b.String(), nil
}

func (s *DocumentService) readPdf(path string) (string, int, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("打开PDF失败: %w", err)
	}
	defer f.Close()

	totalPage := r.NumPage()
	var b strings.Builder

	for i := 1; i <= totalPage; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err == nil {
			b.WriteString(fmt.Sprintf("\n--- 第%d页 ---\n", i))
			b.WriteString(strings.TrimSpace(text))
			b.WriteString("\n")
		}
	}

	return b.String(), totalPage, nil
}

func (s *DocumentService) readDocx(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("打开DOCX失败: %w", err)
	}
	defer r.Close()

	var b strings.Builder
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			defer rc.Close()
			data, _ := io.ReadAll(rc)
			content := string(data)
			text := extractDocxText(content)
			b.WriteString(text)
			break
		}
	}
	return b.String(), nil
}

func (s *DocumentService) readXlsx(path string) (string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", fmt.Errorf("打开XLSX失败: %w", err)
	}
	defer f.Close()

	var b strings.Builder
	for _, sheet := range f.GetSheetList() {
		b.WriteString(fmt.Sprintf("\n=== 工作表: %s ===\n", sheet))
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		for i, row := range rows {
			b.WriteString(fmt.Sprintf("第%d行: %s\n", i+1, strings.Join(row, ", ")))
		}
	}
	return b.String(), nil
}

// ── 从内存 bytes 解析（兼容 Vercel 只读文件系统）──

func (s *DocumentService) readCsvFromBytes(data []byte) (string, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i, row := range records {
		b.WriteString(fmt.Sprintf("第%d行: %s\n", i+1, strings.Join(row, ", ")))
	}
	return b.String(), nil
}

func (s *DocumentService) readPdfFromBytes(data []byte) (string, int, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", 0, fmt.Errorf("打开PDF失败: %w", err)
	}

	totalPage := r.NumPage()
	var b strings.Builder
	for i := 1; i <= totalPage; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err == nil {
			b.WriteString(fmt.Sprintf("\n--- 第%d页 ---\n", i))
			b.WriteString(strings.TrimSpace(text))
			b.WriteString("\n")
		}
	}
	return b.String(), totalPage, nil
}

func (s *DocumentService) readDocxFromBytes(data []byte) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("打开DOCX失败: %w", err)
	}

	var b strings.Builder
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			defer rc.Close()
			contentData, _ := io.ReadAll(rc)
			text := extractDocxText(string(contentData))
			b.WriteString(text)
			break
		}
	}
	return b.String(), nil
}

func (s *DocumentService) readXlsxFromBytes(data []byte) (string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("打开XLSX失败: %w", err)
	}
	defer f.Close()

	var b strings.Builder
	for _, sheet := range f.GetSheetList() {
		b.WriteString(fmt.Sprintf("\n=== 工作表: %s ===\n", sheet))
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		for i, row := range rows {
			b.WriteString(fmt.Sprintf("第%d行: %s\n", i+1, strings.Join(row, ", ")))
		}
	}
	return b.String(), nil
}

// extractDocxText 从 word/document.xml 中提取文本
func extractDocxText(xmlContent string) string {
	var texts []string
	for {
		start := strings.Index(xmlContent, "<w:t")
		if start == -1 {
			break
		}
		// 找到 > 开头标签结束
		gt := strings.Index(xmlContent[start:], ">")
		if gt == -1 {
			break
		}
		contentStart := start + gt + 1
		end := strings.Index(xmlContent[contentStart:], "</w:t>")
		if end == -1 {
			break
		}
		text := html.UnescapeString(xmlContent[contentStart : contentStart+end])
		text = strings.TrimSpace(text)
		if text == "" {
			xmlContent = xmlContent[contentStart+end+6:]
			continue
		}
		texts = append(texts, text)
		xmlContent = xmlContent[contentStart+end+6:]
	}
	return strings.Join(texts, " ")
}

// ConvertToKnowledgeFormat 将文档内容转换为知识库资源 JSON
func (s *DocumentService) ConvertToKnowledgeFormat(result *DocumentResult, resourceType string) (map[string]interface{}, error) {
	if resourceType == "" {
		resourceType = "FAQ"
	}
	title := strings.TrimSuffix(result.FileName, filepath.Ext(result.FileName))

	// 确定 scope
	scope := "college"
	ownerID := "cs"

	resource := map[string]interface{}{
		"resource_id":   fmt.Sprintf("upload-%d", time.Now().UnixMilli()),
		"resource_type": resourceType,
		"owner_scope":   scope,
		"owner_id":      ownerID,
		"role_scope":    []string{"student", "counselor", "student_union", "college_admin"},
		"version":       "v1.0",
		"status":        "published",
		"title":         title,
		"summary":       fmt.Sprintf("上传文档：%s（%s, %d KB）", result.FileName, result.FileType, result.FileSize/1024),
		"content":       result.TextContent,
		"tags":          []string{"上传文档", result.FileType},
		"updated_by":    "upload",
		"updated_at":    time.Now().Format("2006-01-02 15:04:05"),
	}

	return resource, nil
}

// SaveResult 将 DocumentResult 保存为 JSON 日志（可选）
func (s *DocumentService) SaveResult(result *DocumentResult) error {
	path := filepath.Join(s.uploadDir, fmt.Sprintf("result_%d.json", time.Now().UnixMilli()))
	data, _ := json.MarshalIndent(result, "", "  ")
	return os.WriteFile(path, data, 0644)
}

// GetTextPreview 截取文本前 N 字符用作摘要
func GetTextPreview(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "..."
}

// BytesToContentSize 返回可读的文件大小描述
func BytesToContentSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}

var _ = bytes.MinRead // 确保 bytes 被引用
