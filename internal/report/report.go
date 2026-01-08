package report

import "cogni/internal/runner"

// BuildReportHTML renders a simple HTML report for runs.
func BuildReportHTML(runs []runner.Results) string {
	html, err := renderReportHTML(runs)
	if err != nil {
		return ""
	}
	return html
}
