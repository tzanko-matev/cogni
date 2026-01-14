package runner

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"cogni/internal/agent"
	"cogni/internal/question"
	"cogni/internal/spec"
	"cogni/internal/testutil"
	"cogni/internal/tools"
	"cogni/internal/vcs"
	"cogni/pkg/ratelimiter"
)

// TestRunObserverEmitsQuestionLifecycle verifies ordered question events.
func TestRunObserverEmitsQuestionLifecycle(t *testing.T) {
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
		Repo:         spec.RepoConfig{OutputDir: "./out"},
		Agents:       []spec.AgentConfig{{ID: "agent-1", Type: "builtin", Provider: "openrouter", Model: "model"}},
		DefaultAgent: "agent-1",
		Tasks:        []spec.TaskConfig{{ID: "task-1", Type: "question_eval", Agent: "agent-1", QuestionsFile: "questions.yml"}},
	}
	observer := &recordingObserver{}
	ctx := testutil.Context(t, time.Second)
	_, err := Run(ctx, cfg, RunParams{
		RepoRoot: repoRoot,
		Observer: observer,
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
		t.Fatalf("run: %v", err)
	}

	events := observer.eventsForQuestion(0)
	expected := []QuestionEventType{
		QuestionQueued,
		QuestionScheduled,
		QuestionReserving,
		QuestionRunning,
		QuestionParsing,
		QuestionCorrect,
	}
	assertSequence(t, events, expected)
}

// TestRunObserverReportsRateLimitWait verifies retry events are emitted.
func TestRunObserverReportsRateLimitWait(t *testing.T) {
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
		Repo:         spec.RepoConfig{OutputDir: "./out"},
		Agents:       []spec.AgentConfig{{ID: "agent-1", Type: "builtin", Provider: "openrouter", Model: "model"}},
		DefaultAgent: "agent-1",
		Tasks:        []spec.TaskConfig{{ID: "task-1", Type: "question_eval", Agent: "agent-1", QuestionsFile: "questions.yml"}},
	}
	observer := &recordingObserver{}
	limiter := &retryLimiter{}
	ctx := testutil.Context(t, time.Second)
	_, err := Run(ctx, cfg, RunParams{
		RepoRoot: repoRoot,
		Observer: observer,
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

	waitEvent := observer.findEvent(QuestionWaitingRateLimit)
	if waitEvent == nil {
		t.Fatalf("expected waiting_rate_limit event")
	}
	if waitEvent.RetryAfterMs <= 0 {
		t.Fatalf("expected retry_after_ms > 0")
	}
}

// TestObservedToolExecutorEmitsEvents verifies tool events are emitted.
func TestObservedToolExecutorEmitsEvents(t *testing.T) {
	observer := &recordingObserver{}
	jobObserver := newQuestionJobObserver(observer, "task-1", []question.Question{{
		ID:     "q1",
		Prompt: "Question",
	}})
	executor := newObservedToolExecutor(jobObserver, 0, stubExecutor{})
	_ = executor.Execute(context.Background(), agent.ToolCall{Name: "search", Args: agent.ToolCallArgs{}})

	events := observer.eventsForQuestion(0)
	expected := []QuestionEventType{QuestionToolStart, QuestionToolFinish}
	assertSequence(t, events, expected)
}

// recordingObserver stores events for assertions.
type recordingObserver struct {
	mu     sync.Mutex
	events []QuestionEvent
}

// OnRunStart records run starts.
func (o *recordingObserver) OnRunStart(_ string, _ string) {}

// OnTaskStart records task starts.
func (o *recordingObserver) OnTaskStart(_ string, _ string, _ string, _ string, _ string) {}

// OnQuestionEvent stores question events.
func (o *recordingObserver) OnQuestionEvent(event QuestionEvent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
}

// OnTaskEnd records task ends.
func (o *recordingObserver) OnTaskEnd(_ string, _ string, _ *string) {}

// OnRunEnd records run completion.
func (o *recordingObserver) OnRunEnd(_ Results) {}

// eventsForQuestion returns ordered event types for a question index.
func (o *recordingObserver) eventsForQuestion(index int) []QuestionEventType {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]QuestionEventType, 0, len(o.events))
	for _, event := range o.events {
		if event.QuestionIndex == index {
			out = append(out, event.Type)
		}
	}
	return out
}

// findEvent returns the first event matching a type.
func (o *recordingObserver) findEvent(kind QuestionEventType) *QuestionEvent {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, event := range o.events {
		if event.Type == kind {
			copy := event
			return &copy
		}
	}
	return nil
}

// assertSequence ensures expected events appear in order.
func assertSequence(t *testing.T, events []QuestionEventType, expected []QuestionEventType) {
	t.Helper()
	pos := 0
	for _, event := range events {
		if pos < len(expected) && event == expected[pos] {
			pos++
		}
	}
	if pos != len(expected) {
		t.Fatalf("expected sequence %v, got %v", expected, events)
	}
}

// stubExecutor returns a successful tool result.
type stubExecutor struct{}

// Execute returns a simple tool result for tests.
func (stubExecutor) Execute(_ context.Context, call agent.ToolCall) tools.CallResult {
	return tools.CallResult{
		Tool:     call.Name,
		Duration: 10 * time.Millisecond,
	}
}

// retryLimiter denies the first reserve to force retry.
type retryLimiter struct {
	mu    sync.Mutex
	calls int
}

// Reserve denies the first call and allows subsequent requests.
func (l *retryLimiter) Reserve(_ context.Context, _ ratelimiter.ReserveRequest) (ratelimiter.ReserveResponse, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.calls == 0 {
		l.calls++
		return ratelimiter.ReserveResponse{Allowed: false, RetryAfterMs: 1}, nil
	}
	return ratelimiter.ReserveResponse{Allowed: true}, nil
}

// Complete accepts completion requests.
func (l *retryLimiter) Complete(_ context.Context, _ ratelimiter.CompleteRequest) (ratelimiter.CompleteResponse, error) {
	return ratelimiter.CompleteResponse{Ok: true}, nil
}

// BatchReserve accepts batch reserves.
func (l *retryLimiter) BatchReserve(_ context.Context, req ratelimiter.BatchReserveRequest) (ratelimiter.BatchReserveResponse, error) {
	results := make([]ratelimiter.BatchReserveResult, 0, len(req.Requests))
	for range req.Requests {
		results = append(results, ratelimiter.BatchReserveResult{Allowed: true})
	}
	return ratelimiter.BatchReserveResponse{Results: results}, nil
}

// BatchComplete accepts batch completes.
func (l *retryLimiter) BatchComplete(_ context.Context, req ratelimiter.BatchCompleteRequest) (ratelimiter.BatchCompleteResponse, error) {
	results := make([]ratelimiter.BatchCompleteResult, 0, len(req.Requests))
	for range req.Requests {
		results = append(results, ratelimiter.BatchCompleteResult{Ok: true})
	}
	return ratelimiter.BatchCompleteResponse{Results: results}, nil
}
