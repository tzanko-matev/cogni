package runner

import (
	"context"
	"fmt"
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
}

func (p cucumberProvider) Stream(_ context.Context, prompt agent.Prompt) (agent.Stream, error) {
	exampleID := extractExampleID(prompt)
	implemented := p.implementedByID[exampleID]
	message := fmt.Sprintf(`{"example_id":"%s","implemented":%t}`, exampleID, implemented)
	return &fakeStream{events: []agent.StreamEvent{{Type: agent.StreamEventMessage, Message: message}}}, nil
}

func extractExampleID(prompt agent.Prompt) string {
	for i := len(prompt.InputItems) - 1; i >= 0; i-- {
		item := prompt.InputItems[i]
		if item.Role != "user" {
			continue
		}
		text, ok := item.Content.(string)
		if !ok {
			continue
		}
		if strings.Contains(text, "example_id:") {
			parts := strings.SplitN(text, "example_id:", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
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
			PromptTemplate: "example_id: {example_id}",
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
	if task.Cucumber.Summary.ExamplesCorrect != 2 {
		t.Fatalf("expected 2 correct, got %+v", task.Cucumber.Summary)
	}
	if task.Cucumber.Summary.Accuracy != 1.0 {
		t.Fatalf("expected accuracy 1.0, got %v", task.Cucumber.Summary.Accuracy)
	}
}
