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

// TestRunCommandParsesFlags verifies CLI flag parsing for run.
func TestRunCommandParsesFlags(t *testing.T) {
	specDir := t.TempDir()
	specPath := filepath.Join(specDir, ".cogni", "config.yml")
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
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(specPath, []byte(specBody), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	var gotParams runner.RunParams
	logPath := filepath.Join(specDir, "run.log")
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
	exitCode := cmd.Run([]string{"--spec", specPath, "--agent", "default", "--verbose", "--no-color", "--log", logPath, "task-1"}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("unexpected exit: %d, stderr: %s", exitCode, stderr.String())
	}
	if gotParams.AgentOverride != "default" {
		t.Fatalf("unexpected agent override: %s", gotParams.AgentOverride)
	}
	if !gotParams.Verbose {
		t.Fatalf("expected verbose enabled")
	}
	if gotParams.VerboseWriter != &stdout {
		t.Fatalf("expected verbose writer to be stdout")
	}
	if gotParams.VerboseLogWriter == nil {
		t.Fatalf("expected verbose log writer to be set")
	}
	if !gotParams.NoColor {
		t.Fatalf("expected no-color enabled")
	}
	if len(gotParams.Selectors) != 1 || gotParams.Selectors[0].TaskID != "task-1" {
		t.Fatalf("unexpected selectors: %+v", gotParams.Selectors)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected log file to exist: %v", err)
	}
}
