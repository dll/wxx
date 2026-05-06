// Package api 提供 Vercel serverless 函数入口。
// Vercel Go 运行时将每个请求路由到此 Handler。
package api

import (
	"net/http"

	"github.com/dll/wxx/server/pkg/app"
)

var handler http.Handler

func init() {
	h, err := app.New()
	if err != nil {
		panic("初始化应用失败: " + err.Error())
	}
	handler = h
}

// Handler 是 Vercel serverless 入口，接收所有 HTTP 请求。
func Handler(w http.ResponseWriter, r *http.Request) {
	handler.ServeHTTP(w, r)
}
