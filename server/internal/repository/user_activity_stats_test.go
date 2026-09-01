package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
)

// TestUserActivityStats_CollegeScope 回归（2026-09-01 安全修复）：
// college_admin 读取学生注册/打卡统计必须限定本院（users.college），
// 不可把全校其它学院数据带入。全校（scopeType=""）计数应含全部，
// 学院范围（scopeType="college"）仅含对应学院。
func TestUserActivityStats_CollegeScope(t *testing.T) {
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { _ = db.Close() })

	userRepo := NewUserRepo(db)
	create := func(username, college, role string) int64 {
		id, err := userRepo.Create(&model.User{
			Username:    username,
			DisplayName: username,
			Role:        role,
			College:     college,
			OwnerScope:  "college",
			OwnerID:     college,
			Status:      "active",
		})
		if err != nil {
			t.Fatalf("创建用户失败 %s: %v", username, err)
		}
		return id
	}

	// 两个学院各 2 名学生
	const cs = "计算机学院"
	const wl = "外国语学院"
	create("cs1", cs, "student")
	create("cs2", cs, "student")
	s1 := create("wl1", wl, "student")
	create("wl2", wl, "student")

	// 今日打卡：本院 cs 一人、外院 wl 一人
	if _, err := db.Exec("INSERT INTO student_checkins (user_id, check_date) VALUES (?, date('now'))", s1); err != nil {
		t.Fatalf("插入打卡失败: %v", err)
	}

	repo := NewUserActivityStatsRepo(db)

	// 全校口径（含迁移 016 的 seed 学生，故 ≥4，这里精确断言 ≥4 即可）
	global, err := repo.GetUserActivityStats("", "")
	if err != nil {
		t.Fatalf("全校统计失败: %v", err)
	}
	if global.RegisteredTotal < 4 {
		t.Fatalf("全校累计注册应 ≥4，得到 %d", global.RegisteredTotal)
	}

	// 本院口径（计算机学院）
	csStats, err := repo.GetUserActivityStats("college", cs)
	if err != nil {
		t.Fatalf("本院统计失败: %v", err)
	}
	if csStats.RegisteredTotal != 2 {
		t.Fatalf("计算机学院累计注册应为 2，得到 %d（跨学院泄漏）", csStats.RegisteredTotal)
	}

	// 外院口径
	wlStats, err := repo.GetUserActivityStats("college", wl)
	if err != nil {
		t.Fatalf("外院统计失败: %v", err)
	}
	if wlStats.RegisteredTotal != 2 {
		t.Fatalf("外国语学院累计注册应为 2，得到 %d", wlStats.RegisteredTotal)
	}
}
