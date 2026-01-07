package eval

// QAConfig configures validation for QA-style responses.
type QAConfig struct {
	JSONSchemaPath    string
	MustContain       []string
	ValidateCitations bool
	RepoRoot          string
}

// QAArtifacts captures validation errors and missing requirements.
type QAArtifacts struct {
	SchemaErrors       []string
	CitationErrors     []string
	MustContainMissing []string
}

// QAResult summarizes validation status and artifacts.
type QAResult struct {
	Status        string
	FailureReason string
	SchemaValid   bool
	CitationValid bool
	Artifacts     QAArtifacts
}

// Citation describes a file path and line range reference.
type Citation struct {
	Path  string
	Start int
	End   int
}
