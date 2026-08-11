package repository

import (
	"database/sql"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
)

func briefingFixture(n int) *model.AIBriefing {
	category := "ai_teaching"
	keyword := "大模型,AI"
	if n == 2 {
		category = "ai_tool"
		keyword = "智能体,AI工具"
	}
	return &model.AIBriefing{
		Source:      "AI内参",
		Category:    category,
		Topic:       "大模型版本更新简报" + string(rune('0'+n)),
		Summary:     "AI 行业热点摘要" + string(rune('0'+n)),
		Link:        "https://example.com/brief/" + string(rune('0'+n)),
		Keyword:     keyword,
		PublishedAt: "2026-08-10 09:00:00",
		Status:      1,
		CreatedBy:   1,
	}
}

func sourceFixture() *model.AIBriefingSource {
	return &model.AIBriefingSource{
		Name:         "AI内参",
		URL:          "https://example.com/rss",
		Category:     "ai_teaching",
		Enabled:      1,
		FetchEnabled: 1,
		FetchTime:    "08:00",
	}
}

// TestAIBriefingRepo_CRUD 校验资讯 CRUD 与查询过滤
func TestAIBriefingRepo_CRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	clearBriefings(t, db)
	r := NewAIBriefingRepo(db)

	// 新增
	id, err := r.Create(briefingFixture(1))
	if err != nil || id == 0 {
		t.Fatalf("Create 失败 id=%d err=%v", id, err)
	}
	_, _ = r.Create(briefingFixture(2))

	// 列表（全部）
	list, total, err := r.List("", "", "", 1, 10)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("List 应为 2 条，实际 total=%d len=%d", total, len(list))
	}

	// 按分类过滤
	_, totalAI, _ := r.List("", "ai_tool", "", 1, 10)
	if totalAI != 1 {
		t.Fatalf("ai_tool 分类应 1 条，实际 %d", totalAI)
	}

	// 关键词搜索
	_, totalQ, _ := r.List("", "", "智能体", 1, 10)
	if totalQ != 1 {
		t.Fatalf("关键词'智能体'应 1 条，实际 %d", totalQ)
	}

	// 更新
	item, _ := r.Get(id)
	item.Topic = "更新后的主题"
	if err := r.Update(item); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	got, _ := r.Get(id)
	if got.Topic != "更新后的主题" {
		t.Fatalf("Update 未生效: %s", got.Topic)
	}

	// 上下架
	if err := r.UpdateStatus(id, 0); err != nil {
		t.Fatalf("UpdateStatus 失败: %v", err)
	}
	// 用户端列表只显示上架
	visible, _ := r.ListUserVisible("", "", 10)
	if len(visible) != 1 {
		t.Fatalf("上架资讯应 1 条，实际 %d", len(visible))
	}

	// 批量删除
	n, err := r.DeleteMany([]int64{id})
	if err != nil || n != 1 {
		t.Fatalf("DeleteMany 失败 n=%d err=%v", n, err)
	}
	// 清空
	n, _ = r.ClearAll()
	if n != 1 {
		t.Fatalf("ClearAll 应删除 1 条，实际 %d", n)
	}
}

// TestAIBriefingRepo_Stats 校验汇总统计
func TestAIBriefingRepo_Stats(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	clearBriefings(t, db)
	r := NewAIBriefingRepo(db)

	_, _ = r.Create(briefingFixture(1))
	id2, _ := r.Create(briefingFixture(2))
	// 其中一条下架
	b, _ := r.Get(id2)
	if b == nil {
		t.Fatal("id2 查询为空")
	}
	_ = r.UpdateStatus(b.ID, 0)

	s, err := r.Stats()
	if err != nil {
		t.Fatalf("Stats 失败: %v", err)
	}
	if s.Total != 2 || s.Published != 1 || s.Draft != 1 {
		t.Fatalf("统计不符: %+v", s)
	}
	if s.ByCategory["ai_tool"] != 1 || s.ByCategory["ai_teaching"] != 1 {
		t.Fatalf("分类统计不符: %+v", s.ByCategory)
	}
	if s.BySource["AI内参"] != 2 {
		t.Fatalf("来源统计不符: %+v", s.BySource)
	}
}

// TestAIBriefingRepo_Sources 校验来源 CRUD 与抓取列表
func TestAIBriefingRepo_Sources(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	r := NewAIBriefingRepo(db)

	id, err := r.CreateSource(sourceFixture())
	if err != nil || id == 0 {
		t.Fatalf("CreateSource 失败 id=%d err=%v", id, err)
	}

	list, _ := r.ListSources()
	if len(list) != 1 {
		t.Fatalf("来源应 1 条，实际 %d", len(list))
	}

	// 更新
	src := list[0]
	src.FetchTime = "09:30"
	if err := r.UpdateSource(src); err != nil {
		t.Fatalf("UpdateSource 失败: %v", err)
	}
	// 可抓取来源
	fetchList, _ := r.ListEnabledFetchSources()
	if len(fetchList) != 1 {
		t.Fatalf("可抓取来源应 1 条，实际 %d", len(fetchList))
	}
	if fetchList[0].FetchTime != "09:30" {
		t.Fatalf("FetchTime 更新未生效: %s", fetchList[0].FetchTime)
	}

	if err := r.DeleteSource(id); err != nil {
		t.Fatalf("DeleteSource 失败: %v", err)
	}
}

// clearBriefings 清空 seed 资讯（067 迁移默认插入 20 条，测试需隔离）
func clearBriefings(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("DELETE FROM ai_briefings"); err != nil {
		t.Fatalf("清空 ai_briefings 失败: %v", err)
	}
}
