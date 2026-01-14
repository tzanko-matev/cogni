package runner

import (
	"context"
	"time"

	"cogni/internal/agent"
	"cogni/internal/tools"
)

// observedToolExecutor wraps a ToolExecutor to emit tool activity events.
type observedToolExecutor struct {
	observer *questionJobObserver
	index    int
	inner    agent.ToolExecutor
}

// newObservedToolExecutor wraps an executor when observation is enabled.
func newObservedToolExecutor(observer *questionJobObserver, index int, inner agent.ToolExecutor) agent.ToolExecutor {
	if observer == nil {
		return inner
	}
	return observedToolExecutor{observer: observer, index: index, inner: inner}
}

// Execute emits tool start/finish events around tool execution.
func (e observedToolExecutor) Execute(ctx context.Context, call agent.ToolCall) tools.CallResult {
	if e.observer != nil {
		e.observer.Emit(e.index, questionEventOptions{EventType: QuestionToolStart, ToolName: call.Name})
	}
	start := time.Now()
	result := e.inner.Execute(ctx, call)
	duration := result.Duration
	if duration <= 0 {
		duration = time.Since(start)
	}
	if e.observer != nil {
		e.observer.Emit(e.index, questionEventOptions{
			EventType:    QuestionToolFinish,
			ToolName:     call.Name,
			ToolDuration: duration,
			ToolError:    result.Error,
		})
	}
	return result
}
