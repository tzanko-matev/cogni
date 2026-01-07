package runner

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"golang.org/x/term"
)

const verbosePrefix = "[verbose]"

const (
	ansiReset = "\x1b[0m"
	ansiBold  = "\x1b[1m"
	ansiDim   = "\x1b[2m"
	ansiGray  = "\x1b[90m"
	ansiGreen = "\x1b[32m"
	ansiRed   = "\x1b[31m"
	ansiBlue  = "\x1b[34m"
)

// verboseStyle selects how verbose output lines are styled.
type verboseStyle int

const (
	styleDefault verboseStyle = iota
	styleTask
	styleMetrics
	styleError
)

type verboseSink struct {
	writer  io.Writer
	noColor bool
}

func collectVerboseSinks(enabled bool, writer io.Writer, logWriter io.Writer, noColor bool) []verboseSink {
	sinks := make([]verboseSink, 0, 2)
	if enabled && writer != nil {
		sinks = append(sinks, verboseSink{writer: writer, noColor: noColor})
	}
	if logWriter != nil {
		sinks = append(sinks, verboseSink{writer: logWriter, noColor: true})
	}
	return sinks
}

// logVerbose emits a styled verbose line when verbosity is enabled.
func logVerbose(enabled bool, writer io.Writer, logWriter io.Writer, noColor bool, style verboseStyle, message string) {
	for _, sink := range collectVerboseSinks(enabled, writer, logWriter, noColor) {
		palette := paletteFor(sink.writer, sink.noColor)
		fmt.Fprintf(sink.writer, "%s %s\n", palette.prefix(verbosePrefix), palette.apply(style, message))
	}
}

// formatToolCounts renders tool call counts in a deterministic order.
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

// verbosePalette holds whether styling is enabled for verbose output.
type verbosePalette struct {
	enabled bool
}

// paletteFor determines whether styling should be enabled for a writer.
func paletteFor(writer io.Writer, noColor bool) verbosePalette {
	if noColor {
		return verbosePalette{enabled: false}
	}
	return verbosePalette{enabled: shouldUseStyling(writer)}
}

// shouldUseStyling returns true when ANSI styles should be used.
func shouldUseStyling(writer io.Writer) bool {
	if writer == nil {
		return false
	}
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	if strings.EqualFold(os.Getenv("CLICOLOR"), "0") {
		return false
	}
	if file, ok := writer.(*os.File); ok {
		return term.IsTerminal(int(file.Fd()))
	}
	if fder, ok := writer.(interface{ Fd() uintptr }); ok {
		return term.IsTerminal(int(fder.Fd()))
	}
	return false
}

// prefix styles the verbose prefix marker.
func (p verbosePalette) prefix(text string) string {
	if !p.enabled {
		return text
	}
	return ansiDim + ansiGray + text + ansiReset
}

// apply applies the selected style to text when enabled.
func (p verbosePalette) apply(style verboseStyle, text string) string {
	if !p.enabled {
		return text
	}
	switch style {
	case styleTask:
		return ansiBold + ansiBlue + text + ansiReset
	case styleMetrics:
		return ansiBold + ansiGreen + text + ansiReset
	case styleError:
		return ansiBold + ansiRed + text + ansiReset
	default:
		return text
	}
}
