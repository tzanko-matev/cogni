package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"cogni/internal/tools"
)

type fakeStream struct {
	events []StreamEvent
	index  int
}

func (s *fakeStream) Recv() (StreamEvent, error) {
	if s.index >= len(s.events) {
		return StreamEvent{}, io.EOF
	}
	event := s.events[s.index]
	s.index++
	return event, nil
}

type fakeProvider struct {
	streams [][]StreamEvent
	calls   int
}

func (p *fakeProvider) Stream(_ context.Context, _ Prompt) (Stream, error) {
	if p.calls >= len(p.streams) {
		return nil, fmt.Errorf("no more streams")
	}
	stream := &fakeStream{events: p.streams[p.calls]}
	p.calls++
	return stream, nil
}

type fakeExecutor struct {
	calls int
}

func (e *fakeExecutor) Execute(_ context.Context, call ToolCall) tools.CallResult {
	e.calls++
	return tools.CallResult{Tool: call.Name, Output: "ok", OutputBytes: 2}
}

type verboseExecutor struct {
	output string
}

func (e *verboseExecutor) Execute(_ context.Context, call ToolCall) tools.CallResult {
	return tools.CallResult{Tool: call.Name, Output: e.output, OutputBytes: len(e.output)}
}

func TestRunTurnHandlesToolCalls(t *testing.T) {
	session := &Session{
		Ctx: TurnContext{
			ModelFamily: ModelFamily{BaseInstructionsTemplate: "base"},
		},
	}
	provider := &fakeProvider{
		streams: [][]StreamEvent{
			{{Type: StreamEventToolCall, ToolCall: ToolCall{Name: "list_files", Args: map[string]any{}}}},
			{{Type: StreamEventMessage, Message: "done"}},
		},
	}
	executor := &fakeExecutor{}

	metrics, err := RunTurn(context.Background(), session, provider, executor, "run", RunOptions{})
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
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
	if session.History[3].Content != "done" {
		t.Fatalf("expected assistant message, got %v", session.History[3].Content)
	}
	if metrics.Steps != 2 {
		t.Fatalf("expected 2 steps, got %d", metrics.Steps)
	}
	if metrics.ToolCalls["list_files"] != 1 {
		t.Fatalf("expected tool call count, got %v", metrics.ToolCalls)
	}
}

func TestRunTurnBudgetExceeded(t *testing.T) {
	session := &Session{
		Ctx: TurnContext{
			ModelFamily: ModelFamily{BaseInstructionsTemplate: "base"},
		},
	}
	provider := &fakeProvider{
		streams: [][]StreamEvent{
			{{Type: StreamEventToolCall, ToolCall: ToolCall{Name: "list_files", Args: map[string]any{}}}},
		},
	}
	executor := &fakeExecutor{}

	_, err := RunTurn(context.Background(), session, provider, executor, "run", RunOptions{
		Limits: RunLimits{MaxSteps: 1},
	})
	if err != ErrBudgetExceeded {
		t.Fatalf("expected budget exceeded, got %v", err)
	}
}

func TestRunTurnVerboseLogs(t *testing.T) {
	session := &Session{
		Ctx: TurnContext{
			ModelFamily: ModelFamily{BaseInstructionsTemplate: "base"},
		},
	}
	provider := &fakeProvider{
		streams: [][]StreamEvent{
			{{Type: StreamEventToolCall, ToolCall: ToolCall{Name: "list_files", Args: map[string]any{}}}},
			{{Type: StreamEventMessage, Message: "done"}},
		},
	}
	executor := &verboseExecutor{output: strings.Join([]string{"one", "two", "three", "four", "five", "six", "seven"}, "\n")}
	var logs bytes.Buffer

	_, err := RunTurn(context.Background(), session, provider, executor, "run", RunOptions{
		Verbose:       true,
		VerboseWriter: &logs,
	})
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
