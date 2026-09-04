package app

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dll/wxx/server/internal/config"
	"github.com/gin-gonic/gin"
)

// registerBaseRoutes 注册根路由、健康检查和可选的 Flutter 静态资源。
// API 路由仍由 routes.go 统一注册，保证中间件和路径顺序不变。
func registerBaseRoutes(router *gin.Engine, cfg *config.Config, health gin.HandlerFunc) {
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "蔚小芯",
			"version": "0.0.1",
			"docs":    "/health",
		})
	})
	router.GET("/health", health)

	staticDir := cfg.FrontendStaticDir
	if staticDir == "" {
		return
	}
	if _, err := os.Stat(staticDir); err != nil {
		log.Printf("警告: 前端静态目录不存在，跳过静态文件服务: %s", staticDir)
		return
	}

	noCachePaths := map[string]bool{
		"/main.dart.js":              true,
		"/index.html":                true,
		"/flutter_bootstrap.js":      true,
		"/flutter_service_worker.js": true,
	}
	router.Use(func(c *gin.Context) {
		if noCachePaths[c.Request.URL.Path] {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		c.Next()
	})
	router.NoRoute(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/") && !strings.HasPrefix(c.Request.URL.Path, "/health") {
			indexPath := filepath.Join(staticDir, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
				c.File(indexPath)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "接口不存在"})
	})
	router.Static("/assets", filepath.Join(staticDir, "assets"))
	router.Static("/canvaskit", filepath.Join(staticDir, "canvaskit"))
	router.StaticFile("/index.html", filepath.Join(staticDir, "index.html"))
	router.StaticFile("/main.dart.js", filepath.Join(staticDir, "main.dart.js"))
	router.StaticFile("/favicon.png", filepath.Join(staticDir, "favicon.png"))
	router.StaticFile("/favicon.ico", filepath.Join(staticDir, "favicon.ico"))
	router.StaticFile("/flutter_bootstrap.js", filepath.Join(staticDir, "flutter_bootstrap.js"))
	router.StaticFile("/flutter_service_worker.js", filepath.Join(staticDir, "flutter_service_worker.js"))
	router.StaticFile("/manifest.json", filepath.Join(staticDir, "manifest.json"))
	router.StaticFile("/version.json", filepath.Join(staticDir, "version.json"))
	log.Printf("前端静态文件已挂载: %s", staticDir)
}
