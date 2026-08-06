// resetSeedPasswords — 一次性运维工具：重置内置/学生账号密码为 Wxx@2026
// 用法：
//
//	reset-seed-passwords <sqlite_path>                    # 仅重置 14 个内置种子账号
//	reset-seed-passwords <sqlite_path> --all-students     # 重置所有 student 角色账号
//	reset-seed-passwords <sqlite_path> <year>             # 重置种子账号 + 补种子学生入学年份
package main

import (
	"database/sql"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "用法: reset-seed-passwords <sqlite_path> [--all-students|enrollment_year]")
		os.Exit(1)
	}
	path := os.Args[1]
	arg2 := ""
	if len(os.Args) >= 3 {
		arg2 = os.Args[2]
	}
	allStudents := arg2 == "--all-students"
	year := ""
	if !allStudents && arg2 != "" {
		year = arg2
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

	tx, err := db.Begin()
	if err != nil {
		fmt.Fprintln(os.Stderr, "开启事务失败:", err)
		os.Exit(1)
	}
	defer tx.Rollback()

	updated := 0
	if allStudents {
		// 重置全部 student 角色账号（导入的学生学号账号）
		res, err := tx.Exec(
			"UPDATE users SET password_hash = ?, updated_at = datetime('now') WHERE role = 'student'",
			string(newHash),
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, "重置学生密码失败:", err)
			os.Exit(1)
		}
		n, _ := res.RowsAffected()
		updated = int(n)
		fmt.Printf("重置 %d 个学生账号密码为 %s\n", updated, seedPassword)
	} else {
		// 内置种子账号（含旧版补充账号）
		seedUsers := []string{
			"sysadmin", "schooladmin", "collegeadmin",
			"counselor_cs", "counselor_math", "counselor1", "counselor2",
			"stunion", "student_cs", "student_math", "student1",
			"teacher1", "assistant1", "admin",
		}
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
		fmt.Printf("完成：%d 个种子账号已重置为 %s\n", updated, seedPassword)
	}

	if err := tx.Commit(); err != nil {
		fmt.Fprintln(os.Stderr, "提交失败:", err)
		os.Exit(1)
	}

	// 可选：给种子学生补入学年份（用于年级主题）
	if !allStudents && year != "" {
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
