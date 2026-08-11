package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// pollutedOOXML 模拟 docx 解析泄漏的 OOXML 标记
const pollutedOOXML = `学 生 手 册 <w:p w:rsidR="00884AF8" w:rsidRDefault="00884AF8"/>` +
	`</w:txbxContent><wps:bodyPr rot="0" wrap="square"/>请假需提前提交申请`

// testAdminCtx 返回系统管理员上下文，用于清洗/导入类测试（scope 校验恒通过）
func testAdminCtx() *model.UserContext {
	return &model.UserContext{
		Username:   "admin",
		Role:       "sys_admin",
		Status:     "active",
		OwnerScope: "school",
		OwnerID:    "all",
	}
}

// TestKBServiceCreateSanitizesContent 入库前必须清洗，避免污染内容进入 FTS 索引
func TestKBServiceCreateSanitizesContent(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	created, err := svc.Create(context.Background(), &model.KBCreateRequest{
		ResourceType: "Policy",
		OwnerScope:   "college",
		OwnerID:      "cs",
		Title:        pollutedOOXML,
		Summary:      pollutedOOXML,
		Content:      pollutedOOXML,
	}, testAdminCtx())
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if created == nil {
		t.Fatal("创建结果为空")
	}

	for field, val := range map[string]string{
		"title":   created.Title,
		"summary": created.Summary,
		"content": created.Content,
	} {
		for _, bad := range []string{"<w:", "</w:", "w:rsidR", "wps:bodyPr", "rot="} {
			if strings.Contains(val, bad) {
				t.Errorf("%s 字段残留标记 %q：%q", field, bad, val)
			}
		}
		if !strings.Contains(val, "请假需提前提交申请") {
			t.Errorf("%s 字段正文丢失：%q", field, val)
		}
	}
}

// TestKBServiceUpdateSanitizesContent 更新路径同样需要清洗
func TestKBServiceUpdateSanitizesContent(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	created, err := svc.Create(context.Background(), &model.KBCreateRequest{
		ResourceType: "Policy",
		OwnerScope:   "college",
		OwnerID:      "cs",
		Title:        "初始标题",
		Content:      "初始正文",
	}, testAdminCtx())
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}

	updated, err := svc.Update(context.Background(), created.ResourceID, &model.KBUpdateRequest{
		Title:   pollutedOOXML,
		Content: pollutedOOXML,
	}, testAdminCtx())
	if err != nil {
		t.Fatalf("更新失败: %v", err)
	}

	if strings.Contains(updated.Title, "<w:") || strings.Contains(updated.Content, "<w:") {
		t.Errorf("更新后残留标记\ntitle=%q\ncontent=%q", updated.Title, updated.Content)
	}
}

// TestKBServiceCreateFAQPreservesJSON FAQ 的 content 是 AnswerCard JSON，清洗不得破坏结构
func TestKBServiceCreateFAQPreservesJSON(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	card := map[string]interface{}{
		"answer":  "请假需提前提交申请，经辅导员审批。",
		"sources": []string{"学生手册 第12条"},
		"note":    "条件 a < b",
	}
	raw, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("构造 JSON 失败: %v", err)
	}

	created, err := svc.Create(context.Background(), &model.KBCreateRequest{
		ResourceType: "FAQ",
		OwnerScope:   "college",
		OwnerID:      "cs",
		Title:        "请假怎么办",
		Content:      string(raw),
	}, testAdminCtx())
	if err != nil {
		t.Fatalf("创建 FAQ 失败: %v", err)
	}

	var back map[string]interface{}
	if err := json.Unmarshal([]byte(created.Content), &back); err != nil {
		t.Fatalf("FAQ content 清洗后 JSON 损坏: %v\n实际内容: %s", err, created.Content)
	}
	if back["answer"] != card["answer"] {
		t.Errorf("answer 字段被破坏：%v", back["answer"])
	}
}

// TestKBRepoUpsertSanitizesContent NDJSON 导入与 FAQ 缓存走 Upsert，也需清洗
func TestKBRepoUpsertSanitizesContent(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	repo := repository.NewKBRepo(db)

	kb := &model.KBResource{
		ResourceID:   "test-upsert-sanitize",
		ResourceType: "Policy",
		OwnerScope:   "college",
		OwnerID:      "cs",
		Version:      "1.0",
		Status:       "published",
		Title:        pollutedOOXML,
		Summary:      pollutedOOXML,
		Content:      pollutedOOXML,
		UpdatedBy:    "tester",
	}

	if _, _, err := repo.Upsert(kb); err != nil {
		t.Fatalf("Upsert 失败: %v", err)
	}

	got, err := repo.GetByResourceID("test-upsert-sanitize")
	if err != nil {
		t.Fatalf("回查失败: %v", err)
	}
	if got == nil {
		t.Fatal("回查结果为空")
	}

	for field, val := range map[string]string{
		"title":   got.Title,
		"summary": got.Summary,
		"content": got.Content,
	} {
		if strings.Contains(val, "<w:") || strings.Contains(val, "wps:") {
			t.Errorf("%s 字段残留标记：%q", field, val)
		}
	}
}
