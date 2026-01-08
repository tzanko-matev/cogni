package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

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

// ErrBudgetExceeded signals that a run exceeded configured limits.
var ErrBudgetExceeded = errors.New("budget_exceeded")

// RunLimits bounds steps, time, and token usage.
type RunLimits struct {
	MaxSteps   int
	MaxSeconds time.Duration
	MaxTokens  int
}

// RunOptions configures per-run behavior and logging.
type RunOptions struct {
	TokenCounter     TokenCounter
	Compaction       CompactionConfig
	Limits           RunLimits
	Verbose          bool
	VerboseWriter    io.Writer
	VerboseLogWriter io.Writer
	NoColor          bool
}

// RunMetrics captures execution effort for a run.
type RunMetrics struct {
	ToolCalls map[string]int
	WallTime  time.Duration
	Tokens    int
	Steps     int
}

// RunTurn executes a single agent turn, including any follow-up tool calls.
func RunTurn(ctx context.Context, session *Session, provider Provider, executor ToolExecutor, userText string, opts RunOptions) (RunMetrics, error) {
	start := time.Now()
	metrics := RunMetrics{ToolCalls: map[string]int{}}

	session.History = append(session.History, HistoryItem{Role: "user", Content: HistoryText{Text: userText}})
	for {
		if opts.TokenCounter != nil {
			compacted, stats, err := CompactHistory(ctx, session.History, provider, opts.TokenCounter, opts.Compaction)
			if err != nil {
				metrics.WallTime = time.Since(start)
				if opts.TokenCounter != nil {
					metrics.Tokens = opts.TokenCounter(session.History)
				}
				return metrics, err
			}
			if stats != nil {
				session.History = compacted
				logVerbose(opts, styleHeadingMetrics, fmt.Sprintf("Compaction tokens=%d->%d summary_tokens=%d", stats.BeforeTokens, stats.AfterTokens, stats.SummaryTokens))
			}
		}
		if exceededLimits(start, opts.Limits, opts.TokenCounter, session.History, metrics.Steps) {
			metrics.WallTime = time.Since(start)
			if opts.TokenCounter != nil {
				metrics.Tokens = opts.TokenCounter(session.History)
			}
			return metrics, ErrBudgetExceeded
		}

		prompt := BuildPrompt(session.Ctx, session.History)
		logVerbosePrompt(opts, prompt, metrics.Steps+1)
		stream, err := provider.Stream(ctx, prompt)
		if err != nil {
			metrics.WallTime = time.Since(start)
			if opts.TokenCounter != nil {
				metrics.Tokens = opts.TokenCounter(session.History)
			}
			return metrics, err
		}
		metrics.Steps++
		needsFollowUp, err := HandleResponseStream(ctx, session, stream, executor, &metrics, opts)
		if err != nil {
			metrics.WallTime = time.Since(start)
			if opts.TokenCounter != nil {
				metrics.Tokens = opts.TokenCounter(session.History)
			}
			return metrics, err
		}
		if !needsFollowUp {
			break
		}
	}

	metrics.WallTime = time.Since(start)
	if opts.TokenCounter != nil {
		metrics.Tokens = opts.TokenCounter(session.History)
	}
	return metrics, nil
}

// HandleResponseStream consumes streamed output and executes any tools.
func HandleResponseStream(ctx context.Context, session *Session, stream Stream, executor ToolExecutor, metrics *RunMetrics, opts RunOptions) (bool, error) {
	needsFollowUp := false
	for {
		event, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return needsFollowUp, err
		}
		switch event.Type {
		case StreamEventMessage:
			session.History = append(session.History, HistoryItem{Role: "assistant", Content: HistoryText{Text: event.Message}})
			logVerboseBlock(opts, "LLM output", event.Message, styleHeadingOutput, styleDefault)
		case StreamEventToolCall:
			if event.ToolCall.ID == "" {
				event.ToolCall.ID = fmt.Sprintf("call-%d", len(session.History))
			}
			logVerbose(opts, styleHeadingToolCall, fmt.Sprintf("Tool call id=%s name=%s args=%s", event.ToolCall.ID, event.ToolCall.Name, formatArgs(event.ToolCall.Args)))
			session.History = append(session.History, HistoryItem{Role: "assistant", Content: event.ToolCall})
			result := executor.Execute(ctx, event.ToolCall)
			session.History = append(session.History, HistoryItem{Role: "tool", Content: ToolOutput{
				ToolCallID: event.ToolCall.ID,
				Result:     result,
			}})
			logVerboseToolOutput(opts, fmt.Sprintf("Tool result id=%s name=%s duration=%s bytes=%d truncated=%t error=%s", event.ToolCall.ID, result.Tool, result.Duration, result.OutputBytes, result.Truncated, result.Error), result.Output)
			if metrics != nil {
				if metrics.ToolCalls == nil {
					metrics.ToolCalls = map[string]int{}
				}
				metrics.ToolCalls[event.ToolCall.Name]++
			}
			needsFollowUp = true
		default:
			return needsFollowUp, fmt.Errorf("unknown stream event type: %d", event.Type)
		}
	}
	return needsFollowUp, nil
}

// exceededLimits reports whether any limits have been exceeded.
func exceededLimits(start time.Time, limits RunLimits, counter TokenCounter, history []HistoryItem, steps int) bool {
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
