package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cogni/internal/agent"
	"cogni/internal/spec"
	"cogni/internal/testutil"
	"cogni/internal/tools"
	"cogni/internal/vcs"
	"cogni/pkg/ratelimiter"
)

// TestRemoteRateLimiterIntegration verifies remote limiter requests are issued.
func TestRemoteRateLimiterIntegration(t *testing.T) {
	ctx := testutil.Context(t, 2*time.Second)
	var reserveCount int32
	var completeCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/v1/reserve"):
			atomic.AddInt32(&reserveCount, 1)
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
				writeJSONResponse(t, w, ratelimiter.BatchReserveResponse{Results: results})
				return
			}
			writeJSONResponse(t, w, ratelimiter.ReserveResponse{Allowed: true})
		case strings.HasPrefix(r.URL.Path, "/v1/complete"):
			atomic.AddInt32(&completeCount, 1)
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
				writeJSONResponse(t, w, ratelimiter.BatchCompleteResponse{Results: results})
				return
			}
			writeJSONResponse(t, w, ratelimiter.CompleteResponse{Ok: true})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

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
		RateLimiter: spec.RateLimiterConfig{
			Mode:             "remote",
			BaseURL:          server.URL,
			Workers:          1,
			RequestTimeoutMs: 2000,
			Batch:            spec.BatchConfig{Size: 1, FlushMs: 1},
		},
		Agents:       []spec.AgentConfig{{ID: "agent-1", Type: "builtin", Provider: "openrouter", Model: "model"}},
		DefaultAgent: "agent-1",
		Tasks:        []spec.TaskConfig{{ID: "task-1", Type: "question_eval", Agent: "agent-1", QuestionsFile: "questions.yml"}},
	}

	_, err := Run(ctx, cfg, RunParams{
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
		t.Fatalf("run: %v", err)
	}
	if atomic.LoadInt32(&reserveCount) < 1 {
		t.Fatalf("expected at least 1 reserve request")
	}
	if atomic.LoadInt32(&completeCount) < 1 {
		t.Fatalf("expected at least 1 complete request")
	}
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
