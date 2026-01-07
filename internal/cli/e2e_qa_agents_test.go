//go:build live
// +build live

package cli

import (
	"testing"

	"cogni/internal/spec"
)

// TestE2EMultipleAgentsModelOverride checks model override per agent.
func TestE2EMultipleAgentsModelOverride(t *testing.T) {
	model := requireLiveLLM(t)
	override := modelOverride(model)
	repoRoot := simpleRepo(t)
	agents := []spec.AgentConfig{
		defaultAgent("default", model),
		defaultAgent("secondary", model),
	}
	prompt := "Read README.md and report the project name. The answer must include the exact phrase \"Sample Service\". Cite README.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", agents, "default", []spec.TaskConfig{{
		ID:     "t6a",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"Sample Service", "README.md"}},
	}, {
		ID:     "t6b",
		Type:   "qa",
		Agent:  "secondary",
		Model:  override,
		Prompt: prompt,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"Sample Service", "README.md"}},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if len(results.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(results.Tasks))
	}
	if results.Tasks[0].Attempts[0].AgentID != "default" {
		t.Fatalf("unexpected task 1 agent: %+v", results.Tasks[0].Attempts[0])
	}
	if results.Tasks[1].Attempts[0].AgentID != "secondary" || results.Tasks[1].Attempts[0].Model != override {
		t.Fatalf("unexpected task 2 agent/model: %+v", results.Tasks[1].Attempts[0])
	}
}

// TestE2EBudgetLimitFailure verifies budget exceeded failures are reported.
func TestE2EBudgetLimitFailure(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Before answering, call the list_files tool with an empty glob. Do not answer until after the tool result. After the tool response, return ONLY JSON with keys \"answer\" and \"citations\".\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t7",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Budget: spec.TaskBudget{MaxSteps: 1},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if results.Tasks[0].Status != "fail" || results.Tasks[0].FailureReason == nil || *results.Tasks[0].FailureReason != "budget_exceeded" {
		t.Fatalf("expected budget exceeded failure, got %+v", results.Tasks[0])
	}
}
