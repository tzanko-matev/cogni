//go:build live
// +build live

package cli

import (
	"path/filepath"
	"testing"

	"cogni/internal/spec"
)

// TestE2EMultipleAgentsModelOverride checks model override per agent.
func TestE2EMultipleAgentsModelOverride(t *testing.T) {
	model := requireLiveLLM(t)
	override := modelOverride(model)
	repoRoot := simpleRepo(t)
	questionsPath := filepath.Join("spec", "questions", "agents.yml")
	writeFile(t, repoRoot, questionsPath, `version: 1
questions:
  - id: q1
    question: "What is the project name in README.md?"
    answers: ["Sample Service", "Other"]
    correct_answers: ["Sample Service"]
`)
	agents := []spec.AgentConfig{
		defaultAgent("default", model),
		defaultAgent("secondary", model),
	}
	cfg := baseConfig("./cogni-results", agents, "default", []spec.TaskConfig{{
		ID:            "t6a",
		Type:          "question_eval",
		Agent:         "default",
		QuestionsFile: questionsPath,
	}, {
		ID:            "t6b",
		Type:          "question_eval",
		Agent:         "secondary",
		Model:         override,
		QuestionsFile: questionsPath,
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
	if len(results.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(results.Agents))
	}
	foundDefault := false
	foundSecondary := false
	for _, agent := range results.Agents {
		switch agent.ID {
		case "default":
			foundDefault = true
		case "secondary":
			foundSecondary = true
			if agent.Model != override {
				t.Fatalf("expected override model %q, got %q", override, agent.Model)
			}
		}
	}
	if !foundDefault || !foundSecondary {
		t.Fatalf("expected agents default and secondary, got %+v", results.Agents)
	}
}

// TestE2EBudgetLimitFailure verifies budget exceeded failures are reported.
func TestE2EBudgetLimitFailure(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	questionsPath := filepath.Join("spec", "questions", "budget.yml")
	writeFile(t, repoRoot, questionsPath, `version: 1
questions:
  - id: q1
    question: "What is the project name in README.md?"
    answers: ["Sample Service", "Other"]
    correct_answers: ["Sample Service"]
`)
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:            "t7",
		Type:          "question_eval",
		Agent:         "default",
		QuestionsFile: questionsPath,
		Budget:        spec.TaskBudget{MaxTokens: 1},
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
