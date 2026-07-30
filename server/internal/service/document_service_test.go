package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildDocxXML 构造一段包含各类 "w:t*" 干扰标签的 OOXML 正文
func buildDocxXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p w:rsidR="00884AF8" w:rsidRDefault="00884AF8">
  <w:pPr><w:rPr><w:rFonts w:ascii="宋体" w:hAnsi="宋体"/><w:sz w:val="21"/><w:szCs w:val="21"/></w:rPr></w:pPr>
  <w:r><w:t>学生手册</w:t></w:r>
</w:p>
<w:tbl>
  <w:tblPr><w:tblW w:w="0" w:type="auto"/><w:tblLayout w:type="fixed"/></w:tblPr>
  <w:tr><w:trPr><w:trHeight w:val="300"/></w:trPr>
    <w:tc><w:tcPr><w:tcW w:w="100" w:type="dxa"/></w:tcPr>
      <w:p><w:r><w:t>请假流程</w:t></w:r></w:p>
    </w:tc>
  </w:tr>
</w:tbl>
<w:p>
  <w:r><w:t>第一条</w:t></w:r><w:r><w:tab/></w:r><w:r><w:t>学生应遵守规定</w:t></w:r>
  <w:r><w:br/></w:r><w:r><w:t>第二条</w:t></w:r>
</w:p>
<w:p><w:r><w:instrText> HYPERLINK "http://example.com" </w:instrText></w:r></w:p>
<w:p><w:r><w:delText>已删除的旧条款</w:delText></w:r></w:p>
<w:p><w:r><w:t>转义内容 &lt;b&gt;加粗&lt;/b&gt;</w:t></w:r></w:p>
<w:p><w:r><w:t xml:space="preserve">  多余    空格  </w:t></w:r></w:p>
</w:body>
</w:document>`
}

// TestExtractDocxTextNoMarkupLeak 回归：不得泄漏任何 OOXML 标签
func TestExtractDocxTextNoMarkupLeak(t *testing.T) {
	got := extractDocxText(buildDocxXML())

	for _, bad := range []string{"<w:", "</w:", "rsid", "w:rsidR", "rFonts", "szCs", "tcPr", "tblPr", "w:val"} {
		if strings.Contains(got, bad) {
			t.Errorf("提取结果泄漏 OOXML 标记 %q\n实际输出:\n%s", bad, got)
		}
	}
}

// TestExtractDocxTextSemantics 校验标签语义映射与跳过规则
func TestExtractDocxTextSemantics(t *testing.T) {
	got := extractDocxText(buildDocxXML())

	// w:t 文本必须保留
	for _, want := range []string{"学生手册", "请假流程", "第一条", "学生应遵守规定", "第二条"} {
		if !strings.Contains(got, want) {
			t.Errorf("缺少正文文本 %q\n实际输出:\n%s", want, got)
		}
	}

	// w:instrText（域代码）与 w:delText（修订删除）必须跳过
	if strings.Contains(got, "HYPERLINK") || strings.Contains(got, "example.com") {
		t.Errorf("未跳过 w:instrText 域代码\n实际输出:\n%s", got)
	}
	if strings.Contains(got, "已删除的旧条款") {
		t.Errorf("未跳过 w:delText 修订删除内容\n实际输出:\n%s", got)
	}

	// w:tab 转制表符，w:br 转换行
	if !strings.Contains(got, "第一条\t学生应遵守规定") {
		t.Errorf("w:tab 未转换为制表符\n实际输出:\n%q", got)
	}
	if !strings.Contains(got, "学生应遵守规定\n第二条") {
		t.Errorf("w:br 未转换为换行\n实际输出:\n%q", got)
	}

	// w:p 之间按空行分段
	if !strings.Contains(got, "学生手册\n\n请假流程") {
		t.Errorf("段落未按 w:p 分隔\n实际输出:\n%q", got)
	}

	// 段内多余空格应折叠
	if strings.Contains(got, "多余    空格") {
		t.Errorf("段内多余空格未折叠\n实际输出:\n%q", got)
	}
	if !strings.Contains(got, "多余 空格") {
		t.Errorf("空格折叠结果不符合预期\n实际输出:\n%q", got)
	}
}

// TestExtractDocxTextNoDoubleUnescape 实体只解码一次。
//
// 原实现在 encoding/xml 之外又调用了 html.UnescapeString，导致文档里写的
// "&amp;lt;script&amp;gt;" 会被解码两次，变成可执行标签 "<script>"。
// 单次解码只应还原到 "&lt;script&gt;"。
func TestExtractDocxTextNoDoubleUnescape(t *testing.T) {
	xmlDoc := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body><w:p><w:r><w:t>双重转义 &amp;lt;script&amp;gt;alert(1)&amp;lt;/script&amp;gt;</w:t></w:r></w:p></w:body>` +
		`</w:document>`

	got := extractDocxText(xmlDoc)

	if strings.Contains(got, "<script>") {
		t.Errorf("实体被二次解码为真实标签，存在注入风险\n实际输出:\n%s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("单次解码结果不符合预期\n实际输出:\n%s", got)
	}
}

// TestExtractDocxTextSingleUnescape 单层转义正常还原为字面文本
func TestExtractDocxTextSingleUnescape(t *testing.T) {
	got := extractDocxText(buildDocxXML())
	// 文档中写的 &lt;script&gt; 语义上就是字面的 "<script>" 文本，应正常还原；
	// 阻断注入由入库层 util.SanitizeKnowledgeContent 负责，不在提取阶段做
	if !strings.Contains(got, "转义内容") {
		t.Errorf("转义段落文本丢失\n实际输出:\n%s", got)
	}
}

// TestExtractDocxTextEmpty 空输入与非法 XML 不应 panic
func TestExtractDocxTextEmpty(t *testing.T) {
	if got := extractDocxText(""); got != "" {
		t.Errorf("空输入应返回空字符串，实际 %q", got)
	}
	// 截断的 XML 应保留已解析部分而非崩溃
	partial := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body><w:p><w:r><w:t>可用文本</w:t></w:r></w:p><w:p><w:r><w:t>截断`
	got := extractDocxText(partial)
	if !strings.Contains(got, "可用文本") {
		t.Errorf("截断 XML 应保留已解析内容，实际 %q", got)
	}
}

// TestReadDocxFromBytesInvalid 非 DOCX 输入应返回明确错误而非空内容
func TestReadDocxFromBytesInvalid(t *testing.T) {
	svc := NewDocumentService(t.TempDir(), 10)

	if _, err := svc.readDocxFromBytes([]byte("not a zip")); err == nil {
		t.Error("非 ZIP 数据应返回错误")
	}
}

// TestExtractDocTitleSkipsNoisePrefix 「附件：」这类前缀行不应成为标题
func TestExtractDocTitleSkipsNoisePrefix(t *testing.T) {
	content := "附件：\n\n滁州学院本科生“第二课堂”学分认证标准\n\n一、项目定级标准\n"
	got := extractDocTitle(content, "标准.docx")
	want := "滁州学院本科生“第二课堂”学分认证标准"
	if got != want {
		t.Errorf("标题提取错误\n期望: %q\n实际: %q", want, got)
	}
}

// TestExtractDocTitleInlinePrefix 「附件：标题」同行时应剥离前缀
func TestExtractDocTitleInlinePrefix(t *testing.T) {
	content := "附件：学生请假管理办法\n\n第一条 总则\n"
	got := extractDocTitle(content, "x.docx")
	if got != "学生请假管理办法" {
		t.Errorf("未剥离行内前缀，实际: %q", got)
	}
}

// TestExtractDocTitleSkipsSentence 以句末标点结尾的正文句子不作标题
func TestExtractDocTitleSkipsSentence(t *testing.T) {
	content := "为规范学生管理，特制定本办法。\n\n学生手册管理规定\n"
	got := extractDocTitle(content, "x.docx")
	if got != "学生手册管理规定" {
		t.Errorf("句子被误判为标题，实际: %q", got)
	}
}

// TestExtractDocTitleFallbackFileName 无正文时回退文件名
func TestExtractDocTitleFallbackFileName(t *testing.T) {
	if got := extractDocTitle("", "学生手册.docx"); got != "学生手册" {
		t.Errorf("未回退文件名，实际: %q", got)
	}
}

// TestExtractDocKeywordsNoMarkup 关键词不得出现 XML 属性名
func TestExtractDocKeywordsNoMarkup(t *testing.T) {
	content := extractDocxText(buildDocxXML())
	got := extractDocKeywords(content, 10)
	for _, kw := range got {
		for _, bad := range []string{"rpr", "rfonts", "hansi", "szcs", "val", "ascii", "tcpr"} {
			if strings.EqualFold(kw, bad) {
				t.Errorf("关键词包含 XML 属性名 %q，全部结果: %v", kw, got)
			}
		}
	}
}

// TestExtractDocKeywordsFiltersOrdinals 条款序号不应占据关键词位置
func TestExtractDocKeywordsFiltersOrdinals(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 30; i++ {
		b.WriteString("第二条 第十条 第二十三条 学生管理规定 ")
	}
	got := extractDocKeywords(b.String(), 10)

	for _, kw := range got {
		if isOrdinalPhrase(kw) {
			t.Errorf("关键词包含条款序号 %q，全部结果: %v", kw, got)
		}
	}
	// 实义词应当入选
	found := false
	for _, kw := range got {
		if strings.Contains(kw, "学生") || strings.Contains(kw, "管理") || strings.Contains(kw, "规定") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("实义词未入选，实际结果: %v", got)
	}
}

// TestExtractDocKeywordsSuppressesSubstrings 重叠 n-gram 碎片应被抑制
func TestExtractDocKeywordsSuppressesSubstrings(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 30; i++ {
		b.WriteString("一二三等奖 ")
	}
	got := extractDocKeywords(b.String(), 10)

	// 长词入选后，其纯子串不应再出现
	for i, a := range got {
		for j, c := range got {
			if i != j && strings.Contains(a, c) {
				t.Errorf("关键词 %q 是 %q 的子串，未被抑制。全部结果: %v", c, a, got)
			}
		}
	}
}

// TestIsOrdinalPhrase 序号判定的正负样本
func TestIsOrdinalPhrase(t *testing.T) {
	ordinals := []string{"第二", "第十", "二十", "三十四", "第一条", "十二"}
	for _, w := range ordinals {
		if !isOrdinalPhrase(w) {
			t.Errorf("%q 应判定为序号词组", w)
		}
	}
	// 含数字但为实义词的情况不能被误杀
	meaningful := []string{"学生", "管理规定", "第二课堂", "一表通", "十佳青年"}
	for _, w := range meaningful {
		if isOrdinalPhrase(w) {
			t.Errorf("%q 不应判定为序号词组", w)
		}
	}
}

// TestExtractDocKeywordsPerformance 大文档关键词提取不得退化到平方级
// （原实现为冒泡排序，187k 字文档约需 9.5 亿次比较）
func TestExtractDocKeywordsPerformance(t *testing.T) {
	matches, _ := filepath.Glob(filepath.Join("..", "..", "..", "data", "*.docx"))
	if len(matches) == 0 {
		t.Skip("data/ 下无 .docx 样本，跳过性能回归")
	}

	svc := NewDocumentService(t.TempDir(), 10)
	var content string
	for _, p := range matches {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		text, err := svc.readDocxFromBytes(data)
		if err == nil && len([]rune(text)) > len([]rune(content)) {
			content = text
		}
	}
	if content == "" {
		t.Skip("无可用样本内容")
	}

	start := time.Now()
	got := extractDocKeywords(content, 10)
	elapsed := time.Since(start)

	t.Logf("文档长度 %d 字符，关键词提取耗时 %v，结果: %v", len([]rune(content)), elapsed, got)
	if elapsed > 10*time.Second {
		t.Errorf("关键词提取耗时 %v，疑似性能退化", elapsed)
	}
	if len(got) == 0 {
		t.Error("真实文档未提取到任何关键词")
	}
}
// TestHasSignificantOverlap 交错窗口判定的正负样本
func TestHasSignificantOverlap(t *testing.T) {
	overlapping := [][2]string{
		{"二三等奖", "三等奖和"}, // 共享「三等奖」
		{"三等奖和", "等奖和优"}, // 共享「等奖和」
		{"学生管理", "生管理规"}, // 共享「生管理」
	}
	for _, pair := range overlapping {
		if !hasSignificantOverlap(pair[0], pair[1]) {
			t.Errorf("%q 与 %q 应判定为交错窗口", pair[0], pair[1])
		}
	}

	independent := [][2]string{
		{"学生", "学校"},
		{"请假", "奖学金"},
		{"管理规定", "考试纪律"},
	}
	for _, pair := range independent {
		if hasSignificantOverlap(pair[0], pair[1]) {
			t.Errorf("%q 与 %q 不应判定为交错窗口", pair[0], pair[1])
		}
	}
}

// TestExtractDocKeywordsNoOverlappingWindows 结果中不得出现首尾交错的滑动窗口
func TestExtractDocKeywordsNoOverlappingWindows(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString("一二三等奖和优秀奖 ")
	}
	got := extractDocKeywords(b.String(), 10)

	for i, a := range got {
		for j, c := range got {
			if i >= j {
				continue
			}
			if hasSignificantOverlap(a, c) {
				t.Errorf("关键词 %q 与 %q 为交错窗口，未被抑制。全部结果: %v", a, c, got)
			}
		}
	}
}

// TestReadDocxRealFile 用仓库 data/ 下真实文档做回归（文件缺失时跳过）
func TestReadDocxRealFile(t *testing.T) {
	matches, _ := filepath.Glob(filepath.Join("..", "..", "..", "data", "*.docx"))
	if len(matches) == 0 {
		t.Skip("data/ 下无 .docx 样本，跳过真实文档回归")
	}

	svc := NewDocumentService(t.TempDir(), 10)
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("读取样本失败 %s: %v", path, err)
		}
		text, err := svc.readDocxFromBytes(data)
		if err != nil {
			t.Fatalf("解析样本失败 %s: %v", path, err)
		}
		if strings.TrimSpace(text) == "" {
			t.Errorf("样本 %s 解析结果为空", path)
		}
		for _, bad := range []string{"<w:", "rsid", "rFonts", "szCs"} {
			if strings.Contains(text, bad) {
				t.Errorf("样本 %s 泄漏 OOXML 标记 %q", filepath.Base(path), bad)
			}
		}
	}
}
