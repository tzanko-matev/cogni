package live

import "cogni/internal/runner"

// EventKind identifies the type of live UI event.
type EventKind int

const (
	// EventRunStart signals the start of a run.
	EventRunStart EventKind = iota
	// EventTaskStart signals the start of a task.
	EventTaskStart
	// EventQuestion delivers a question status update.
	EventQuestion
	// EventTaskEnd signals task completion.
	EventTaskEnd
	// EventRunEnd signals run completion.
	EventRunEnd
)

// Event carries a UI update payload.
type Event struct {
	Kind          EventKind
	RunID         string
	Repo          string
	TaskID        string
	QuestionsFile string
	AgentID       string
	Model         string
	TaskStatus    string
	TaskReason    *string
	Question      runner.QuestionEvent
}
