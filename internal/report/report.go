package report

import (
	"context"

	"cogni/internal/runner"
)

// BuildReportHTML renders a simple HTML report for runs.
func BuildReportHTML(runs []runner.Results) string {
	html, err := RenderReportHTML(context.Background(), runs)
	if err != nil {
		return ""
	}
	return html
}
