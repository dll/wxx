// Package repository 分页动态查询脚手架（P4-a）。
//
// 背景：全仓库约 60 处 List 函数复制粘贴同一骨架——
// 动态拼 WHERE → COUNT → SELECT ... LIMIT ? OFFSET ?。
// 本文件收敛公共部分；各仓库保留自己的 Scan 循环（类型安全优先于反射魔法）。
//
// 用法：
//
//	where, args := []string{"is_active = 1"}, []interface{}{}
//	if college != "" {
//	    where = append(where, "college = ?")
//	    args = append(args, college)
//	}
//	whereSQL := buildWhereClause(where)
//	total, err := countPaged(r.db, "advisors", whereSQL, args)
//	rows, err := r.db.Query(fmt.Sprintf(
//	    "SELECT ... FROM advisors %s ORDER BY name LIMIT ? OFFSET ?", whereSQL),
//	    append(args, pageSize, offset)...)
package repository

import (
	"database/sql"
	"fmt"
	"strings"
)

// buildWhereClause 拼接 WHERE 子句。
// 空条件返回 "WHERE 1=1"（恒真兜底）：避免空切片产生残尾 "WHERE" 的 SQL 语法错误，
// 也让调用方可以无条件地用 %s 追加，无需分支判断。
func buildWhereClause(where []string) string {
	clauses := make([]string, 0, len(where))
	for _, w := range where {
		if strings.TrimSpace(w) != "" {
			clauses = append(clauses, w)
		}
	}
	if len(clauses) == 0 {
		return "WHERE 1=1"
	}
	return "WHERE " + strings.Join(clauses, " AND ")
}

// countPaged 执行动态条件的 COUNT 查询。
func countPaged(db *sql.DB, table string, whereSQL string, args []interface{}) (int, error) {
	var total int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", table, whereSQL)
	if err := db.QueryRow(query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("统计分页数量失败: %w", err)
	}
	return total, nil
}

// pageOffset 分页偏移（页码从 1 起；非法值归一为第 1 页）。
func pageOffset(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	return (page - 1) * pageSize
}
