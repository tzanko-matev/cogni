//go:build cucumber
// +build cucumber

package cucumber

import (
	"fmt"
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
