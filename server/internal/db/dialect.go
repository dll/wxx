// Package db 提供数据库方言工具：SQLite ↔ MySQL 的 DDL 转换与方言判断。
// 蔚小芯迁移 MySQL 时，原 SQLite 迁移文件（server/migrations/*.sql）由本包
// 在运行时转换为 MySQL 方言，避免维护两套迁移文件。
package db

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// Driver 数据库方言标识
type Driver string

const (
	DriverSQLite Driver = "sqlite"
	DriverMySQL  Driver = "mysql"
	DriverTurso  Driver = "turso"
)

// IsMySQL 判断 *sql.DB 底层驱动是否为 MySQL
func IsMySQL(db *sql.DB) bool {
	return strings.Contains(fmt.Sprintf("%T", db.Driver()), "mysql")
}

// DriverOf 探测 *sql.DB 底层方言
func DriverOf(db *sql.DB) Driver {
	switch {
	case strings.Contains(fmt.Sprintf("%T", db.Driver()), "mysql"):
		return DriverMySQL
	case strings.Contains(fmt.Sprintf("%T", db.Driver()), "libsql"):
		return DriverTurso
	default:
		return DriverSQLite
	}
}

// InsertIgnore 返回方言对应的"插入或忽略"语句前缀（含 INTO 前导空格）
// SQLite: INSERT OR IGNORE INTO；MySQL: INSERT IGNORE INTO
func InsertIgnore(d Driver) string {
	if d == DriverMySQL {
		return "INSERT IGNORE INTO"
	}
	return "INSERT OR IGNORE INTO"
}

// longTextColumns 已知长文本列：MySQL 的 TEXT 不允许默认值，这些列转为
// TEXT 并去掉 DEFAULT（保留 NOT NULL，INSERT 必须显式提供，项目代码均显式赋值）。
var longTextColumns = map[string]bool{
	"content": true, "description": true, "detail": true, "notes": true,
	"summary": true, "body": true, "text": true, "message": true,
	"answer": true, "payload": true, "remark": true, "materials": true,
	"data": true, "reason": true, "suggestion": true, "html": true,
	"markdown": true, "feedback": true, "screenshot": true, "report": true,
	"extra": true, "input": true, "output": true, "record": true,
	"plan": true, "template": true, "script": true, "sql": true,
	"code": true, "analysis": true, "snapshot": true, "metadata": true,
	"trace": true, "title": true, "query": true,
	"question": true, "context": true, "message_text": true,
	// 实际数据超过 MySQL TEXT(64KB) 上限的列 → LONGTEXT（见 longLongTextColumns）
	"analysis_json": true, "gap_analysis": true,
}

// longLongTextColumns 已知超长文本列：MySQL TEXT 上限 64KB，这些列必须用 LONGTEXT
// （如 feedback.screenshot_url 实际存储 base64 内联数据，可达数百 KB）
var longLongTextColumns = map[string]bool{
	"screenshot_url": true,
}

// keyTextColumns 虽然语义上可能是文本，但参与 UNIQUE/键约束、需保留默认值的列，
// 不能留在 longTextColumns 中（MySQL TEXT 不能作键）。这些列按短文本转 VARCHAR(255)。
var keyTextColumns = map[string]bool{
	"code": true,
}

var (
	// 时间列：SQLite `TEXT DEFAULT (datetime('now'))` / `datetime('now','localtime')`
	// → MySQL `DATETIME DEFAULT CURRENT_TIMESTAMP`
	timeColRe = regexp.MustCompile(`(?i)\b([a-z_][a-z0-9_]*)\s+TEXT(\s+NOT\s+NULL)?\s+DEFAULT\s+\(datetime\('now'(,\s*'localtime')?\)\)`)
	// datetime('now','localtime') 表达式 → CURRENT_TIMESTAMP
	nowLocalRe = regexp.MustCompile(`(?i)datetime\('now'\s*,\s*'localtime'\)`)
	// datetime('now', '+N unit') 表达式 → DATE_ADD(NOW(), INTERVAL N UNIT)
	nowModRe = regexp.MustCompile(`(?i)datetime\('now'\s*,\s*'([+-]?\d+)\s*([a-z]+)'\)`)
	// 主键自增：SQLite `INTEGER PRIMARY KEY AUTOINCREMENT` → MySQL `BIGINT PRIMARY KEY AUTO_INCREMENT`
	pkAutoRe = regexp.MustCompile(`(?i)INTEGER\s+PRIMARY\s+KEY\s+AUTOINCREMENT`)
	// 剩余 AUTOINCREMENT
	autoIncRe = regexp.MustCompile(`(?i)AUTOINCREMENT`)
	// 其余 INTEGER 列类型 → BIGINT（与自增主键 BIGINT 保持一致，避免外键类型不匹配）
	integerRe = regexp.MustCompile(`(?i)\bINTEGER\b`)
	// TEXT 带默认值：`TEXT [NOT NULL] DEFAULT 'xxx'`（短文本列 → VARCHAR(255)）
	// 列名允许带反引号（如 `` `references` ``），词边界在反引号之后
	textDefaultRe = regexp.MustCompile("(?i)\\b(`?[a-z_][a-z0-9_]*`?)\\s+TEXT(\\s+NOT\\s+NULL)?\\s+DEFAULT\\s+('[^']*'|\"[^\"]*\"|\\w+|\\d+)")
	// 剩余 TEXT 列（无默认值/作键/作索引）：MySQL 不允许 TEXT 直接建键或索引，
	// 且不允许默认值。长文本列（content/description 等）保留 TEXT；其余统一转 VARCHAR(255)。
	plainTextRe = regexp.MustCompile("(?i)\\b(`?[a-z_][a-z0-9_]*`?)\\s+TEXT\\b")
	// BLOB 带默认值：MySQL BLOB 不允许默认值，去掉 DEFAULT
	blobDefaultRe = regexp.MustCompile(`(?i)\b([a-z_][a-z0-9_]*)\s+BLOB(\s+NOT\s+NULL)?\s+DEFAULT\s+('[^']*'|"[^"]*"|\w+|\d+)`)
	// CREATE [UNIQUE] INDEX IF NOT EXISTS → MySQL 8 不支持 IF NOT EXISTS，去掉该子句
	// （重复索引报错由 execSQL 的 "Duplicate key name" 容错跳过）
	createIndexRe = regexp.MustCompile(`(?i)^\s*CREATE\s+(UNIQUE\s+)?INDEX\s+IF\s+NOT\s+EXISTS\s+([a-z_][a-z0-9_]*)\s+ON\s+([a-z_][a-z0-9_]*)\s*(.*)`)
	// INSERT OR IGNORE → INSERT IGNORE
	insertOrIgnoreRe = regexp.MustCompile(`(?i)INSERT\s+OR\s+IGNORE`)
	// SQLite PRAGMA 语句（MySQL 无对应物，跳过）
	pragmaRe = regexp.MustCompile(`(?i)^\s*PRAGMA\s+`)
	// 双引号标识符列定义（SQLite 允许 `"references"` 等）：MySQL 双引号是字符串字面量，
	// 需转为反引号。仅匹配行首的标识符位置，避免误伤字符串内容中的中文引号。
	doubleQuoteIdentRe = regexp.MustCompile(`(?m)^(\s*)"([a-z_][a-z0-9_]*)"`)	// MySQL 保留字列名（key/rank/value 等）在列定义位置加反引号（SQLite 也接受反引号）
	// MySQL 保留字列名（key/rank/value 等）在列定义位置加反引号（SQLite 也接受反引号）
	reservedColRe = regexp.MustCompile(`(?m)^(\s*)(key|rank|value)(\s+(TEXT|INTEGER|BIGINT|VARCHAR|REAL|DATETIME|BLOB))`)
	// INSERT 列清单中的保留字列名加反引号，如 `INSERT INTO system_settings (key, value, ...)`
	insertColsRe = regexp.MustCompile(`(?i)(\bINSERT\s+(?:IGNORE\s+)?INTO\s+[a-z_][a-z0-9_]*\s*\()([^)]*)\)`)
	// 运行时 DML 适配（AdaptForDriver 用）
	// SQLite ON CONFLICT(...) DO UPDATE SET → MySQL ON DUPLICATE KEY UPDATE
	onConflictRe = regexp.MustCompile(`(?i)ON\s+CONFLICT\s*\([^)]*\)\s+DO\s+UPDATE\s+SET`)
	// SQLite excluded.col → MySQL VALUES(col)
	excludedRe = regexp.MustCompile(`(?i)\bexcluded\.([a-z_][a-z0-9_]*)`)
	// SQLite || 拼接 → MySQL CONCAT(a, b)
	concatRe = regexp.MustCompile(`([a-z_][a-z0-9_]*)\s*\|\|\s*(\?|'[^']*'|\w+)`)
)

// AdaptForDriver 将 Go 代码中运行时执行的 SQLite 方言 DML 适配为 MySQL 方言。
// 与 ToMySQL（迁移文件用）互补；非 MySQL 方言原样返回。
// 处理：ON CONFLICT ... DO UPDATE SET → ON DUPLICATE KEY UPDATE、excluded.col → VALUES(col)、
// datetime('now','localtime') → NOW()、|| 拼接 → CONCAT。
func AdaptForDriver(stmt string, driver Driver) string {
	if driver != DriverMySQL {
		return stmt
	}
	s := stmt
	s = strings.ReplaceAll(s, "datetime('now','localtime')", "NOW()")
	s = onConflictRe.ReplaceAllString(s, "ON DUPLICATE KEY UPDATE")
	s = excludedRe.ReplaceAllString(s, "VALUES($1)")
	s = concatRe.ReplaceAllString(s, "CONCAT($1, $2)")
	return s
}

// ToMySQL 将单条 SQLite DDL/DML 语句转换为 MySQL 方言。
// 仅处理迁移文件中的常见差异；FTS5 虚拟表/触发器由调用方跳过。
// PRAGMA 等 SQLite 专有语句返回空串（由调用方跳过）。
func ToMySQL(stmt string) string {
	// PRAGMA（如 foreign_keys）在 MySQL 无对应物，跳过
	if pragmaRe.MatchString(stmt) {
		return ""
	}

	// 0. 双引号标识符列定义 → 反引号（如 `"references"` 列）。
	// 必须最先执行：否则后续 TEXT 默认值/键列处理因列名前有双引号而无法匹配。
	s := doubleQuoteIdentRe.ReplaceAllString(stmt, "${1}`${2}`")

	// 1. 时间列：TEXT DEFAULT (CURRENT_TIMESTAMP) → DATETIME DEFAULT CURRENT_TIMESTAMP
	s = timeColRe.ReplaceAllString(s, "${1} DATETIME${2} DEFAULT CURRENT_TIMESTAMP")

	// 2. 主键自增
	s = pkAutoRe.ReplaceAllString(s, "BIGINT PRIMARY KEY AUTO_INCREMENT")
	s = autoIncRe.ReplaceAllString(s, "AUTO_INCREMENT")
	// 2.5 其余 INTEGER → BIGINT（外键引用类型一致）
	s = integerRe.ReplaceAllString(s, "BIGINT")

	// 3. datetime('now') 及变体 → 双方言通用时间表达式
	// 3a. datetime('now', 'localtime') → CURRENT_TIMESTAMP
	s = nowLocalRe.ReplaceAllString(s, "CURRENT_TIMESTAMP")
	// 3b. datetime('now', '+N unit') → DATE_ADD(NOW(), INTERVAL N UNIT)
	s = nowModRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := nowModRe.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		num, unit := sub[1], strings.TrimSuffix(strings.ToUpper(sub[2]), "S")
		return fmt.Sprintf("DATE_ADD(NOW(), INTERVAL %s %s)", num, unit)
	})
	// 3c. 剩余 datetime('now') → CURRENT_TIMESTAMP
	s = strings.ReplaceAll(s, "datetime('now')", "CURRENT_TIMESTAMP")

	// 4. TEXT 默认值处理
	s = textDefaultRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := textDefaultRe.FindStringSubmatch(m)
		if len(sub) < 4 {
			return m
		}
		col := strings.ToLower(sub[1])
		notNull := sub[2]
		def := sub[3]
		if longLongTextColumns[col] {
			// 超长文本：保留 LONGTEXT，去掉默认值（MySQL TEXT/BLOB 不允许 DEFAULT）
			return fmt.Sprintf("%s LONGTEXT%s", sub[1], notNull)
		}
		if longTextColumns[col] {
			// 长文本：保留 TEXT，去掉默认值（MySQL TEXT 不允许 DEFAULT）
			return fmt.Sprintf("%s TEXT%s", sub[1], notNull)
		}
		// 短文本：转为 VARCHAR(128) 保留默认值（utf8mb4 下 128*4=512B，复合唯一键 4 列 < 3072B 上限）
		return fmt.Sprintf("%s VARCHAR(128)%s DEFAULT %s", sub[1], notNull, def)
	})

	// 4.5 剩余 TEXT 列：长文本保留 TEXT，其余（含键列/索引列）转 VARCHAR(128)
	// 必须在 textDefaultRe 之后：带默认值的短文本列此时已是 VARCHAR，不会重复匹配；
	// 带默认值的长文本列已是 TEXT（去默认值），这里按列名判定保留 TEXT。
	s = plainTextRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := plainTextRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		col := strings.ToLower(sub[1])
		if longLongTextColumns[col] {
			return fmt.Sprintf("%s LONGTEXT", sub[1])
		}
		// 参与键约束的列（code 等）即使语义上是文本也不能保留 TEXT
		if longTextColumns[col] && !keyTextColumns[col] {
			return fmt.Sprintf("%s TEXT", sub[1])
		}
		// utf8mb4 下 128*4=512B，复合唯一键 4 列 < InnoDB 3072B 上限
		return fmt.Sprintf("%s VARCHAR(128)", sub[1])
	})

	// 4.6 保留字列名（key/rank/value）加反引号（MySQL 8 中 KEY/RANK/VALUE 为保留字）
	s = reservedColRe.ReplaceAllString(s, "${1}`${2}`${3}")

	// 5. BLOB 默认值：去掉 DEFAULT
	s = blobDefaultRe.ReplaceAllString(s, "${1} BLOB${2}")

	// 6. CREATE INDEX IF NOT EXISTS → 去掉 IF NOT EXISTS
	s = createIndexRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := createIndexRe.FindStringSubmatch(m)
		if len(sub) < 5 {
			return m
		}
		unique, idx, tbl, rest := sub[1], sub[2], sub[3], sub[4]
		return fmt.Sprintf("CREATE %sINDEX %s ON %s%s", unique, idx, tbl, rest)
	})

	// 7. INSERT OR IGNORE → INSERT IGNORE
	s = insertOrIgnoreRe.ReplaceAllString(s, "INSERT IGNORE")

	// 8. INSERT 列清单中的保留字列名加反引号（如 system_settings 的 key/value）
	// 必须在步骤 7 之后：INSERT OR IGNORE → INSERT IGNORE 转换完成后再识别列清单
	s = insertColsRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := insertColsRe.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		prefix, cols := sub[1], sub[2]
		parts := strings.Split(cols, ",")
		for i, p := range parts {
			p = strings.TrimSpace(p)
			switch strings.ToLower(p) {
			case "key", "value", "rank":
				parts[i] = "`" + p + "`"
			}
		}
		return prefix + strings.Join(parts, ", ") + ")"
	})

	return s
}
