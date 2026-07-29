package service

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
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
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("打开DOCX失败: %w", err)
	}
	return s.readDocxFromBytes(data)
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

	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("读取DOCX正文失败: %w", err)
		}
		defer rc.Close()
		contentData, err := io.ReadAll(rc)
		if err != nil {
			return "", fmt.Errorf("读取DOCX正文失败: %w", err)
		}
		return extractDocxText(string(contentData)), nil
	}

	// 缺少 word/document.xml 说明不是有效的 DOCX（可能是 .doc 改名）
	return "", fmt.Errorf("DOCX 缺少 word/document.xml，可能是旧版 .doc 格式或文件损坏")
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

// wordprocessingNS 是 OOXML 正文命名空间
const wordprocessingNS = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"

// extractDocxText 从 word/document.xml 中提取纯文本。
//
// 必须按命名空间 + 标签名精确匹配，不能用字符串前缀匹配：
// OOXML 中以 "w:t" 开头的标签还有 w:tc / w:tcPr / w:tab / w:tbl / w:txbxContent
// 等上百种（表格、制表位、文本框），前缀匹配会把整段原始 XML 当作正文提取出来。
//
// 语义映射：w:t 取文本，w:tab 转制表符，w:br / w:cr 转换行，w:p 结束作段落分隔；
// w:instrText（域代码）与 w:delText（修订删除内容）不属于正文，整段跳过。
func extractDocxText(xmlContent string) string {
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	// OOXML 常含 HTML 实体与非标准字符集声明，放宽解码约束避免整篇解析失败
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	var (
		paragraphs []string
		buf        strings.Builder
		// 记录当前是否位于需要采集文本的 w:t 元素内
		inText bool
		// 需要整体跳过的元素嵌套深度（w:instrText / w:delText）
		skipDepth int
	)

	flushParagraph := func() {
		text := normalizeDocxParagraph(buf.String())
		buf.Reset()
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			// io.EOF 为正常结束；其他错误保留已解析内容，避免整篇丢失
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Space != wordprocessingNS {
				continue
			}
			if skipDepth > 0 {
				skipDepth++
				continue
			}
			switch t.Name.Local {
			case "instrText", "delText":
				skipDepth = 1
			case "t":
				inText = true
			case "tab":
				buf.WriteString("\t")
			case "br", "cr":
				buf.WriteString("\n")
			}
		case xml.EndElement:
			if t.Name.Space != wordprocessingNS {
				continue
			}
			if skipDepth > 0 {
				skipDepth--
				continue
			}
			switch t.Name.Local {
			case "t":
				inText = false
			case "p":
				flushParagraph()
			}
		case xml.CharData:
			// 仅采集 w:t 内的字符数据，其余（如属性间空白）一律忽略
			if inText && skipDepth == 0 {
				buf.Write(t)
			}
		}
	}

	// 文档结尾可能缺少闭合 w:p，兜底收尾
	flushParagraph()

	return strings.Join(paragraphs, "\n\n")
}

// normalizeDocxParagraph 规整单个段落：折叠段内多余空格，保留制表符与换行
func normalizeDocxParagraph(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	lines := strings.Split(s, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		// 制表符是 w:tab 的语义（常用于目录、表格对齐），必须保留，
		// 因此按制表符切分后只折叠各段内部的空格
		cells := strings.Split(line, "\t")
		for i, cell := range cells {
			fields := strings.FieldsFunc(cell, func(r rune) bool {
				return r == ' ' || r == '\v' || r == '\f' || r == 0xA0
			})
			cells[i] = strings.Join(fields, " ")
		}
		joined := strings.Join(cells, "\t")
		if strings.TrimSpace(joined) != "" {
			kept = append(kept, joined)
		}
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
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

// extractDocTitle 从文本中提取标题。
//
// 原实现直接取第一个非空行，实测《学分认证标准》会得到「附件：」这类无意义前缀。
// 改为在前若干行中挑选长度合理、不以标点结尾的候选行，均不合适时回退文件名。
func extractDocTitle(content, fileName string) string {
	const (
		scanLines    = 10 // 仅在前若干行内寻找标题
		minTitleRune = 4  // 过短的行（如「附件：」「目录」）不作标题
		maxTitleRune = 60
	)

	// 明显的附属前缀行，不能作为标题
	prefixNoise := []string{"附件", "附录", "附表", "目录", "目 录", "前言", "前 言"}

	var fallback string
	lines := strings.Split(content, "\n")
	scanned := 0
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		scanned++
		if scanned > scanLines {
			break
		}

		// 记录第一个非空行作为次选，保持与原行为兼容
		if fallback == "" {
			fallback = line
		}

		// 去掉「附件：」这类前缀后再判断
		candidate := line
		isNoise := false
		for _, p := range prefixNoise {
			if strings.HasPrefix(candidate, p) {
				trimmed := strings.TrimSpace(strings.TrimLeft(candidate[len(p):], "：: 　"))
				if trimmed == "" {
					isNoise = true // 整行就是「附件：」，跳过
				} else {
					candidate = trimmed
				}
				break
			}
		}
		if isNoise {
			continue
		}

		runes := []rune(candidate)
		if len(runes) < minTitleRune || len(runes) > maxTitleRune {
			continue
		}
		// 以句末标点结尾的多为正文句子，不是标题
		if strings.ContainsRune("。！？；，、,.;!?", runes[len(runes)-1]) {
			continue
		}
		return candidate
	}

	if fallback != "" {
		runes := []rune(fallback)
		if len(runes) > 100 {
			return string(runes[:100])
		}
		return fallback
	}

	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
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
			if isChinese && !stopWords[w] && !isOrdinalPhrase(w) {
				wordFreq[w]++
			}
		}
	}

	// 按频率排序（原实现为冒泡排序，实测 187k 字文档产生 4.3 万候选词、
	// 约 9.5 亿次比较，必须用 O(n log n) 排序）
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

	sort.Slice(wcList, func(i, j int) bool {
		if wcList[i].count != wcList[j].count {
			return wcList[i].count > wcList[j].count
		}
		// 同频优先取更长的词（信息量更大），再按字典序保证结果稳定
		ri, rj := []rune(wcList[i].word), []rune(wcList[j].word)
		if len(ri) != len(rj) {
			return len(ri) > len(rj)
		}
		return wcList[i].word < wcList[j].word
	})

	// 子串抑制：n-gram 全量枚举会产生大量重叠噪声
	// （实测「分别」「别计」「分别计」、「一二三等奖」「二三等」「三等奖」同时入选）。
	//
	// 必须对「全部候选词」而非「已选词」做判断：碎片词频总是 >= 母词词频，
	// 排序后碎片反而在前，只跟已选词比较会让碎片先入选、母词随后被误判为冗余。
	//
	// 为避免在大文档上退化为平方级比较（实测候选词可达 4 万），
	// 只在排序后的前若干高频候选中做互相包含检查——低频词本就取不到 topN。
	inspectLimit := topN * 40
	if inspectLimit > len(wcList) {
		inspectLimit = len(wcList)
	}
	head := wcList[:inspectLimit]

	isFragment := make(map[string]bool, inspectLimit)
	for _, short := range head {
		shortLen := len([]rune(short.word))
		for _, long := range head {
			if len([]rune(long.word)) <= shortLen {
				continue
			}
			// 母词包含碎片，且词频接近（碎片没有独立于母词的用法）
			if strings.Contains(long.word, short.word) &&
				short.count <= long.count+substringFreqTolerance {
				isFragment[short.word] = true
				break
			}
		}
	}

	result := make([]string, 0, topN)
	for _, wc := range head {
		if len(result) >= topN {
			break
		}
		if isFragment[wc.word] {
			continue
		}
		// 已选词之间做互不重叠校验：
		// 滑动窗口会在同一短语上产生「二三等奖」「三等奖和」「等奖和优」这类
		// 首尾交错的窗口，彼此互不包含，仅靠包含判断无法过滤。
		redundant := false
		for _, picked := range result {
			if strings.Contains(picked, wc.word) || strings.Contains(wc.word, picked) ||
				hasSignificantOverlap(picked, wc.word) {
				redundant = true
				break
			}
		}
		if !redundant {
			result = append(result, wc.word)
		}
	}

	return result
}

// hasSignificantOverlap 判断两个词是否为同一短语上的交错窗口。
//
// 例如「二三等奖」与「三等奖和」共享后缀/前缀「三等奖」，长度 3 已达到较短词的
// 多数字符，应视为重复窗口而非两个独立词。
func hasSignificantOverlap(a, b string) bool {
	ra, rb := []rune(a), []rune(b)
	minLen := len(ra)
	if len(rb) < minLen {
		minLen = len(rb)
	}
	if minLen < 2 {
		return false
	}

	// 重叠长度达到较短词的一半以上即认为是同一短语的窗口
	threshold := (minLen + 1) / 2
	for overlap := minLen; overlap >= threshold; overlap-- {
		// a 的后缀 == b 的前缀
		if string(ra[len(ra)-overlap:]) == string(rb[:overlap]) {
			return true
		}
		// b 的后缀 == a 的前缀
		if string(rb[len(rb)-overlap:]) == string(ra[:overlap]) {
			return true
		}
	}
	return false
}

// substringFreqTolerance 子串抑制的词频容差：
// 子串词频不超过「包含它的长词词频 + 容差」时视为碎片
const substringFreqTolerance = 1

// isOrdinalPhrase 判断是否为条款序号类词组（如「第二」「第十」「二十三」）。
// 制度文档中这类词高频出现但无检索价值，实测会挤占 top10 关键词位置。
func isOrdinalPhrase(w string) bool {
	runes := []rune(w)
	if len(runes) == 0 {
		return false
	}

	// 中文数字与量词字符集
	numerals := map[rune]bool{
		'零': true, '一': true, '二': true, '三': true, '四': true, '五': true,
		'六': true, '七': true, '八': true, '九': true, '十': true, '百': true,
		'千': true, '万': true, '两': true, '〇': true,
	}
	// 序号结构常见的前后缀
	affixes := map[rune]bool{
		'第': true, '条': true, '款': true, '项': true, '章': true, '节': true,
		'编': true, '页': true, '次': true, '届': true, '等': true,
	}

	numeralCount := 0
	for _, r := range runes {
		switch {
		case numerals[r]:
			numeralCount++
		case affixes[r]:
			// 前后缀本身不计入数字，但允许出现
		default:
			// 含有普通汉字，说明是实义词
			return false
		}
	}

	// 全部由数字与序号前后缀组成，且至少含一个数字
	return numeralCount > 0
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
