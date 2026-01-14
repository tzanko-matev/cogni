package runner

import "time"

// QuestionEventType identifies a question status update for observers.
type QuestionEventType string

const (
	// QuestionQueued marks a question known but not yet submitted.
	QuestionQueued QuestionEventType = "queued"
	// QuestionScheduled marks a question submitted to the scheduler.
	QuestionScheduled QuestionEventType = "scheduled"
	// QuestionReserving marks a reserve attempt in progress.
	QuestionReserving QuestionEventType = "reserving"
	// QuestionWaitingRateLimit marks a reserve denial with retry_after_ms.
	QuestionWaitingRateLimit QuestionEventType = "waiting_rate_limit"
	// QuestionWaitingLimitDecreasing marks a reserve denial for limit_decreasing.
	QuestionWaitingLimitDecreasing QuestionEventType = "waiting_limit_decreasing"
	// QuestionWaitingLimiterError marks a reserve error retry.
	QuestionWaitingLimiterError QuestionEventType = "waiting_limiter_error"
	// QuestionRunning marks an active model call.
	QuestionRunning QuestionEventType = "running"
	// QuestionParsing marks parsing the model response.
	QuestionParsing QuestionEventType = "parsing"
	// QuestionCorrect marks a correct answer.
	QuestionCorrect QuestionEventType = "correct"
	// QuestionIncorrect marks an incorrect answer.
	QuestionIncorrect QuestionEventType = "incorrect"
	// QuestionParseError marks a parse failure.
	QuestionParseError QuestionEventType = "parse_error"
	// QuestionBudgetExceeded marks a budget exceeded failure.
	QuestionBudgetExceeded QuestionEventType = "budget_exceeded"
	// QuestionRuntimeError marks a runtime error.
	QuestionRuntimeError QuestionEventType = "runtime_error"
	// QuestionSkipped marks a question skipped due to task-level failure.
	QuestionSkipped QuestionEventType = "skipped"
	// QuestionToolStart marks the start of a tool call.
	QuestionToolStart QuestionEventType = "tool_start"
	// QuestionToolFinish marks the completion of a tool call.
	QuestionToolFinish QuestionEventType = "tool_finish"
)

// QuestionEvent carries a single status update for a question.
type QuestionEvent struct {
	TaskID        string
	QuestionIndex int
	QuestionID    string
	QuestionText  string
	Type          QuestionEventType
	RetryAfterMs  int
	ToolName      string
	ToolDuration  time.Duration
	ToolError     string
	Tokens        int
	WallTime      time.Duration
	Error         string
	EmittedAt     time.Time
}

// RunObserver receives run lifecycle events for UI or logging.
type RunObserver interface {
	// OnRunStart signals the start of a run.
	OnRunStart(runID string, repo string)
	// OnTaskStart signals the start of a task.
	OnTaskStart(taskID string, taskType string, questionsFile string, agentID string, model string)
	// OnQuestionEvent delivers a question status update.
	OnQuestionEvent(event QuestionEvent)
	// OnTaskEnd signals task completion.
	OnTaskEnd(taskID string, status string, reason *string)
	// OnRunEnd signals run completion.
	OnRunEnd(results Results)
}
