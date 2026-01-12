package call

import (
	"context"
	"fmt"
	"time"

	"cogni/internal/agent"
)

// RunCall executes a single agent turn, including any follow-up tool calls.
func RunCall(ctx context.Context, session *agent.Session, provider agent.Provider, executor agent.ToolExecutor, userText string, opts RunOptions, hooks []CallHook) (CallResult, error) {
	start := time.Now()
	metrics := RunMetrics{ToolCalls: map[string]int{}}
	session.History = append(session.History, agent.HistoryItem{Role: "user", Content: agent.HistoryText{Text: userText}})

	var hookInput CallInput
	hooksReady := false
	var runErr error

	for {
		if opts.TokenCounter != nil {
			compacted, stats, err := agent.CompactHistory(ctx, session.History, provider, opts.TokenCounter, opts.Compaction)
			if err != nil {
				runErr = err
				break
			}
			if stats != nil {
				session.History = compacted
				metrics.Compactions++
				metrics.LastSummaryTokens = stats.SummaryTokens
				logVerbose(opts, styleHeadingMetrics, fmt.Sprintf("Compaction tokens=%d->%d summary_tokens=%d", stats.BeforeTokens, stats.AfterTokens, stats.SummaryTokens))
			}
		}
		if exceededLimits(start, opts.Limits, opts.TokenCounter, session.History, metrics.Steps) {
			runErr = ErrBudgetExceeded
			break
		}

		prompt := agent.BuildPrompt(session.Ctx, session.History)
		if !hooksReady {
			hookInput = CallInput{Prompt: prompt, ToolDefs: session.Ctx.Tools, Limits: opts.Limits}
			if err := callBeforeHooks(ctx, hooks, hookInput); err != nil {
				runErr = err
				break
			}
			hooksReady = true
		}

		logVerbosePrompt(opts, prompt, metrics.Steps+1)
		stream, err := provider.Stream(ctx, prompt)
		if err != nil {
			runErr = err
			break
		}
		metrics.Steps++
		needsFollowUp, err := handleResponseStream(ctx, session, stream, executor, &metrics, opts)
		if err != nil {
			runErr = err
			break
		}
		if !needsFollowUp {
			break
		}
	}

	finalizeMetrics(start, opts, session.History, &metrics)
	result := CallResult{
		Output:        latestAssistantMessage(session.History),
		Metrics:       metrics,
		FailureReason: failureReasonForError(runErr),
	}
	if hooksReady {
		callAfterHooks(ctx, hooks, hookInput, result)
	}
	return result, runErr
}
