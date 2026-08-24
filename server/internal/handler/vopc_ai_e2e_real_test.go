package handler

// vOPC 虚拟向导模板渲染确定性验证（v2.0）：不再调用真实 DeepSeek。
// 原 v1.0 真实端到端 E2E 已停用（L4 真实 AI 执行不实现），改为验证模板渲染确定性。

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestVOPCB1VirtualGuideTemplateDeterministic 验证模板渲染确定性且含预期结构。
// 无外部模型、无 .env 依赖，结果可复现。
func TestVOPCB1VirtualGuideTemplateDeterministic(t *testing.T) {
	db := vopcB1DB(t)
	r := vopcB1Router(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	id := mustA4Project(t, r, owner)

	instruction := "为校园失物招领小程序撰写需求基线"
	w := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/ai-tasks", id), owner, map[string]any{
		"role_key": "project_manager", "instruction": instruction,
	})
	if w.Code != 201 {
		t.Fatalf("virtual guide create got %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
			Output string `json:"output_content"`
			Model  string `json:"model"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out.Data.Status != "succeeded" {
		t.Fatalf("status=%s want succeeded", out.Data.Status)
	}
	if out.Data.Model != "virtual_guide" {
		t.Fatalf("model=%s want virtual_guide", out.Data.Model)
	}
	// 草稿必须包含结构化骨架与用户指令回显，且不依赖任何外部调用。
	for _, frag := range []string{"产品经理向导", "结构化草稿", instruction} {
		if !strings.Contains(out.Data.Output, frag) {
			t.Fatalf("draft missing %q: %s", frag, out.Data.Output)
		}
	}
}
