package main

import (
	"log"
	"os"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/pkg/app"
)

func main() {
	log.Println("蔚小芯 数据库同步工具 (SQLite ↔ Turso)")
	log.Println("========================================")

	// 强制 debug 模式，避免 JWT 等验证干扰数据同步
	os.Setenv("APP_MODE", "debug")
	cfg := config.Load()

	if dsn := cfg.TursoDSN(); dsn == "" {
		log.Println("错误: TURSO_DB_URL 或 TURSO_DB_TOKEN 未配置")
		log.Println("请在 .env 中添加:")
		log.Println("  TURSO_DB_URL=libsql://your-db.turso.io")
		log.Println("  TURSO_DB_TOKEN=your-token")
		os.Exit(1)
	}

	if err := app.SyncDB(cfg); err != nil {
		log.Fatalf("同步失败: %v", err)
	}
}
