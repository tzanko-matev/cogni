package runner

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"cogni/internal/cucumber"
	"cogni/internal/spec"
)

// cucumberGroundTruth captures whether an example is implemented.
type cucumberGroundTruth struct {
	Implemented bool
}

// loadCucumberGroundTruth builds ground truth data for cucumber evaluation.
func loadCucumberGroundTruth(
	ctx context.Context,
	repoRoot string,
	adapter spec.AdapterConfig,
	featurePaths []string,
	index cucumber.ExampleIndex,
	examples []cucumber.Example,
) (map[string]cucumberGroundTruth, error) {
	groundTruth := make(map[string]cucumberGroundTruth)
	switch adapter.Type {
	case "cucumber":
		features, err := cucumber.RunGodogJSON(ctx, repoRoot, featurePaths, adapter.Tags)
		if err != nil {
			return nil, err
		}
		normalized, err := cucumber.NormalizeGodogResults(repoRoot, features, index)
		if err != nil {
			return nil, err
		}
		for _, entry := range normalized {
			groundTruth[entry.ExampleID] = cucumberGroundTruth{Implemented: entry.Status == "passed"}
		}
	case "cucumber_manual":
		expectationsDir := strings.TrimSpace(adapter.ExpectationsDir)
		if expectationsDir != "" && !filepath.IsAbs(expectationsDir) {
			expectationsDir = filepath.Join(repoRoot, expectationsDir)
		}
		expectations, err := cucumber.LoadExpectations(expectationsDir)
		if err != nil {
			return nil, err
		}
		if err := cucumber.ValidateExpectations(expectations, examples); err != nil {
			return nil, err
		}
		for id, expectation := range expectations {
			groundTruth[id] = cucumberGroundTruth{Implemented: expectation.Implemented}
		}
	default:
		return nil, fmt.Errorf("unsupported adapter type %q", adapter.Type)
	}
	return groundTruth, nil
}
