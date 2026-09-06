package service

import (
	"fmt"
	"mime/multipart"

	"github.com/dll/wxx/server/internal/util"
)

// ParseDocument 增强解析：统一提取标题、摘要、关键词、字数、段落数和质量评估。
// 解析出口统一清洗内容，供文档预览和知识库入库复用，避免标签残留进入下游。
func (s *DocumentService) ParseDocument(file *multipart.FileHeader) (*DocumentParseResult, error) {
	result, err := s.ProcessUpload(file)
	if err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("%s", result.Error)
	}

	content := util.SanitizeKnowledgeContent(result.TextContent)
	title := extractDocTitle(content, result.FileName)
	summary := extractDocSummary(content, 200)
	keywords := extractDocKeywordsWithTitle(content, title, 10)
	wordCount := countDocWords(content)
	paragraphs := countDocParagraphs(content)
	quality := assessDocQuality(stripStructuralMarkers(content))

	return &DocumentParseResult{
		Title: title, Content: content, Summary: summary, Keywords: keywords,
		WordCount: wordCount, Paragraphs: paragraphs, FileName: result.FileName,
		FileType: result.FileType, FileSize: result.FileSize, Pages: result.Pages,
		Quality: quality, ParseWarning: result.Warning,
	}, nil
}
