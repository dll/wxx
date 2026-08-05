package app

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// SyncDB 同步本地 SQLite 到 Turso：schema + 数据双向同步
func SyncDB(cfg *config.Config) error {
	dsn := cfg.TursoDSN()
	if dsn == "" {
		return fmt.Errorf("TURSO_DB_URL 或 TURSO_DB_TOKEN 未配置")
	}

	log.Println("正在连接本地 SQLite...")
	localDB, err := initDB(cfg.SQLitePath)
	if err != nil {
		return fmt.Errorf("本地 SQLite 连接失败: %w", err)
	}
	defer localDB.Close()

	log.Println("正在连接 Turso 云数据库...")
	remoteDB, err := initDB(dsn)
	if err != nil {
		return fmt.Errorf("Turso 连接失败: %w", err)
	}
	defer remoteDB.Close()

	log.Println("━━━ Phase 1: Schema 迁移（Turso） ━━━")
	if err := runMigrations(remoteDB, true); err != nil {
		return fmt.Errorf("Turso 迁移失败: %w", err)
	}

	log.Println("━━━ Phase 2: 数据同步 ━━━")
	if err := syncAllData(localDB, remoteDB); err != nil {
		return fmt.Errorf("数据同步失败: %w", err)
	}

	log.Println("━━━ Phase 3: 密码哈希修复 ━━━")
	if err := fixPasswordHashes(localDB); err != nil {
		return fmt.Errorf("本地密码修复失败: %w", err)
	}
	if err := fixPasswordHashes(remoteDB); err != nil {
		return fmt.Errorf("远程密码修复失败: %w", err)
	}

	log.Println("Turso 同步完成 ✓")
	return nil
}

func syncAllData(localDB, remoteDB *sql.DB) error {
	tables := []struct {
		name     string
		bizKey   string // 业务唯一键（空=用 id）
		strategy string // last_write_wins | one_way | append_union
	}{
		{"users", "username", "last_write_wins"},
		{"kb_resources", "resource_id", "last_write_wins"},
		{"agents", "agent_id", "last_write_wins"},
		{"sessions", "", "one_way"},
		{"messages", "", "append_union"},
		{"process_steps", "", "one_way"},
		{"audit_logs", "", "append_union"},
		{"emotion_logs", "", "append_union"},
		{"export_logs", "", "append_union"},
	}

	for _, t := range tables {
		log.Printf("  同步表: %s (%s)", t.name, t.strategy)
		start := time.Now()
		var err error
		switch t.strategy {
		case "last_write_wins":
			err = syncLastWriteWins(localDB, remoteDB, t.name, t.bizKey)
		case "one_way":
			err = syncOneWay(localDB, remoteDB, t.name)
		case "append_union":
			err = syncAppendUnion(localDB, remoteDB, t.name)
		}
		if err != nil {
			return fmt.Errorf("表 %s 同步失败: %w", t.name, err)
		}
		log.Printf("  ✓ %s 完成 (%v)", t.name, time.Since(start))
	}
	return nil
}

func syncLastWriteWins(localDB, remoteDB *sql.DB, table, bizKey string) error {
	localRows, err := queryAllRows(localDB, table)
	if err != nil {
		return err
	}
	remoteRows, err := queryAllRows(remoteDB, table)
	if err != nil {
		return err
	}

	localByKey := indexRows(localRows, bizKey)
	remoteByKey := indexRows(remoteRows, bizKey)

	if len(localByKey) == 0 && len(remoteByKey) == 0 {
		return nil
	}

	var cols []string
	if len(localRows) > 0 {
		cols = localRows[0].cols
	} else {
		cols = remoteRows[0].cols
	}

	txLocal, err := localDB.Begin()
	if err != nil {
		return err
	}
	defer txLocal.Rollback()

	txRemote, err := remoteDB.Begin()
	if err != nil {
		return err
	}
	defer txRemote.Rollback()

	for key, localRow := range localByKey {
		remoteRow, exists := remoteByKey[key]
		if !exists {
			if err := insertRow(txRemote, table, cols, localRow.values, bizKey, key); err != nil {
				return fmt.Errorf("远程 INSERT %s[%s] 失败: %w", table, key, err)
			}
			log.Printf("    -> 远程新增 %s[%s]", table, key)
		} else {
			localTime := parseTime(localRow.values["updated_at"])
			remoteTime := parseTime(remoteRow.values["updated_at"])
			if localTime.After(remoteTime) {
				if err := updateRowByKey(txRemote, table, cols, bizKey, key, localRow.values); err != nil {
					return fmt.Errorf("远程 UPDATE %s[%s] 失败: %w", table, key, err)
				}
				log.Printf("    -> 远程更新 %s[%s] (本地更新)", table, key)
			}
		}
	}

	for key, remoteRow := range remoteByKey {
		if _, exists := localByKey[key]; exists {
			continue
		}
		if err := insertRow(txLocal, table, cols, remoteRow.values, bizKey, key); err != nil {
			return fmt.Errorf("本地 INSERT %s[%s] 失败: %w", table, key, err)
		}
		log.Printf("    <- 本地新增 %s[%s]", table, key)
	}

	if err := txLocal.Commit(); err != nil {
		return err
	}
	return txRemote.Commit()
}

func syncOneWay(localDB, remoteDB *sql.DB, table string) error {
	localRows, err := queryAllRows(localDB, table)
	if err != nil {
		return err
	}
	remoteRows, err := queryAllRows(remoteDB, table)
	if err != nil {
		return err
	}

	remotePKs := make(map[string]bool)
	for _, r := range remoteRows {
		remotePKs[r.key] = true
	}

	if len(localRows) == 0 {
		return nil
	}
	cols := localRows[0].cols

	txRemote, err := remoteDB.Begin()
	if err != nil {
		return err
	}
	defer txRemote.Rollback()

	for _, row := range localRows {
		if remotePKs[row.key] {
			continue
		}
		if err := insertRow(txRemote, table, cols, row.values, "", row.key); err != nil {
			return fmt.Errorf("远程 INSERT %s[%s] 失败: %w", table, row.key, err)
		}
		log.Printf("    -> 远程新增 %s[%s]", table, row.key)
	}

	return txRemote.Commit()
}

func syncAppendUnion(localDB, remoteDB *sql.DB, table string) error {
	localRows, err := queryAllRows(localDB, table)
	if err != nil {
		return err
	}
	remoteRows, err := queryAllRows(remoteDB, table)
	if err != nil {
		return err
	}

	remotePKs := make(map[string]bool)
	for _, r := range remoteRows {
		remotePKs[r.key] = true
	}
	localPKs := make(map[string]bool)
	for _, r := range localRows {
		localPKs[r.key] = true
	}

	var cols []string
	if len(localRows) > 0 {
		cols = localRows[0].cols
	} else if len(remoteRows) > 0 {
		cols = remoteRows[0].cols
	} else {
		return nil
	}

	txLocal, err := localDB.Begin()
	if err != nil {
		return err
	}
	defer txLocal.Rollback()

	txRemote, err := remoteDB.Begin()
	if err != nil {
		return err
	}
	defer txRemote.Rollback()

	for _, row := range localRows {
		if remotePKs[row.key] {
			continue
		}
		if err := insertRow(txRemote, table, cols, row.values, "", row.key); err != nil {
			return fmt.Errorf("远程 INSERT %s[%s] 失败: %w", table, row.key, err)
		}
		log.Printf("    -> 远程新增 %s[%s]", table, row.key)
	}

	for _, row := range remoteRows {
		if localPKs[row.key] {
			continue
		}
		if err := insertRow(txLocal, table, cols, row.values, "", row.key); err != nil {
			return fmt.Errorf("本地 INSERT %s[%s] 失败: %w", table, row.key, err)
		}
		log.Printf("    <- 本地新增 %s[%s]", table, row.key)
	}

	if err := txLocal.Commit(); err != nil {
		return err
	}
	return txRemote.Commit()
}

// ── 辅助类型 ──

type rowData struct {
	key    string
	cols   []string
	values map[string]string
}

func queryAllRows(db *sql.DB, table string) ([]rowData, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", quoteIdent(table)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []rowData
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		valPtrs := make([]interface{}, len(cols))
		for i := range vals {
			valPtrs[i] = &vals[i]
		}
		if err := rows.Scan(valPtrs...); err != nil {
			return nil, err
		}

		row := rowData{
			cols:   cols,
			values: make(map[string]string, len(cols)),
		}
		for i, col := range cols {
			if vals[i] == nil {
				row.values[col] = ""
			} else {
				switch v := vals[i].(type) {
				case []byte:
					row.values[col] = string(v)
				default:
					row.values[col] = fmt.Sprintf("%v", v)
				}
			}
		}
		// key 默认用第一列（通常是 id）
		if len(cols) > 0 {
			row.key = row.values[cols[0]]
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func indexRows(rows []rowData, bizKey string) map[string]rowData {
	m := make(map[string]rowData, len(rows))
	for _, r := range rows {
		key := r.key
		if bizKey != "" {
			if v, ok := r.values[bizKey]; ok && v != "" {
				key = v
			}
		}
		r.key = key
		m[key] = r
	}
	return m
}

func insertRow(tx *sql.Tx, table string, cols []string, values map[string]string, bizKey, bizKeyValue string) error {
	var insertCols []string
	var placeholders []string
	var args []interface{}

	hasPK := false
	for _, col := range cols {
		if col == "id" {
			hasPK = true
			continue
		}
		insertCols = append(insertCols, quoteIdent(col))
		placeholders = append(placeholders, "?")
		args = append(args, values[col])
	}

	if len(insertCols) == 0 {
		return nil
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdent(table),
		strings.Join(insertCols, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := tx.Exec(query, args...)
	if err != nil && hasPK && bizKey != "" {
		return tryUpdateOnConflict(tx, table, cols, values, bizKey, bizKeyValue)
	}
	return err
}

func tryUpdateOnConflict(tx *sql.Tx, table string, cols []string, values map[string]string, bizKey, bizKeyValue string) error {
	var setClauses []string
	var args []interface{}
	for _, col := range cols {
		if col == "id" || col == bizKey {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", quoteIdent(col)))
		args = append(args, values[col])
	}
	args = append(args, bizKeyValue)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
		quoteIdent(table),
		strings.Join(setClauses, ", "),
		quoteIdent(bizKey),
	)

	_, err := tx.Exec(query, args...)
	return err
}

func updateRowByKey(tx *sql.Tx, table string, cols []string, bizKey, keyValue string, values map[string]string) error {
	var setClauses []string
	var args []interface{}
	for _, col := range cols {
		if col == "id" || col == bizKey {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", quoteIdent(col)))
		args = append(args, values[col])
	}
	args = append(args, keyValue)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
		quoteIdent(table),
		strings.Join(setClauses, ", "),
		quoteIdent(bizKey),
	)

	_, err := tx.Exec(query, args...)
	return err
}

func quoteIdent(name string) string {
	return `"` + name + `"`
}

func parseTime(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

// fixPasswordHashes 修复数据库中的密码哈希：
//   - 空哈希 → 替换为 "wxx123456" 的 bcrypt 哈希
//   - $2b$ 前缀 → 重新哈希（兼容 bcryptjs）
func fixPasswordHashes(db *sql.DB) error {
	rows, err := db.Query("SELECT id, password_hash FROM users")
	if err != nil {
		return err
	}
	defer rows.Close()

	type fix struct {
		id   int64
		hash string
	}
	var fixes []fix
	const seedPassword = "wxx123456"

	for rows.Next() {
		var id int64
		var hash string
		if err := rows.Scan(&id, &hash); err != nil {
			return err
		}
		hash = strings.TrimSpace(hash)

		needsFix := hash == "" || strings.HasPrefix(hash, "$2b$")
		if !needsFix {
			continue
		}

		// 验证当前哈希是否匹配种子密码
		if hash != "" {
			if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(seedPassword)); err == nil {
				continue // 哈希有效，无需修复
			}
		}

		fixes = append(fixes, fix{id, hash})
	}

	if len(fixes) == 0 {
		return nil
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(seedPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成密码哈希失败: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, f := range fixes {
		_, err := tx.Exec("UPDATE users SET password_hash = ?, updated_at = datetime('now') WHERE id = ?", string(newHash), f.id)
		if err != nil {
			return fmt.Errorf("更新用户 %d 密码失败: %w", f.id, err)
		}
		log.Printf("  密码重置: user_id=%d (旧哈希: %.20s...)", f.id, f.hash)
	}

	return tx.Commit()
}
