//go:build cucumber
// +build cucumber

package cucumber

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cogni/internal/cli"
)

// iRunCommand executes a CLI command for the scenario.
func (s *featureState) iRunCommand(command string) error {
	args := strings.Fields(command)
	if len(args) == 0 {
		return fmt.Errorf("command is empty")
	}
	if args[0] == "cogni" {
		args = args[1:]
	}
	s.stdout.Reset()
	s.stderr.Reset()
	s.exitCode = cli.Run(args, &s.stdout, &s.stderr)
	return nil
}

// initGitRepo initializes a git repo with a README.
func (s *featureState) initGitRepo(dir string) error {
	if err := s.runGit(dir, "-c", "init.defaultBranch=main", "init"); err != nil {
		return err
	}
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("fixture"), 0o644); err != nil {
		return fmt.Errorf("write README: %w", err)
	}
	if err := s.runGit(dir, "add", "README.md"); err != nil {
		return err
	}
	if err := s.runGit(dir, "commit", "-m", "initial"); err != nil {
		return err
	}
	return nil
}

// runGit executes git commands in a repo with fixed author metadata.
func (s *featureState) runGit(dir string, args ...string) error {
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
		return fmt.Errorf("git %s failed: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
