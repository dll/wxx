package repository

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/dll/wxx/server/internal/testutil"
)

// setupTwinAggTestDB 构造含 student_profile_snapshot 表（NewTestDB 已含 043 迁移）的 TwinRepo。
// 返回 repo 与 db（插入快照需先建满足外键的 users 记录）。
func setupTwinAggTestDB(t *testing.T) (*TwinRepo, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })
	return NewTwinRepo(db), db
}

// createStudent 在 users 表建一个真实学生（满足外键），返回其 id。
func createStudent(t *testing.T, db *sql.DB, seq int) int64 {
	t.Helper()
	uname := fmt.Sprintf("aggstu%d", seq)
	res, err := db.Exec(`INSERT OR IGNORE INTO users (username, role, display_name) VALUES (?, 'student', ?)`, uname, "测试学生"+fmt.Sprint(seq))
	if err != nil {
		t.Fatalf("建学生失败: %v", err)
	}
	id, _ := res.LastInsertId()
	if id <= 0 {
		// OR IGNORE 命中已存在：回查
		_ = db.QueryRow(`SELECT id FROM users WHERE username=?`, uname).Scan(&id)
	}
	return id
}

// insertSnapshot 直接插入一条快照（约定全部五维分一致，便于断言 AVG）。
func insertSnapshot(t *testing.T, db *sql.DB, repo *TwinRepo, ownerID, ownerScope, major, class string, dim float64) {
	t.Helper()
	uid := createStudent(t, db, uidCounter())
	if err := repo.UpsertSnapshot(&TwinSnapshot{
		UserID: uid, OwnerScope: ownerScope, OwnerID: ownerID,
		College: ownerID, Major: major, ClassName: class,
		AcademicScore: dim, AbilityScore: dim, IdeologicalScore: dim,
		EmotionalScore: dim, SocialScore: dim,
		AIInterpretation: "", GapAnalysis: "[]", StageAdvice: "[]",
	}); err != nil {
		t.Fatalf("写入快照失败: %v", err)
	}
}

var aggUIDSeed = 900000

func uidCounter() int {
	aggUIDSeed++
	return aggUIDSeed
}

// TestAggregateSnapshotsByScope_ZeroSamples 覆盖 0 样本：整体 Count=0、均值=0（调用方按 0 样本走 not_available）。
func TestAggregateSnapshotsByScope_ZeroSamples(t *testing.T) {
	repo, db := setupTwinAggTestDB(t)
	_ = db
	agg, err := repo.AggregateSnapshotsByScope("college", "cs", "", "", "")
	if err != nil {
		t.Fatalf("聚合失败: %v", err)
	}
	if agg.Overall.Count != 0 {
		t.Errorf("0 样本应 Count=0，得到 %d", agg.Overall.Count)
	}
	if agg.Overall.Academic != 0 || agg.Overall.Emotional != 0 {
		t.Errorf("空库五维均值应为 0，得到 %+v", agg.Overall)
	}
}

// TestAggregateSnapshotsByScope_Over500 覆盖 >500 学生去限：插入 600 条，AVG 与 COUNT 完整无漏样本。
func TestAggregateSnapshotsByScope_Over500(t *testing.T) {
	repo, db := setupTwinAggTestDB(t)
	const n = 600
	for i := 0; i < n; i++ {
		insertSnapshot(t, db, repo, "cs", "college", "cs-major", "CS2501", float64(i%101))
	}
	agg, err := repo.AggregateSnapshotsByScope("college", "cs", "", "", "")
	if err != nil {
		t.Fatalf("聚合失败: %v", err)
	}
	if agg.Overall.Count != n {
		t.Fatalf(">500 学生应全部计入（去 limit 上限），期望 %d 实际 %d", n, agg.Overall.Count)
	}
	// dim 取值 i%101（0..100），理论均值 = Σ(i%101)/600
	want := 0.0
	for i := 0; i < n; i++ {
		want += float64(i % 101)
	}
	want /= n
	if diff := abs(want - agg.Overall.Academic); diff > 1e-6 {
		t.Errorf("学业均值偏差过大: want≈%.3f got %.3f", want, agg.Overall.Academic)
	}
}

// TestAggregateSnapshotsByScope_ScopedByOwner 覆盖跨院隔离：不同 ownerID 互不可见。
func TestAggregateSnapshotsByScope_ScopedByOwner(t *testing.T) {
	repo, db := setupTwinAggTestDB(t)
	insertSnapshot(t, db, repo, "cs", "college", "cs-major", "CS2501", 90)
	insertSnapshot(t, db, repo, "cs", "college", "cs-major", "CS2501", 70)
	// 其他学院的快照（不应被 cs 读到）
	insertSnapshot(t, db, repo, "math", "college", "math-major", "MA2501", 40)

	aggCS, err := repo.AggregateSnapshotsByScope("college", "cs", "", "", "")
	if err != nil {
		t.Fatalf("聚合 cs 失败: %v", err)
	}
	if aggCS.Overall.Count != 2 {
		t.Errorf("cs 应只统计本院 2 条（跨院隔离），得到 %d", aggCS.Overall.Count)
	}
	// cs 均值 = (90+70)/2 = 80
	if got := aggCS.Overall.Academic; abs(got-80) > 1e-6 {
		t.Errorf("cs 学业均值应 80，得到 %.2f", got)
	}

	aggMath, _ := repo.AggregateSnapshotsByScope("college", "math", "", "", "")
	if aggMath.Overall.Count != 1 || aggMath.Overall.Social != 40 {
		t.Errorf("math 应独立统计，得到 %+v", aggMath.Overall)
	}
}

// TestAggregateSnapshotsByScope_GroupByMajor 覆盖按 major 分组聚合。
func TestAggregateSnapshotsByScope_GroupByMajor(t *testing.T) {
	repo, db := setupTwinAggTestDB(t)
	insertSnapshot(t, db, repo, "cs", "college", "软件工程", "SE2501", 80)
	insertSnapshot(t, db, repo, "cs", "college", "软件工程", "SE2502", 60)
	insertSnapshot(t, db, repo, "cs", "college", "计算机科学与技术", "CS2501", 100)

	agg, err := repo.AggregateSnapshotsByScope("college", "cs", "", "", "major")
	if err != nil {
		t.Fatalf("聚合失败: %v", err)
	}
	if len(agg.ByGroup) != 2 {
		t.Fatalf("应有 2 个专业分组，得到 %d", len(agg.ByGroup))
	}
	se := agg.ByGroup["软件工程"]
	if se.Count != 2 || abs(se.Academic-70) > 1e-6 {
		t.Errorf("软件工程均值应 70（(80+60)/2），得到 %+v", se)
	}
	cst := agg.ByGroup["计算机科学与技术"]
	if cst.Count != 1 || cst.Academic != 100 {
		t.Errorf("计算机专业异常: %+v", cst)
	}
}

// 辅助断言工具
func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
