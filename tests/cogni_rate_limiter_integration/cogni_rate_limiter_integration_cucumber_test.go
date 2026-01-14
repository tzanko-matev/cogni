//go:build cucumber

package cogniratelimiterintegration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cucumber/godog"

	"cogni/internal/agent"
	"cogni/internal/runner"
	"cogni/internal/spec"
	"cogni/internal/tools"
	"cogni/internal/vcs"
	"cogni/pkg/ratelimiter"
)

// TestCogniRateLimiterIntegrationFeatures executes the Cogni rate limiter scenarios via godog.
func TestCogniRateLimiterIntegrationFeatures(t *testing.T) {
	featurePath := filepath.Join("..", "..", "spec", "features", "cogni-rate-limiter-integration", "testing.feature")
	suite := godog.TestSuite{
		Name:                "cogni-rate-limiter-integration",
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:    "pretty",
			Paths:     []string{featurePath},
			Strict:    true,
			TestingT:  t,
			Randomize: 0,
		},
	}
	if suite.Run() != 0 {
		t.Fatalf("non-zero godog status")
	}
}

// InitializeScenario wires step definitions for the feature tests.
func InitializeScenario(ctx *godog.ScenarioContext) {
	state := &cogniRateLimiterState{}
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		return ctx, state.reset()
	})
	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		state.close()
		return ctx, nil
	})

	ctx.Step(`^a question spec with (\d+) questions$`, state.givenQuestionSpec)
	ctx.Step(`^a question spec with (\d+) question$`, state.givenQuestionSpec)
	ctx.Step(`^a fake provider that sleeps (\d+) milliseconds per call$`, state.givenSleepProvider)
	ctx.Step(`^a config with rate_limiter mode "([^"]+)" and workers (\d+)$`, state.givenConfigWithModeAndWorkers)
	ctx.Step(`^a limits file with concurrency capacity (\d+) for provider "([^"]+)" model "([^"]+)"$`, state.givenLimitsFile)
	ctx.Step(`^a stub ratelimiterd server that always allows$`, state.givenStubServer)
	ctx.Step(`^a config with rate_limiter mode "([^"]+)" and the stub base URL$`, state.givenConfigWithModeAndBaseURL)
	ctx.Step(`^I run "([^"]+)"$`, state.whenIRunCommand)
	ctx.Step(`^the run completes within (\d+) milliseconds$`, state.thenRunCompletesWithin)
	ctx.Step(`^the run completes within (\d+) second$`, state.thenRunCompletesWithinSeconds)
	ctx.Step(`^no more than (\d+) call is in flight at any time$`, state.thenMaxInFlight)
	ctx.Step(`^the server receives at least (\d+) reserve request$`, state.thenServerReceivesReserves)
	ctx.Step(`^the server receives at least (\d+) complete request$`, state.thenServerReceivesCompletes)
}

type cogniRateLimiterState struct {
	repoRoot      string
	questionsPath string
	limitsPath    string
	config        spec.Config
	provider      *sleepProvider
	runDuration   time.Duration
	runErr        error
	server        *httptest.Server
	reserveCount  int32
	completeCount int32
}

func (s *cogniRateLimiterState) reset() error {
	s.close()
	repoRoot, err := os.MkdirTemp("", "cogni-bdd-")
	if err != nil {
		return err
	}
	s.repoRoot = repoRoot
	s.questionsPath = filepath.Join("spec", "questions", "sample.yml")
	s.config = spec.Config{
		Version: 1,
		Repo:    spec.RepoConfig{OutputDir: repoRoot},
		Agents: []spec.AgentConfig{{
			ID:       "default",
			Type:     "builtin",
			Provider: "openrouter",
			Model:    "model",
		}},
		DefaultAgent: "default",
		Tasks: []spec.TaskConfig{{
			ID:            "question-eval",
			Type:          "question_eval",
			Agent:         "default",
			QuestionsFile: s.questionsPath,
		}},
	}
	s.provider = nil
	s.runDuration = 0
	s.runErr = nil
	s.reserveCount = 0
	s.completeCount = 0
	return nil
}

func (s *cogniRateLimiterState) close() {
	if s.server != nil {
		s.server.Close()
		s.server = nil
	}
	if s.repoRoot != "" {
		_ = os.RemoveAll(s.repoRoot)
		s.repoRoot = ""
	}
}

func (s *cogniRateLimiterState) givenQuestionSpec(count int) error {
	if count < 1 {
		return fmt.Errorf("question count must be >= 1")
	}
	dir := filepath.Join(s.repoRoot, filepath.Dir(s.questionsPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var builder strings.Builder
	builder.WriteString("version: 1\nquestions:\n")
	for i := 1; i <= count; i++ {
		builder.WriteString(fmt.Sprintf("  - id: q%d\n", i))
		builder.WriteString(fmt.Sprintf("    question: \"Question %d?\"\n", i))
		builder.WriteString("    answers: [\"4\", \"5\"]\n")
		builder.WriteString("    correct_answers: [\"4\"]\n")
	}
	path := filepath.Join(s.repoRoot, s.questionsPath)
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func (s *cogniRateLimiterState) givenSleepProvider(milliseconds int) error {
	if milliseconds < 0 {
		return fmt.Errorf("sleep duration must be >= 0")
	}
	s.provider = newSleepProvider(time.Duration(milliseconds) * time.Millisecond)
	return nil
}

func (s *cogniRateLimiterState) givenConfigWithModeAndWorkers(mode string, workers int) error {
	if workers < 1 {
		return fmt.Errorf("workers must be >= 1")
	}
	s.config.RateLimiter = spec.RateLimiterConfig{
		Mode:             mode,
		Workers:          workers,
		RequestTimeoutMs: 2000,
		MaxOutputTokens:  2048,
		Batch:            spec.BatchConfig{Size: 1, FlushMs: 1},
	}
	s.config.Tasks[0].Concurrency = workers
	if mode == "embedded" {
		if s.limitsPath == "" {
			return fmt.Errorf("limits file not configured")
		}
		s.config.RateLimiter.LimitsPath = s.limitsPath
	}
	return nil
}

func (s *cogniRateLimiterState) givenLimitsFile(capacity int, provider, model string) error {
	if capacity < 1 {
		return fmt.Errorf("capacity must be >= 1")
	}
	limitsDir := filepath.Join(s.repoRoot, ".cogni")
	if err := os.MkdirAll(limitsDir, 0o755); err != nil {
		return err
	}
	s.limitsPath = filepath.Join(limitsDir, "limits.json")

	defs := []ratelimiter.LimitDefinition{
		{
			Key:           ratelimiter.LimitKey(fmt.Sprintf("global:llm:%s:%s:rpm", provider, model)),
			Kind:          ratelimiter.KindRolling,
			Capacity:      1000,
			WindowSeconds: 60,
			Unit:          "requests",
			Overage:       ratelimiter.OverageDebt,
		},
		{
			Key:           ratelimiter.LimitKey(fmt.Sprintf("global:llm:%s:%s:tpm", provider, model)),
			Kind:          ratelimiter.KindRolling,
			Capacity:      100000,
			WindowSeconds: 60,
			Unit:          "tokens",
			Overage:       ratelimiter.OverageDebt,
		},
		{
			Key:            ratelimiter.LimitKey(fmt.Sprintf("global:llm:%s:%s:concurrency", provider, model)),
			Kind:           ratelimiter.KindConcurrency,
			Capacity:       uint64(capacity),
			TimeoutSeconds: 1,
			Unit:           "requests",
			Overage:        ratelimiter.OverageDebt,
		},
	}
	states := make([]ratelimiter.LimitState, 0, len(defs))
	for _, def := range defs {
		states = append(states, ratelimiter.LimitState{Definition: def, Status: ratelimiter.LimitStatusActive})
	}
	payload, err := json.Marshal(states)
	if err != nil {
		return err
	}
	return os.WriteFile(s.limitsPath, payload, 0o644)
}

func (s *cogniRateLimiterState) givenStubServer() error {
	if s.server != nil {
		s.server.Close()
	}
	atomic.StoreInt32(&s.reserveCount, 0)
	atomic.StoreInt32(&s.completeCount, 0)
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/v1/reserve"):
			atomic.AddInt32(&s.reserveCount, 1)
			if strings.HasSuffix(r.URL.Path, "/batch") {
				req := ratelimiter.BatchReserveRequest{}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				results := make([]ratelimiter.BatchReserveResult, 0, len(req.Requests))
				for range req.Requests {
					results = append(results, ratelimiter.BatchReserveResult{Allowed: true})
				}
				writeJSONResponse(w, ratelimiter.BatchReserveResponse{Results: results})
				return
			}
			writeJSONResponse(w, ratelimiter.ReserveResponse{Allowed: true})
		case strings.HasPrefix(r.URL.Path, "/v1/complete"):
			atomic.AddInt32(&s.completeCount, 1)
			if strings.HasSuffix(r.URL.Path, "/batch") {
				req := ratelimiter.BatchCompleteRequest{}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				results := make([]ratelimiter.BatchCompleteResult, 0, len(req.Requests))
				for range req.Requests {
					results = append(results, ratelimiter.BatchCompleteResult{Ok: true})
				}
				writeJSONResponse(w, ratelimiter.BatchCompleteResponse{Results: results})
				return
			}
			writeJSONResponse(w, ratelimiter.CompleteResponse{Ok: true})
		default:
			http.NotFound(w, r)
		}
	}))
	return nil
}

func (s *cogniRateLimiterState) givenConfigWithModeAndBaseURL(mode string) error {
	if s.server == nil {
		return fmt.Errorf("stub server not configured")
	}
	s.config.RateLimiter = spec.RateLimiterConfig{
		Mode:             mode,
		BaseURL:          s.server.URL,
		Workers:          1,
		RequestTimeoutMs: 2000,
		MaxOutputTokens:  2048,
		Batch:            spec.BatchConfig{Size: 1, FlushMs: 1},
	}
	s.config.Tasks[0].Concurrency = 1
	return nil
}

func (s *cogniRateLimiterState) whenIRunCommand(_ string) error {
	if s.provider == nil {
		s.provider = newSleepProvider(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	_, err := runner.Run(ctx, s.config, runner.RunParams{
		RepoRoot: s.repoRoot,
		Deps: runner.RunDependencies{
			ProviderFactory: func(_ spec.AgentConfig, _ string) (agent.Provider, error) {
				return s.provider, nil
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
	s.runDuration = time.Since(start)
	s.runErr = err
	return err
}

func (s *cogniRateLimiterState) thenRunCompletesWithin(limitMs int) error {
	if s.runErr != nil {
		return fmt.Errorf("run failed: %w", s.runErr)
	}
	limit := time.Duration(limitMs) * time.Millisecond
	if s.runDuration > limit {
		return fmt.Errorf("run duration %s exceeded %s", s.runDuration, limit)
	}
	return nil
}

func (s *cogniRateLimiterState) thenRunCompletesWithinSeconds(limitSeconds int) error {
	return s.thenRunCompletesWithin(limitSeconds * 1000)
}

func (s *cogniRateLimiterState) thenMaxInFlight(max int) error {
	if s.provider == nil {
		return fmt.Errorf("provider not configured")
	}
	observed := s.provider.MaxInFlight()
	if int(observed) > max {
		return fmt.Errorf("observed max in-flight %d exceeds %d", observed, max)
	}
	return nil
}

func (s *cogniRateLimiterState) thenServerReceivesReserves(min int) error {
	count := atomic.LoadInt32(&s.reserveCount)
	if int(count) < min {
		return fmt.Errorf("expected at least %d reserve requests, got %d", min, count)
	}
	return nil
}

func (s *cogniRateLimiterState) thenServerReceivesCompletes(min int) error {
	count := atomic.LoadInt32(&s.completeCount)
	if int(count) < min {
		return fmt.Errorf("expected at least %d complete requests, got %d", min, count)
	}
	return nil
}

// sleepProvider returns a stream that sleeps per call and tracks concurrency.
type sleepProvider struct {
	sleep       time.Duration
	inFlight    int32
	maxInFlight int32
}

func newSleepProvider(sleep time.Duration) *sleepProvider {
	return &sleepProvider{sleep: sleep}
}

// Stream waits for the sleep duration, emits a response, and tracks in-flight counts.
func (p *sleepProvider) Stream(_ context.Context, _ agent.Prompt) (agent.Stream, error) {
	current := atomic.AddInt32(&p.inFlight, 1)
	for {
		max := atomic.LoadInt32(&p.maxInFlight)
		if current <= max || atomic.CompareAndSwapInt32(&p.maxInFlight, max, current) {
			break
		}
	}
	return &sleepStream{sleep: p.sleep, onDone: func() {
		atomic.AddInt32(&p.inFlight, -1)
	}}, nil
}

// MaxInFlight reports the maximum concurrent calls observed.
func (p *sleepProvider) MaxInFlight() int32 {
	return atomic.LoadInt32(&p.maxInFlight)
}

type sleepStream struct {
	sleep  time.Duration
	sent   bool
	onDone func()
	once   sync.Once
}

// Recv emits a single answer after sleeping, then EOF.
func (s *sleepStream) Recv() (agent.StreamEvent, error) {
	if !s.sent {
		time.Sleep(s.sleep)
		s.sent = true
		return agent.StreamEvent{Type: agent.StreamEventMessage, Message: "Reasoning.\n<answer>4</answer>"}, nil
	}
	s.once.Do(func() {
		if s.onDone != nil {
			s.onDone()
		}
	})
	return agent.StreamEvent{}, io.EOF
}

func writeJSONResponse(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
