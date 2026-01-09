//go:build live
// +build live

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cogni/internal/spec"
)

// requireGit skips tests when git is unavailable.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

// runGit executes a git command and returns trimmed stdout.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(output))
	}
	return strings.TrimSpace(string(output))
}

// writeFile writes a file under the repo root with required directories.
func writeFile(t *testing.T, root, rel, contents string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

// simpleRepo builds a git repo with sample files for QA tests.
func simpleRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	root := t.TempDir()
	runGit(t, root, "-c", "init.defaultBranch=main", "init")

	writeFile(t, root, "README.md", "# Sample Service\nThis repo exists only for Cogni integration tests.\n")
	writeFile(t, root, "app.md", "# App Notes\nService owner: Platform Team\n")
	writeFile(t, root, "config.yml", "service_name: Sample Service\nowner: Platform Team\n")
	writeFile(t, root, filepath.Join("config", "app-config.yml"), "mode: sample\n")

	runGit(t, root, "add", "README.md", "app.md", "config.yml", "config/app-config.yml")
	runGit(t, root, "commit", "-m", "init")
	return root
}

// historyRepo prepares a git repo with two commits for compare tests.
func historyRepo(t *testing.T) (string, string, string) {
	t.Helper()
	requireGit(t)
	root := t.TempDir()
	runGit(t, root, "-c", "init.defaultBranch=main", "init")

	writeFile(t, root, "README.md", "# Sample Service\nRelease stage: alpha\n")
	writeFile(t, root, "change-log.md", "- 0.1.0: initial\n")
	runGit(t, root, "add", "README.md", "change-log.md")
	runGit(t, root, "commit", "-m", "init")
	first := runGit(t, root, "rev-parse", "HEAD")

	writeFile(t, root, "README.md", "# Sample Service\nRelease stage: beta\n")
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "update release stage")
	second := runGit(t, root, "rev-parse", "HEAD")

	return root, first, second
}

// baseConfig returns a baseline Cogni config for tests.
func baseConfig(output string, agents []spec.AgentConfig, defaultAgent string, tasks []spec.TaskConfig) spec.Config {
	return spec.Config{
		Version:      1,
		Repo:         spec.RepoConfig{OutputDir: output},
		Agents:       agents,
		DefaultAgent: defaultAgent,
		Tasks:        tasks,
	}
}
