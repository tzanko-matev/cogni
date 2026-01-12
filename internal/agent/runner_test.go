package agent

import (
	"bytes"
	"strings"
	"testing"

	"cogni/internal/agent/call"
	"cogni/internal/testutil"
)

// TestRunTurnHandlesToolCalls verifies tool-call execution flow.
func TestRunTurnHandlesToolCalls(t *testing.T) {
	session := &Session{
		Ctx: TurnContext{
			ModelFamily: ModelFamily{BaseInstructionsTemplate: "base"},
		},
	}
	provider := &fakeProvider{
		streams: [][]StreamEvent{
			{{Type: StreamEventToolCall, ToolCall: ToolCall{Name: "list_files", Args: ToolCallArgs{}}}},
			{{Type: StreamEventMessage, Message: "done"}},
		},
	}
	executor := &fakeExecutor{}

	ctx := testutil.Context(t, 0)
	result, err := call.RunCall(ctx, session, provider, executor, "run", call.RunOptions{}, nil)
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
	metrics := result.Metrics
	if provider.calls != 2 {
		t.Fatalf("expected 2 provider calls, got %d", provider.calls)
	}
	if executor.calls != 1 {
		t.Fatalf("expected 1 tool call, got %d", executor.calls)
	}
	if len(session.History) != 4 {
		t.Fatalf("expected 4 history items, got %d", len(session.History))
	}
	roles := []string{"user", "assistant", "tool", "assistant"}
	for i, role := range roles {
		if session.History[i].Role != role {
			t.Fatalf("unexpected role at %d: %s", i, session.History[i].Role)
		}
	}
	call, ok := session.History[1].Content.(ToolCall)
	if !ok {
		t.Fatalf("expected tool call content")
	}
	if call.Name != "list_files" || call.ID == "" {
		t.Fatalf("unexpected tool call: %+v", call)
	}
	output, ok := session.History[2].Content.(ToolOutput)
	if !ok {
		t.Fatalf("expected tool output content")
	}
	if output.ToolCallID != call.ID {
		t.Fatalf("expected tool output to reference call id")
	}
	if session.History[3].Content != (HistoryText{Text: "done"}) {
		t.Fatalf("expected assistant message, got %v", session.History[3].Content)
	}
	if metrics.Steps != 2 {
		t.Fatalf("expected 2 steps, got %d", metrics.Steps)
	}
	if metrics.ToolCalls["list_files"] != 1 {
		t.Fatalf("expected tool call count, got %v", metrics.ToolCalls)
	}
}

// TestRunTurnBudgetExceeded verifies budget enforcement behavior.
func TestRunTurnBudgetExceeded(t *testing.T) {
	session := &Session{
		Ctx: TurnContext{
			ModelFamily: ModelFamily{BaseInstructionsTemplate: "base"},
		},
	}
	provider := &fakeProvider{
		streams: [][]StreamEvent{
			{{Type: StreamEventToolCall, ToolCall: ToolCall{Name: "list_files", Args: ToolCallArgs{}}}},
		},
	}
	executor := &fakeExecutor{}

	ctx := testutil.Context(t, 0)
	_, err := call.RunCall(ctx, session, provider, executor, "run", call.RunOptions{
		Limits: call.RunLimits{MaxSteps: 1},
	}, nil)
	if err != call.ErrBudgetExceeded {
		t.Fatalf("expected budget exceeded, got %v", err)
	}
}

// TestRunTurnVerboseLogs verifies verbose logging output formatting.
func TestRunTurnVerboseLogs(t *testing.T) {
	session := &Session{
		Ctx: TurnContext{
			ModelFamily: ModelFamily{BaseInstructionsTemplate: "base"},
		},
	}
	provider := &fakeProvider{
		streams: [][]StreamEvent{
			{{Type: StreamEventToolCall, ToolCall: ToolCall{Name: "list_files", Args: ToolCallArgs{}}}},
			{{Type: StreamEventMessage, Message: "done"}},
		},
	}
	executor := &verboseExecutor{output: verboseOutput()}
	var logs bytes.Buffer

	ctx := testutil.Context(t, 0)
	_, err := call.RunCall(ctx, session, provider, executor, "run", call.RunOptions{
		Verbose:       true,
		VerboseWriter: &logs,
	}, nil)
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
	output := logs.String()
	for _, needle := range []string{"LLM prompt", "Tool call", "Tool result", "LLM output"} {
		if !strings.Contains(output, needle) {
			t.Fatalf("expected verbose logs to include %q, got %s", needle, output)
		}
	}
	if strings.Contains(output, "six") {
		t.Fatalf("expected tool output to be truncated to 5 lines, got %s", output)
	}
	if !strings.Contains(output, "[truncated]") {
		t.Fatalf("expected truncation marker, got %s", output)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	toolResultIndex := -1
	endIndex := len(lines)
	for i, line := range lines {
		if toolResultIndex == -1 && strings.Contains(line, "Tool result id=") {
			toolResultIndex = i
			continue
		}
		if toolResultIndex != -1 {
			if strings.Contains(line, "LLM prompt") || strings.Contains(line, "LLM output") || strings.Contains(line, "Tool call") {
				endIndex = i
				break
			}
		}
	}
	if toolResultIndex == -1 {
		t.Fatalf("expected tool result log, got %s", output)
	}
	toolLines := lines[toolResultIndex+1 : endIndex]
	if len(toolLines) > 5 {
		t.Fatalf("expected at most 5 tool output lines, got %d: %s", len(toolLines), output)
	}
}

// TestRunTurnVerboseLogWriterCapturesFullOutput verifies log writer output is not truncated.
func TestRunTurnVerboseLogWriterCapturesFullOutput(t *testing.T) {
	session := &Session{
		Ctx: TurnContext{
			ModelFamily: ModelFamily{BaseInstructionsTemplate: "base"},
		},
	}
	provider := &fakeProvider{
		streams: [][]StreamEvent{
			{{Type: StreamEventToolCall, ToolCall: ToolCall{Name: "list_files", Args: ToolCallArgs{}}}},
			{{Type: StreamEventMessage, Message: "done"}},
		},
	}
	executor := &verboseExecutor{output: verboseOutput()}
	var logBuffer bytes.Buffer

	ctx := testutil.Context(t, 0)
	_, err := call.RunCall(ctx, session, provider, executor, "run", call.RunOptions{
		Verbose:          false,
		VerboseLogWriter: &logBuffer,
	}, nil)
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "six") || !strings.Contains(logOutput, "seven") {
		t.Fatalf("expected log output to include full tool output, got %s", logOutput)
	}
	if strings.Contains(logOutput, "[truncated]") {
		t.Fatalf("expected log output to avoid truncation markers, got %s", logOutput)
	}
}
