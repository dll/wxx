package repository

import (
	"strings"
	"testing"
)

func TestEscapeQuery_Chinese(t *testing.T) {
	result := escapeQuery("奖学金怎么申请")
	// 每个汉字后应有 * 并用 OR 连接
	if !strings.Contains(result, "奖*") {
		t.Errorf("中文应含通配符: %s", result)
	}
	if !strings.Contains(result, "学*") {
		t.Errorf("应包含 学*: %s", result)
	}
	if !strings.Contains(result, " OR ") {
		t.Errorf("中文查询应用 OR 连接: %s", result)
	}
	if strings.Contains(result, "\"") {
		t.Errorf("中文查询不应有引号: %s", result)
	}
}

func TestEscapeQuery_English(t *testing.T) {
	result := escapeQuery("GPA")
	if result != "\"GPA\"" {
		t.Errorf("英文应加双引号: 期望 \"GPA\" 得到 %s", result)
	}
}

func TestEscapeQuery_Mixed(t *testing.T) {
	// 混合中英文仍应检测到中文并走中文路径
	result := escapeQuery("GPA奖学金要求")
	if strings.Contains(result, " OR ") {
		t.Log("含中文时走中文路径")
	} else {
		t.Error("含中文字符应走中文路径")
	}
}

func TestEscapeQuery_Empty(t *testing.T) {
	result := escapeQuery("")
	if result != "\"\"" {
		t.Errorf("空查询应返回 \"\": 得到 %s", result)
	}
}

func TestEscapeQuery_WhitespaceOnly(t *testing.T) {
	result := escapeQuery("   ")
	if result != "\"\"" {
		t.Errorf("纯空白应返回 \"\": 得到 %s", result)
	}
}

func TestEscapeQuery_NoChineseNoEnglish(t *testing.T) {
	// 全非 CJK 字符
	result := escapeQuery("12345")
	if !strings.HasPrefix(result, "\"") {
		t.Error("非中文应加双引号")
	}
}

func TestEscapeQuery_RemovesQuotes(t *testing.T) {
	result := escapeQuery("奖\"学金")
	if strings.Contains(result, "\"") {
		t.Error("应移除引号")
	}
	if !strings.Contains(result, "奖*") {
		t.Error("应保留中文字符")
	}
}

func TestEscapeQuery_SingleChar(t *testing.T) {
	result := escapeQuery("奖")
	if result != "奖*" {
		t.Errorf("单个中文字应为 奖*，得到 %s", result)
	}
}

func TestEscapeQuery_AllCJKCharacters(t *testing.T) {
	result := escapeQuery("奖学金")
	// 验证每个中文字后跟 *
	if !strings.HasPrefix(result, "奖*") {
		t.Errorf("应以 奖* 开头: %s", result)
	}
	if !strings.Contains(result, "学*") {
		t.Errorf("应包含 学*: %s", result)
	}
}

func TestEscapeQuery_SpecialNonCJKChars(t *testing.T) {
	// 中文缺测场景：纯数字加中文
	result := escapeQuery("2026奖学金")
	if !strings.Contains(result, " OR ") {
		t.Errorf("含中文应走中文路径: %s", result)
	}
	// 2026 不是中文，不应添加 *
	if strings.Contains(result, "2026*") {
		t.Errorf("数字不应加通配符: %s", result)
	}
}
