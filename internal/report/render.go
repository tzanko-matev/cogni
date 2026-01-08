package report

import (
	"context"
	"strings"

	"cogni/internal/runner"
)

// renderReportHTML renders the report template into a string.
func renderReportHTML(runs []runner.Results) (string, error) {
	var builder strings.Builder
	if err := ReportPage(runs).Render(context.Background(), &builder); err != nil {
		return "", err
	}
	return builder.String(), nil
}
