package runner

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"cogni/internal/agent"
	"cogni/internal/spec"
	"cogni/internal/testutil"
	"cogni/internal/tools"
	"cogni/internal/vcs"
	"cogni/pkg/ratelimiter"
)

// TestRunUsesRateLimiter verifies question_eval tasks reserve and complete per question.
func TestRunUsesRateLimiter(t *testing.T) {
	ctx := testutil.Context(t, time.Second)
	repoRoot := t.TempDir()
	specPath := filepath.Join(repoRoot, "questions.yml")
	specBody := `version: 1
questions:
  - id: q1
    question: "What is 2+2?"
    answers: ["4", "5"]
    correct_answers: ["4"]
  - id: q2
    question: "Pick a color"
    answers: ["blue", "green"]
    correct_answers: ["blue"]
`
	if err := os.WriteFile(specPath, []byte(specBody), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	cfg := spec.Config{
		Repo:         spec.RepoConfig{OutputDir: "./out"},
		Agents:       []spec.AgentConfig{{ID: "agent-1", Type: "builtin", Provider: "openrouter", Model: "model"}},
		DefaultAgent: "agent-1",
		Tasks:        []spec.TaskConfig{{ID: "task-1", Type: "question_eval", Agent: "agent-1", QuestionsFile: "questions.yml"}},
	}

	limiter := &recordingLimiter{}
	_, err := Run(ctx, cfg, RunParams{
		RepoRoot: repoRoot,
		Deps: RunDependencies{
			ProviderFactory: func(_ spec.AgentConfig, _ string) (agent.Provider, error) {
				return fakeProvider{message: "Reasoning.\n<answer>4</answer>"}, nil
			},
			LimiterFactory: func(_ spec.Config, _ string) (ratelimiter.Limiter, error) {
				return limiter, nil
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
		t.Fatalf("run: %v", err)
	}
	if limiter.reserves() != 2 {
		t.Fatalf("expected 2 reserves, got %d", limiter.reserves())
	}
	if limiter.completes() != 2 {
		t.Fatalf("expected 2 completes, got %d", limiter.completes())
	}
}

// recordingLimiter tracks reserve and complete calls for tests.
type recordingLimiter struct {
	mu         sync.Mutex
	reservesN  int
	completesN int
}

// Reserve records a reservation and allows it.
func (l *recordingLimiter) Reserve(_ context.Context, _ ratelimiter.ReserveRequest) (ratelimiter.ReserveResponse, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reservesN++
	return ratelimiter.ReserveResponse{Allowed: true}, nil
}

// Complete records a completion and accepts it.
func (l *recordingLimiter) Complete(_ context.Context, _ ratelimiter.CompleteRequest) (ratelimiter.CompleteResponse, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.completesN++
	return ratelimiter.CompleteResponse{Ok: true}, nil
}

// BatchReserve allows batch reservations without recording.
func (l *recordingLimiter) BatchReserve(_ context.Context, req ratelimiter.BatchReserveRequest) (ratelimiter.BatchReserveResponse, error) {
	results := make([]ratelimiter.BatchReserveResult, 0, len(req.Requests))
	for range req.Requests {
		results = append(results, ratelimiter.BatchReserveResult{Allowed: true})
	}
	return ratelimiter.BatchReserveResponse{Results: results}, nil
}

// BatchComplete allows batch completions without recording.
func (l *recordingLimiter) BatchComplete(_ context.Context, req ratelimiter.BatchCompleteRequest) (ratelimiter.BatchCompleteResponse, error) {
	results := make([]ratelimiter.BatchCompleteResult, 0, len(req.Requests))
	for range req.Requests {
		results = append(results, ratelimiter.BatchCompleteResult{Ok: true})
	}
	return ratelimiter.BatchCompleteResponse{Results: results}, nil
}

// reserves returns the recorded reserve count.
func (l *recordingLimiter) reserves() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reservesN
}

// completes returns the recorded complete count.
func (l *recordingLimiter) completes() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.completesN
}
