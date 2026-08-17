package app

// 健康检查与占位 handler（从原 app.go 拆分，行为不变）

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"time"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/gin-gonic/gin"
)

func healthHandler(db *sql.DB) gin.HandlerFunc {
	startTime := time.Now()
	return func(c *gin.Context) {
		// 数据库连通性
		dbStatus := "ok"
		dbLatency := ""
		t0 := time.Now()
		if err := db.Ping(); err != nil {
			dbStatus = "error: " + err.Error()
		} else {
			dbLatency = time.Since(t0).String()
		}

		// FTS5 可用性（MySQL 无 FTS5 虚拟表，报 unavailable）
		ftsStatus := "ok"
		if !dbutil.IsMySQL(db) {
			var ftsCheck int
			if err := db.QueryRow("SELECT 1 FROM kb_fts LIMIT 1").Scan(&ftsCheck); err != nil {
				ftsStatus = "unavailable"
			}
		} else {
			ftsStatus = "unavailable (mysql)"
		}

		// LLM API 配置（仅检查 key 是否配置，不实际调用）
		llmStatus := "configured"
		if os.Getenv("ZHIPU_API_KEY") == "" && os.Getenv("DEEPSEEK_API_KEY") == "" && os.Getenv("SPARK_API_KEY") == "" {
			llmStatus = "no_api_key"
		}

		// Redis 状态（可选组件）
		redisStatus := "disabled"
		if appRedis != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			if err := appRedis.Ping(ctx).Err(); err != nil {
				redisStatus = "error: " + err.Error()
			} else {
				redisStatus = "ok"
			}
		}

		// 总体状态
		overall := "healthy"
		if dbStatus != "ok" {
			overall = "degraded"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  overall,
			"service": "蔚小芯",
			"version": "0.0.1",
			"uptime":  time.Since(startTime).String(),
			"dependencies": gin.H{
				"database": gin.H{"status": dbStatus, "latency": dbLatency, "driver": string(dbutil.DriverOf(db))},
				"redis":    gin.H{"status": redisStatus},
				"fts5":     gin.H{"status": ftsStatus},
				"llm_api":  gin.H{"status": llmStatus},
			},
			"time": time.Now().Format(time.RFC3339),
		})
	}
}

func placeholderHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"code":    501,
			"message": name + " 待实现",
		})
	}
}
