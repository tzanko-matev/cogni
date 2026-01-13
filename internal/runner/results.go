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
	TaskID        string        `json:"task_id"`
	Type          string        `json:"type"`
	Status        string        `json:"status"`
	FailureReason *string       `json:"failure_reason"`
	QuestionEval  *QuestionEval `json:"question_eval,omitempty"`
}

// RunSummary aggregates run-level metrics.
type RunSummary struct {
	TasksTotal         int     `json:"tasks_total"`
	TasksPassed        int     `json:"tasks_passed"`
	TasksFailed        int     `json:"tasks_failed"`
	PassRate           float64 `json:"pass_rate"`
	TokensTotal        int     `json:"tokens_total"`
	QuestionsTotal     int     `json:"questions_total,omitempty"`
	QuestionsCorrect   int     `json:"questions_correct,omitempty"`
	QuestionsIncorrect int     `json:"questions_incorrect,omitempty"`
	QuestionAccuracy   float64 `json:"question_accuracy,omitempty"`
}
