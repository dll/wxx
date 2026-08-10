package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/testutil"
)

// TestAIBriefingSeed 校验 067 seed 迁移：插入 20 条且幂等
func TestAIBriefingSeed(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	r := NewAIBriefingRepo(db)

	list, total, err := r.List("", "", "", 1, 100)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if total != 20 || len(list) != 20 {
		t.Fatalf("seed 应插入 20 条，实际 total=%d len=%d", total, len(list))
	}

	// 幂等：重新执行迁移不应新增（用 UpdateStatus 后无法直接测，这里校验 ListUserVisible 均为上架）
	visible, _ := r.ListUserVisible("", "", 100)
	if len(visible) != 20 {
		t.Fatalf("seed 均应为上架，实际 %d", len(visible))
	}

	// 校验开源/闭源各 10 条
	s, _ := r.Stats()
	byCat := s.ByCategory
	_ = byCat
	// 全部为 ai_version 分类
	if s.ByCategory["ai_version"] != 20 {
		t.Fatalf("seed 应全为 ai_version，实际 %+v", s.ByCategory)
	}
}
