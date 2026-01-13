package runner

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogni/internal/agent"
	"cogni/internal/spec"
	"cogni/internal/testutil"
	"cogni/internal/tools"
	"cogni/internal/vcs"
)

// fakeStream replays a predetermined stream of events for tests.
type fakeStream struct {
	events []agent.StreamEvent
	index  int
}

// Recv returns the next event or io.EOF for fakeStream.
func (s *fakeStream) Recv() (agent.StreamEvent, error) {
	if s.index >= len(s.events) {
		return agent.StreamEvent{}, io.EOF
	}
	event := s.events[s.index]
	s.index++
	return event, nil
}

// fakeProvider returns a static assistant message for tests.
type fakeProvider struct {
	message string
}

// Stream returns a stream containing the configured message.
func (p fakeProvider) Stream(_ context.Context, _ agent.Prompt) (agent.Stream, error) {
	return &fakeStream{events: []agent.StreamEvent{{Type: agent.StreamEventMessage, Message: p.message}}}, nil
}

// TestRunExecutesTask verifies a question_eval task run completes successfully.
func TestRunExecutesTask(t *testing.T) {
	repoRoot := t.TempDir()
	specPath := filepath.Join(repoRoot, "questions.yml")
	specBody := `version: 1
questions:
  - id: q1
    question: "What is 2+2?"
    answers: ["4", "5"]
    correct_answers: ["4"]
`
	if err := os.WriteFile(specPath, []byte(specBody), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	cfg := spec.Config{
		Repo: spec.RepoConfig{OutputDir: "./out"},
		Agents: []spec.AgentConfig{
			{ID: "agent-1", Type: "builtin", Provider: "openrouter", Model: "model"},
		},
		DefaultAgent: "agent-1",
		Tasks: []spec.TaskConfig{
			{ID: "task-1", Type: "question_eval", Agent: "agent-1", QuestionsFile: "questions.yml"},
		},
	}

	fixedTime := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	ctx := testutil.Context(t, 0)
	results, err := Run(ctx, cfg, RunParams{
		RepoRoot: repoRoot,
		Deps: RunDependencies{
			ProviderFactory: func(_ spec.AgentConfig, _ string) (agent.Provider, error) {
				return fakeProvider{message: "Reasoning.\n<answer>4</answer>"}, nil
			},
			ToolRunnerFactory: func(root string) (*tools.Runner, error) {
				return tools.NewRunner(root)
			},
			RepoRootResolver: func(_ context.Context, root string) (string, error) {
				return root, nil
			},
			RepoMetadataLoader: func(_ context.Context, root string) (vcs.Metadata, error) {
				return vcs.Metadata{Name: filepath.Base(root), VCS: "git", Commit: "commit", Branch: "main", Dirty: false}, nil
			},
			RunID: func() (string, error) { return "run-1", nil },
			Now:   func() time.Time { return fixedTime },
		},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if results.RunID != "run-1" {
		t.Fatalf("unexpected run id: %s", results.RunID)
	}
	if len(results.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(results.Tasks))
	}
	if results.Tasks[0].Status != "pass" {
		t.Fatalf("expected pass, got %+v", results.Tasks[0])
	}
	if results.Summary.TasksPassed != 1 {
		t.Fatalf("expected summary pass, got %+v", results.Summary)
	}
}

// TestRunAndWriteOutputs verifies output files are written.
func TestRunAndWriteOutputs(t *testing.T) {
	repoRoot := t.TempDir()
	outputDir := t.TempDir()
	specPath := filepath.Join(repoRoot, "questions.yml")
	specBody := `version: 1
questions:
  - id: q1
    question: "What is 2+2?"
    answers: ["4", "5"]
    correct_answers: ["4"]
`
	if err := os.WriteFile(specPath, []byte(specBody), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	cfg := spec.Config{
		Repo: spec.RepoConfig{OutputDir: outputDir},
		Agents: []spec.AgentConfig{
			{ID: "agent-1", Type: "builtin", Provider: "openrouter", Model: "model"},
		},
		DefaultAgent: "agent-1",
		Tasks: []spec.TaskConfig{
			{ID: "task-1", Type: "question_eval", Agent: "agent-1", QuestionsFile: "questions.yml"},
		},
	}
	ctx := testutil.Context(t, 0)
	_, paths, err := RunAndWrite(ctx, cfg, RunParams{
		RepoRoot: repoRoot,
		Deps: RunDependencies{
			ProviderFactory: func(_ spec.AgentConfig, _ string) (agent.Provider, error) {
				return fakeProvider{message: "Reasoning.\n<answer>4</answer>"}, nil
			},
			ToolRunnerFactory: func(root string) (*tools.Runner, error) {
				return tools.NewRunner(root)
			},
			RepoRootResolver: func(_ context.Context, root string) (string, error) {
				return root, nil
			},
			RepoMetadataLoader: func(_ context.Context, root string) (vcs.Metadata, error) {
				return vcs.Metadata{Name: filepath.Base(root), VCS: "git", Commit: "commit", Branch: "main", Dirty: false}, nil
			},
			RunID: func() (string, error) { return "run-1", nil },
			Now:   func() time.Time { return time.Now() },
		},
	})
	if err != nil {
		t.Fatalf("run and write: %v", err)
	}
	if _, err := os.Stat(paths.ResultsPath()); err != nil {
		t.Fatalf("missing results: %v", err)
	}
	if _, err := os.Stat(paths.ReportPath()); err != nil {
		t.Fatalf("missing report: %v", err)
	}
}
