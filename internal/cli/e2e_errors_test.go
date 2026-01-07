//go:build live
// +build live

package cli

import (
	"testing"

	"cogni/internal/spec"
)

// TestE2EProviderFailureHandling asserts runtime errors are surfaced.
func TestE2EProviderFailureHandling(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Read README.md and report the project name. The answer must include the exact phrase \"Sample Service\". Cite README.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t12",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	t.Setenv("LLM_API_KEY", "invalid-key")
	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if results.Tasks[0].Status != "error" || results.Tasks[0].FailureReason == nil || *results.Tasks[0].FailureReason != "runtime_error" {
		t.Fatalf("expected runtime error failure, got %+v", results.Tasks[0])
	}
}
