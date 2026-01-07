package agent

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cogni/internal/tools"
)

// fakeStream replays a predetermined stream of events for tests.
type fakeStream struct {
	events []StreamEvent
	index  int
}

// Recv returns the next event or io.EOF for fakeStream.
func (s *fakeStream) Recv() (StreamEvent, error) {
	if s.index >= len(s.events) {
		return StreamEvent{}, io.EOF
	}
	event := s.events[s.index]
	s.index++
	return event, nil
}

// fakeProvider returns scripted streams for agent tests.
type fakeProvider struct {
	streams [][]StreamEvent
	calls   int
}

// Stream returns the next scripted stream for fakeProvider.
func (p *fakeProvider) Stream(_ context.Context, _ Prompt) (Stream, error) {
	if p.calls >= len(p.streams) {
		return nil, fmt.Errorf("no more streams")
	}
	stream := &fakeStream{events: p.streams[p.calls]}
	p.calls++
	return stream, nil
}

// fakeExecutor counts tool executions for tests.
type fakeExecutor struct {
	calls int
}

// Execute records tool calls and returns a minimal response.
func (e *fakeExecutor) Execute(_ context.Context, call ToolCall) tools.CallResult {
	e.calls++
	return tools.CallResult{Tool: call.Name, Output: "ok", OutputBytes: 2}
}

// verboseExecutor emits configurable output for verbose logging tests.
type verboseExecutor struct {
	output string
}

// Execute returns tool output with the configured payload.
func (e *verboseExecutor) Execute(_ context.Context, call ToolCall) tools.CallResult {
	return tools.CallResult{Tool: call.Name, Output: e.output, OutputBytes: len(e.output)}
}

// verboseOutput returns multiline output for verbose logging tests.
func verboseOutput() string {
	return strings.Join([]string{"one", "two", "three", "four", "five", "six", "seven"}, "\n")
}
