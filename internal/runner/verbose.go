package runner

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

const verbosePrefix = "[verbose]"

func logVerbose(enabled bool, writer io.Writer, format string, args ...any) {
	if !enabled || writer == nil {
		return
	}
	fmt.Fprintf(writer, "%s %s\n", verbosePrefix, fmt.Sprintf(format, args...))
}

func formatToolCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, " ")
}
