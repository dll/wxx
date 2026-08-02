package util

import (
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestDecodeToUTF8_UTF8Passthrough(t *testing.T) {
	raw := "奖学金评选办法，每人每年8000元。\n第二条 适用范围"
	if got := DecodeToUTF8([]byte(raw)); got != raw {
		t.Errorf("UTF-8 应原样直通，得到 %q", got)
	}
}

func TestDecodeToUTF8_Empty(t *testing.T) {
	if got := DecodeToUTF8(nil); got != "" {
		t.Errorf("空输入应返回空串，得到 %q", got)
	}
}

func TestDecodeToUTF8_GBK(t *testing.T) {
	raw := "滁州学院新生入学须知：报到时请携带录取通知书、身份证与一寸照片。"
	gbk, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(raw))
	if err != nil {
		t.Fatalf("GBK 编码失败: %v", err)
	}
	if utf8.Valid(gbk) {
		t.Fatal("样本应验证 GBK 编码字节并非合法 UTF-8，否则测试失去意义")
	}
	got := DecodeToUTF8(gbk)
	if got != raw {
		t.Errorf("GBK 解码失败：\n got=%q\nwant=%q", got, raw)
	}
}

func TestDecodeToUTF8_GB18030(t *testing.T) {
	// GB18030 编码（GBK 超集），包含简体中文与全角标点
	raw := "关于做好2026届毕业生离校工作的通知（滁州学院教务处）"
	encoded, err := simplifiedchinese.GB18030.NewEncoder().Bytes([]byte(raw))
	if err != nil {
		t.Fatalf("GB18030 编码失败: %v", err)
	}
	got := DecodeToUTF8(encoded)
	if got != raw {
		t.Errorf("GB18030 解码失败：\n got=%q\nwant=%q", got, raw)
	}
}

func TestDecodeToUTF8_BinaryFallback(t *testing.T) {
	// 非文本二进制（含大量 NUL/控制字节）：解码应回退原始字节而非返回乱码
	data := []byte{0x00, 0x01, 0x02, 0x89, 0x50, 0x4E, 0x47}
	got := DecodeToUTF8(data)
	if !strings.Contains(got, "\x00") {
		t.Errorf("二进制输入应回退原始字节，得到 %q", got)
	}
}
