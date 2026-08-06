// resetSeedPasswords — 一次性工具：将内置种子账号密码重置为 Wxx@2026
// 用法：reset-seed-passwords <sqlite_path>
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "用法: reset-seed-passwords <sqlite_path> [enrollment_year]")
		fmt.Fprintln(os.Stderr, "  第二个参数可选：给种子学生补入学年份（如 2025）")
		os.Exit(1)
	}
	path := os.Args[1]
	year := ""
	if len(os.Args) >= 3 {
		year = os.Args[2]
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "打开数据库失败:", err)
		os.Exit(1)
	}
	defer db.Close()

	const seedPassword = "Wxx@2026"
	newHash, err := bcrypt.GenerateFromPassword([]byte(seedPassword), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "生成哈希失败:", err)
		os.Exit(1)
	}

	// 内置种子账号（含旧版补充账号）
	seedUsers := []string{
		"sysadmin", "schooladmin", "collegeadmin",
		"counselor_cs", "counselor_math", "counselor1", "counselor2",
		"stunion", "student_cs", "student_math", "student1",
		"teacher1", "assistant1", "admin",
	}

	tx, err := db.Begin()
	if err != nil {
		fmt.Fprintln(os.Stderr, "开启事务失败:", err)
		os.Exit(1)
	}
	defer tx.Rollback()

	updated := 0
	for _, u := range seedUsers {
		res, err := tx.Exec(
			"UPDATE users SET password_hash = ?, updated_at = datetime('now') WHERE username = ?",
			string(newHash), u,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "更新 %s 失败: %v\n", u, err)
			os.Exit(1)
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			updated++
			fmt.Printf("  重置: %s\n", u)
		}
	}

	if err := tx.Commit(); err != nil {
		fmt.Fprintln(os.Stderr, "提交失败:", err)
		os.Exit(1)
	}
	fmt.Printf("完成：%d 个种子账号已重置为 %s\n", updated, seedPassword)

	// 可选：给种子学生补入学年份（用于年级主题）
	if year != "" {
		studentUsers := []string{
			"student_cs", "student_math", "student1", "stunion",
		}
		by, err := db.Begin()
		if err != nil {
			fmt.Fprintln(os.Stderr, "开启事务失败:", err)
			os.Exit(1)
		}
		defer by.Rollback()
		n := 0
		for _, u := range studentUsers {
			res, err := by.Exec(
				"UPDATE users SET enrollment_year = ?, updated_at = datetime('now') WHERE username = ?",
				year, u,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "更新 %s 年份失败: %v\n", u, err)
				os.Exit(1)
			}
			c, _ := res.RowsAffected()
			if c > 0 {
				n++
				fmt.Printf("  补年份 %s = %s 级\n", u, year)
			}
		}
		if err := by.Commit(); err != nil {
			fmt.Fprintln(os.Stderr, "提交失败:", err)
			os.Exit(1)
		}
		fmt.Printf("完成：%d 个学生账号已设置入学年份 %s\n", n, year)
	}
}
