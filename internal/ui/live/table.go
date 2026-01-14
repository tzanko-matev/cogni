package live

import (
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// tableStyles returns table styles for the UI.
func tableStyles(noColor bool) table.Styles {
	if noColor {
		return table.DefaultStyles()
	}
	styles := table.DefaultStyles()
	styles.Header = styles.Header.Foreground(lipgloss.Color("252"))
	return styles
}

// columnsForWidth builds column definitions sized for the terminal width.
func columnsForWidth(width int) []table.Column {
	if width <= 0 {
		return defaultColumns()
	}
	idWidth := 4
	statusWidth := 24
	timeWidth := 9
	tokenWidth := 8
	retryWidth := 7
	separatorWidth := 5
	if width >= 140 {
		statusWidth = 30
	}
	if width <= 100 {
		statusWidth = 18
	}
	fixedWidth := idWidth + statusWidth + timeWidth + tokenWidth + retryWidth + separatorWidth
	questionWidth := width - fixedWidth
	if questionWidth < 1 {
		questionWidth = 1
	}
	if questionWidth < 10 && width >= fixedWidth+10 {
		questionWidth = 10
	}
	return []table.Column{
		{Title: "ID", Width: idWidth},
		{Title: "Question", Width: questionWidth},
		{Title: "Status", Width: statusWidth},
		{Title: "Time", Width: timeWidth},
		{Title: "Tokens", Width: tokenWidth},
		{Title: "Retries", Width: retryWidth},
	}
}

// defaultColumns returns the base column sizing.
func defaultColumns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Question", Width: 60},
		{Title: "Status", Width: 24},
		{Title: "Time", Width: 9},
		{Title: "Tokens", Width: 8},
		{Title: "Retries", Width: 7},
	}
}

// rowsForState converts UI state into table rows.
func rowsForState(state State, now time.Time, noColor bool) []table.Row {
	rows := make([]table.Row, 0, len(state.Rows))
	for _, row := range state.Rows {
		status := formatStatus(row, now, noColor)
		elapsed := formatRowDuration(row, now)
		tokens := formatTokens(row.Tokens)
		retries := formatRetries(row.RetryCount)
		rows = append(rows, table.Row{
			formatQuestionID(row),
			formatQuestionText(row.Text),
			status,
			elapsed,
			tokens,
			retries,
		})
	}
	return rows
}
