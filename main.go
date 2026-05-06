// 蔚小芯 — 本地开发入口（Vercel 部署使用 api/index.go）
// 用法: go run . 或 make dev
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/dll/wxx/server/pkg/app"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler, err := app.New()
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	log.Printf("蔚小芯服务已启动 → http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
