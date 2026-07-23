package util

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestParseXLSX_SharedStrings(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	files := map[string]string{
		"xl/sharedStrings.xml": `<?xml version="1.0" encoding="UTF-8"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <si><t>学号</t></si><si><t>姓名</t></si><si><t>2023211981</t></si>
  <si><r><t>张</t></r><r><t>明远</t></r></si>
</sst>`,
		"xl/worksheets/sheet1.xml": `<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row>
    <row><c r="A2" t="s"><v>2</v></c><c r="B2" t="s"><v>3</v></c></row>
  </sheetData>
</worksheet>`,
	}
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("创建测试 xlsx 条目失败: %v", err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("写入测试 xlsx 条目失败: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭测试 xlsx 失败: %v", err)
	}

	reader := bytes.NewReader(buffer.Bytes())
	rows, err := ParseXLSX(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("解析 xlsx 失败: %v", err)
	}
	if len(rows) != 2 || rows[1]["A"] != "2023211981" || rows[1]["B"] != "张明远" {
		t.Fatalf("解析结果不符合预期: %#v", rows)
	}
}
