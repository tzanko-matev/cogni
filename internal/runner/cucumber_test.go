package runner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogni/internal/agent"
	"cogni/internal/spec"
	"cogni/internal/tools"
)

type cucumberProvider struct {
	implementedByID map[string]bool
	responseIDs     []string
}

func (p cucumberProvider) Stream(_ context.Context, prompt agent.Prompt) (agent.Stream, error) {
	exampleIDs := p.responseIDs
	if len(exampleIDs) == 0 {
		exampleIDs = extractExampleIDs(prompt)
	}
	results := make([]map[string]any, 0, len(exampleIDs))
	for _, exampleID := range exampleIDs {
		results = append(results, map[string]any{
			"example_id":  exampleID,
			"implemented": p.implementedByID[exampleID],
		})
	}
	payload, err := json.Marshal(map[string]any{"results": results})
	if err != nil {
		return nil, err
	}
	message := string(payload)
	return &fakeStream{events: []agent.StreamEvent{{Type: agent.StreamEventMessage, Message: message}}}, nil
}

func extractExampleIDs(prompt agent.Prompt) []string {
	for i := len(prompt.InputItems) - 1; i >= 0; i-- {
		item := prompt.InputItems[i]
		if item.Role != "user" {
			continue
		}
		text, ok := item.Content.(string)
		if !ok {
			continue
		}
		if strings.Contains(text, "example_ids:") {
			parts := strings.SplitN(text, "example_ids:", 2)
			if len(parts) != 2 {
				return nil
			}
			lines := strings.Split(parts[1], "\n")
			ids := make([]string, 0, len(lines))
			for _, line := range lines {
				id := strings.TrimSpace(line)
				if id != "" {
					ids = append(ids, id)
				}
			}
			return ids
		}
	}
	return nil
}

func TestRunCucumberEvalManual(t *testing.T) {
	repoRoot := initGitRepo(t)
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
			PromptTemplate: "example_ids:\n{example_ids}",
			Features:       []string{featurePath},
		}},
	}

	results, err := Run(context.Background(), cfg, RunParams{
		RepoRoot: repoRoot,
		Deps: RunDependencies{
			ProviderFactory: func(_ spec.AgentConfig, _ string) (agent.Provider, error) {
				return cucumberProvider{implementedByID: map[string]bool{"alpha:1": true, "beta:1": false}}, nil
			},
			ToolRunnerFactory: func(root string) (*tools.Runner, error) {
				return tools.NewRunner(root)
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
	if task.Status != "pass" {
		t.Fatalf("expected pass, got %+v", task)
	}
	if task.Cucumber == nil {
		t.Fatalf("expected cucumber results")
	}
	if len(task.Cucumber.FeatureRuns) != 1 {
		t.Fatalf("expected 1 feature run, got %d", len(task.Cucumber.FeatureRuns))
	}
	if task.Cucumber.FeatureRuns[0].ExamplesTotal != 2 {
		t.Fatalf("expected 2 examples in feature run, got %+v", task.Cucumber.FeatureRuns[0])
	}
	if task.Cucumber.Summary.ExamplesCorrect != 2 {
		t.Fatalf("expected 2 correct, got %+v", task.Cucumber.Summary)
	}
	if task.Cucumber.Summary.Accuracy != 1.0 {
		t.Fatalf("expected accuracy 1.0, got %v", task.Cucumber.Summary.Accuracy)
	}
}

func TestRunCucumberEvalManualMissingIDs(t *testing.T) {
	repoRoot := initGitRepo(t)
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
			PromptTemplate: "example_ids:\n{example_ids}",
			Features:       []string{featurePath},
		}},
	}

	results, err := Run(context.Background(), cfg, RunParams{
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
