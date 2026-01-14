package live

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderHeader renders the run header line.
func renderHeader(state State, now time.Time, noColor bool) string {
	elapsed := ""
	if !state.StartedAt.IsZero() {
		elapsed = now.Sub(state.StartedAt).Round(100 * time.Millisecond).String()
	}
	line := "Run " + state.RunID
	if state.Repo != "" {
		line += " | Repo: " + state.Repo
	}
	if elapsed != "" {
		line += " | Elapsed: " + elapsed
	}
	return stylize(line, noColor, lipgloss.Color("33"))
}

// renderSummary renders the status counts line.
func renderSummary(state State, noColor bool) string {
	counts := state.Counts
	line := "Queued: " + fmtInt(counts.Queued) +
		" Waiting: " + fmtInt(counts.Waiting) +
		" Running: " + fmtInt(counts.Running) +
		" Done: " + fmtInt(counts.Done) +
		" Correct: " + fmtInt(counts.Correct) +
		" Incorrect: " + fmtInt(counts.Incorrect) +
		" ParseErr: " + fmtInt(counts.ParseError) +
		" Budget: " + fmtInt(counts.BudgetExceeded) +
		" Error: " + fmtInt(counts.RuntimeError)
	return stylize(line, noColor, lipgloss.Color("242"))
}

// renderTaskLine renders the current task line.
func renderTaskLine(state State, noColor bool) string {
	if state.TaskID == "" {
		return ""
	}
	line := "Task " + state.TaskID
	if state.QuestionsFile != "" {
		line += " | " + state.QuestionsFile
	}
	if state.AgentID != "" || state.Model != "" {
		line += " | " + state.AgentID + " / " + state.Model
	}
	return stylize(line, noColor, lipgloss.Color("240"))
}

// renderFooter renders the last event line.
func renderFooter(state State, noColor bool) string {
	if state.LastEvent == "" {
		return ""
	}
	return stylize("Last event: "+state.LastEvent, noColor, lipgloss.Color("244"))
}

// stylize applies optional color styling.
func stylize(text string, noColor bool, color lipgloss.Color) string {
	if noColor {
		return text
	}
	return lipgloss.NewStyle().Foreground(color).Render(text)
}
