package util

import (
	"regexp"
	"strings"
)

// 知识内容清洗工具。
//
// 背景：知识资源 content 从入库到拼装进大模型提示词的全链路原先没有任何清洗。
// 文档解析若泄漏 OOXML 标记（如 <w:p w:rsidR="00884AF8"/>），污染内容会被
// migrations/001_init.sql 的 FTS5 触发器自动索引（draft 状态也索引），
// 随后进入 ChatService 与各 agent 的系统提示词、sources[] 摘要、推荐卡片与导出。
//
// 本文件提供入库前的兜底清洗，覆盖手动粘贴、NDJSON 导入、/kb/upload 自动入库等所有路径。

// markupTagPattern 匹配 XML/HTML 标签。
// 仅匹配「合法标签名开头」的结构（字母、下划线、命名空间前缀），
// 避免把正文里的数学比较（如 "a < b"）误判为标签。
var markupTagPattern = regexp.MustCompile(`(?s)</?[A-Za-z_][A-Za-z0-9_.:-]*(\s[^<>]*)?/?>`)

// xmlDeclPattern 匹配 XML 声明与处理指令，如 <?xml version="1.0"?>
var xmlDeclPattern = regexp.MustCompile(`(?s)<\?.*?\?>`)

// commentPattern 匹配 HTML/XML 注释
var commentPattern = regexp.MustCompile(`(?s)<!--.*?-->`)

// cdataPattern 匹配 CDATA 包装，保留其中内容
var cdataPattern = regexp.MustCompile(`(?s)<!\[CDATA\[(.*?)]]>`)

// doctypePattern 匹配 DOCTYPE 声明
var doctypePattern = regexp.MustCompile(`(?s)<!DOCTYPE[^>]*>`)

// blankLinesPattern 匹配 3 行及以上连续空行
var blankLinesPattern = regexp.MustCompile(`\n{3,}`)

// StripMarkup 移除文本中残留的 XML/HTML 标签、注释、XML 声明与 CDATA 包装。
//
// 注意：本函数会移除形如 <tag> 的结构，因此不可用于 JSON 内容
// （FAQ 类型的 content 存的是序列化 AnswerCard JSON）。
// 入库层请统一走 SanitizeKnowledgeContent，由它按类型决定是否调用本函数。
func StripMarkup(s string) string {
	if s == "" {
		return s
	}

	// CDATA 先解包，保留内部文本
	s = cdataPattern.ReplaceAllString(s, "$1")
	s = commentPattern.ReplaceAllString(s, "")
	s = xmlDeclPattern.ReplaceAllString(s, "")
	s = doctypePattern.ReplaceAllString(s, "")

	// 标签替换为空格而非直接删除，避免 "甲</w:t><w:t>乙" 粘连成 "甲乙"
	s = markupTagPattern.ReplaceAllString(s, " ")

	return s
}

// NormalizeText 规整空白与控制字符：
//   - 统一换行为 \n
//   - 移除除 \t \n 外的 C0/C1 控制字符与零宽字符
//   - 折叠行内连续空格、去除行尾空白
//   - 连续空行压缩为最多一个空行
func NormalizeText(s string) string {
	if s == "" {
		return s
	}

	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// 移除控制字符与零宽/BOM 字符，制表符与换行保留
	s = strings.Map(func(r rune) rune {
		switch r {
		case '\t', '\n':
			return r
		case '\u200b', '\u200c', '\u200d', '\ufeff':
			// 零宽字符会干扰 FTS 分词与关键词统计
			return -1
		}
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return -1
		}
		// 全角空格统一为半角，便于后续折叠
		if r == '\u3000' || r == '\u00a0' {
			return ' '
		}
		return r
	}, s)

	// 逐行折叠空格并去除行尾空白
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		cells := strings.Split(line, "\t")
		for j, cell := range cells {
			cells[j] = strings.Join(strings.Fields(cell), " ")
		}
		joined := strings.Join(cells, "\t")
		lines[i] = strings.TrimRight(joined, " \t")
	}
	s = strings.Join(lines, "\n")

	// 连续空行压缩为一个空行（保留段落结构）
	s = blankLinesPattern.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}

// SanitizeKnowledgeContent 清洗知识资源的标题、摘要与正文。
//
// 对普通资源（Policy / Process / Activity）执行标签剥离 + 空白规整；
// 调用方若处理 FAQ 类型，应改用 SanitizeKnowledgeContentByType，
// 因为 FAQ 的 content 是 JSON，剥离标签会破坏结构。
func SanitizeKnowledgeContent(s string) string {
	return NormalizeText(StripMarkup(s))
}

// SanitizeKnowledgeContentByType 按资源类型清洗内容。
//
// FAQ 类型的 content 存放序列化后的 AnswerCard JSON（ChatService 会 json.Unmarshal），
// 因此只做控制字符与空白规整，不剥离标签，避免破坏合法 JSON。
func SanitizeKnowledgeContentByType(s, resourceType string) string {
	if strings.EqualFold(resourceType, "FAQ") {
		return NormalizeText(s)
	}
	return SanitizeKnowledgeContent(s)
}

// ContainsMarkup 判断文本是否仍含 XML/HTML 标签，供审核与告警使用
func ContainsMarkup(s string) bool {
	return markupTagPattern.MatchString(s) ||
		xmlDeclPattern.MatchString(s) ||
		commentPattern.MatchString(s)
}
