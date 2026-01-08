package agent

import (
	"strings"
	"testing"

	"cogni/internal/testutil"
	"cogni/internal/tools"
)

// TestCompactHistoryRebuildsWithSummary verifies summary insertion and ordering.
func TestCompactHistoryRebuildsWithSummary(t *testing.T) {
	ctx := testutil.Context(t, 0)
	history := []HistoryItem{
		{Role: "developer", Content: HistoryText{Text: "dev"}},
		{Role: "user", Content: HistoryText{Text: "# AGENTS.md instructions for /repo\n\n<INSTRUCTIONS>\nuser\n</INSTRUCTIONS>"}},
		{Role: "user", Content: HistoryText{Text: "<environment_context>\n  <cwd>/repo</cwd>\n</environment_context>"}},
		{Role: "user", Content: HistoryText{Text: "question one"}},
		{Role: "assistant", Content: HistoryText{Text: "answer one"}},
		{Role: "user", Content: HistoryText{Text: "question two"}},
	}
	provider := &fakeProvider{
		streams: [][]StreamEvent{{{Type: StreamEventMessage, Message: "summary text"}}},
	}

	compacted, stats, err := CompactHistory(ctx, history, provider, ApproxTokenCount, CompactionConfig{
		SoftLimit:             1,
		HardLimit:             100,
		RecentUserTokenBudget: 1,
	})
	if err != nil {
		t.Fatalf("compact history: %v", err)
	}
	if stats == nil {
		t.Fatalf("expected compaction stats")
	}
	if len(compacted) != 5 {
		t.Fatalf("expected 5 items, got %d", len(compacted))
	}
	if compacted[3].Content != (HistoryText{Text: "question two"}) {
		t.Fatalf("expected latest user message, got %v", compacted[3].Content)
	}
	summaryItem, ok := compacted[4].Content.(HistoryText)
	if !ok {
		t.Fatalf("expected summary text, got %T", compacted[4].Content)
	}
	if !strings.HasPrefix(summaryItem.Text, SummaryPrefix) {
		t.Fatalf("expected summary prefix, got %q", summaryItem.Text)
	}
}

// TestCompactHistoryRetainsToolOutputs verifies tool output retention rules.
func TestCompactHistoryRetainsToolOutputs(t *testing.T) {
	ctx := testutil.Context(t, 0)
	history := []HistoryItem{
		{Role: "user", Content: HistoryText{Text: "question one"}},
		{Role: "assistant", Content: ToolCall{ID: "call-1", Name: "list_files"}},
		{Role: "tool", Content: ToolOutput{ToolCallID: "call-1", Result: tools.CallResult{Tool: "list_files", Output: "out-1", OutputBytes: 5}}},
		{Role: "assistant", Content: ToolCall{ID: "call-2", Name: "read_file"}},
		{Role: "tool", Content: ToolOutput{ToolCallID: "call-2", Result: tools.CallResult{Tool: "read_file", Output: "out-2", OutputBytes: 5}}},
		{Role: "user", Content: HistoryText{Text: "question two"}},
	}
	provider := &fakeProvider{
		streams: [][]StreamEvent{{{Type: StreamEventMessage, Message: "summary text"}}},
	}

	compacted, _, err := CompactHistory(ctx, history, provider, ApproxTokenCount, CompactionConfig{
		SoftLimit:             1,
		HardLimit:             100,
		RecentUserTokenBudget: 1,
		RecentToolOutputLimit: 1,
	})
	if err != nil {
		t.Fatalf("compact history: %v", err)
	}

	var keptCall ToolCall
	var keptOutput ToolOutput
	for _, item := range compacted {
		switch content := item.Content.(type) {
		case ToolCall:
			keptCall = content
		case ToolOutput:
			keptOutput = content
		}
	}
	if keptCall.ID != "call-2" {
		t.Fatalf("expected retained tool call call-2, got %q", keptCall.ID)
	}
	if keptOutput.ToolCallID != "call-2" {
		t.Fatalf("expected retained tool output call-2, got %q", keptOutput.ToolCallID)
	}
}

// TestCompactHistoryDropsPriorSummary verifies old summaries are removed.
func TestCompactHistoryDropsPriorSummary(t *testing.T) {
	ctx := testutil.Context(t, 0)
	history := []HistoryItem{
		{Role: "user", Content: HistoryText{Text: "question one"}},
		{Role: "assistant", Content: HistoryText{Text: SummaryPrefix + "old summary"}},
		{Role: "user", Content: HistoryText{Text: "question two"}},
	}
	provider := &fakeProvider{
		streams: [][]StreamEvent{{{Type: StreamEventMessage, Message: "new summary"}}},
	}

	compacted, _, err := CompactHistory(ctx, history, provider, ApproxTokenCount, CompactionConfig{
		SoftLimit:             1,
		HardLimit:             100,
		RecentUserTokenBudget: 1,
	})
	if err != nil {
		t.Fatalf("compact history: %v", err)
	}
	count := 0
	for _, item := range compacted {
		if isSummaryItem(item) {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 summary item, got %d", count)
	}
}

// TestCompactHistorySkipsUnderSoftLimit verifies compaction is skipped when under the limit.
func TestCompactHistorySkipsUnderSoftLimit(t *testing.T) {
	ctx := testutil.Context(t, 0)
	history := []HistoryItem{
		{Role: "user", Content: HistoryText{Text: "small"}},
	}
	compacted, stats, err := CompactHistory(ctx, history, nil, ApproxTokenCount, CompactionConfig{
		SoftLimit: 1000,
	})
	if err != nil {
		t.Fatalf("compact history: %v", err)
	}
	if stats != nil {
		t.Fatalf("expected no compaction stats")
	}
	if len(compacted) != len(history) {
		t.Fatalf("expected history unchanged")
	}
}
