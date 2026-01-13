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

	questionsPath := filepath.Join("spec", "questions", "init.yml")
	writeFile(t, repoRoot, questionsPath, `version: 1
questions:
  - id: q1
    question: "What is the project name in README.md?"
    answers: ["init", "other"]
    correct_answers: ["init"]
`)
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:            "t11",
		Type:          "question_eval",
		Agent:         "default",
		QuestionsFile: questionsPath,
	}})
	writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode = runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("run failed: %s", stderr)
	}
}
