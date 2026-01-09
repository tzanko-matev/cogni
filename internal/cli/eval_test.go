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

// TestEvalCommandBuildsConfig verifies eval builds a question_eval task.
func TestEvalCommandBuildsConfig(t *testing.T) {
	repoRoot := t.TempDir()
	specPath := filepath.Join(repoRoot, ".cogni", "config.yml")
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
tasks: []
`
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(specPath, []byte(specBody), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	questionsPath := filepath.Join(repoRoot, "questions.yml")
	questionsBody := `version: 1
questions:
  - question: "What is 1+1?"
    answers: ["2"]
    correct_answers: ["2"]
`
	if err := os.WriteFile(questionsPath, []byte(questionsBody), 0o644); err != nil {
		t.Fatalf("write questions: %v", err)
	}

	var gotParams runner.RunParams
	var gotConfig spec.Config
	origRun := runEvalAndWrite
	runEvalAndWrite = func(_ context.Context, cfg spec.Config, params runner.RunParams) (runner.Results, runner.OutputPaths, error) {
		gotConfig = cfg
		gotParams = params
		return runner.Results{RunID: "run-1"}, runner.OutputPaths{Root: repoRoot, Commit: "abc", RunID: "run-1"}, nil
	}
	t.Cleanup(func() { runEvalAndWrite = origRun })

	cmd := findCommand("eval")
	if cmd == nil {
		t.Fatalf("eval command not found")
	}
	var stdout, stderr bytes.Buffer
	exitCode := cmd.Run([]string{"--spec", specPath, "--agent", "default", questionsPath}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("unexpected exit: %d, stderr: %s", exitCode, stderr.String())
	}
	if gotParams.RepoRoot != repoRoot {
		t.Fatalf("unexpected repo root: %q", gotParams.RepoRoot)
	}
	if len(gotConfig.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(gotConfig.Tasks))
	}
	task := gotConfig.Tasks[0]
	if task.Type != "question_eval" {
		t.Fatalf("expected question_eval task, got %q", task.Type)
	}
	if task.QuestionsFile == "" {
		t.Fatalf("expected questions file to be set")
	}
}

// TestEvalCommandAllowsFlagsAfterQuestionsFile ensures eval accepts flags after the questions file.
func TestEvalCommandAllowsFlagsAfterQuestionsFile(t *testing.T) {
	repoRoot := t.TempDir()
	specPath := filepath.Join(repoRoot, ".cogni", "config.yml")
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
tasks: []
`
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(specPath, []byte(specBody), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	questionsPath := filepath.Join(repoRoot, "questions.yml")
	questionsBody := `version: 1
questions:
  - question: "What is 1+1?"
    answers: ["2"]
    correct_answers: ["2"]
`
	if err := os.WriteFile(questionsPath, []byte(questionsBody), 0o644); err != nil {
		t.Fatalf("write questions: %v", err)
	}

	origRun := runEvalAndWrite
	runEvalAndWrite = func(_ context.Context, _ spec.Config, _ runner.RunParams) (runner.Results, runner.OutputPaths, error) {
		return runner.Results{RunID: "run-1"}, runner.OutputPaths{Root: repoRoot, Commit: "abc", RunID: "run-1"}, nil
	}
	t.Cleanup(func() { runEvalAndWrite = origRun })

	cmd := findCommand("eval")
	if cmd == nil {
		t.Fatalf("eval command not found")
	}
	var stdout, stderr bytes.Buffer
	exitCode := cmd.Run([]string{questionsPath, "--spec", specPath, "--agent", "default", "--verbose", "--log", filepath.Join(repoRoot, "run.log")}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("unexpected exit: %d, stderr: %s", exitCode, stderr.String())
	}
}
