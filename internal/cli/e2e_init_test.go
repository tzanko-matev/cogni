//go:build live
// +build live

package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"cogni/internal/spec"
)

// TestE2EInitToRunFlow validates init followed by a run.
func TestE2EInitToRunFlow(t *testing.T) {
	model := requireLiveLLM(t)
	requireGit(t)
	repoRoot := t.TempDir()
	runGit(t, repoRoot, "-c", "init.defaultBranch=main", "init")
	writeFile(t, repoRoot, "README.md", "init\n")
	runGit(t, repoRoot, "add", "README.md")
	runGit(t, repoRoot, "commit", "-m", "init")

	specPath := filepath.Join(repoRoot, ".cogni", "config.yml")
	origInput := initInput
	initInput = strings.NewReader("y\n\nn\n")
	t.Cleanup(func() { initInput = origInput })
	_, stderr, exitCode := runCLI(t, []string{"init", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("init failed: %s", stderr)
	}

	prompt := "Read README.md and return the project name. The answer must include the exact word \"init\". Cite README.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t11",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"init", "README.md"}},
	}})
	writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode = runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("run failed: %s", stderr)
	}
}
