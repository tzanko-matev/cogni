package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// rgRunner defines how ripgrep commands are executed.
type rgRunner interface {
	Run(ctx context.Context, dir string, args ...string) (string, error)
}

// execRGRunner executes ripgrep via the system binary.
type execRGRunner struct{}

// Run executes ripgrep and returns stdout or an error.
func (execRGRunner) Run(ctx context.Context, dir string, args ...string) (string, error) {
	if _, err := exec.LookPath("rg"); err != nil {
		return "", fmt.Errorf("rg not found")
	}
	cmd := exec.CommandContext(ctx, "rg", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return stdout.String(), nil
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = "no stderr"
		}
		return "", fmt.Errorf("rg %s: %w (%s)", strings.Join(args, " "), err, msg)
	}
	return stdout.String(), nil
}
