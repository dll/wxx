package util

import (
	"bytes"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// DecodeToUTF8 将文档字节流安全地转换为 UTF-8 字符串。
//
// 背景：上传的 TXT/MD/CSV 常为 GBK/GB18030 编码（中文 Windows 默认），
// 直接 string(data) 会产生 mojibake（乱码）。assessDocQuality 只能检出乱码、无法修复。
//
// 策略：
//   - 已是合法 UTF-8 → 原样直通（UTF-8 中文与绝大多数现代文本）；
//   - 非 UTF-8 → 尝试 GB18030（GBK 超集）解码，解码后不含替换符 U+FFFD 则采用；
//   - 解码仍含替换符（如 UTF-16/二进制）→ 回退原始字节，由质量门槛判乱码。
func DecodeToUTF8(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if utf8.Valid(data) {
		return string(data)
	}

	out, _, err := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), data)
	if err == nil && bytes.IndexRune(out, '\uFFFD') < 0 {
		return string(out)
	}
	return string(data)
}
