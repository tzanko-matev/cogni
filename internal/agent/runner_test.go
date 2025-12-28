package agent

import (
	"context"
	"fmt"
	"io"
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

	if err := RunTurn(context.Background(), session, provider, executor, "run", nil, 0); err != nil {
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
}
