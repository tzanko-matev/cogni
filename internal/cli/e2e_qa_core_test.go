//go:build live
// +build live

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"cogni/internal/spec"
)

// TestE2EProviderConnectivity validates a live run produces expected artifacts.
func TestE2EProviderConnectivity(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	questionsPath := filepath.Join("spec", "questions", "core.yml")
	writeFile(t, repoRoot, questionsPath, `version: 1
questions:
  - id: q1
    question: "What is the project name in README.md?"
    answers: ["Sample Service", "Other"]
    correct_answers: ["Sample Service"]
`)
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:            "t1",
		Type:          "question_eval",
		Agent:         "default",
		QuestionsFile: questionsPath,
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	results, runDir := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if len(results.Tasks) != 1 || results.Tasks[0].Status != "pass" {
		t.Fatalf("expected pass result, got %+v", results.Tasks)
	}
	if _, err := os.Stat(filepath.Join(runDir, "results.json")); err != nil {
		t.Fatalf("missing results.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(runDir, "report.html")); err != nil {
		t.Fatalf("missing report.html: %v", err)
	}
}

// TestE2EBasicQuestionEval checks question evaluation results for a single question.
func TestE2EBasicQuestionEval(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	questionsPath := filepath.Join("spec", "questions", "owner.yml")
	writeFile(t, repoRoot, questionsPath, `version: 1
questions:
  - id: q1
    question: "Who owns the service in app.md?"
    answers: ["Platform Team", "Other"]
    correct_answers: ["Platform Team"]
`)
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:            "t2",
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

// TestE2EMultiQuestionEval validates multi-question evaluation results.
func TestE2EMultiFileEvidence(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	questionsPath := filepath.Join("spec", "questions", "multi.yml")
	writeFile(t, repoRoot, questionsPath, `version: 1
questions:
  - id: q1
    question: "What is the project name in README.md?"
    answers: ["Sample Service", "Other"]
    correct_answers: ["Sample Service"]
  - id: q2
    question: "Who owns the service in app.md?"
    answers: ["Platform Team", "Other"]
    correct_answers: ["Platform Team"]
`)
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:            "t3",
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
	if results.Tasks[0].QuestionEval == nil || results.Tasks[0].QuestionEval.Summary.QuestionsTotal != 2 {
		t.Fatalf("expected 2 questions, got %+v", results.Tasks[0].QuestionEval)
	}
}
