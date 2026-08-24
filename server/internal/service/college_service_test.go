package service

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// setupCollegeTwinService 构造仅含 TwinRepo 的 CollegeService（其余依赖置 nil）。
// 返回 service、TwinRepo、db（建满足外键的学生用）。
func setupCollegeTwinService(t *testing.T) (*CollegeService, *repository.TwinRepo, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })
	twinRepo := repository.NewTwinRepo(db)
	svc := NewCollegeService(nil, nil, twinRepo, nil)
	return svc, twinRepo, db
}

func insertSnap(t *testing.T, db *sql.DB, repo *repository.TwinRepo, ownerID, major, class string, dim float64) {
	t.Helper()
	uid := createStu(t, db)
	if err := repo.UpsertSnapshot(&repository.TwinSnapshot{
		UserID: uid, OwnerScope: "college", OwnerID: ownerID,
		College: ownerID, Major: major, ClassName: class,
		AcademicScore: dim, AbilityScore: dim, IdeologicalScore: dim,
		EmotionalScore: dim, SocialScore: dim,
		AIInterpretation: "", GapAnalysis: "[]", StageAdvice: "[]",
	}); err != nil {
		t.Fatalf("写入快照失败: %v", err)
	}
}

var stuSeed = 700000

func createStu(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	stuSeed++
	uname := fmt.Sprintf("svcstu%d", stuSeed)
	res, err := db.Exec(`INSERT OR IGNORE INTO users (username, role, display_name) VALUES (?, 'student', ?)`, uname, "测试学生")
	if err != nil {
		t.Fatalf("建学生失败: %v", err)
	}
	id, _ := res.LastInsertId()
	if id <= 0 {
		_ = db.QueryRow(`SELECT id FROM users WHERE username=?`, uname).Scan(&id)
	}
	return id
}

// TestCollegeFiveDim_ZeroSamples 覆盖 0 样本：无快照 → FiveDim=nil，绝不可见编造均值。
func TestCollegeFiveDim_ZeroSamples(t *testing.T) {
	svc, _, _ := setupCollegeTwinService(t)
	fd := svc.aggregateCollegeFiveDim("cs", "", "")
	if fd != nil {
		t.Fatalf("无快照时应返回 nil（前端渲染数据积累中），得到 %+v", fd)
	}
	// 整体 Fallback 语义不变
	data := svc.GenerateTwinScreen(context.Background(), "计算机学院", "cs", "", "")
	if data.DataSource != "fallback" || data.FiveDim != nil {
		t.Errorf("空库应 fallback 且 FiveDim=nil，got data_source=%s five_dim=%+v", data.DataSource, data.FiveDim)
	}
	if v, _ := data.Overview["total_students"]; v != 0 {
		t.Errorf("空库 overview 四字段应保持 fallback，total_students=%v", v)
	}
}

// TestCollegeFiveDim_MixedAvailability 覆盖部分维度机制：有快照时各维均为 real。
// （此处全部维度同分，验证均值 = 指定值、sample_count 正确、data_source=real）
func TestCollegeFiveDim_RealMeans(t *testing.T) {
	svc, repo, db := setupCollegeTwinService(t)
	insertSnap(t, db, repo, "cs", "软件工程", "SE2501", 90)
	insertSnap(t, db, repo, "cs", "软件工程", "SE2502", 70)
	insertSnap(t, db, repo, "cs", "计算机科学与技术", "CS2501", 80)

	fd := svc.aggregateCollegeFiveDim("cs", "", "")
	if fd == nil {
		t.Fatal("有快照时应返回 FiveDim")
	}
	if fd.SampleCount != 3 {
		t.Errorf("sample_count 应 3，得到 %d", fd.SampleCount)
	}
	if len(fd.Dimensions) != 5 {
		t.Fatalf("应 5 维，得到 %d", len(fd.Dimensions))
	}
	// 学业均值 = (90+70+80)/3 = 80
	academic := findDim(fd, "academic")
	if academic == nil || academic.Score == nil || *academic.Score != 80 {
		t.Errorf("学业均值应 80，得到 %+v", academic)
	}
	if academic.SampleCount != 3 || academic.DataSource != "real" {
		t.Errorf("学业维样本/来源错误: %+v", academic)
	}
}

// TestCollegeFiveDim_Over500 覆盖 >500 去限：mean 完整不因 500 上限失真。
func TestCollegeFiveDim_Over500(t *testing.T) {
	svc, repo, db := setupCollegeTwinService(t)
	const n = 600
	for i := 0; i < n; i++ {
		insertSnap(t, db, repo, "cs", "cs-major", "CS2501", float64(i%101))
	}
	data := svc.GenerateTwinScreen(context.Background(), "计算机学院", "cs", "", "")
	if data.FiveDim == nil || data.FiveDim.SampleCount != n {
		t.Fatalf(">500 学生应全部计入 FiveDim，期望 %d 快照，得到 %+v", n, data.FiveDim)
	}
	// 理论学业均值 = Σ(i%101)/600
	want := 0.0
	for i := 0; i < n; i++ {
		want += float64(i % 101)
	}
	want /= n
	academic := findDim(data.FiveDim, "academic")
	if academic == nil || academic.Score == nil {
		t.Fatal("学业维应有分数（去 limit 后非空）")
	}
	if d := diff(*academic.Score, roundTo1(want)); d > 1e-6 {
		t.Errorf("去 limit 均值偏差: want≈%.2f got %.2f", roundTo1(want), *academic.Score)
	}
}

// TestCollegeFiveDim_CrossCollege 覆盖跨院隔离：ownerID 只读本院，看不到他院快照。
func TestCollegeFiveDim_CrossCollege(t *testing.T) {
	svc, repo, db := setupCollegeTwinService(t)
	insertSnap(t, db, repo, "cs", "cs-major", "CS2501", 100)
	insertSnap(t, db, repo, "math", "math-major", "MA2501", 20)

	cs := svc.aggregateCollegeFiveDim("cs", "", "")
	if cs == nil || cs.SampleCount != 1 {
		t.Fatalf("cs 应只统计本院 1 条（跨院隔离），得到 %+v", cs)
	}
	academic := findDim(cs, "academic")
	if academic == nil || academic.Score == nil || *academic.Score != 100 {
		t.Errorf("cs 学业均值应 100（他院 20 不应混入），得到 %+v", academic)
	}
}

// TestTwinScreen_OverviewUnchanged 行为不变守卫：overview 四字段在有无数据两态下保持既有语义。
func TestTwinScreen_OverviewUnchanged(t *testing.T) {
	svc, repo, db := setupCollegeTwinService(t)

	// 空库：fallback 四字段全 0
	data := svc.GenerateTwinScreen(context.Background(), "计算机学院", "cs", "", "")
	if data.DataSource != "fallback" {
		t.Fatalf("空库应 fallback，得到 %s", data.DataSource)
	}
	if v, _ := data.Overview["total_students"]; v != 0 {
		t.Errorf("fallback total_students 应 0，得到 %v", v)
	}

	// 有学生：overview 由真实聚合得出，健康度 = 五维均值（每维同分 80 → 健康度 80）
	insertSnap(t, db, repo, "cs", "cs-major", "CS2501", 80)
	insertSnap(t, db, repo, "cs", "cs-major", "CS2501", 80)
	data2 := svc.GenerateTwinScreen(context.Background(), "计算机学院", "cs", "", "")
	if data2.DataSource != "real" {
		t.Fatalf("有学生应 real，得到 %s", data2.DataSource)
	}
	if data2.Overview == nil || data2.Overview["total_students"] != 0 {
		t.Errorf("health 由 twin 聚合，但 userRepo=nil 时 total_students=0；请勿误填。got %+v", data2.Overview)
	}
	// 健康度仅由 twinRepo 聚合得出（userRepo=nil 不影响 health_score）
	if data2.Overview["health_score"] != nil {
		if v, _ := data2.Overview["health_score"].(float64); diff(v, 80) > 1e-6 {
			t.Errorf("健康度应为 80（每维同分 80），得到 %v", data2.Overview["health_score"])
		}
	}
}

// findDim 按 key 查找维度条目
func findDim(fd *CollegeFiveDim, key string) *FiveDimEntry {
	for i := range fd.Dimensions {
		if fd.Dimensions[i].Key == key {
			return &fd.Dimensions[i]
		}
	}
	return nil
}

// diff 浮点差绝对值
func diff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}
