package runner

import "cogni/internal/cucumber"

// groupExamplesByFeature groups examples by feature path and preserves order.
func groupExamplesByFeature(examples []cucumber.Example) ([]string, map[string][]cucumber.Example) {
	features := make([]string, 0)
	examplesByFeature := make(map[string][]cucumber.Example)
	seenFeatures := make(map[string]struct{})
	for _, example := range examples {
		if _, seen := seenFeatures[example.FeaturePath]; !seen {
			features = append(features, example.FeaturePath)
			seenFeatures[example.FeaturePath] = struct{}{}
		}
		examplesByFeature[example.FeaturePath] = append(examplesByFeature[example.FeaturePath], example)
	}
	return features, examplesByFeature
}
