package agent

import (
	"context"
	"strings"
	"testing"

	"cogni/internal/tools"
)

func TestRunnerExecutorMissingArgs(t *testing.T) {
	root := t.TempDir()
	runner, err := tools.NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	executor := RunnerExecutor{Runner: runner}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "search",
		Args: ToolCallArgs{},
	})
	if !strings.Contains(result.Error, "query is required") {
		t.Fatalf("unexpected error: %v", result.Error)
	}
}

func TestRunnerExecutorUnknownTool(t *testing.T) {
	root := t.TempDir()
	runner, err := tools.NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	executor := RunnerExecutor{Runner: runner}

	result := executor.Execute(context.Background(), ToolCall{Name: "nope"})
	if !strings.Contains(result.Error, "unknown tool") {
		t.Fatalf("unexpected error: %v", result.Error)
	}
}
