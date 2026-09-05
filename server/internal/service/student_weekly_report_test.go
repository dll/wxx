package service

import (
	"context"
	"testing"
)

func TestStudentServiceWeeklyReportFallbackContract(t *testing.T) {
	report := (&StudentService{}).GenerateWeeklyReport(context.Background(), 42)
	if report == nil || report.Week == "" || report.DataSource != "reference" {
		t.Fatalf("weekly report fallback contract failed: %#v", report)
	}
	if len(report.Highlights) == 0 || len(report.NextWeekGoals) == 0 || len(report.TimeDistribution) == 0 {
		t.Fatalf("weekly report should contain actionable sections: %#v", report)
	}
}
