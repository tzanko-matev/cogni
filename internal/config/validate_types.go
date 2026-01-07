package config

import (
	"fmt"
	"strings"
)

// Issue captures a validation problem with a config field.
type Issue struct {
	Field   string
	Message string
}

// ValidationError aggregates config validation issues.
type ValidationError struct {
	Issues []Issue
}

// Error renders validation errors as a multi-line string.
func (err *ValidationError) Error() string {
	if err == nil || len(err.Issues) == 0 {
		return "config validation failed"
	}
	lines := make([]string, 0, len(err.Issues))
	for _, issue := range err.Issues {
		lines = append(lines, fmt.Sprintf("%s: %s", issue.Field, issue.Message))
	}
	return strings.Join(lines, "\n")
}
