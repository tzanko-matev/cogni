package report

import "fmt"

// formatPassRate returns a percentage string for report output.
func formatPassRate(rate float64) string {
	return fmt.Sprintf("%.2f", rate*100)
}
