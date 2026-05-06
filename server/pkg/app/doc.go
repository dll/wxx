// Package app 封装 Gin 初始化 + 迁移 + 路由，供本地 main.go 和 Vercel serverless 入口复用。
// 核心函数 New(cfg) 返回完全初始化的 http.Handler（*gin.Engine）。
package app
