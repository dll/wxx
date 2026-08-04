package service

import (
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/stretchr/testify/require"
)

func TestExportAnswerFormats(t *testing.T) {
	svc := NewExportService()
	card := &model.AnswerCard{
		Conclusion: "这是一段测试回答。",
		Sources: []model.Source{
			{Title: "测试来源", Version: "v1", SourceLink: "https://example.edu"},
		},
		TraceID:  "trace-export",
		Fallback: false,
	}

	cases := []struct {
		format ExportFormat
		magic  string
	}{
		{ExportJSON, "{"},
		{ExportMD, "#"},
		{ExportDOCX, "PK"},
		{ExportXLSX, "PK"},
		{ExportICS, "BEGIN:VCALENDAR"},
		{ExportPDF, "%PDF"},
	}
	for _, tc := range cases {
		data, mime, err := svc.ExportAnswer(card, tc.format, true)
		require.NoError(t, err, "format %s", tc.format)
		require.NotEmpty(t, data)
		require.NotEmpty(t, mime)
		require.True(t, strings.HasPrefix(string(data), tc.magic), "format %s magic", tc.format)
	}
}

func TestExportPNGWithCJKFont(t *testing.T) {
	svc := NewExportService()
	if svc.resolveCJKFontPath() == "" {
		t.Skip("未找到中文字体，跳过 PNG 导出测试")
	}
	card := &model.AnswerCard{Conclusion: "中文导出测试"}
	data, mime, err := svc.ExportAnswer(card, ExportPNG, false)
	require.NoError(t, err)
	require.Equal(t, "image/png", mime)
	require.True(t, strings.HasPrefix(string(data), "\x89PNG"))
}
