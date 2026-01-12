package agent

import (
	"context"

	"cogni/internal/tools"
)

// StreamEventType identifies streamed event kinds.
type StreamEventType int

const (
	StreamEventMessage StreamEventType = iota
	StreamEventToolCall
)

// StreamEvent carries either a message or tool call from the model stream.
type StreamEvent struct {
	Type     StreamEventType
	Message  string
	ToolCall ToolCall
}

// Stream yields incremental model events.
type Stream interface {
	Recv() (StreamEvent, error)
}

// Provider streams model responses for a prompt.
type Provider interface {
	Stream(ctx context.Context, prompt Prompt) (Stream, error)
}

// ToolCall describes a tool invocation emitted by the model.
type ToolCall struct {
	ID   string
	Name string
	Args ToolCallArgs
}

// ToolOutput represents the result of a tool invocation.
type ToolOutput struct {
	ToolCallID string
	Result     tools.CallResult
}

// ToolExecutor executes tool calls.
type ToolExecutor interface {
	Execute(ctx context.Context, call ToolCall) tools.CallResult
}
