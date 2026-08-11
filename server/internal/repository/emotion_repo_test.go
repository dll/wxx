package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
)

func setupEmotionTestDB(t *testing.T) *EmotionRepo {
	t.Helper()

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	// 001_init.sql 创建的 emotion_logs 缺少 alert_id/username/status 等字段
	// 直接删除旧表并用完整 schema 重建
	db.Exec("DROP TABLE IF EXISTS emotion_logs")
	db.Exec(`CREATE TABLE emotion_logs (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id         INTEGER NOT NULL REFERENCES users(id),
		session_id      TEXT    NOT NULL,
		message_text    TEXT    NOT NULL DEFAULT '',
		score           REAL    NOT NULL DEFAULT 0,
		risk_level      TEXT    NOT NULL DEFAULT 'low'
		                CHECK(risk_level IN ('low','medium','high','urgent')),
		analysis_json   TEXT    NOT NULL DEFAULT '{}',
		notified        INTEGER NOT NULL DEFAULT 0,
		status          TEXT    NOT NULL DEFAULT 'pending'
		                CHECK(status IN ('pending','acknowledged','resolved')),
		acknowledged_by TEXT    NOT NULL DEFAULT '',
		acknowledged_at TEXT    DEFAULT NULL,
		alert_id        TEXT    NOT NULL DEFAULT '',
		username        TEXT    NOT NULL DEFAULT '',
		created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
	)`)

	// 创建测试用户（ListAlerts 和 GetStats 需要 JOIN users）。
	// 注意：016_fix_seed_users.sql 已插入种子用户 student1(id=1, college/cs)、
	// counselor2(id=3, college/math) 等。测试日志 user_id 固定用 id=1（归属 college/cs），
	// 跨学院隔离测试用 id=3（college/math）做"他人学院"。
	db.Exec(`UPDATE users SET owner_scope='college', owner_id='cs' WHERE id=1`)

	return NewEmotionRepo(db)
}

func TestEmotionRepo_Create(t *testing.T) {
	repo := setupEmotionTestDB(t)

	log := &model.EmotionLog{
		AlertID:      "alert-001",
		UserID:       1,
		Username:     "testuser",
		SessionID:    "sess-1",
		MessageText:  "最近压力很大",
		Score:        -0.6,
		RiskLevel:    "medium",
		AnalysisJSON: `{"reason":"学业压力"}`,
		Notified:     0,
		Status:       "pending",
	}
	id, err := repo.Create(log)
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if id <= 0 {
		t.Errorf("期望有效 id，得到 %d", id)
	}

	// 回查
	created, err := repo.GetByAlertID("alert-001", "school", "", "sys_admin")
	if err != nil {
		t.Fatalf("GetByAlertID 失败: %v", err)
	}
	if created.MessageText != "最近压力很大" {
		t.Errorf("期望 MessageText=最近压力很大，得到 %s", created.MessageText)
	}
	if created.RiskLevel != "medium" {
		t.Errorf("期望 RiskLevel=medium，得到 %s", created.RiskLevel)
	}
}

func TestEmotionRepo_GetByAlertID_NotFound(t *testing.T) {
	repo := setupEmotionTestDB(t)

	log, err := repo.GetByAlertID("nonexistent", "school", "", "sys_admin")
	if err != nil {
		t.Fatalf("GetByAlertID 失败: %v", err)
	}
	if log != nil {
		t.Error("不存在的 alert_id 应返回 nil")
	}
}

func TestEmotionRepo_ListAlerts(t *testing.T) {
	repo := setupEmotionTestDB(t)

	repo.Create(&model.EmotionLog{AlertID: "al-1", UserID: 1, Username: "testuser", MessageText: "消息1", Score: -0.8, RiskLevel: "high", Status: "pending"})
	repo.Create(&model.EmotionLog{AlertID: "al-2", UserID: 1, Username: "testuser", MessageText: "消息2", Score: -0.3, RiskLevel: "low", Status: "pending"})

	alerts, total, err := repo.ListAlerts("", "", "school", "", "sys_admin", 1, 10)
	if err != nil {
		t.Fatalf("ListAlerts 失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望 total=2，得到 %d", total)
	}
	if len(alerts) != 2 {
		t.Errorf("期望 2 条告警，得到 %d", len(alerts))
	}
}

func TestEmotionRepo_ListAlerts_FilterByRisk(t *testing.T) {
	repo := setupEmotionTestDB(t)

	repo.Create(&model.EmotionLog{AlertID: "al-h", UserID: 1, Username: "testuser", MessageText: "高危", Score: -0.9, RiskLevel: "high", Status: "pending"})
	repo.Create(&model.EmotionLog{AlertID: "al-l", UserID: 1, Username: "testuser", MessageText: "低危", Score: -0.1, RiskLevel: "low", Status: "pending"})

	_, total, err := repo.ListAlerts("high", "", "school", "", "sys_admin", 1, 10)
	if err != nil {
		t.Fatalf("ListAlerts 过滤失败: %v", err)
	}
	if total != 1 {
		t.Errorf("过滤 high 后期望 total=1，得到 %d", total)
	}
}

func TestEmotionRepo_GetStats(t *testing.T) {
	repo := setupEmotionTestDB(t)

	repo.Create(&model.EmotionLog{AlertID: "st-1", UserID: 1, Username: "testuser", MessageText: "紧急", Score: -1.0, RiskLevel: "urgent", Status: "pending"})
	repo.Create(&model.EmotionLog{AlertID: "st-2", UserID: 1, Username: "testuser", MessageText: "高", Score: -0.7, RiskLevel: "high", Status: "pending"})
	repo.Create(&model.EmotionLog{AlertID: "st-3", UserID: 1, Username: "testuser", MessageText: "中", Score: -0.4, RiskLevel: "medium", Status: "resolved"})

	stats, err := repo.GetStats("school", "", "sys_admin")
	if err != nil {
		t.Fatalf("GetStats 失败: %v", err)
	}
	if stats.Pending != 2 {
		t.Errorf("期望 Pending=2，得到 %d", stats.Pending)
	}
	if stats.Urgent != 1 {
		t.Errorf("期望 Urgent=1，得到 %d", stats.Urgent)
	}
	if stats.High != 1 {
		t.Errorf("期望 High=1，得到 %d", stats.High)
	}
}

func TestEmotionRepo_UpdateStatus(t *testing.T) {
	repo := setupEmotionTestDB(t)

	repo.Create(&model.EmotionLog{AlertID: "upd-st", UserID: 1, Username: "testuser", MessageText: "更新测试", Score: -0.5, RiskLevel: "medium", Status: "pending"})

	n, err := repo.UpdateStatus("upd-st", "acknowledged", "counselor1", "school", "", "sys_admin")
	if err != nil {
		t.Fatalf("UpdateStatus 失败: %v", err)
	}
	if n != 1 {
		t.Errorf("期望更新 1 行，得到 %d", n)
	}

	updated, _ := repo.GetByAlertID("upd-st", "school", "", "sys_admin")
	if updated.Status != "acknowledged" {
		t.Errorf("期望 Status=acknowledged，得到 %s", updated.Status)
	}
	if updated.AcknowledgedBy != "counselor1" {
		t.Errorf("期望 AcknowledgedBy=counselor1，得到 %s", updated.AcknowledgedBy)
	}
}

// TestEmotionRepo_UpdateStatus_CrossCollegeBlocked P0-05 回归：
// cs 学院辅导员不得更新 math 学院学生的告警（越权写），更新行数必须为 0。
// 测试数据：user_id=1（student1, college/cs）为本人，跨学院用 college/math。
func TestEmotionRepo_UpdateStatus_CrossCollegeBlocked(t *testing.T) {
	repo := setupEmotionTestDB(t)

	// user_id=1 归属 college/cs（seed student1）
	repo.Create(&model.EmotionLog{AlertID: "cs-alert", UserID: 1, Username: "student1", MessageText: "cs学院学生消息", Score: -0.8, RiskLevel: "high", Status: "pending"})

	// math 学院的辅导员（owner_id=math）尝试更新 cs 学院的告警 → 必须 0 行
	n, err := repo.UpdateStatus("cs-alert", "acknowledged", "counselor_math", "college", "math", "counselor")
	if err != nil {
		t.Fatalf("UpdateStatus 失败: %v", err)
	}
	if n != 0 {
		t.Errorf("跨学院更新应返回 0 行，得到 %d", n)
	}

	// 且告警状态未被篡改
	got, err := repo.GetByAlertID("cs-alert", "school", "", "sys_admin")
	if err != nil || got == nil {
		t.Fatalf("回查失败: %v", err)
	}
	if got.Status != "pending" {
		t.Errorf("跨学院更新不应生效，期望仍为 pending，得到 %s", got.Status)
	}

	// 同学院辅导员更新成功
	n, err = repo.UpdateStatus("cs-alert", "acknowledged", "counselor_cs", "college", "cs", "counselor")
	if err != nil {
		t.Fatalf("UpdateStatus 失败: %v", err)
	}
	if n != 1 {
		t.Errorf("同学院更新应返回 1 行，得到 %d", n)
	}
}

// TestEmotionRepo_GetByAlertID_ScopeFiltered P0-05 回归：
// 跨学院读取原始敏感消息必须返回 nil（越权读）。
func TestEmotionRepo_GetByAlertID_ScopeFiltered(t *testing.T) {
	repo := setupEmotionTestDB(t)

	repo.Create(&model.EmotionLog{AlertID: "cs-read", UserID: 1, Username: "student1", MessageText: "cs学院敏感消息", Score: -0.9, RiskLevel: "urgent", Status: "pending"})

	// math 学院辅导员读 cs 学院告警 → nil
	got, err := repo.GetByAlertID("cs-read", "college", "math", "counselor")
	if err != nil {
		t.Fatalf("GetByAlertID 失败: %v", err)
	}
	if got != nil {
		t.Error("跨学院读取告警应返回 nil（越权读被拒绝）")
	}

	// cs 学院辅导员读自己学院告警 → 命中
	got, err = repo.GetByAlertID("cs-read", "college", "cs", "counselor")
	if err != nil {
		t.Fatalf("GetByAlertID 失败: %v", err)
	}
	if got == nil {
		t.Error("同学院读取告警应命中")
	}
	if got.MessageText != "cs学院敏感消息" {
		t.Errorf("同学院应能读取消息文本，得到 %s", got.MessageText)
	}
}

func TestEmotionRepo_GetTrends(t *testing.T) {
	repo := setupEmotionTestDB(t)

	repo.Create(&model.EmotionLog{AlertID: "tr-1", UserID: 1, Username: "testuser", MessageText: "趋势1", Score: -0.8, RiskLevel: "high", Status: "pending"})
	repo.Create(&model.EmotionLog{AlertID: "tr-2", UserID: 1, Username: "testuser", MessageText: "趋势2", Score: -0.2, RiskLevel: "low", Status: "pending"})

	points, err := repo.GetTrends(7, "school", "", "sys_admin")
	if err != nil {
		t.Fatalf("GetTrends 失败: %v", err)
	}
	if len(points) == 0 {
		t.Error("应有至少 1 个数据点")
	}
}
