//go:build live
// +build live

package cli

import (
	"testing"

	"cogni/internal/spec"
)

// TestE2ERepositoryNavigation exercises repository path reads.
func TestE2ERepositoryNavigation(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Read config/app-config.yml and report its path. The answer must include the exact string \"config/app-config.yml\". Cite config/app-config.yml.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t4",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval: spec.TaskEval{
			ValidateCitations: true,
			MustContainStrings: []string{
				"config/app-config.yml",
			},
		},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if results.Tasks[0].Status != "pass" {
		t.Fatalf("expected pass, got %+v", results.Tasks[0])
	}
}

// TestE2EMultipleTasksSummary verifies summary counts across tasks.
func TestE2EMultipleTasksSummary(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	promptA := "Read README.md and report the project name. The answer must include the exact phrase \"Sample Service\". Cite README.md.\n\n" + jsonRules
	promptB := "Read app.md and report the service owner. The answer must include the exact phrase \"Platform Team\". Cite app.md.\n\n" + jsonRules
	promptC := "Read config/app-config.yml and report the mode value. The answer must include the exact word \"sample\". Cite config/app-config.yml.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t5a",
		Type:   "qa",
		Agent:  "default",
		Prompt: promptA,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"Sample Service", "README.md"}},
	}, {
		ID:     "t5b",
		Type:   "qa",
		Agent:  "default",
		Prompt: promptB,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"Platform Team", "app.md"}},
	}, {
		ID:     "t5c",
		Type:   "qa",
		Agent:  "default",
		Prompt: promptC,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"sample", "config/app-config.yml"}},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if results.Summary.TasksTotal != 3 || results.Summary.TasksPassed != 3 || results.Summary.TasksFailed != 0 {
		t.Fatalf("unexpected summary: %+v", results.Summary)
	}
	if len(results.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(results.Tasks))
	}
}
