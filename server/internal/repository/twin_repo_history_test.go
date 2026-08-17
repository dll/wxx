package repository

// P1-2 A 路径：快照历史留痕（snapshot_history）repo 层单测（2026-08-17）。
// NewTestDB 已含 091_student_profile_snapshot_history 迁移（testutil/db.go），故此处无需手动建表。

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/dll/wxx/server/internal/testutil"
)

func setupTwinHistoryDB(t *testing.T) (*TwinRepo, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })
	return NewTwinRepo(db), db
}

// insertHistoryDirect 直接向 snapshot_history 插入一条历史（绕过 repo 幂等，便于构造测试场景）。
func insertHistoryDirect(t *testing.T, db *sql.DB, uid int64, ownerScope, ownerID, day string, base, bonus float64) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO snapshot_history
		(user_id, owner_scope, owner_id, college, major, class_name,
		 academic_score, ability_score, ideological_score, emotional_score, social_score, computed_at)
		VALUES (?, ?, ?, ?, '', '', ?, ?, ?, ?, ?, ?)`,
		uid, ownerScope, ownerID, ownerID, base+bonus, base+bonus, base+bonus, base+bonus, base+bonus, day)
	if err != nil {
		t.Fatalf("插入快照历史失败: %v", err)
	}
}

// TestInsertSnapshotHistory_DedupByDay 幂等去抖：同一学生同一天多次插入只保留一条。
func TestInsertSnapshotHistory_DedupByDay(t *testing.T) {
	repo, db := setupTwinHistoryDB(t)
	uid := createStudent(t, db, uidCounter())
	snap := &TwinSnapshot{
		UserID: uid, OwnerScope: "college", OwnerID: "cs", College: "cs",
		AcademicScore: 60, AbilityScore: 60, IdeologicalScore: 60,
		EmotionalScore: 60, SocialScore: 60, ComputedAt: "2026-08-01T10:00:00Z",
	}
	// 同一天插入多次（computed_at 不同时刻，但均归一化到 08-01）
	for i := 0; i < 3; i++ {
		sap := *snap
		sap.ComputedAt = fmt.Sprintf("2026-08-01T12:0%d:00Z", i)
		if err := repo.InsertSnapshotHistory(&sap); err != nil {
			t.Fatalf("第 %d 次插入历史失败: %v", i, err)
		}
	}

	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM snapshot_history WHERE user_id=?`, uid).Scan(&cnt); err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("同一天多次插入应只留 1 条历史（去抖），实际 %d 条", cnt)
	}
	// 保留首条分值 60
	var ac float64
	_ = db.QueryRow(`SELECT academic_score FROM snapshot_history WHERE user_id=?`, uid).Scan(&ac)
	if ac != 60 {
		t.Fatalf("保留首条分数应为 60，得到 %v", ac)
	}
}

// TestGetGrowthTrend_Empty 空历史 → HasData=false，各维 delta=0（不造数）。
func TestGetGrowthTrend_Empty(t *testing.T) {
	repo, db := setupTwinHistoryDB(t)
	_ = db
	gt, err := repo.GetGrowthTrend("cs", 4)
	if err != nil {
		t.Fatalf("空历史查询失败: %v", err)
	}
	if gt.HasData {
		t.Fatalf("空历史不应 HasData")
	}
	if gt.SampleCount != 0 {
		t.Fatalf("空历史 SampleCount 应为 0，得到 %d", gt.SampleCount)
	}
	if gt.Academic != 0 || gt.Social != 0 {
		t.Fatalf("空历史五维 delta 应为 0，得到 %+v", gt)
	}
}

// TestGetGrowthTrend_SingleEnd 仅单端（一个采样日）→ 不算 HasData（缺两端对比，诚实）。
func TestGetGrowthTrend_SingleEnd(t *testing.T) {
	repo, db := setupTwinHistoryDB(t)
	uid := createStudent(t, db, uidCounter())
	insertHistoryDirect(t, db, uid, "college", "cs", "2026-08-01 00:00:00", 50, 0)
	gt, err := repo.GetGrowthTrend("cs", 4)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if gt.HasData || gt.SampleCount != 0 {
		t.Fatalf("仅单端不应 HasData，得到 sample=%d hasData=%v", gt.SampleCount, gt.HasData)
	}
}

// TestGetGrowthTrend_Delta 构造双端历史 → 正确差分：学生 A 各维 +5，B 各维 -2 → 均值各维 1.5。
func TestGetGrowthTrend_Delta(t *testing.T) {
	repo, db := setupTwinHistoryDB(t)
	uidA := createStudent(t, db, uidCounter())
	uidB := createStudent(t, db, uidCounter())
	oldDay := time.Now().AddDate(0, 0, -21).Format("2006-01-02") + " 00:00:00"
	newDay := time.Now().Format("2006-01-02") + " 00:00:00"

	insertHistoryDirect(t, db, uidA, "college", "cs", oldDay, 50, 0)
	insertHistoryDirect(t, db, uidA, "college", "cs", newDay, 50, 5)
	insertHistoryDirect(t, db, uidB, "college", "cs", oldDay, 40, 0)
	insertHistoryDirect(t, db, uidB, "college", "cs", newDay, 40, -2)

	gt, err := repo.GetGrowthTrend("cs", 4)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if !gt.HasData || gt.SampleCount != 2 {
		t.Fatalf("应有 2 名双端学生，得到 sample=%d hasData=%v", gt.SampleCount, gt.HasData)
	}
	for _, dim := range []float64{gt.Academic, gt.Ability, gt.Ideological, gt.Emotional, gt.Social} {
		if abs(dim-1.5) > 1e-6 {
			t.Fatalf("各维平均变化应 1.5，得到 %v", dim)
		}
	}
}

// TestGetGrowthTrend_WindowCutoff 超出窗口（>N 周）的早期历史不计入，避免跨窗口误导。
func TestGetGrowthTrend_WindowCutoff(t *testing.T) {
	repo, db := setupTwinHistoryDB(t)
	uid := createStudent(t, db, uidCounter())
	// 10 周前（超出 4 周窗口）与 今天
	tooOldDay := time.Now().AddDate(0, 0, -70).Format("2006-01-02") + " 00:00:00"
	newDay := time.Now().Format("2006-01-02") + " 00:00:00"
	insertHistoryDirect(t, db, uid, "college", "cs", tooOldDay, 50, 0)
	insertHistoryDirect(t, db, uid, "college", "cs", newDay, 50, 5)
	gt, err := repo.GetGrowthTrend("cs", 4)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	// 窗口内只有 1 个采样日（今天），无双端 → 不算 HasData
	if gt.HasData || gt.SampleCount != 0 {
		t.Fatalf("窗口外早期历史不应计入双端，得到 sample=%d hasData=%v", gt.SampleCount, gt.HasData)
	}
}

// TestGetGrowthTrend_CrossOwner 越权隔离：只读本院，跨院历史不入计。
func TestGetGrowthTrend_CrossOwner(t *testing.T) {
	repo, db := setupTwinHistoryDB(t)
	uidCS := createStudent(t, db, uidCounter())
	uidZD := createStudent(t, db, uidCounter())
	oldDay := time.Now().AddDate(0, 0, -21).Format("2006-01-02") + " 00:00:00"
	newDay := time.Now().Format("2006-01-02") + " 00:00:00"

	insertHistoryDirect(t, db, uidCS, "college", "cs", oldDay, 50, 0)
	insertHistoryDirect(t, db, uidCS, "college", "cs", newDay, 50, 5)
	// 外院生双端（不应被 cs 读到）
	insertHistoryDirect(t, db, uidZD, "college", "zd", oldDay, 30, 0)
	insertHistoryDirect(t, db, uidZD, "college", "zd", newDay, 30, 30)

	gt, err := repo.GetGrowthTrend("cs", 4)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if gt.SampleCount != 1 || gt.Academic != 5 {
		t.Fatalf("cs 应只统计 1 名本院双端学生且学业变化 5，得到 sample=%d academic=%v", gt.SampleCount, gt.Academic)
	}
}
