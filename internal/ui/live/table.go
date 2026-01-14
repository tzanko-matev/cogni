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

// rowsForState converts UI state into table rows.
func rowsForState(state State, now time.Time) []table.Row {
	rows := make([]table.Row, 0, len(state.Rows))
	for _, row := range state.Rows {
		status := formatStatus(row)
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
