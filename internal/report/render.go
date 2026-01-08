package report

import (
	"context"
	"strings"

	"cogni/internal/runner"
)

// RenderReportHTML renders the report template into a string.
func RenderReportHTML(ctx context.Context, runs []runner.Results) (string, error) {
	var builder strings.Builder
	if err := ReportPage(runs).Render(ctx, &builder); err != nil {
		return "", err
	}
	return builder.String(), nil
}
