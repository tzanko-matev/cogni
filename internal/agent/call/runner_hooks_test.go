package call

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"cogni/internal/agent"
	"cogni/internal/testutil"
)

type hookRecorder struct {
	calls      []string
	beforeErr  error
	afterErr   error
	lastInput  CallInput
	lastResult CallResult
}

func (h *hookRecorder) BeforeCall(_ context.Context, input CallInput) error {
	h.calls = append(h.calls, "before")
	h.lastInput = input
	return h.beforeErr
}

func (h *hookRecorder) AfterCall(_ context.Context, _ CallInput, result CallResult) error {
	h.calls = append(h.calls, "after")
	h.lastResult = result
	return h.afterErr
}

type stubStream struct {
	events []agent.StreamEvent
	index  int
}

func (s *stubStream) Recv() (agent.StreamEvent, error) {
	if s.index >= len(s.events) {
		return agent.StreamEvent{}, io.EOF
	}
	event := s.events[s.index]
	s.index++
	return event, nil
}

type stubProvider struct {
	streams [][]agent.StreamEvent
	calls   int
	err     error
}

func (p *stubProvider) Stream(_ context.Context, _ agent.Prompt) (agent.Stream, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	if p.calls > len(p.streams) {
		return nil, errors.New("no streams")
	}
	events := p.streams[p.calls-1]
	return &stubStream{events: events}, nil
}

// TestRunCallHooksInvoked verifies hooks run around the call.
func TestRunCallHooksInvoked(t *testing.T) {
	ctx := testutil.Context(t, 2*time.Second)
	session := &agent.Session{Ctx: agent.TurnContext{ModelFamily: agent.ModelFamily{BaseInstructionsTemplate: "base"}}}
	provider := &stubProvider{streams: [][]agent.StreamEvent{{{Type: agent.StreamEventMessage, Message: "done"}}}}
	recorder := &hookRecorder{}

	result, err := RunCall(ctx, session, provider, nil, "run", RunOptions{}, []CallHook{recorder})
	if err != nil {
		t.Fatalf("run call: %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("expected 1 provider call, got %d", provider.calls)
	}
	if len(recorder.calls) != 2 || recorder.calls[0] != "before" || recorder.calls[1] != "after" {
		t.Fatalf("unexpected hook order: %v", recorder.calls)
	}
	if recorder.lastInput.Prompt.Instructions == "" {
		t.Fatalf("expected hook input to include prompt instructions")
	}
	if result.Output != "done" {
		t.Fatalf("expected output to be recorded, got %q", result.Output)
	}
}

// TestRunCallBeforeHookErrorStopsRun verifies before-hook errors abort execution.
func TestRunCallBeforeHookErrorStopsRun(t *testing.T) {
	ctx := testutil.Context(t, 2*time.Second)
	session := &agent.Session{Ctx: agent.TurnContext{ModelFamily: agent.ModelFamily{BaseInstructionsTemplate: "base"}}}
	provider := &stubProvider{streams: [][]agent.StreamEvent{{{Type: agent.StreamEventMessage, Message: "done"}}}}
	recorder := &hookRecorder{beforeErr: errors.New("stop")}

	_, err := RunCall(ctx, session, provider, nil, "run", RunOptions{}, []CallHook{recorder})
	if err == nil {
		t.Fatalf("expected error")
	}
	if provider.calls != 0 {
		t.Fatalf("expected provider not to be called, got %d", provider.calls)
	}
	if len(recorder.calls) != 1 || recorder.calls[0] != "before" {
		t.Fatalf("unexpected hook calls: %v", recorder.calls)
	}
}

// TestRunCallAfterHookRunsOnError verifies after-hook runs even when the call fails.
func TestRunCallAfterHookRunsOnError(t *testing.T) {
	ctx := testutil.Context(t, 2*time.Second)
	session := &agent.Session{Ctx: agent.TurnContext{ModelFamily: agent.ModelFamily{BaseInstructionsTemplate: "base"}}}
	provider := &stubProvider{err: errors.New("stream failed")}
	recorder := &hookRecorder{}

	result, err := RunCall(ctx, session, provider, nil, "run", RunOptions{}, []CallHook{recorder})
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(recorder.calls) != 2 || recorder.calls[0] != "before" || recorder.calls[1] != "after" {
		t.Fatalf("unexpected hook calls: %v", recorder.calls)
	}
	if result.FailureReason != "runtime_error" {
		t.Fatalf("expected runtime_error failure reason, got %q", result.FailureReason)
	}
}
