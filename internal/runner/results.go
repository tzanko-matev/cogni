package runner

import "time"

type Results struct {
	RunID      string       `json:"run_id"`
	Repo       RepoMetadata `json:"repo"`
	Agents     []AgentInfo  `json:"agents"`
	StartedAt  time.Time    `json:"started_at"`
	FinishedAt time.Time    `json:"finished_at"`
	Tasks      []TaskResult `json:"tasks"`
	Summary    RunSummary   `json:"summary"`
}

type RepoMetadata struct {
	Name   string `json:"name"`
	VCS    string `json:"vcs"`
	Commit string `json:"commit"`
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
}

type AgentInfo struct {
	ID             string  `json:"id"`
	Type           string  `json:"type"`
	Provider       string  `json:"provider"`
	Model          string  `json:"model"`
	Temperature    float64 `json:"temperature"`
	MaxSteps       int     `json:"max_steps"`
	ToolingVersion string  `json:"tooling_version"`
}

type TaskResult struct {
	TaskID        string          `json:"task_id"`
	Type          string          `json:"type"`
	Status        string          `json:"status"`
	FailureReason *string         `json:"failure_reason"`
	Attempts      []AttemptResult `json:"attempts"`
}

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

type EvalResult struct {
	SchemaValid        bool     `json:"schema_valid"`
	CitationValid      bool     `json:"citation_valid"`
	SchemaErrors       []string `json:"schema_errors,omitempty"`
	CitationErrors     []string `json:"citation_errors,omitempty"`
	MustContainMissing []string `json:"must_contain_missing,omitempty"`
}

type RunSummary struct {
	TasksTotal  int     `json:"tasks_total"`
	TasksPassed int     `json:"tasks_passed"`
	TasksFailed int     `json:"tasks_failed"`
	PassRate    float64 `json:"pass_rate"`
	TokensTotal int     `json:"tokens_total"`
}
