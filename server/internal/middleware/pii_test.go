package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPIIMask_DetectsButDoesNotModifyRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(PIIMask())
	router.POST("/login", func(c *gin.Context) {
		var request map[string]string
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"username":     request["username"],
			"pii_detected": c.GetBool("pii_detected"),
		})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"username":"2023211981","password":"2023211981"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("请求失败: %d %s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if response["username"] != "2023211981" {
		t.Fatalf("请求体不应被脱敏修改: %#v", response)
	}
	if response["pii_detected"] != true {
		t.Fatalf("应记录 PII 检测结果: %#v", response)
	}
}
