package cucumber

import (
	"fmt"
	"strings"
)

// NormalizeGodogResults maps godog output to example ids and statuses.
func NormalizeGodogResults(repoRoot string, features []CukeFeatureJSON, index ExampleIndex) ([]GodogScenarioResult, error) {
	results := make([]GodogScenarioResult, 0)
	for _, feature := range features {
		featurePath := normalizePath(repoRoot, feature.URI)
		for _, element := range feature.Elements {
			example, ok := index.FindByLine(repoRoot, feature.URI, element.Line)
			if !ok {
				return nil, fmt.Errorf("no example matches %s:%d", featurePath, element.Line)
			}
			status := deriveScenarioStatus(element.Steps)
			results = append(results, GodogScenarioResult{
				ExampleID:    example.ID,
				Status:       status,
				FeaturePath:  featurePath,
				ScenarioName: element.Name,
				Line:         element.Line,
			})
		}
	}
	return results, nil
}

// deriveScenarioStatus reduces step statuses to a scenario status.
func deriveScenarioStatus(steps []CukeStep) string {
	hasPending := false
	hasUndefined := false
	hasSkipped := false
	for _, step := range steps {
		switch strings.ToLower(strings.TrimSpace(step.Result.Status)) {
		case "failed":
			return "failed"
		case "undefined":
			hasUndefined = true
		case "pending":
			hasPending = true
		case "skipped":
			hasSkipped = true
		}
	}
	switch {
	case hasUndefined:
		return "undefined"
	case hasPending:
		return "pending"
	case hasSkipped:
		return "skipped"
	default:
		return "passed"
	}
}
