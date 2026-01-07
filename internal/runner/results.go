package runner

import "time"

// Results captures the output of a cogni run.
type Results struct {
	RunID      string       `json:"run_id"`
	Repo       RepoMetadata `json:"repo"`
	Agents     []AgentInfo  `json:"agents"`
	StartedAt  time.Time    `json:"started_at"`
	FinishedAt time.Time    `json:"finished_at"`
	Tasks      []TaskResult `json:"tasks"`
	Summary    RunSummary   `json:"summary"`
}

// RepoMetadata describes repository state at run time.
type RepoMetadata struct {
	Name   string `json:"name"`
	VCS    string `json:"vcs"`
	Commit string `json:"commit"`
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
}

// AgentInfo captures agent configuration used in a run.
type AgentInfo struct {
	ID             string  `json:"id"`
	Type           string  `json:"type"`
	Provider       string  `json:"provider"`
	Model          string  `json:"model"`
	Temperature    float64 `json:"temperature"`
	MaxSteps       int     `json:"max_steps"`
	ToolingVersion string  `json:"tooling_version"`
}

// TaskResult records outcomes for a task.
type TaskResult struct {
	TaskID        string          `json:"task_id"`
	Type          string          `json:"type"`
	Status        string          `json:"status"`
	FailureReason *string         `json:"failure_reason"`
	Attempts      []AttemptResult `json:"attempts"`
	Cucumber      *CucumberEval   `json:"cucumber,omitempty"`
}

// AttemptResult records effort and validation for one attempt.
type AttemptResult struct {
	Attempt         int            `json:"attempt"`
	Status          string         `json:"status"`
	AgentID         string         `json:"agent_id"`
	Model           string         `json:"model"`
	TokensIn        int            `json:"tokens_in"`
	TokensOut       int            `json:"tokens_out"`
	TokensTotal     int            `json:"tokens_total"`
	WallTimeSeconds float64        `json:"wall_time_seconds"`
	AgentSteps      int            `json:"agent_steps"`
	ToolCalls       map[string]int `json:"tool_calls"`
	UniqueFilesRead int            `json:"unique_files_read"`
	Eval            EvalResult     `json:"eval"`
}

// EvalResult summarizes schema/citation validation.
type EvalResult struct {
	SchemaValid        bool     `json:"schema_valid"`
	CitationValid      bool     `json:"citation_valid"`
	SchemaErrors       []string `json:"schema_errors,omitempty"`
	CitationErrors     []string `json:"citation_errors,omitempty"`
	MustContainMissing []string `json:"must_contain_missing,omitempty"`
}

// CucumberEval contains per-example Cucumber evaluation results.
type CucumberEval struct {
	AdapterID   string               `json:"adapter_id"`
	AdapterType string               `json:"adapter_type"`
	FeatureRuns []CucumberFeatureRun `json:"feature_runs,omitempty"`
	Examples    []CucumberExample    `json:"examples"`
	Summary     CucumberSummary      `json:"summary"`
}

// CucumberFeatureRun captures per-feature execution metrics.
type CucumberFeatureRun struct {
	FeaturePath     string         `json:"feature_path"`
	ExamplesTotal   int            `json:"examples_total"`
	TokensTotal     int            `json:"tokens_total,omitempty"`
	WallTimeSeconds float64        `json:"wall_time_seconds,omitempty"`
	AgentSteps      int            `json:"agent_steps,omitempty"`
	ToolCalls       map[string]int `json:"tool_calls,omitempty"`
}

// CucumberExample records evaluation results for a single example.
type CucumberExample struct {
	ExampleID    string               `json:"example_id"`
	FeaturePath  string               `json:"feature_path"`
	ScenarioName string               `json:"scenario_name"`
	ScenarioLine int                  `json:"scenario_line,omitempty"`
	ExampleLine  int                  `json:"example_line,omitempty"`
	GroundTruth  string               `json:"ground_truth"`
	Agent        *CucumberAgentResult `json:"agent,omitempty"`
	Correct      bool                 `json:"correct"`
}

// CucumberAgentResult captures agent output for an example.
type CucumberAgentResult struct {
	ExampleID   string             `json:"example_id"`
	Implemented bool               `json:"implemented"`
	Evidence    []CucumberEvidence `json:"evidence,omitempty"`
	Notes       string             `json:"notes,omitempty"`
	ParseError  string             `json:"parse_error,omitempty"`
}

// CucumberEvidence describes a file/line reference from the agent.
type CucumberEvidence struct {
	Path  string `json:"path"`
	Lines []int  `json:"lines,omitempty"`
}

// CucumberSummary aggregates evaluation accuracy metrics.
type CucumberSummary struct {
	ExamplesTotal     int     `json:"examples_total"`
	ExamplesCorrect   int     `json:"examples_correct"`
	ExamplesIncorrect int     `json:"examples_incorrect"`
	Accuracy          float64 `json:"accuracy"`
	ImplementedTotal  int     `json:"implemented_total"`
	NotImplemented    int     `json:"not_implemented_total"`
}

// RunSummary aggregates run-level metrics.
type RunSummary struct {
	TasksTotal                int     `json:"tasks_total"`
	TasksPassed               int     `json:"tasks_passed"`
	TasksFailed               int     `json:"tasks_failed"`
	PassRate                  float64 `json:"pass_rate"`
	TokensTotal               int     `json:"tokens_total"`
	CucumberExamplesTotal     int     `json:"cucumber_examples_total,omitempty"`
	CucumberExamplesCorrect   int     `json:"cucumber_examples_correct,omitempty"`
	CucumberExamplesIncorrect int     `json:"cucumber_examples_incorrect,omitempty"`
	CucumberAccuracy          float64 `json:"cucumber_accuracy,omitempty"`
}
