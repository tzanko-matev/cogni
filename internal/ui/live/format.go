package live

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

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
	normalized := strings.Join(strings.Fields(text), " ")
	if normalized == "" {
		return ""
	}
	const limit = 80
	if len(normalized) <= limit {
		return normalized
	}
	return normalized[:limit-3] + "..."
}

// formatStatus renders a status string for a row.
func formatStatus(row QuestionRow, now time.Time, noColor bool) string {
	primary := formatPrimaryStatus(row)
	primary = stylizeStatus(primary, row.Status, noColor)
	tool := formatToolStatus(row, now, noColor)
	if tool == "" {
		return primary
	}
	return primary + " | " + tool
}

// formatPrimaryStatus renders the primary status text.
func formatPrimaryStatus(row QuestionRow) string {
	switch row.Status {
	case runner.QuestionWaitingRateLimit:
		if row.RetryAfterMs > 0 {
			return "waiting rate limit (" + formatRetryAfter(row.RetryAfterMs) + ")"
		}
		return "waiting rate limit"
	case runner.QuestionWaitingLimitDecreasing:
		return "waiting limit decreasing"
	case runner.QuestionWaitingLimiterError:
		return "waiting limiter error"
	case runner.QuestionParseError:
		return "parse error"
	case runner.QuestionBudgetExceeded:
		return "budget exceeded"
	case runner.QuestionRuntimeError:
		return "runtime error"
	default:
		return statusLabel(row.Status)
	}
}

// statusLabel maps status codes to display labels.
func statusLabel(status runner.QuestionEventType) string {
	switch status {
	case runner.QuestionQueued:
		return "queued"
	case runner.QuestionScheduled:
		return "scheduled"
	case runner.QuestionReserving:
		return "reserving"
	case runner.QuestionRunning:
		return "running"
	case runner.QuestionParsing:
		return "parsing"
	case runner.QuestionCorrect:
		return "correct"
	case runner.QuestionIncorrect:
		return "incorrect"
	case runner.QuestionSkipped:
		return "skipped"
	default:
		return string(status)
	}
}

// formatToolStatus renders the tool sub-status text.
func formatToolStatus(row QuestionRow, now time.Time, noColor bool) string {
	if !row.HasTool || row.Tool.Name == "" {
		return ""
	}
	label := "tool:" + row.Tool.Name
	switch row.Tool.State {
	case "running":
		if !row.Tool.StartedAt.IsZero() {
			elapsed := now.Sub(row.Tool.StartedAt)
			label = label + " running " + formatDuration(elapsed)
		} else {
			label = label + " running"
		}
	case "done":
		if row.Tool.Error != "" {
			label = label + " error"
		} else if row.Tool.Duration > 0 {
			label = label + " done " + formatDuration(row.Tool.Duration)
		} else {
			label = label + " done"
		}
	default:
		label = label + " " + row.Tool.State
	}
	return stylizeTool(label, noColor)
}

// formatRetryAfter renders retry delays in human readable units.
func formatRetryAfter(ms int) string {
	if ms <= 0 {
		return ""
	}
	return formatDuration(time.Duration(ms) * time.Millisecond)
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

// stylizeStatus applies status coloring when enabled.
func stylizeStatus(text string, status runner.QuestionEventType, noColor bool) string {
	if noColor {
		return text
	}
	return statusStyle(status).Render(text)
}

// stylizeTool applies muted styling to tool sub-status.
func stylizeTool(text string, noColor bool) string {
	if noColor {
		return text
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(text)
}

// statusStyle selects a style for a given status.
func statusStyle(status runner.QuestionEventType) lipgloss.Style {
	color := lipgloss.Color("244")
	switch status {
	case runner.QuestionCorrect:
		color = lipgloss.Color("42")
	case runner.QuestionIncorrect:
		color = lipgloss.Color("220")
	case runner.QuestionParseError,
		runner.QuestionBudgetExceeded,
		runner.QuestionRuntimeError:
		color = lipgloss.Color("196")
	case runner.QuestionWaitingRateLimit,
		runner.QuestionWaitingLimitDecreasing,
		runner.QuestionWaitingLimiterError:
		color = lipgloss.Color("39")
	case runner.QuestionRunning:
		color = lipgloss.Color("33")
	case runner.QuestionParsing:
		color = lipgloss.Color("201")
	case runner.QuestionQueued,
		runner.QuestionScheduled,
		runner.QuestionReserving,
		runner.QuestionSkipped:
		color = lipgloss.Color("246")
	}
	return lipgloss.NewStyle().Foreground(color)
}
