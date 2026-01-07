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
	prompt := "Read README.md and report the project name. The answer must include the exact phrase \"Sample Service\". Cite README.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t1",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval: spec.TaskEval{
			ValidateCitations: true,
			MustContainStrings: []string{
				"Sample Service",
				"README.md",
			},
		},
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

// TestE2EBasicQACitations checks citation validation for a simple prompt.
func TestE2EBasicQACitations(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Read app.md and report the service owner. The answer must include the exact phrase \"Platform Team\". Cite app.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t2",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval: spec.TaskEval{
			ValidateCitations: true,
			MustContainStrings: []string{
				"Platform Team",
				"app.md",
			},
		},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if results.Tasks[0].Status != "pass" || !results.Tasks[0].Attempts[0].Eval.CitationValid {
		t.Fatalf("expected citation pass, got %+v", results.Tasks[0])
	}
}

// TestE2EMultiFileEvidence validates multi-file citations and answers.
func TestE2EMultiFileEvidence(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Using README.md and app.md, report the project name and service owner in one sentence. The answer must include \"Sample Service\" and \"Platform Team\". Include citations entries for README.md and app.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t3",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval: spec.TaskEval{
			ValidateCitations: true,
			MustContainStrings: []string{
				"Sample Service",
				"Platform Team",
				"README.md",
				"app.md",
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
