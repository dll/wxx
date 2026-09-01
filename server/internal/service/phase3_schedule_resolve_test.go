package service

import (
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// TestPhase3_ImportSchedules_ResolveOwnerByUsername 回归（2026-09-01 修复「课表挂错账号」）：
// 课表批量导入按 username(学号/工号) 解析归属 user_id，杜绝运营填错内部 user_id
// 导致登录后显示的课程不对；未知 username 或缺失归属一律拒绝，绝不落到幽灵账号。
func TestPhase3_ImportSchedules_ResolveOwnerByUsername(t *testing.T) {
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	// 建 course_schedules 表（testutil 未含迁移 037；含 owner_username 模拟迁移 116）
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS course_schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		course_id TEXT NOT NULL,
		course_name TEXT NOT NULL,
		semester_code TEXT NOT NULL,
		weekday INTEGER NOT NULL,
		start_period INTEGER NOT NULL,
		end_period INTEGER NOT NULL,
		weeks_pattern TEXT NOT NULL DEFAULT '1-20',
		location TEXT,
		teacher TEXT,
		color TEXT DEFAULT '#1565C0',
		owner_username TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
		UNIQUE(user_id, course_id, weekday, start_period, semester_code)
	)`); err != nil {
		t.Fatalf("建 course_schedules 表失败: %v", err)
	}

	userRepo := repository.NewUserRepo(db)
	// 教师 120001、学生 2023001
	tid, _ := userRepo.Create(&model.User{Username: "120001", DisplayName: "胡老师", Role: "teacher", OwnerScope: "college", OwnerID: "cs", Status: "active"})
	_, _ = userRepo.Create(&model.User{Username: "2023001", DisplayName: "张同学", Role: "student", OwnerScope: "college", OwnerID: "cs", Status: "active"})

	dataRepo := repository.NewDataImportRepo(db)
	svc := NewPhase3Service(dataRepo)
	svc.SetUserRepo(userRepo)

	// 1）按 username 导入 + user_id 故意填错 → 应以 username 解析为准（权威）
	res := svc.ImportSchedules([]*repository.ScheduleRow{{
		Username: "120001", CourseID: "CS101", CourseName: "数据结构",
		SemesterCode: "2025-2026-2", Weekday: 1, StartPeriod: 1, EndPeriod: 2,
		UserID: 99999, // 故意错——应被 username 解析覆盖
	}})
	if len(res.Errors) != 0 || res.Created != 1 {
		t.Fatalf("username 导入应成功且忽略错误 user_id: %+v", res)
	}
	// 校验挂到了教师账号（120001 的 user_id）
	info, _ := userRepo.GetByUsername("120001")
	if info.ID != tid {
		t.Fatalf("120001 应解析为 tid=%d，得到 %d", tid, info.ID)
	}
	// 校验挂到了教师账号（120001 的 user_id）：
	// 表内含迁移 037 种子课表（user 1 的 CS101）；这里只验证存在 tid 名下的 CS101 行。
	list, _ := dataRepo.ListSchedules("")
	foundCS := false
	for _, s := range list {
		if s["course_id"] == "CS101" && s["user_id"] == int64(tid) {
			foundCS = true
		}
	}
	if !foundCS {
		t.Fatalf("应存在挂到教师 user_id=%d 的 CS101 课表，当前列表 %v", tid, list)
	}

	// 2）未知 username → 拒绝，不创建
	res2 := svc.ImportSchedules([]*repository.ScheduleRow{{
		Username: "ghost_user", CourseID: "CS999", CourseName: "幽灵课",
		SemesterCode: "2025-2026-2", Weekday: 2, StartPeriod: 3, EndPeriod: 4,
	}})
	if len(res2.Errors) != 1 || res2.Created != 0 {
		t.Fatalf("未知用户名应拒绝导入: %+v", res2)
	}

	// 3）无 username 无 user_id → 拒绝
	res3 := svc.ImportSchedules([]*repository.ScheduleRow{{
		CourseID: "CS200", CourseName: "孤儿课", SemesterCode: "2025-2026-2",
		Weekday: 3, StartPeriod: 1, EndPeriod: 2,
	}})
	if len(res3.Errors) != 1 {
		t.Fatalf("缺失归属应拒绝导入: %+v", res3)
	}

	// 4）学生用自己 username 导入正常
	res4 := svc.ImportSchedules([]*repository.ScheduleRow{{
		Username: "2023001", CourseID: "CS102", CourseName: "操作系统",
		SemesterCode: "2025-2026-2", Weekday: 1, StartPeriod: 3, EndPeriod: 4,
	}})
	if len(res4.Errors) != 0 {
		t.Fatalf("学生 username 导入应成功: %+v", res4)
	}

	// 5）模拟历史错挂：把教师课表直接以错误 user_id 写入（owner_username=120001）
	// 后通过归位接口修正——验证 owner_username 追溯归位
	if _, err := db.Exec(`INSERT INTO course_schedules (user_id, course_id, course_name, semester_code, weekday, start_period, end_period, weeks_pattern, location, teacher, owner_username)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		99999, "CS_HIST", "错挂课", "2025-2026-2", 2, 1, 2, "1-16", "X", "胡老师", "120001"); err != nil {
		t.Fatalf("插入历史错挂课表失败: %v", err)
	}
	affected, total, err := svc.ReassignSchedulesByUsername("120001")
	if err != nil {
		t.Fatalf("归位失败: %v", err)
	}
	// 归位目标：该工号名下全部课表（CS101 已正确 + CS_HIST 错挂，共 2 条）
	if affected != 2 {
		t.Fatalf("应归位 2 条（CS101 已正确 + CS_HIST 错挂），得到 %d", affected)
	}
	if total != 2 {
		t.Fatalf("该工号名下应有 2 条课表，得到 %d", total)
	}
	// 校验已归位到教师正确账号
	list2, _ := dataRepo.ListSchedules("")
	found := false
	for _, s := range list2 {
		if s["course_id"] == "CS_HIST" {
			found = true
			if s["user_id"] != int64(tid) {
				t.Fatalf("历史错挂课表应归位到教师 user_id=%d，实际 %v", tid, s["user_id"])
			}
		}
	}
	if !found {
		t.Fatal("未找到历史错挂课表")
	}
}
