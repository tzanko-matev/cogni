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
)

// TestQuestionEvalConcurrencyVerifiesSpeed checks concurrent execution reduces runtime.
func TestQuestionEvalConcurrencyVerifiesSpeed(t *testing.T) {
	ctx := testutil.Context(t, 2*time.Second)
	cfg := concurrencyConfig(t)
	cfg.RateLimiter.Mode = "disabled"
	cfg.RateLimiter.Workers = 2
	cfg.Tasks[0].Concurrency = 2

	provider := newBlockingProvider(2, 150*time.Millisecond)
	start := time.Now()
	_, err := Run(ctx, cfg, RunParams{
		RepoRoot: cfg.Repo.OutputDir,
		Deps:     runnerDepsForConcurrency(provider),
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed >= 300*time.Millisecond {
		t.Fatalf("expected concurrent runtime <300ms, got %s", elapsed)
	}
}

// TestQuestionEvalSequentialVerifiesSpeed checks sequential execution takes longer.
func TestQuestionEvalSequentialVerifiesSpeed(t *testing.T) {
	ctx := testutil.Context(t, 2*time.Second)
	cfg := concurrencyConfig(t)
	cfg.RateLimiter.Mode = "disabled"
	cfg.RateLimiter.Workers = 1
	cfg.Tasks[0].Concurrency = 1

	provider := newBlockingProvider(1, 150*time.Millisecond)
	start := time.Now()
	_, err := Run(ctx, cfg, RunParams{
		RepoRoot: cfg.Repo.OutputDir,
		Deps:     runnerDepsForConcurrency(provider),
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 300*time.Millisecond {
		t.Fatalf("expected sequential runtime >=300ms, got %s", elapsed)
	}
}

func concurrencyConfig(t *testing.T) spec.Config {
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
	return spec.Config{
		Repo: spec.RepoConfig{OutputDir: repoRoot},
		Agents: []spec.AgentConfig{
			{ID: "agent-1", Type: "builtin", Provider: "openrouter", Model: "model"},
		},
		DefaultAgent: "agent-1",
		Tasks: []spec.TaskConfig{
			{ID: "task-1", Type: "question_eval", Agent: "agent-1", QuestionsFile: "questions.yml"},
		},
	}
}

func runnerDepsForConcurrency(provider agent.Provider) RunDependencies {
	return RunDependencies{
		ProviderFactory: func(_ spec.AgentConfig, _ string) (agent.Provider, error) {
			return provider, nil
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
	}
}

// blockingProvider waits until a target number of calls begin, then sleeps per call.
type blockingProvider struct {
	releaseCount int
	sleep        time.Duration
	releaseOnce  sync.Once
	releaseCh    chan struct{}
	mu           sync.Mutex
	started      int
}

func newBlockingProvider(releaseCount int, sleep time.Duration) *blockingProvider {
	if releaseCount < 1 {
		releaseCount = 1
	}
	return &blockingProvider{
		releaseCount: releaseCount,
		sleep:        sleep,
		releaseCh:    make(chan struct{}),
	}
}

// Stream waits until enough calls start, then returns a response after sleeping.
func (p *blockingProvider) Stream(_ context.Context, _ agent.Prompt) (agent.Stream, error) {
	p.mu.Lock()
	p.started++
	if p.started >= p.releaseCount {
		p.releaseOnce.Do(func() { close(p.releaseCh) })
	}
	p.mu.Unlock()

	<-p.releaseCh
	time.Sleep(p.sleep)
	return &fakeStream{events: []agent.StreamEvent{{Type: agent.StreamEventMessage, Message: "Reasoning.\n<answer>4</answer>"}}}, nil
}
