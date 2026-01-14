package live

import (
	"strconv"
	"time"

	"cogni/internal/runner"
)

// formatQuestionID returns the display id for a question row.
func formatQuestionID(row QuestionRow) string {
	if row.ID != "" {
		return row.ID
	}
	return formatIndex(row.Index)
}

// formatIndex formats a question index.
func formatIndex(index int) string {
	return "Q" + pad2(index+1)
}

// pad2 left-pads a number to two digits when needed.
func pad2(value int) string {
	if value >= 10 {
		return fmtInt(value)
	}
	return "0" + fmtInt(value)
}

// fmtInt converts an int to string.
func fmtInt(value int) string {
	return strconv.Itoa(value)
}

// formatQuestionText truncates question text for display.
func formatQuestionText(text string) string {
	if text == "" {
		return ""
	}
	const limit = 80
	if len(text) <= limit {
		return text
	}
	return text[:limit-3] + "..."
}

// formatStatus renders a status string for a row.
func formatStatus(row QuestionRow) string {
	status := string(row.Status)
	if row.HasTool {
		status = status + " * tool:" + row.Tool.Name + " " + row.Tool.State
	}
	if row.RetryAfterMs > 0 &&
		(row.Status == runner.QuestionWaitingRateLimit ||
			row.Status == runner.QuestionWaitingLimitDecreasing ||
			row.Status == runner.QuestionWaitingLimiterError) {
		status = status + " (" + fmtInt(row.RetryAfterMs) + "ms)"
	}
	return status
}

// formatRowDuration returns elapsed or total time for a row.
func formatRowDuration(row QuestionRow, now time.Time) string {
	if !row.FinishedAt.IsZero() && !row.StartedAt.IsZero() {
		return row.FinishedAt.Sub(row.StartedAt).Round(100 * time.Millisecond).String()
	}
	if !row.StartedAt.IsZero() {
		return now.Sub(row.StartedAt).Round(100 * time.Millisecond).String()
	}
	return ""
}

// formatTokens formats token counts for display.
func formatTokens(tokens int) string {
	if tokens <= 0 {
		return "n/a"
	}
	return fmtInt(tokens)
}

// formatRetries formats retry counts for display.
func formatRetries(retries int) string {
	if retries <= 0 {
		return ""
	}
	return fmtInt(retries)
}

// formatTaskEnd formats a task completion message.
func formatTaskEnd(taskID, status string, reason *string) string {
	if reason != nil {
		return "Task " + taskID + " " + status + " (" + *reason + ")"
	}
	return "Task " + taskID + " " + status
}
