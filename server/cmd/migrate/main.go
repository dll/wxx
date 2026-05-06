package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

func main() {
	log.Println("蔚小芯 SQLite 迁移工具")

	// 加载环境变量
	_ = godotenv.Load("../../.env")

	dbPath := os.Getenv("SQLITE_PATH")
	if dbPath == "" {
		dbPath = "./data/wxx.db"
	}

	// 确保数据目录存在
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}

	// 打开数据库连接
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()

	// 启用 WAL 模式（提高并发性能）
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("警告：设置 WAL 模式失败: %v", err)
	}

	// 读取 migrations 目录
	migrationsDir := "migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// 尝试相对于 server/ 目录
		migrationsDir = "../../server/migrations"
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		log.Fatalf("读取迁移文件失败: %v", err)
	}

	if len(files) == 0 {
		log.Println("未找到迁移文件")
		return
	}

	// 按文件名排序（确保 001 在 002 之前执行）
	sort.Strings(files)

	// 创建迁移记录表（追踪已执行的迁移）
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL UNIQUE,
		executed_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		log.Fatalf("创建迁移记录表失败: %v", err)
	}

	// 逐个执行迁移文件
	executed := 0
	for _, file := range files {
		filename := filepath.Base(file)

		// 检查是否已执行过
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM _migrations WHERE filename = ?", filename).Scan(&count)
		if err != nil {
			log.Fatalf("查询迁移记录失败: %v", err)
		}
		if count > 0 {
			log.Printf("跳过（已执行）: %s", filename)
			continue
		}

		// 读取 SQL 文件
		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("读取文件 %s 失败: %v", filename, err)
		}

		// 按分号分割并逐条执行（SQLite 不支持多语句 Exec）
		statements := splitSQL(string(content))
		for i, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				log.Fatalf("执行 %s 第 %d 条语句失败: %v\nSQL: %s", filename, i+1, err, truncate(stmt, 200))
			}
		}

		// 记录迁移
		_, err = db.Exec("INSERT INTO _migrations (filename) VALUES (?)", filename)
		if err != nil {
			log.Fatalf("记录迁移 %s 失败: %v", filename, err)
		}

		log.Printf("已执行: %s", filename)
		executed++
	}

	if executed == 0 {
		log.Println("所有迁移已是最新状态")
	} else {
		fmt.Printf("成功执行 %d 个迁移文件，数据库: %s\n", executed, dbPath)
	}
}

// splitSQL 按分号分割 SQL 语句，但忽略触发器等复合语句中的分号
func splitSQL(content string) []string {
	var statements []string
	var current strings.Builder
	inTrigger := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		// 跳过纯注释行
		if strings.HasPrefix(trimmed, "--") {
			continue
		}

		// 检测触发器开始/结束
		upper := strings.ToUpper(trimmed)
		if strings.Contains(upper, "CREATE TRIGGER") {
			inTrigger = true
		}

		current.WriteString(line)
		current.WriteString("\n")

		// 触发器以 END; 结束
		if inTrigger && strings.HasSuffix(trimmed, "END;") {
			statements = append(statements, current.String())
			current.Reset()
			inTrigger = false
			continue
		}

		// 非触发器上下文中，分号是语句终结符
		if !inTrigger && strings.HasSuffix(trimmed, ";") {
			statements = append(statements, current.String())
			current.Reset()
		}
	}

	// 处理末尾无分号的语句
	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		statements = append(statements, remaining)
	}

	return statements
}

// truncate 截断字符串用于日志输出
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
