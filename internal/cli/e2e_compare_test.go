//go:build live
// +build live

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogni/internal/spec"
)

// TestE2ECompareAcrossCommits validates compare/report across git history.
func TestE2ECompareAcrossCommits(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot, baseCommit, headCommit := historyRepo(t)
	questionsPath := filepath.Join("spec", "questions", "compare.yml")
	writeFile(t, repoRoot, questionsPath, `version: 1
questions:
  - id: q1
    question: "What is the project name in README.md?"
    answers: ["Sample Service", "Other"]
    correct_answers: ["Sample Service"]
`)
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:            "t9",
		Type:          "question_eval",
		Agent:         "default",
		QuestionsFile: questionsPath,
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	runGit(t, repoRoot, "checkout", baseCommit)
	if _, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath}); exitCode != ExitOK {
		t.Fatalf("base run failed: %s", stderr)
	}

	runGit(t, repoRoot, "checkout", headCommit)
	if _, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath}); exitCode != ExitOK {
		t.Fatalf("head run failed: %s", stderr)
	}

	stdout, stderr, exitCode := runCLI(t, []string{"compare", "--spec", specPath, "--base", baseCommit, "--head", headCommit})
	if exitCode != ExitOK {
		t.Fatalf("compare failed: %s", stderr)
	}
	if !strings.Contains(stdout, "Delta") {
		t.Fatalf("expected compare output, got %q", stdout)
	}

	reportPath := filepath.Join(outputDir(repoRoot, cfg.Repo.OutputDir), "report.html")
	stdout, stderr, exitCode = runCLI(t, []string{"report", "--spec", specPath, "--range", baseCommit + ".." + headCommit, "--output", reportPath})
	if exitCode != ExitOK {
		t.Fatalf("report failed: %s", stderr)
	}
	if !strings.Contains(stdout, "Report written") {
		t.Fatalf("expected report output, got %q", stdout)
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("missing report output: %v", err)
	}
}
