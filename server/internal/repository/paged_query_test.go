package repository

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestBuildWhereClause(t *testing.T) {
	cases := []struct {
		name  string
		where []string
		want  string
	}{
		{"空条件恒真兜底", nil, "WHERE 1=1"},
		{"含空白片段被剔除", []string{"", "  "}, "WHERE 1=1"},
		{"单条件", []string{"is_active = 1"}, "WHERE is_active = 1"},
		{"多条件 AND 连接", []string{"is_active = 1", "college = ?"}, "WHERE is_active = 1 AND college = ?"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildWhereClause(tc.where); got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestPageOffset(t *testing.T) {
	cases := []struct {
		page, size, want int
	}{
		{1, 20, 0},
		{2, 20, 20},
		{3, 15, 30},
		{0, 20, 0},  // 非法页码归一第 1 页
		{-1, 20, 0}, // 负页码归一
		{2, 0, 20},  // 非法页大小归一 20 → 第 2 页偏移 20
		{2, -5, 20}, // 同上
	}
	for _, tc := range cases {
		if got := pageOffset(tc.page, tc.size); got != tc.want {
			t.Errorf("pageOffset(%d,%d) = %d, want %d", tc.page, tc.size, got, tc.want)
		}
	}
}

func TestCountPaged(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// :memory: 每连接独立库，限单连接
	db.SetMaxOpenConns(1)
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE advisors (id INTEGER PRIMARY KEY, college TEXT, is_active INTEGER);
		INSERT INTO advisors (college, is_active) VALUES ('cs', 1), ('cs', 1), ('math', 1), ('cs', 0)`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	where := []string{"is_active = 1", "college = ?"}
	whereSQL := buildWhereClause(where)
	total, err := countPaged(db, "advisors", whereSQL, []interface{}{"cs"})
	if err != nil {
		t.Fatalf("countPaged: %v", err)
	}
	if total != 2 {
		t.Fatalf("want 2, got %d", total)
	}
}
