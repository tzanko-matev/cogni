package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runSetupCommands executes configured repo setup commands.
func runSetupCommands(ctx context.Context, root string, commands []string) error {
	for _, command := range commands {
		if strings.TrimSpace(command) == "" {
			continue
		}
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("setup command failed: %w", err)
		}
	}
	return nil
}
