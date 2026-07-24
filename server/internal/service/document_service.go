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

// DocumentParseResult 文档解析增强结果
type DocumentParseResult struct {
	Title      string   `json:"title"`      // 文档标题
	Content    string   `json:"content"`    // 提取的纯文本内容
	Summary    string   `json:"summary"`    // 自动生成的摘要
	Keywords   []string `json:"keywords"`   // 关键词列表
	WordCount  int      `json:"word_count"` // 字数
	Paragraphs int      `json:"paragraphs"` // 段落数
	FileName   string   `json:"file_name"`  // 文件名
	FileType   string   `json:"file_type"`  // 文件类型
	FileSize   int64    `json:"file_size"`  // 文件大小
	Pages      int      `json:"pages,omitempty"`
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
	case ".txt", ".md":
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

// ParseDocument 增强解析：提取标题、摘要、关键词、字数、段落数
func (s *DocumentService) ParseDocument(file *multipart.FileHeader) (*DocumentParseResult, error) {
	result, err := s.ProcessUpload(file)
	if err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("%s", result.Error)
	}

	content := strings.TrimSpace(result.TextContent)
	title := extractDocTitle(content, result.FileName)
	summary := generateDocSummary(content, 200)
	keywords := extractDocKeywords(content, 10)
	wordCount := countDocWords(content)
	paragraphs := countDocParagraphs(content)

	return &DocumentParseResult{
		Title:      title,
		Content:    content,
		Summary:    summary,
		Keywords:   keywords,
		WordCount:  wordCount,
		Paragraphs: paragraphs,
		FileName:   result.FileName,
		FileType:   result.FileType,
		FileSize:   result.FileSize,
		Pages:      result.Pages,
	}, nil
}

// extractDocTitle 从文本中提取标题，优先使用正文第一行，其次使用文件名
func extractDocTitle(content, fileName string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			runes := []rune(line)
			if len(runes) > 100 {
				return string(runes[:100])
			}
			return line
		}
	}
	title := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	return title
}

// generateDocSummary 生成摘要（截取前 maxLen 个字符）
func generateDocSummary(content string, maxLen int) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}

// extractDocKeywords 提取关键词（基于词频统计，返回前 topN 个）
func extractDocKeywords(content string, topN int) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return []string{}
	}

	stopWords := map[string]bool{
		"的": true, "了": true, "在": true, "是": true, "我": true, "有": true,
		"和": true, "就": true, "不": true, "人": true, "都": true, "一": true,
		"一个": true, "上": true, "也": true, "很": true, "到": true, "说": true,
		"要": true, "去": true, "你": true, "会": true, "着": true, "没有": true,
		"看": true, "好": true, "自己": true, "这": true, "那": true, "里": true,
		"个": true, "为": true, "来": true, "他": true, "她": true, "它": true,
		"们": true, "而": true, "与": true, "或": true, "但": true, "可": true,
		"以": true, "及": true, "其": true, "之": true, "所": true, "然": true,
		"the": true, "a": true, "an": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "being": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "can": true,
		"and": true, "or": true, "but": true, "not": true, "no": true, "yes": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "into": true,
		"this": true, "that": true, "these": true, "those": true, "it": true,
		"he": true, "she": true, "they": true, "we": true, "you": true, "i": true,
	}

	wordFreq := make(map[string]int)

	// 中文按 2-4 字组合提取，英文按空格分词
	words := strings.FieldsFunc(content, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})

	// 统计英文单词
	for _, w := range words {
		w = strings.ToLower(w)
		if len(w) >= 3 && !stopWords[w] {
			wordFreq[w]++
		}
	}

	// 提取中文 2-4 字词组
	runes := []rune(content)
	for length := 2; length <= 4; length++ {
		for i := 0; i <= len(runes)-length; i++ {
			w := string(runes[i : i+length])
			isChinese := true
			for _, r := range w {
				if !(r >= 0x4e00 && r <= 0x9fff) {
					isChinese = false
					break
				}
			}
			if isChinese && !stopWords[w] {
				wordFreq[w]++
			}
		}
	}

	// 按频率排序
	type wordCount struct {
		word  string
		count int
	}
	var wcList []wordCount
	for w, c := range wordFreq {
		if c >= 2 {
			wcList = append(wcList, wordCount{w, c})
		}
	}

	for i := 0; i < len(wcList)-1; i++ {
		for j := i + 1; j < len(wcList); j++ {
			if wcList[j].count > wcList[i].count {
				wcList[i], wcList[j] = wcList[j], wcList[i]
			}
		}
	}

	result := make([]string, 0, topN)
	for i := 0; i < len(wcList) && i < topN; i++ {
		result = append(result, wcList[i].word)
	}

	return result
}

// countDocWords 统计字数（中文按字符，英文按单词）
func countDocWords(content string) int {
	content = strings.TrimSpace(content)
	if content == "" {
		return 0
	}

	count := 0
	for _, r := range content {
		if r >= 0x4e00 && r <= 0x9fff {
			count++
		}
	}

	words := strings.FieldsFunc(content, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})
	for _, w := range words {
		if len(w) > 0 {
			count++
		}
	}

	return count
}

// countDocParagraphs 统计段落数（按空行分隔）
func countDocParagraphs(content string) int {
	content = strings.TrimSpace(content)
	if content == "" {
		return 0
	}

	paragraphs := 0
	inParagraph := false
	for _, r := range content {
		if r == '\n' {
			if inParagraph {
				paragraphs++
				inParagraph = false
			}
		} else if r != ' ' && r != '\t' && r != '\r' {
			inParagraph = true
		}
	}
	if inParagraph {
		paragraphs++
	}

	return paragraphs
}

var _ = bytes.MinRead // 确保 bytes 被引用
