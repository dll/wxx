// Package api 是 Vercel serverless 函数入口，将所有 HTTP 请求转交给 Gin 引擎处理。
package api

import (
	"net/http"

	"github.com/dll/wxx/server/pkg/app"
)

var handler http.Handler

func init() {
	h, err := app.New()
	if err != nil {
		panic("蔚小芯初始化失败: " + err.Error())
	}
	handler = h
}

// Handler 接收所有 HTTP 请求并转交 Gin 路由。
func Handler(w http.ResponseWriter, r *http.Request) {
	handler.ServeHTTP(w, r)
}
