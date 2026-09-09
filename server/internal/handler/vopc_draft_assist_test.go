package handler

import (
	"encoding/json"
	"testing"
)

func TestAssistProjectDraftReturnsEditableGroundedFields(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	student := token(t, 1, "student", "college", "cs", "active")
	w := request(r, "POST", "/api/v1/vopc/project-draft/assist", student, map[string]any{
		"idea":         "帮助新生找到可靠的校园办事信息",
		"project_type": "校园服务创新",
	})
	if w.Code != 200 {
		t.Fatalf("assist got %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			Fields         map[string]string `json:"fields"`
			DataSource     string            `json:"data_source"`
			ReviewRequired bool              `json:"review_required"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"name", "summary", "problem_statement", "target_users", "validation_plan", "acceptance_criteria"} {
		if out.Data.Fields[key] == "" {
			t.Fatalf("missing suggested field %q", key)
		}
	}
	if out.Data.DataSource != "template" || !out.Data.ReviewRequired {
		t.Fatalf("unsafe metadata: source=%q review=%v", out.Data.DataSource, out.Data.ReviewRequired)
	}
}

func TestAssistProjectDraftRequiresIdea(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	student := token(t, 1, "student", "college", "cs", "active")
	w := request(r, "POST", "/api/v1/vopc/project-draft/assist", student, map[string]any{"idea": "  "})
	if w.Code != 422 {
		t.Fatalf("empty idea got %d: %s", w.Code, w.Body.String())
	}
}
