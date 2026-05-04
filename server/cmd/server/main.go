package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// 加载 .env（开发环境）
	_ = godotenv.Load("../../.env")

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("蔚小芯后端启动中 port=%s ...", port)

	// 待实现：初始化配置、数据库、路由、启动服务
	// router := setupRouter()
	// router.Run(":" + port)
}
