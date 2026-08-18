package service

import (
	"context"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

func TestKnowledgeGovernance_Audit_Runs(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	svc := NewKnowledgeGovernanceService(repository.NewKBRepo(db))
	res := svc.GovernanceAudit(context.Background(), "", "", false, 200)

	if res == nil {
		t.Fatal("GovernanceAudit 返回 nil")
	}
	if res.Summary.Scanned < 1 {
		t.Errorf("应至少扫描到测试资源，得到 scanned=%d", res.Summary.Scanned)
	}
	if res.DataSource != "real" {
		t.Errorf("数据源应为 real，得到 %s", res.DataSource)
	}
}

func TestKnowledgeGovernance_Deterministic_FindsMissingField(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	repo := repository.NewKBRepo(db)
	svc := NewKnowledgeGovernanceService(repo)

	// 插入一条缺标题的资源，确定性检查应能发现
	_, err := repo.Create(&model.KBResource{
		ResourceID:   "kg-test-missing-title",
		ResourceType: "Policy",
		OwnerScope:   "school",
		Status:       "published",
		Title:        "",
		Summary:      "测试摘要",
		Content:      "测试正文内容足够长以便通过过短检测。",
		Tags:         "[\"测试\"]",
	})
	if err != nil {
		t.Fatalf("插入测试资源失败: %v", err)
	}

	res := svc.GovernanceAudit(context.Background(), "", "", false, 200)
	found := false
	for _, it := range res.Issues {
		if it.ResourceID == "kg-test-missing-title" && it.Category == "missing_field" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("应发现缺标题资源的 missing_field 问题，实际 issues=%d", len(res.Issues))
	}
}

func TestKnowledgeGovernance_Audit_NoLLMNoDB(t *testing.T) {
	// 无 DB 时不应 panic，应诚实返回 0 扫描 + 提示
	svc := NewKnowledgeGovernanceService(nil)
	res := svc.GovernanceAudit(context.Background(), "", "", true, 10)
	if res == nil {
		t.Fatal("GovernanceAudit 返回 nil")
	}
	if res.Summary.Scanned != 0 {
		t.Errorf("无 DB 时应 scanned=0，得到 %d", res.Summary.Scanned)
	}
	// 无资源时也应返回非空 issues（至少一条审计提示）
	if len(res.Issues) == 0 {
		t.Errorf("无资源应返回审计提示")
	}
}
