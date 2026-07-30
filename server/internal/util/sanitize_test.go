package util

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestStripMarkupRemovesOOXML 真实污染样本：docx 解析泄漏的 OOXML 标记
func TestStripMarkupRemovesOOXML(t *testing.T) {
	polluted := `学 生 手 册 学生工作部（处）编印 <w:p w:rsidR="00884AF8" ` +
		`w:rsidRDefault="00884AF8"/></w:txbxContent></wps:txbx>` +
		`<wps:bodyPr rot="0" vert="horz" wrap="square"/>正文内容`

	got := StripMarkup(polluted)

	for _, bad := range []string{"<w:", "</w:", "w:rsidR", "wps:txbx", "bodyPr", "rot=", "/>"} {
		if strings.Contains(got, bad) {
			t.Errorf("未移除标记 %q\n实际: %q", bad, got)
		}
	}
	for _, want := range []string{"学生工作部", "正文内容"} {
		if !strings.Contains(got, want) {
			t.Errorf("正文文本丢失 %q\n实际: %q", want, got)
		}
	}
}

// TestStripMarkupNoConcatenation 标签替换为空格，不得让相邻文字粘连
func TestStripMarkupNoConcatenation(t *testing.T) {
	got := StripMarkup(`甲</w:t><w:t>乙`)
	if strings.Contains(got, "甲乙") {
		t.Errorf("标签移除导致文字粘连，实际: %q", got)
	}
}

// TestStripMarkupKeepsMathComparison 正文里的数学比较不应被误判为标签
func TestStripMarkupKeepsMathComparison(t *testing.T) {
	src := "绩点 a < b 且 c > d 的情况"
	got := StripMarkup(src)
	if !strings.Contains(got, "a < b") || !strings.Contains(got, "c > d") {
		t.Errorf("数学比较符被误删\n原文: %q\n实际: %q", src, got)
	}
}

// TestStripMarkupCDATAAndComments CDATA 保留内容，注释整体移除
func TestStripMarkupCDATAAndComments(t *testing.T) {
	got := StripMarkup(`<!-- 内部批注 -->正文<![CDATA[保留文本]]>`)
	if strings.Contains(got, "内部批注") {
		t.Errorf("注释未移除，实际: %q", got)
	}
	if !strings.Contains(got, "保留文本") {
		t.Errorf("CDATA 内容丢失，实际: %q", got)
	}
	if strings.Contains(got, "CDATA") {
		t.Errorf("CDATA 包装未剥离，实际: %q", got)
	}
}

// TestNormalizeTextWhitespace 空白规整：折叠空格、压缩空行、去行尾空白
func TestNormalizeTextWhitespace(t *testing.T) {
	got := NormalizeText("第一条    学生   应遵守规定   \n\n\n\n第二条\n")
	want := "第一条 学生 应遵守规定\n\n第二条"
	if got != want {
		t.Errorf("空白规整错误\n期望: %q\n实际: %q", want, got)
	}
}

// TestNormalizeTextRemovesControlChars 控制字符与零宽字符必须移除
func TestNormalizeTextRemovesControlChars(t *testing.T) {
	// 用转义写法避免源码内出现真实的零宽/BOM 字符
	const (
		zeroWidthSpace = "\u200b"
		bom            = "\ufeff"
	)
	src := "正常\x00文本\x07内容" + zeroWidthSpace + "带零宽" + bom
	got := NormalizeText(src)

	for name, bad := range map[string]string{
		"NUL":  "\x00",
		"BEL":  "\x07",
		"零宽空格": zeroWidthSpace,
		"BOM":  bom,
	} {
		if strings.Contains(got, bad) {
			t.Errorf("未移除%s字符，实际: %q", name, got)
		}
	}
	if !strings.Contains(got, "正常") || !strings.Contains(got, "带零宽") {
		t.Errorf("正文丢失，实际: %q", got)
	}
}

// TestNormalizeTextPreservesTabAndNewline 制表符与换行是有语义的，必须保留
func TestNormalizeTextPreservesTabAndNewline(t *testing.T) {
	got := NormalizeText("前言\t1\n第一章\t3")
	if !strings.Contains(got, "\t") {
		t.Errorf("制表符被移除，实际: %q", got)
	}
	if !strings.Contains(got, "\n") {
		t.Errorf("换行被移除，实际: %q", got)
	}
}

// TestNormalizeTextFullWidthSpace 全角空格应折叠，避免干扰 FTS 分词
func TestNormalizeTextFullWidthSpace(t *testing.T) {
	got := NormalizeText("学　生　手　册")
	if got != "学 生 手 册" {
		t.Errorf("全角空格未规整，实际: %q", got)
	}
}

// TestSanitizeKnowledgeContentFAQPreservesJSON
// FAQ 的 content 是序列化 AnswerCard JSON，清洗后必须仍可反序列化
func TestSanitizeKnowledgeContentFAQPreservesJSON(t *testing.T) {
	type answerCard struct {
		Answer  string   `json:"answer"`
		Sources []string `json:"sources"`
		Note    string   `json:"note"`
	}
	original := answerCard{
		Answer:  "请假需提前提交申请，经辅导员审批。",
		Sources: []string{"学生手册 第12条", "请假管理办法"},
		Note:    "条件为 a < b 且 c > d",
	}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("构造测试数据失败: %v", err)
	}

	cleaned := SanitizeKnowledgeContentByType(string(raw), "FAQ")

	var back answerCard
	if err := json.Unmarshal([]byte(cleaned), &back); err != nil {
		t.Fatalf("FAQ 内容清洗后 JSON 损坏: %v\n清洗结果: %s", err, cleaned)
	}
	if back.Answer != original.Answer {
		t.Errorf("answer 字段被破坏\n期望: %q\n实际: %q", original.Answer, back.Answer)
	}
	if len(back.Sources) != len(original.Sources) {
		t.Errorf("sources 数量不符，期望 %d 实际 %d", len(original.Sources), len(back.Sources))
	}
	if back.Note != original.Note {
		t.Errorf("note 字段被破坏\n期望: %q\n实际: %q", original.Note, back.Note)
	}
}

// TestSanitizeKnowledgeContentByTypeStripsNonFAQ 非 FAQ 类型仍需剥离标签
func TestSanitizeKnowledgeContentByTypeStripsNonFAQ(t *testing.T) {
	polluted := `<w:p w:rsidR="00884AF8"/>请假流程说明`
	for _, rt := range []string{"Policy", "Process", "Activity", ""} {
		got := SanitizeKnowledgeContentByType(polluted, rt)
		if strings.Contains(got, "<w:") {
			t.Errorf("类型 %q 未剥离标签，实际: %q", rt, got)
		}
		if !strings.Contains(got, "请假流程说明") {
			t.Errorf("类型 %q 正文丢失，实际: %q", rt, got)
		}
	}
}

// TestSanitizeKnowledgeContentByTypeFAQCaseInsensitive FAQ 类型判断忽略大小写
func TestSanitizeKnowledgeContentByTypeFAQCaseInsensitive(t *testing.T) {
	jsonContent := `{"answer":"<b>加粗</b>"}`
	for _, rt := range []string{"FAQ", "faq", "Faq"} {
		got := SanitizeKnowledgeContentByType(jsonContent, rt)
		if !strings.Contains(got, "<b>") {
			t.Errorf("类型 %q 应跳过标签剥离以保护 JSON，实际: %q", rt, got)
		}
	}
}

// TestSanitizeKnowledgeContentEmpty 空输入不应 panic
func TestSanitizeKnowledgeContentEmpty(t *testing.T) {
	if got := SanitizeKnowledgeContent(""); got != "" {
		t.Errorf("空输入应返回空，实际: %q", got)
	}
	if got := StripMarkup(""); got != "" {
		t.Errorf("空输入应返回空，实际: %q", got)
	}
	if got := NormalizeText(""); got != "" {
		t.Errorf("空输入应返回空，实际: %q", got)
	}
}

// TestContainsMarkup 标记检测的正负样本
func TestContainsMarkup(t *testing.T) {
	if !ContainsMarkup(`<w:p w:rsidR="00884AF8"/>`) {
		t.Error("应检出 OOXML 标记")
	}
	if !ContainsMarkup(`<!-- 注释 -->`) {
		t.Error("应检出注释")
	}
	if ContainsMarkup("纯正文内容，含比较符 a < b") {
		t.Error("纯正文不应被判定含标记")
	}
}
