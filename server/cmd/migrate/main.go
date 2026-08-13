package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dll/wxx/server/internal/config"
	dbutil "github.com/dll/wxx/server/internal/db"
	_ "github.com/go-sql-driver/mysql" // MySQL 驱动
	"github.com/joho/godotenv"
	_ "github.com/tursodatabase/libsql-client-go/libsql" // Turso 云数据库驱动
	_ "modernc.org/sqlite"                               // 本地 SQLite 驱动
)

func main() {
	log.Println("蔚小芯数据库迁移工具")

	// 加载环境变量
	_ = godotenv.Load("../../.env")

	cfg := config.Load()

	// 方言优先级：DB_DRIVER=mysql → MySQL；否则按 DB_PATH 协议判断
	driver := dbutil.DriverSQLite
	if cfg.DBDriver == "mysql" {
		driver = dbutil.DriverMySQL
	}

	// 优先从 DB_PATH 读取，兼容 SQLITE_PATH
	dbPath := cfg.SQLitePath
	if dbPath == "" {
		dbPath = "./data/wxx.db"
	}

	// 根据协议选择驱动
	var driverName, dsn string
	isTurso := strings.HasPrefix(dbPath, "libsql://")

	switch driver {
	case dbutil.DriverMySQL:
		driverName = "mysql"
		dsn = cfg.MySQLDSN()
		log.Printf("使用 MySQL 数据库: %s@%s:%s/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	case dbutil.DriverTurso:
		driverName = "libsql"
		dsn = dbPath
		log.Printf("使用 Turso 云数据库")
	default:
		if isTurso {
			driver = dbutil.DriverTurso
			driverName = "libsql"
			dsn = dbPath
			log.Printf("使用 Turso 云数据库")
		} else {
			driverName = "sqlite"
			dsn = dbPath + "?_journal_mode=WAL&_busy_timeout=5000"
			// 确保数据目录存在
			if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
				log.Fatalf("创建数据目录失败: %v", err)
			}
			log.Printf("使用本地 SQLite 文件: %s", dbPath)
		}
	}

	// 打开数据库连接
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()

	// 本地 SQLite 启用 WAL 模式（Turso/MySQL 不支持 PRAGMA）
	if driver == dbutil.DriverSQLite {
		if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
			log.Printf("警告：设置 WAL 模式失败: %v", err)
		}
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
	if driver == dbutil.DriverMySQL {
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			filename VARCHAR(255) NOT NULL UNIQUE,
			executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	} else {
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL UNIQUE,
			executed_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
		)`)
	}
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

			// MySQL/Turso 不支持 FTS5 虚拟表及其触发器，跳过
			if driver != dbutil.DriverSQLite && (strings.Contains(strings.ToUpper(stmt), "FTS5") || strings.Contains(strings.ToUpper(stmt), "KB_FTS")) {
				log.Printf("跳过 %s 第 %d 条语句（%s 不支持 FTS5）: %.60s...", filename, i+1, driver, stmt)
				continue
			}

			// SQLite → MySQL 方言转换
			if driver == dbutil.DriverMySQL {
				stmt = dbutil.ToMySQL(stmt)
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}
			}

			if _, err := db.Exec(stmt); err != nil {
				// 与生产迁移 runner 保持一致：ALTER TABLE ADD COLUMN 重复列名视为已达目标状态
				if isDuplicateColumnError(err) && strings.Contains(strings.ToUpper(stmt), "ALTER TABLE") {
					log.Printf("跳过 %s 第 %d 条语句（列已存在）: %v", filename, i+1, err)
					continue
				}
				// 重复索引视为非致命错误
				if isDuplicateIndexError(err) && strings.Contains(strings.ToUpper(stmt), "CREATE INDEX") {
					log.Printf("跳过 %s 第 %d 条语句（索引已存在）: %v", filename, i+1, err)
					continue
				}
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

// splitSQL 按分号分割 SQL 语句，但忽略触发器等复合语句中的分号；
// 支持 `; -- 注释` 同行尾注释（以 `;` 结束语句）
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

		// 语句结束判定基于"去掉行尾注释后的内容"
		base := trimmed
		if idx := strings.LastIndex(base, ";"); idx >= 0 {
			after := strings.TrimSpace(base[idx+1:])
			if after == "" || strings.HasPrefix(after, "--") {
				base = base[:idx+1]
			}
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
		if !inTrigger && strings.HasSuffix(base, ";") {
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

// isDuplicateColumnError 检测 "duplicate column name" 错误（SQLite 小写 / MySQL 大写）
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column name") ||
		strings.Contains(msg, "duplicate column")
}

// isDuplicateIndexError 检测重复索引错误（MySQL 1061 Duplicate key name / SQLite）
func isDuplicateIndexError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "duplicate key name") ||
		strings.Contains(lower, "already exists")
}

// truncate 截断字符串用于日志输出
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
