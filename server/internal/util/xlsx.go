package util

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type xlsxSST struct {
	Items []xlsxStringItem `xml:"si"`
}

type xlsxStringItem struct {
	Text string    `xml:"t"`
	Runs []xlsxRun `xml:"r"`
}

type xlsxRun struct {
	Text string `xml:"t"`
}

func (s xlsxStringItem) value() string {
	if s.Text != "" {
		return s.Text
	}
	var b strings.Builder
	for _, run := range s.Runs {
		b.WriteString(run.Text)
	}
	return b.String()
}

type xlsxWorksheet struct {
	SheetData xlsxSheetData `xml:"sheetData"`
}

type xlsxSheetData struct {
	Rows []xlsxRow `xml:"row"`
}

type xlsxRow struct {
	Cells []xlsxCell `xml:"c"`
}

type xlsxCell struct {
	Ref          string         `xml:"r,attr"`
	Type         string         `xml:"t,attr"`
	Value        string         `xml:"v"`
	InlineString xlsxStringItem `xml:"is"`
}

// XLSXRow 表示一行 Excel 数据，键为列字母（A、B、C……）。
type XLSXRow map[string]string

// ParseXLSX 解析第一个工作表，支持共享字符串、富文本和 inlineStr。
func ParseXLSX(r io.ReaderAt, size int64) ([]XLSXRow, error) {
	if size <= 0 {
		return nil, fmt.Errorf("xlsx 文件为空")
	}
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("无法解压 xlsx: %w", err)
	}

	sharedStrings, err := readSharedStrings(zr)
	if err != nil {
		return nil, err
	}

	sheetFile := firstWorksheet(zr)
	if sheetFile == nil {
		return nil, fmt.Errorf("未找到工作表")
	}
	rc, err := sheetFile.Open()
	if err != nil {
		return nil, fmt.Errorf("打开 %s 失败: %w", sheetFile.Name, err)
	}
	defer rc.Close()

	var worksheet xlsxWorksheet
	if err := xml.NewDecoder(rc).Decode(&worksheet); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", sheetFile.Name, err)
	}

	rows := make([]XLSXRow, 0, len(worksheet.SheetData.Rows))
	for _, sourceRow := range worksheet.SheetData.Rows {
		row := make(XLSXRow, len(sourceRow.Cells))
		for index, cell := range sourceRow.Cells {
			column := cellRefToColumn(cell.Ref)
			if column == "" {
				column = columnName(index + 1)
			}
			value, err := xlsxCellValue(cell, sharedStrings)
			if err != nil {
				return nil, fmt.Errorf("单元格 %s 解析失败: %w", cell.Ref, err)
			}
			row[column] = value
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func readSharedStrings(zr *zip.Reader) ([]string, error) {
	for _, file := range zr.File {
		if filepath.ToSlash(file.Name) != "xl/sharedStrings.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("打开 sharedStrings.xml 失败: %w", err)
		}
		defer rc.Close()

		var table xlsxSST
		if err := xml.NewDecoder(rc).Decode(&table); err != nil {
			return nil, fmt.Errorf("解析 sharedStrings.xml 失败: %w", err)
		}
		values := make([]string, len(table.Items))
		for i, item := range table.Items {
			values[i] = item.value()
		}
		return values, nil
	}
	return nil, nil
}

func firstWorksheet(zr *zip.Reader) *zip.File {
	var sheets []*zip.File
	for _, file := range zr.File {
		name := filepath.ToSlash(file.Name)
		if strings.HasPrefix(name, "xl/worksheets/sheet") && strings.HasSuffix(name, ".xml") {
			sheets = append(sheets, file)
		}
	}
	sort.Slice(sheets, func(i, j int) bool { return sheets[i].Name < sheets[j].Name })
	if len(sheets) == 0 {
		return nil
	}
	return sheets[0]
}

func xlsxCellValue(cell xlsxCell, sharedStrings []string) (string, error) {
	switch cell.Type {
	case "s":
		index, err := strconv.Atoi(cell.Value)
		if err != nil || index < 0 || index >= len(sharedStrings) {
			return "", fmt.Errorf("共享字符串索引无效: %q", cell.Value)
		}
		return sharedStrings[index], nil
	case "inlineStr":
		return cell.InlineString.value(), nil
	case "b":
		if cell.Value == "1" {
			return "TRUE", nil
		}
		return "FALSE", nil
	default:
		return cell.Value, nil
	}
}

func cellRefToColumn(ref string) string {
	for i, ch := range ref {
		if ch >= '0' && ch <= '9' {
			return strings.ToUpper(ref[:i])
		}
	}
	return strings.ToUpper(ref)
}

func columnName(index int) string {
	if index <= 0 {
		return ""
	}
	var result string
	for index > 0 {
		index--
		result = string(rune('A'+index%26)) + result
		index /= 26
	}
	return result
}
