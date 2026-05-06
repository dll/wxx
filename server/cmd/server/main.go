package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/pkg/app"
)

func main() {
	cfg := config.Load()

	// 设置运行模式
	if cfg.AppMode == "release" {
		// gin.SetMode 在 app 包内设置，此处仅用于日志
		log.Println("运行模式: release")
	}

	// 初始化应用（含 DB、迁移、路由）
	handler, err := app.NewWithConfig(cfg)
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}

	// 启动 HTTP 服务（支持优雅关闭）
	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: handler,
	}

	go func() {
		log.Printf("蔚小芯服务已启动 → http://localhost:%s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 等待中断信号，优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务关闭异常: %v", err)
	}
	log.Println("蔚小芯服务已安全退出")
}
