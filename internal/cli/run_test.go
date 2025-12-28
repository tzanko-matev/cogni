package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"cogni/internal/runner"
	"cogni/internal/spec"
)

func TestRunCommandParsesFlags(t *testing.T) {
	specDir := t.TempDir()
	specPath := filepath.Join(specDir, ".cogni.yml")
	specBody := `version: 1
repo:
  output_dir: "./out"
agents:
  - id: default
    type: builtin
    provider: openrouter
    model: test-model
    max_steps: 2
    temperature: 0.0
default_agent: default
tasks:
  - id: task-1
    type: qa
    agent: default
    prompt: "hello"
`
	if err := os.WriteFile(specPath, []byte(specBody), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	var gotParams runner.RunParams
	origRun := runAndWrite
	runAndWrite = func(_ context.Context, _ spec.Config, params runner.RunParams) (runner.Results, runner.OutputPaths, error) {
		gotParams = params
		return runner.Results{RunID: "run-1"}, runner.OutputPaths{Root: specDir, Commit: "abc", RunID: "run-1"}, nil
	}
	t.Cleanup(func() { runAndWrite = origRun })

	cmd := findCommand("run")
	if cmd == nil {
		t.Fatalf("run command not found")
	}
	var stdout, stderr bytes.Buffer
	exitCode := cmd.Run([]string{"--spec", specPath, "--agent", "default", "task-1"}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("unexpected exit: %d, stderr: %s", exitCode, stderr.String())
	}
	if gotParams.AgentOverride != "default" {
		t.Fatalf("unexpected agent override: %s", gotParams.AgentOverride)
	}
	if len(gotParams.Selectors) != 1 || gotParams.Selectors[0].TaskID != "task-1" {
		t.Fatalf("unexpected selectors: %+v", gotParams.Selectors)
	}
}
