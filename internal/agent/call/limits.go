package call

import (
	"errors"
	"io"
	"time"

	"cogni/internal/agent"
)

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
	TokenCounter     agent.TokenCounter
	Compaction       agent.CompactionConfig
	Limits           RunLimits
	Verbose          bool
	VerboseWriter    io.Writer
	VerboseLogWriter io.Writer
	NoColor          bool
}

// RunMetrics captures execution effort for a run.
type RunMetrics struct {
	ToolCalls         map[string]int
	WallTime          time.Duration
	Tokens            int
	Steps             int
	Compactions       int
	LastSummaryTokens int
}
