package app

// 数据库初始化与迁移执行（从原 app.go 拆分，行为不变）

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dll/wxx/server"
	"github.com/dll/wxx/server/internal/config"
	dbutil "github.com/dll/wxx/server/internal/db"

	_ "github.com/go-sql-driver/mysql"                   // MySQL 驱动（DB_DRIVER=mysql）
	_ "github.com/tursodatabase/libsql-client-go/libsql" // Turso 云数据库驱动（libsql:// 协议）
	_ "modernc.org/sqlite"                               // 纯 Go SQLite 驱动（本地文件 + FTS5）
)

// initDB 初始化数据库连接
// 自动识别 DSN 协议选择驱动：
//   - DB_DRIVER=mysql → MySQL（go-sql-driver/mysql）
//   - libsql:// 开头 → Turso 云数据库（libsql 驱动）
//   - 其他 → 本地 SQLite 文件（modernc.org/sqlite 驱动）
func initDB(cfg *config.Config, dbPath string, driver dbutil.Driver) (*sql.DB, error) {
	var driverName, dsn string

	switch driver {
	case dbutil.DriverMySQL:
		driverName = "mysql"
		dsn = cfg.MySQLDSN()
		log.Printf("使用 MySQL 数据库: %s@%s:%s/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	case dbutil.DriverTurso:
		driverName = "libsql"
		dsn = dbPath
		log.Printf("使用 Turso 云数据库: %s", dbPath)
	default:
		// 本地 SQLite 文件：确保目录存在，附加 pragma 参数
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return nil, err
		}
		driverName = "sqlite"
		dsn = dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	// 连接池配置：Turso/MySQL 支持并发连接，本地 SQLite 限制单连接
	switch driver {
	case dbutil.DriverTurso, dbutil.DriverMySQL:
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(time.Minute * 5)
	default:
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(2)
		db.SetConnMaxLifetime(0)
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	switch driver {
	case dbutil.DriverMySQL:
		log.Printf("MySQL 数据库已连接: %s@%s:%s/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	case dbutil.DriverTurso:
		log.Printf("Turso 云数据库已连接: %s", dbPath)
	default:
		log.Printf("SQLite 数据库已连接: %s", dbPath)
	}
	return db, nil
}

// RunMigrations 供测试/工具复用的导出入口（评测套件等需要与生产一致的建库行为）。
// skip 可选：按文件名跳过指定迁移（如评测套件跳过内容种子，使用受控语料）。
func RunMigrations(db *sql.DB, driver dbutil.Driver, skip ...string) error {
	skipSet := make(map[string]bool, len(skip))
	for _, s := range skip {
		skipSet[s] = true
	}
	return runMigrations(db, driver, skipSet)
}

// runMigrations 从嵌入的迁移文件执行数据库迁移
// MySQL 模式：对迁移 SQL 做 SQLite→MySQL 方言转换；FTS5 语句被跳过
func runMigrations(db *sql.DB, driver dbutil.Driver, skip map[string]bool) error {
	if driver == dbutil.DriverMySQL {
		// MySQL 方言的迁移记录表
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			filename VARCHAR(255) NOT NULL UNIQUE,
			executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
			return err
		}
	} else {
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL UNIQUE,
			executed_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
		)`); err != nil {
			return err
		}
	}

	entries, err := server.Migrations.ReadDir("migrations")
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	executed := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if skip[entry.Name()] {
			continue
		}

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM _migrations WHERE filename = ?", entry.Name()).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		content, err := server.Migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}

		if err := execSQL(db, string(content), entry.Name(), driver); err != nil {
			return err
		}

		if _, err := db.Exec("INSERT INTO _migrations (filename) VALUES (?)", entry.Name()); err != nil {
			return err
		}

		log.Printf("已执行迁移: %s", entry.Name())
		executed++
	}

	if executed == 0 {
		log.Println("所有迁移已是最新状态")
	} else {
		log.Printf("成功执行 %d 个迁移文件", executed)
	}
	return nil
}

// execSQL 解析并执行 SQL 内容（按分号分割，处理触发器复合语句）
// MySQL 模式：逐条做 SQLite→MySQL 方言转换，跳过 FTS5 语句（MySQL 无 FTS5 虚拟表）
func execSQL(db *sql.DB, content, filename string, driver dbutil.Driver) error {
	statements := splitSQL(content)
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		// MySQL/Turso 不支持 FTS5 虚拟表及其触发器，跳过
		if driver != dbutil.DriverSQLite && (strings.Contains(strings.ToUpper(stmt), "FTS5") || strings.Contains(strings.ToUpper(stmt), "KB_FTS")) {
			log.Printf("迁移 %s 第 %d 条语句跳过（%s 不支持 FTS5）: %.60s...", filename, i+1, driver, stmt)
			continue
		}

		// LONGTEXT 仅 MySQL 需要（SQLite TEXT 无长度限制），非 MySQL 驱动跳过
		if driver != dbutil.DriverMySQL && strings.Contains(strings.ToUpper(stmt), "LONGTEXT") {
			log.Printf("迁移 %s 第 %d 条语句跳过（%s 无需 LONGTEXT）: %.60s...", filename, i+1, driver, stmt)
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
			// ALTER TABLE ADD COLUMN 重复列名视为非致命错误（列已存在 = 目标状态）
			if isDuplicateColumnError(err) && strings.Contains(strings.ToUpper(stmt), "ALTER TABLE") {
				log.Printf("迁移 %s 第 %d 条语句跳过（列已存在）: %v", filename, i+1, err)
				continue
			}
			// 重复索引（MySQL 1061 Duplicate key name / SQLite already exists）
			if isDuplicateIndexError(err) && strings.Contains(strings.ToUpper(stmt), "CREATE INDEX") {
				log.Printf("迁移 %s 第 %d 条语句跳过（索引已存在）: %v", filename, i+1, err)
				continue
			}
			log.Printf("迁移 %s 第 %d 条语句失败: %v", filename, i+1, err)
			return err
		}
	}
	return nil
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

// splitSQL 按 SQL 词法分割语句。分号出现在字符串或标识符引用中时不能结束语句。
func splitSQL(content string) []string {
	var statements []string
	var current strings.Builder
	inTrigger := false
	inSingle, inDouble, inBacktick, inLineComment := false, false, false, false
	for i := 0; i < len(content); i++ {
		ch := content[i]
		if inLineComment {
			if ch == '\n' {
				current.WriteByte(ch)
				inLineComment = false
			}
			continue
		}
		if !inSingle && !inDouble && !inBacktick && ch == '-' && i+1 < len(content) && content[i+1] == '-' {
			inLineComment = true
			current.WriteString("--")
			i++
			continue
		}
		if !inTrigger && !inSingle && !inDouble && !inBacktick && isCreateTriggerStatement(current.String()) {
			inTrigger = true
		}
		if ch == '\'' && !inDouble && !inBacktick {
			if inSingle && i+1 < len(content) && content[i+1] == '\'' {
				current.WriteString("''")
				i++
				continue
			}
			inSingle = !inSingle
		} else if ch == '"' && !inSingle && !inBacktick {
			if inDouble && i+1 < len(content) && content[i+1] == '"' {
				current.WriteString(`""`)
				i++
				continue
			}
			inDouble = !inDouble
		} else if ch == '`' && !inSingle && !inDouble {
			if inBacktick && i+1 < len(content) && content[i+1] == '`' {
				current.WriteString("``")
				i++
				continue
			}
			inBacktick = !inBacktick
		}
		current.WriteByte(ch)
		if ch == ';' && !inSingle && !inDouble && !inBacktick && !inTrigger {
			statements = append(statements, current.String())
			current.Reset()
		}
		// Trigger bodies contain semicolons; END; is the statement terminator.
		if inTrigger && !inSingle && !inDouble && strings.HasSuffix(strings.TrimSpace(current.String()), "END;") {
			statements = append(statements, current.String())
			current.Reset()
			inTrigger = false
		}
	}

	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		for _, line := range strings.Split(remaining, "\n") {
			if text := strings.TrimSpace(line); text != "" && !strings.HasPrefix(text, "--") {
				statements = append(statements, remaining)
				break
			}
		}
	}

	return statements
}

func isCreateTriggerStatement(statement string) bool {
	var sqlText strings.Builder
	for _, line := range strings.Split(statement, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		sqlText.WriteString(trimmed)
		sqlText.WriteByte(' ')
	}
	upper := strings.ToUpper(strings.TrimSpace(sqlText.String()))
	return upper == "CREATE TRIGGER" || strings.HasPrefix(upper, "CREATE TRIGGER ")
}
