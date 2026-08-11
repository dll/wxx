package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
)

// TestAuditSnapshot_Flow 校验快照创建/查询/标记恢复
func TestAuditSnapshot_Flow(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	r := NewAuditRepo(db)

	if err := r.CreateSnapshot(&model.AuditSnapshot{
		OpTable: "users", RecordID: "42", Operation: "user.status",
		BeforeJSON: "active", AfterJSON: "disabled",
	}); err != nil {
		t.Fatalf("CreateSnapshot 失败: %v", err)
	}

	s, err := r.GetSnapshotByID(1)
	if err != nil || s == nil {
		t.Fatalf("GetSnapshotByID 失败: err=%v s=%v", err, s)
	}
	if s.Operation != "user.status" || s.BeforeJSON != "active" || s.AfterJSON != "disabled" {
		t.Fatalf("快照字段不符: %+v", s)
	}

	list, err := r.ListSnapshots(10)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListSnapshots 应 1 条: len=%d err=%v", len(list), err)
	}

	if err := r.MarkSnapshotRestored(s.ID, "admin"); err != nil {
		t.Fatalf("MarkSnapshotRestored 失败: %v", err)
	}
	// 已恢复的不再出现在未恢复列表
	list2, _ := r.ListSnapshots(10)
	if len(list2) != 0 {
		t.Fatalf("已恢复快照不应再列出: len=%d", len(list2))
	}
}
