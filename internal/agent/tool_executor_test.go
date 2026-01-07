package agent

import (
	"strings"
	"testing"

	"cogni/internal/testutil"
	"cogni/internal/tools"
)

// TestRunnerExecutorMissingArgs verifies validation for missing tool args.
func TestRunnerExecutorMissingArgs(t *testing.T) {
	root := t.TempDir()
	runner, err := tools.NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	executor := RunnerExecutor{Runner: runner}

	ctx := testutil.Context(t, 0)
	result := executor.Execute(ctx, ToolCall{
		Name: "search",
		Args: ToolCallArgs{},
	})
	if !strings.Contains(result.Error, "query is required") {
		t.Fatalf("unexpected error: %v", result.Error)
	}
}

// TestRunnerExecutorUnknownTool verifies unknown tools return errors.
func TestRunnerExecutorUnknownTool(t *testing.T) {
	root := t.TempDir()
	runner, err := tools.NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	executor := RunnerExecutor{Runner: runner}

	ctx := testutil.Context(t, 0)
	result := executor.Execute(ctx, ToolCall{Name: "nope"})
	if !strings.Contains(result.Error, "unknown tool") {
		t.Fatalf("unexpected error: %v", result.Error)
	}
}
