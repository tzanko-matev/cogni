package agent

import "testing"

func TestCompactHistoryKeepsRequiredItems(t *testing.T) {
	history := []HistoryItem{
		{Role: "developer", Content: "dev"},
		{Role: "user", Content: "# AGENTS.md instructions for /repo\n\n<INSTRUCTIONS>\nuser\n</INSTRUCTIONS>"},
		{Role: "user", Content: "<environment_context>\n  <cwd>/repo</cwd>\n</environment_context>"},
		{Role: "user", Content: "question one"},
		{Role: "assistant", Content: "answer one"},
		{Role: "tool", Content: "search output"},
		{Role: "user", Content: "question two"},
		{Role: "user", Content: "<environment_diff>\n  <cwd>/repo</cwd>\n</environment_diff>"},
		{Role: "tool", Content: "read_file output"},
	}

	compacted := CompactHistory(history, func(_ []HistoryItem) int { return 100 }, 10)
	if len(compacted) != 6 {
		t.Fatalf("expected 6 items, got %d", len(compacted))
	}

	expectedRoles := []string{"developer", "user", "tool", "user", "user", "tool"}
	for i, role := range expectedRoles {
		if compacted[i].Role != role {
			t.Fatalf("unexpected role at %d: %s", i, compacted[i].Role)
		}
	}
	if compacted[3].Content != "question two" {
		t.Fatalf("expected latest user message, got %v", compacted[3].Content)
	}
}
