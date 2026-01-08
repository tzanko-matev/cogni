package runner

import (
	"context"
	"strings"
)

// renderRunReportHTML renders the single-run report template into a string.
func renderRunReportHTML(results Results) (string, error) {
	var builder strings.Builder
	if err := RunReportStub(results).Render(context.Background(), &builder); err != nil {
		return "", err
	}
	return builder.String(), nil
}
