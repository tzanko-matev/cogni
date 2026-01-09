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
	prompt := "Read README.md and report the project name. The answer must include the exact phrase \"Sample Service\". Cite README.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t9",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"Sample Service", "README.md"}},
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
