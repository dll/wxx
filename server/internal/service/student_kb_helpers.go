package service

import (
	"encoding/json"
	"strings"
)

// parseKnowledgeTags decodes the JSON tags stored on a knowledge resource.
// Invalid or empty values intentionally produce an empty slice for safe UI use.
func parseKnowledgeTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return []string{}
	}
	return tags
}

// parseTags remains as a package-level compatibility alias for KB services.
func parseTags(raw string) []string { return parseKnowledgeTags(raw) }
