//go:build cucumber
// +build cucumber

package cucumber

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cucumber/godog"
)

// theOutputListsCommands asserts the output contains expected command names.
func (s *featureState) theOutputListsCommands(table *godog.Table) error {
	output := s.stdout.String()
	for _, row := range table.Rows {
		for _, cell := range row.Cells {
			command := strings.TrimSpace(cell.Value)
			if command == "" {
				continue
			}
			if !strings.Contains(output, command) {
				return fmt.Errorf("expected command %q in output", command)
			}
		}
	}
	return nil
}

// theExitCodeIsNonZero asserts that the CLI returned an error code.
func (s *featureState) theExitCodeIsNonZero() error {
	if s.exitCode == 0 {
		return fmt.Errorf("expected non-zero exit code")
	}
	return nil
}

// theErrorMessagePointsToInvalidField checks the error output for hints.
func (s *featureState) theErrorMessagePointsToInvalidField() error {
	errOutput := s.stderr.String()
	if !strings.Contains(errOutput, "version") {
		return fmt.Errorf("expected error to mention version, got %q", errOutput)
	}
	return nil
}

func (s *featureState) theLogFileExists(path string) error {
	logPath := s.resolvePath(path)
	if _, err := os.Stat(logPath); err != nil {
		return fmt.Errorf("expected log file %q to exist: %w", logPath, err)
	}
	return nil
}

func (s *featureState) theLogFileContains(path, needle string) error {
	logPath := s.resolvePath(path)
	data, err := os.ReadFile(logPath)
	if err != nil {
		return fmt.Errorf("read log file %q: %w", logPath, err)
	}
	if !strings.Contains(string(data), needle) {
		return fmt.Errorf("expected log file %q to contain %q", logPath, needle)
	}
	return nil
}

func (s *featureState) stdoutDoesNotIncludeVerboseLogs() error {
	if strings.Contains(s.stdout.String(), "[verbose]") {
		return fmt.Errorf("expected stdout to exclude verbose logs")
	}
	return nil
}

func (s *featureState) theConsoleIncludesVerboseLogs() error {
	if !strings.Contains(s.stdout.String(), "[verbose]") {
		return fmt.Errorf("expected stdout to include verbose logs")
	}
	return nil
}

func (s *featureState) resolvePath(path string) string {
	if filepath.IsAbs(path) || s.repoDir == "" {
		return path
	}
	return filepath.Join(s.repoDir, path)
}
