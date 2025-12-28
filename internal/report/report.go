package report

import (
	"fmt"
	"html"
	"strings"

	"cogni/internal/runner"
)

func BuildReportHTML(runs []runner.Results) string {
	var builder strings.Builder
	builder.WriteString("<!doctype html>\n<html><head><meta charset=\"utf-8\"><title>Cogni Report</title></head><body>")
	builder.WriteString("<h1>Cogni Report</h1>")
	builder.WriteString("<table border=\"1\" cellspacing=\"0\" cellpadding=\"6\">")
	builder.WriteString("<thead><tr><th>Commit</th><th>Run ID</th><th>Pass Rate</th><th>Tokens Total</th></tr></thead><tbody>")
	for _, run := range runs {
		passRate := fmt.Sprintf("%.2f", run.Summary.PassRate*100)
		builder.WriteString("<tr>")
		builder.WriteString("<td>")
		builder.WriteString(html.EscapeString(run.Repo.Commit))
		builder.WriteString("</td><td>")
		builder.WriteString(html.EscapeString(run.RunID))
		builder.WriteString("</td><td>")
		builder.WriteString(passRate)
		builder.WriteString("%</td><td>")
		builder.WriteString(fmt.Sprintf("%d", run.Summary.TokensTotal))
		builder.WriteString("</td></tr>")
	}
	builder.WriteString("</tbody></table></body></html>\n")
	return builder.String()
}
