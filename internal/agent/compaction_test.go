package agent

import "testing"

// TestCompactHistoryKeepsRequiredItems verifies required history items are retained.
func TestCompactHistoryKeepsRequiredItems(t *testing.T) {
	history := []HistoryItem{
		{Role: "developer", Content: HistoryText{Text: "dev"}},
		{Role: "user", Content: HistoryText{Text: "# AGENTS.md instructions for /repo\n\n<INSTRUCTIONS>\nuser\n</INSTRUCTIONS>"}},
		{Role: "user", Content: HistoryText{Text: "<environment_context>\n  <cwd>/repo</cwd>\n</environment_context>"}},
		{Role: "user", Content: HistoryText{Text: "question one"}},
		{Role: "assistant", Content: HistoryText{Text: "answer one"}},
		{Role: "tool", Content: HistoryText{Text: "search output"}},
		{Role: "user", Content: HistoryText{Text: "question two"}},
		{Role: "user", Content: HistoryText{Text: "<environment_diff>\n  <cwd>/repo</cwd>\n</environment_diff>"}},
		{Role: "tool", Content: HistoryText{Text: "read_file output"}},
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
	if compacted[3].Content != (HistoryText{Text: "question two"}) {
		t.Fatalf("expected latest user message, got %v", compacted[3].Content)
	}
}
