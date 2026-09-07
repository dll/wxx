package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestCollegeTwinScreenFallbackIsHonest 确保服务不可用时返回诚实空态，
// 不再把固定学生数、健康度、风险数或专业示例伪装为真实画像。
func TestCollegeTwinScreenFallbackIsHonest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollegeHandler(nil)
	r.GET("/twin-screen", h.TwinScreen)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/twin-screen", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}

	var body struct {
		Overview    map[string]interface{}   `json:"overview"`
		Departments []map[string]interface{} `json:"departments"`
		Trends      map[string]interface{}   `json:"trends"`
		FiveDim     interface{}              `json:"five_dim"`
		AIInsight   string                   `json:"ai_insight"`
		DataSource  string                   `json:"data_source"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if body.DataSource != "fallback" {
		t.Fatalf("服务不可用时应标注 fallback，得到 %q", body.DataSource)
	}
	for _, key := range []string{"total_students", "health_score", "risk_students", "active_rate"} {
		value, ok := body.Overview[key].(float64)
		if !ok || value != 0 {
			t.Errorf("overview.%s 应为诚实零值，得到 %#v", key, body.Overview[key])
		}
	}
	if len(body.Departments) != 0 || len(body.Trends) != 0 || body.FiveDim != nil || body.AIInsight != "" {
		t.Errorf("画像衍生字段应为空，得到 departments=%v trends=%v five_dim=%v ai=%q",
			body.Departments, body.Trends, body.FiveDim, body.AIInsight)
	}
}
