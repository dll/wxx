package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dll/wxx/server/pkg/app"
)

var handler http.Handler
var initError error

func init() {
	var err error
	handler, err = app.New()
	if err != nil {
		initError = err
		log.Printf("===== 初始化失败 =====")
		log.Printf("错误: %v", err)
		log.Printf("VERCEL=%q", os.Getenv("VERCEL"))
		log.Printf("===== 初始化失败结束 =====")
	}
}

// Handler Vercel serverless 入口
func Handler(w http.ResponseWriter, r *http.Request) {
	if initError != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    500,
			"message": "后端初始化失败",
			"detail":  fmt.Sprintf("%v", initError),
		})
		return
	}
	handler.ServeHTTP(w, r)
}
