package api

import (
	"log"
	"net/http"

	"github.com/dll/wxx/server/pkg/app"
)

var handler http.Handler

func init() {
	var err error
	handler, err = app.New()
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}
}

// Handler Vercel serverless 入口
func Handler(w http.ResponseWriter, r *http.Request) {
	handler.ServeHTTP(w, r)
}
