package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// shellSetupRunner executes setup commands via a shell.
type shellSetupRunner struct{}

// Run executes a setup command using the shell.
func (shellSetupRunner) Run(ctx context.Context, dir string, command string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("setup command failed: %w", err)
	}
	return nil
}

// runSetupCommands executes configured repo setup commands.
func runSetupCommands(ctx context.Context, root string, commands []string, runner SetupCommandRunner) error {
	if runner == nil {
		runner = shellSetupRunner{}
	}
	for _, command := range commands {
		if strings.TrimSpace(command) == "" {
			continue
		}
		if err := runner.Run(ctx, root, command); err != nil {
			return err
		}
	}
	return nil
}
