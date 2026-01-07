//go:build live
// +build live

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogni/internal/spec"
)

// TestE2EOutputArtifacts checks that result and report files are generated.
func TestE2EOutputArtifacts(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Read README.md and report the project name. The answer must include the exact phrase \"Sample Service\". Cite README.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t8",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"Sample Service", "README.md"}},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	_, runDir := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	resultsPath := filepath.Join(runDir, "results.json")
	reportPath := filepath.Join(runDir, "report.html")

	resultsPayload, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("read results: %v", err)
	}
	if !bytes.Contains(resultsPayload, []byte(`"run_id"`)) || !bytes.Contains(resultsPayload, []byte(`"tasks"`)) {
		t.Fatalf("results.json missing expected fields")
	}

	reportPayload, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(reportPayload), "Cogni Report") {
		t.Fatalf("report.html missing heading")
	}
}
