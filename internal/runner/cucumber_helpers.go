package runner

import (
	"strings"

	"cogni/internal/cucumber"
)

// truthLabel returns the canonical label for truth values.
func truthLabel(implemented bool) string {
	if implemented {
		return "implemented"
	}
	return "not_implemented"
}

// convertEvidence normalizes evidence entries for result output.
func convertEvidence(items []cucumber.Evidence) []CucumberEvidence {
	if len(items) == 0 {
		return nil
	}
	out := make([]CucumberEvidence, 0, len(items))
	for _, item := range items {
		out = append(out, CucumberEvidence{
			Path:  strings.TrimSpace(item.Path),
			Lines: item.Lines,
		})
	}
	return out
}
