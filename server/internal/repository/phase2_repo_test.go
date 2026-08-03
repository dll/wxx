package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/testutil"
)

func setupPhase2TestDB(t *testing.T) (*Phase2Repo, int64, int64) {
	t.Helper()
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS student_points (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			points INTEGER NOT NULL DEFAULT 0,
			reason TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime'))
		);
		CREATE TABLE IF NOT EXISTS qa_posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT '综合',
			answers INTEGER NOT NULL DEFAULT 0,
			views INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'published',
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime'))
		);
		CREATE TABLE IF NOT EXISTS qa_answers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			adopted INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime'))
		);
		CREATE TABLE IF NOT EXISTS talk_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			counselor_id INTEGER NOT NULL,
			student_id INTEGER NOT NULL DEFAULT 0,
			student_name TEXT NOT NULL DEFAULT '',
			topic TEXT NOT NULL DEFAULT '',
			emotion TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			follow_ups TEXT NOT NULL DEFAULT '[]',
			status TEXT NOT NULL DEFAULT 'following',
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime'))
		);
	`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO users (username, role, display_name) VALUES ('stu_qa1','student','张三'), ('stu_qa2','student','李四')`)
	var uid1, uid2 int64
	_ = db.QueryRow(`SELECT id FROM users WHERE username='stu_qa1'`).Scan(&uid1)
	_ = db.QueryRow(`SELECT id FROM users WHERE username='stu_qa2'`).Scan(&uid2)
	return NewPhase2Repo(db), uid1, uid2
}

func TestPhase2Repo_Points(t *testing.T) {
	r, uid, _ := setupPhase2TestDB(t)

	if err := r.AddPoints(uid, 5, "打卡", "checkin"); err != nil {
		t.Fatalf("AddPoints 失败: %v", err)
	}
	if err := r.AddPoints(uid, 10, "发布问题", "qa_post"); err != nil {
		t.Fatalf("AddPoints 失败: %v", err)
	}

	total, count, recent, err := r.GetPointsSummary(uid, 10)
	if err != nil {
		t.Fatalf("GetPointsSummary 失败: %v", err)
	}
	if total != 15 || count != 2 || len(recent) != 2 {
		t.Fatalf("积分汇总不符 total=%d count=%d recent=%d", total, count, len(recent))
	}

	n, _ := r.CountSource(uid, "checkin")
	if n != 1 {
		t.Fatalf("checkin 来源计数应为 1，实际 %d", n)
	}
}

func TestPhase2Repo_QA(t *testing.T) {
	r, uid1, uid2 := setupPhase2TestDB(t)

	postID, err := r.CreateQAPost(uid1, "转专业条件？", "需要什么", "政策")
	if err != nil {
		t.Fatalf("CreateQAPost 失败: %v", err)
	}
	if postID == 0 {
		t.Fatal("postID 不应为 0")
	}

	posts, err := r.ListQAPosts(10)
	if err != nil {
		t.Fatalf("ListQAPosts 失败: %v", err)
	}
	if len(posts) != 1 || posts[0].Title != "转专业条件？" {
		t.Fatalf("帖子列表不符: %+v", posts)
	}
	if posts[0].Author != "张三" {
		t.Fatalf("作者名 JOIN 失败: %s", posts[0].Author)
	}

	if _, err := r.CreateQAAnswer(postID, uid2, "我来回答"); err != nil {
		t.Fatalf("CreateQAAnswer 失败: %v", err)
	}
	answers, err := r.ListQAAnswers(postID)
	if err != nil {
		t.Fatalf("ListQAAnswers 失败: %v", err)
	}
	if len(answers) != 1 || answers[0]["author"] != "李四" {
		t.Fatalf("回答列表不符: %+v", answers)
	}

	post, err := r.GetQAPost(postID)
	if err != nil {
		t.Fatalf("GetQAPost 失败: %v", err)
	}
	if post.Answers != 1 {
		t.Fatalf("回答数应自增为 1，实际 %d", post.Answers)
	}
}

func TestPhase2Repo_TalkRecords(t *testing.T) {
	r, uid, _ := setupPhase2TestDB(t)

	if _, err := r.CreateTalkRecord(100, uid, "张三", "学业困难", "焦虑", "谈话原文", "AI 摘要", `["跟进"]`); err != nil {
		t.Fatalf("CreateTalkRecord 失败: %v", err)
	}

	records, err := r.ListTalkRecords(100, 10)
	if err != nil {
		t.Fatalf("ListTalkRecords 失败: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("记录数应为 1，实际 %d", len(records))
	}
	if records[0]["student_name"] != "张三" || records[0]["topic"] != "学业困难" {
		t.Fatalf("记录内容不符: %+v", records[0])
	}

	other, _ := r.ListTalkRecords(101, 10)
	if len(other) != 0 {
		t.Fatalf("他人记录应不可见，实际 %d", len(other))
	}
}
