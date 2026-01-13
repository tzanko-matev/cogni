//go:build live
// +build live

package cli

import (
	"path/filepath"
	"testing"

	"cogni/internal/spec"
)

// TestE2ERepositoryNavigation exercises repository path reads.
func TestE2ERepositoryNavigation(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	questionsPath := filepath.Join("spec", "questions", "repo.yml")
	writeFile(t, repoRoot, questionsPath, `version: 1
questions:
  - id: q1
    question: "Which config path contains the mode value?"
    answers: ["config/app-config.yml", "config.yml"]
    correct_answers: ["config/app-config.yml"]
`)
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:            "t4",
		Type:          "question_eval",
		Agent:         "default",
		QuestionsFile: questionsPath,
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
	questionsPathA := filepath.Join("spec", "questions", "summary-a.yml")
	questionsPathB := filepath.Join("spec", "questions", "summary-b.yml")
	questionsPathC := filepath.Join("spec", "questions", "summary-c.yml")
	writeFile(t, repoRoot, questionsPathA, `version: 1
questions:
  - id: q1
    question: "What is the project name in README.md?"
    answers: ["Sample Service", "Other"]
    correct_answers: ["Sample Service"]
`)
	writeFile(t, repoRoot, questionsPathB, `version: 1
questions:
  - id: q1
    question: "Who owns the service in app.md?"
    answers: ["Platform Team", "Other"]
    correct_answers: ["Platform Team"]
`)
	writeFile(t, repoRoot, questionsPathC, `version: 1
questions:
  - id: q1
    question: "What is the mode in config/app-config.yml?"
    answers: ["sample", "other"]
    correct_answers: ["sample"]
`)
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:            "t5a",
		Type:          "question_eval",
		Agent:         "default",
		QuestionsFile: questionsPathA,
	}, {
		ID:            "t5b",
		Type:          "question_eval",
		Agent:         "default",
		QuestionsFile: questionsPathB,
	}, {
		ID:            "t5c",
		Type:          "question_eval",
		Agent:         "default",
		QuestionsFile: questionsPathC,
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
