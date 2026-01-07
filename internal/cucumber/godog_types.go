package cucumber

// CukeFeatureJSON matches godog JSON output for a feature.
type CukeFeatureJSON struct {
	URI      string        `json:"uri"`
	Elements []CukeElement `json:"elements"`
}

// CukeElement describes a scenario element from godog JSON.
type CukeElement struct {
	Name  string     `json:"name"`
	Line  int        `json:"line"`
	Steps []CukeStep `json:"steps"`
}

// CukeStep captures step status information from godog.
type CukeStep struct {
	Result CukeResult `json:"result"`
}

// CukeResult contains a step execution status from godog.
type CukeResult struct {
	Status string `json:"status"`
}

// GodogScenarioResult captures a normalized scenario outcome.
type GodogScenarioResult struct {
	ExampleID    string
	Status       string
	FeaturePath  string
	ScenarioName string
	Line         int
}
