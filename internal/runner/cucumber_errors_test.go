package runner

import (
	"context"
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

// TestRunCucumberEvalManualMissingIDs verifies missing ids produce an error.
func TestRunCucumberEvalManualMissingIDs(t *testing.T) {
	repoRoot := t.TempDir()
	featuresDir := filepath.Join(repoRoot, "features")
	if err := os.MkdirAll(featuresDir, 0o755); err != nil {
		t.Fatalf("mkdir features: %v", err)
	}
	featurePath := filepath.Join(featuresDir, "sample.feature")
	feature := `Feature: Sample

  @id:alpha
  Scenario: First
    Given something

  @id:beta
  Scenario: Second
    Given something
`
	if err := os.WriteFile(featurePath, []byte(feature), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	expectationsDir := filepath.Join(repoRoot, "expectations")
	if err := os.MkdirAll(expectationsDir, 0o755); err != nil {
		t.Fatalf("mkdir expectations: %v", err)
	}
	expectations := `examples:
  alpha:1: true
  beta:1: false
`
	if err := os.WriteFile(filepath.Join(expectationsDir, "expectations.yml"), []byte(expectations), 0o644); err != nil {
		t.Fatalf("write expectations: %v", err)
	}

	cfg := spec.Config{
		Repo: spec.RepoConfig{OutputDir: "./out"},
		Agents: []spec.AgentConfig{
			{ID: "agent-1", Type: "builtin", Provider: "openrouter", Model: "model"},
		},
		DefaultAgent: "agent-1",
		Adapters: []spec.AdapterConfig{{
			ID:              "manual",
			Type:            "cucumber_manual",
			FeatureRoots:    []string{"features"},
			ExpectationsDir: expectationsDir,
		}},
		Tasks: []spec.TaskConfig{{
			ID:             "cucumber-task",
			Type:           "cucumber_eval",
			Agent:          "agent-1",
			Adapter:        "manual",
			Features:       []string{featurePath},
		}},
	}

	ctx := testutil.Context(t, 0)
	results, err := Run(ctx, cfg, RunParams{
		RepoRoot: repoRoot,
		Deps: RunDependencies{
			ProviderFactory: func(_ spec.AgentConfig, _ string) (agent.Provider, error) {
				return cucumberProvider{
					implementedByID: map[string]bool{"alpha:1": true, "beta:1": false},
					responseIDs:     []string{"alpha:1"},
				}, nil
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
	if len(results.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(results.Tasks))
	}
	task := results.Tasks[0]
	if task.Status != "error" {
		t.Fatalf("expected error, got %+v", task)
	}
	if task.FailureReason == nil || *task.FailureReason != "invalid_agent_response" {
		t.Fatalf("expected invalid_agent_response, got %+v", task)
	}
}
