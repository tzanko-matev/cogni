package runner

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogni/internal/agent"
	"cogni/internal/spec"
	"cogni/internal/tools"
)

type fakeStream struct {
	events []agent.StreamEvent
	index  int
}

func (s *fakeStream) Recv() (agent.StreamEvent, error) {
	if s.index >= len(s.events) {
		return agent.StreamEvent{}, io.EOF
	}
	event := s.events[s.index]
	s.index++
	return event, nil
}

type fakeProvider struct {
	message string
}

func (p fakeProvider) Stream(_ context.Context, _ agent.Prompt) (agent.Stream, error) {
	return &fakeStream{events: []agent.StreamEvent{{Type: agent.StreamEventMessage, Message: p.message}}}, nil
}

func TestRunExecutesTask(t *testing.T) {
	repoRoot := initGitRepo(t)
	cfg := spec.Config{
		Repo: spec.RepoConfig{OutputDir: "./out"},
		Agents: []spec.AgentConfig{
			{ID: "agent-1", Type: "builtin", Provider: "openrouter", Model: "model"},
		},
		DefaultAgent: "agent-1",
		Tasks: []spec.TaskConfig{
			{ID: "task-1", Type: "qa", Agent: "agent-1", Prompt: "prompt"},
		},
	}

	fixedTime := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	results, err := Run(context.Background(), cfg, RunParams{
		RepoRoot: repoRoot,
		Deps: RunDependencies{
			ProviderFactory: func(_ spec.AgentConfig, _ string) (agent.Provider, error) {
				return fakeProvider{message: `{"ok":true}`}, nil
			},
			ToolRunnerFactory: func(root string) (*tools.Runner, error) {
				return tools.NewRunner(root)
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

func TestRunAndWriteOutputs(t *testing.T) {
	repoRoot := initGitRepo(t)
	outputDir := t.TempDir()
	cfg := spec.Config{
		Repo: spec.RepoConfig{OutputDir: outputDir},
		Agents: []spec.AgentConfig{
			{ID: "agent-1", Type: "builtin", Provider: "openrouter", Model: "model"},
		},
		DefaultAgent: "agent-1",
		Tasks: []spec.TaskConfig{
			{ID: "task-1", Type: "qa", Agent: "agent-1", Prompt: "prompt"},
		},
	}
	_, paths, err := RunAndWrite(context.Background(), cfg, RunParams{
		RepoRoot: repoRoot,
		Deps: RunDependencies{
			ProviderFactory: func(_ spec.AgentConfig, _ string) (agent.Provider, error) {
				return fakeProvider{message: `{"ok":true}`}, nil
			},
			ToolRunnerFactory: func(root string) (*tools.Runner, error) {
				return tools.NewRunner(root)
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

func initGitRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	root := t.TempDir()
	runGit(t, root, "-c", "init.defaultBranch=main", "init")
	path := filepath.Join(root, "README.md")
	if err := os.WriteFile(path, []byte("init"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "init")
	return root
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}
