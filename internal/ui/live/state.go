package live

import (
	"time"

	"cogni/internal/runner"
)

// ToolStatus captures the latest tool call activity for a question.
type ToolStatus struct {
	Name     string
	State    string
	Duration time.Duration
	Error    string
}

// QuestionRow holds UI state for a single question.
type QuestionRow struct {
	Index        int
	ID           string
	Text         string
	Status       runner.QuestionEventType
	Tool         ToolStatus
	HasTool      bool
	RetryCount   int
	RetryAfterMs int
	StartedAt    time.Time
	FinishedAt   time.Time
	Tokens       int
	Error        string
}

// StatusCounts aggregates counts by status bucket.
type StatusCounts struct {
	Queued         int
	Scheduled      int
	Reserving      int
	Waiting        int
	Running        int
	Parsing        int
	Done           int
	Correct        int
	Incorrect      int
	ParseError     int
	BudgetExceeded int
	RuntimeError   int
	Skipped        int
}

// State captures the live UI state for a task run.
type State struct {
	RunID         string
	Repo          string
	TaskID        string
	QuestionsFile string
	AgentID       string
	Model         string
	StartedAt     time.Time
	LastEvent     string
	Rows          []QuestionRow
	Counts        StatusCounts
}
