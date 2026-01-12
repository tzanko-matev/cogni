package call

import (
	"context"
	"errors"
	"time"

	"cogni/internal/agent"
)

// callBeforeHooks executes CallHook.BeforeCall in order.
func callBeforeHooks(ctx context.Context, hooks []CallHook, input CallInput) error {
	for _, hook := range hooks {
		if err := hook.BeforeCall(ctx, input); err != nil {
			return err
		}
	}
	return nil
}

// callAfterHooks executes CallHook.AfterCall in order and ignores errors.
func callAfterHooks(ctx context.Context, hooks []CallHook, input CallInput, result CallResult) {
	for _, hook := range hooks {
		_ = hook.AfterCall(ctx, input, result)
	}
}

// finalizeMetrics populates wall time and tokens for a completed run.
func finalizeMetrics(start time.Time, opts RunOptions, history []agent.HistoryItem, metrics *RunMetrics) {
	metrics.WallTime = time.Since(start)
	if opts.TokenCounter != nil {
		metrics.Tokens = opts.TokenCounter(history)
	}
}

// latestAssistantMessage returns the most recent assistant text message.
func latestAssistantMessage(history []agent.HistoryItem) string {
	for i := len(history) - 1; i >= 0; i-- {
		item := history[i]
		if item.Role != "assistant" {
			continue
		}
		text, ok := item.Content.(agent.HistoryText)
		if ok {
			return text.Text
		}
	}
	return ""
}

// failureReasonForError maps a run error to a CallResult failure reason.
func failureReasonForError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrBudgetExceeded) {
		return "budget_exceeded"
	}
	return "runtime_error"
}

// exceededLimits reports whether any limits have been exceeded.
func exceededLimits(start time.Time, limits RunLimits, counter agent.TokenCounter, history []agent.HistoryItem, steps int) bool {
	if limits.MaxSeconds > 0 && time.Since(start) > limits.MaxSeconds {
		return true
	}
	if limits.MaxSteps > 0 && steps >= limits.MaxSteps {
		return true
	}
	if limits.MaxTokens > 0 && counter != nil && counter(history) > limits.MaxTokens {
		return true
	}
	return false
}
