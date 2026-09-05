package service

import "testing"

func TestParseKnowledgeTags(t *testing.T) {
	if got := parseKnowledgeTags(`["学业","政策"]`); len(got) != 2 || got[0] != "学业" {
		t.Fatalf("valid tags parse failed: %#v", got)
	}
	if got := parseKnowledgeTags("not-json"); len(got) != 0 {
		t.Fatalf("invalid tags should be empty: %#v", got)
	}
	if got := parseKnowledgeTags(" "); got == nil {
		t.Fatal("empty tags should return non-nil empty slice")
	}
}
