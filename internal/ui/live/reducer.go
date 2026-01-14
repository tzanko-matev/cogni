package live

import (
	"fmt"
	"time"

	"cogni/internal/runner"
)

// Reduce applies a question event to the UI state.
func Reduce(state State, event runner.QuestionEvent) State {
	state = ensureRow(state, event)
	state = applyQuestionEvent(state, event)
	state.Counts = recount(state.Rows)
	if message := formatLastEvent(event); message != "" {
		state.LastEvent = message
	}
	return state
}

// ensureRow grows the state rows to include the target index.
func ensureRow(state State, event runner.QuestionEvent) State {
	if event.QuestionIndex < 0 {
		return state
	}
	if event.QuestionIndex < len(state.Rows) {
		return state
	}
	rows := make([]QuestionRow, event.QuestionIndex+1)
	copy(rows, state.Rows)
	for i := len(state.Rows); i < len(rows); i++ {
		rows[i] = QuestionRow{Index: i, Status: runner.QuestionQueued}
	}
	state.Rows = rows
	return state
}

// applyQuestionEvent updates a row with the given event.
func applyQuestionEvent(state State, event runner.QuestionEvent) State {
	if event.QuestionIndex < 0 || event.QuestionIndex >= len(state.Rows) {
		return state
	}
	row := state.Rows[event.QuestionIndex]
	if row.ID == "" {
		row.ID = event.QuestionID
	}
	if row.Text == "" {
		row.Text = event.QuestionText
	}
	switch event.Type {
	case runner.QuestionToolStart:
		row.Tool = ToolStatus{
			Name:      event.ToolName,
			State:     "running",
			StartedAt: event.EmittedAt,
		}
		row.HasTool = true
	case runner.QuestionToolFinish:
		duration := event.ToolDuration
		if duration <= 0 && !row.Tool.StartedAt.IsZero() && !event.EmittedAt.IsZero() {
			duration = event.EmittedAt.Sub(row.Tool.StartedAt)
		}
		row.Tool = ToolStatus{
			Name:       event.ToolName,
			State:      "done",
			Duration:   duration,
			Error:      event.ToolError,
			StartedAt:  row.Tool.StartedAt,
			FinishedAt: event.EmittedAt,
		}
		row.HasTool = true
	default:
		row.Status = event.Type
		row.RetryAfterMs = event.RetryAfterMs
		if event.Type == runner.QuestionWaitingRateLimit ||
			event.Type == runner.QuestionWaitingLimitDecreasing ||
			event.Type == runner.QuestionWaitingLimiterError {
			row.RetryCount++
		}
		if event.Type == runner.QuestionRunning && row.StartedAt.IsZero() {
			row.StartedAt = event.EmittedAt
		}
		if isTerminalStatus(event.Type) {
			if !event.EmittedAt.IsZero() {
				row.FinishedAt = event.EmittedAt
			}
			row.Tokens = event.Tokens
			row.Error = event.Error
		}
	}
	state.Rows[event.QuestionIndex] = row
	return state
}

// isTerminalStatus reports whether a status is final.
func isTerminalStatus(status runner.QuestionEventType) bool {
	switch status {
	case runner.QuestionCorrect,
		runner.QuestionIncorrect,
		runner.QuestionParseError,
		runner.QuestionBudgetExceeded,
		runner.QuestionRuntimeError,
		runner.QuestionSkipped:
		return true
	default:
		return false
	}
}

// recount recomputes status counts for the current rows.
func recount(rows []QuestionRow) StatusCounts {
	var counts StatusCounts
	for _, row := range rows {
		switch row.Status {
		case runner.QuestionQueued:
			counts.Queued++
		case runner.QuestionScheduled:
			counts.Scheduled++
		case runner.QuestionReserving:
			counts.Reserving++
		case runner.QuestionWaitingRateLimit,
			runner.QuestionWaitingLimitDecreasing,
			runner.QuestionWaitingLimiterError:
			counts.Waiting++
		case runner.QuestionRunning:
			counts.Running++
		case runner.QuestionParsing:
			counts.Parsing++
		case runner.QuestionCorrect:
			counts.Done++
			counts.Correct++
		case runner.QuestionIncorrect:
			counts.Done++
			counts.Incorrect++
		case runner.QuestionParseError:
			counts.Done++
			counts.ParseError++
		case runner.QuestionBudgetExceeded:
			counts.Done++
			counts.BudgetExceeded++
		case runner.QuestionRuntimeError:
			counts.Done++
			counts.RuntimeError++
		case runner.QuestionSkipped:
			counts.Done++
			counts.Skipped++
		}
	}
	return counts
}

// formatLastEvent creates a short footer message for the event.
func formatLastEvent(event runner.QuestionEvent) string {
	switch event.Type {
	case runner.QuestionWaitingRateLimit:
		if event.RetryAfterMs > 0 {
			return fmt.Sprintf("Q%d rate limited (retry in %s)", event.QuestionIndex+1, formatRetryAfter(event.RetryAfterMs))
		}
		return fmt.Sprintf("Q%d rate limited", event.QuestionIndex+1)
	case runner.QuestionWaitingLimitDecreasing:
		return fmt.Sprintf("Q%d limit decreasing", event.QuestionIndex+1)
	case runner.QuestionWaitingLimiterError:
		return fmt.Sprintf("Q%d limiter error (retrying)", event.QuestionIndex+1)
	case runner.QuestionToolStart:
		return fmt.Sprintf("Q%d tool %s started", event.QuestionIndex+1, event.ToolName)
	case runner.QuestionToolFinish:
		if event.ToolError != "" {
			return fmt.Sprintf("Q%d tool %s error (%s)", event.QuestionIndex+1, event.ToolName, event.ToolError)
		}
		return fmt.Sprintf("Q%d tool %s finished (%s)", event.QuestionIndex+1, event.ToolName, formatDuration(event.ToolDuration))
	case runner.QuestionRuntimeError:
		return fmt.Sprintf("Q%d runtime error: %s", event.QuestionIndex+1, event.Error)
	case runner.QuestionBudgetExceeded:
		return fmt.Sprintf("Q%d budget exceeded", event.QuestionIndex+1)
	case runner.QuestionParseError:
		return fmt.Sprintf("Q%d parse error: %s", event.QuestionIndex+1, event.Error)
	}
	if event.Type == runner.QuestionCorrect || event.Type == runner.QuestionIncorrect {
		return fmt.Sprintf("Q%d completed", event.QuestionIndex+1)
	}
	return ""
}

// formatDuration renders a rounded duration for display.
func formatDuration(duration time.Duration) string {
	if duration <= 0 {
		return "0s"
	}
	return duration.Round(100 * time.Millisecond).String()
}
