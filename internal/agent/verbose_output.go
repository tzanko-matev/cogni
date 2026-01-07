package agent

import (
	"fmt"
	"io"
	"strings"
)

type verboseSink struct {
	writer             io.Writer
	noColor            bool
	maxBytes           int
	toolOutputMaxLines int
}

func collectVerboseSinks(opts RunOptions) []verboseSink {
	sinks := make([]verboseSink, 0, 2)
	if opts.Verbose && opts.VerboseWriter != nil {
		sinks = append(sinks, verboseSink{
			writer:             opts.VerboseWriter,
			noColor:            opts.NoColor,
			maxBytes:           verboseMaxBytes,
			toolOutputMaxLines: verboseToolOutputMaxLines,
		})
	}
	if opts.VerboseLogWriter != nil {
		sinks = append(sinks, verboseSink{
			writer:             opts.VerboseLogWriter,
			noColor:            true,
			maxBytes:           0,
			toolOutputMaxLines: 0,
		})
	}
	return sinks
}

// logVerbose emits a styled verbose line when verbosity is enabled.
func logVerbose(opts RunOptions, style verboseStyle, message string) {
	for _, sink := range collectVerboseSinks(opts) {
		palette := paletteFor(sink.writer, sink.noColor)
		writeVerboseLine(sink.writer, palette, style, message)
	}
}

// logVerboseBlock writes a header and a multi-line body to the verbose stream.
func logVerboseBlock(opts RunOptions, header, body string, headerStyle, bodyStyle verboseStyle) {
	for _, sink := range collectVerboseSinks(opts) {
		palette := paletteFor(sink.writer, sink.noColor)
		writeVerboseLine(sink.writer, palette, headerStyle, header)
		trimmed := truncateVerboseWithLimit(body, sink.maxBytes, verboseTruncationMarker)
		if strings.TrimSpace(trimmed) == "" {
			continue
		}
		for _, line := range strings.Split(trimmed, "\n") {
			writeVerboseLine(sink.writer, palette, bodyStyle, line)
		}
	}
}

// logVerboseToolOutput writes a tool output block with truncation rules.
func logVerboseToolOutput(opts RunOptions, header, body string) {
	for _, sink := range collectVerboseSinks(opts) {
		palette := paletteFor(sink.writer, sink.noColor)
		writeVerboseLine(sink.writer, palette, styleHeadingToolResult, header)
		trimmed := truncateVerboseInlineWithLimit(limitOutputLines(body, sink.toolOutputMaxLines), sink.maxBytes, verboseInlineTruncationMarker)
		if strings.TrimSpace(trimmed) == "" {
			continue
		}
		for _, line := range strings.Split(trimmed, "\n") {
			writeVerboseLine(sink.writer, palette, styleDefault, line)
		}
	}
}

// logVerbosePrompt renders the prompt for verbose logging.
func logVerbosePrompt(opts RunOptions, prompt Prompt, step int) {
	header := fmt.Sprintf("LLM prompt (step %d)", step)
	for _, sink := range collectVerboseSinks(opts) {
		palette := paletteFor(sink.writer, sink.noColor)
		writeVerboseLine(sink.writer, palette, styleHeadingPrompt, header)
		trimmed := truncateVerboseWithLimit(formatPromptWithLimits(prompt, verboseFormatLimits{
			maxBytes:            sink.maxBytes,
			toolOutputMaxLines:  sink.toolOutputMaxLines,
			trimTrailingNewline: sink.maxBytes > 0,
		}), sink.maxBytes, verboseTruncationMarker)
		if strings.TrimSpace(trimmed) == "" {
			continue
		}
		for _, line := range strings.Split(trimmed, "\n") {
			writeVerboseLine(sink.writer, palette, styleDim, line)
		}
	}
}

// writeVerboseLine writes a single styled verbose line.
func writeVerboseLine(w io.Writer, palette verbosePalette, style verboseStyle, line string) {
	prefix := verbosePrefix
	if palette.enabled {
		prefix = palette.apply(styleDim, prefix)
	}
	fmt.Fprintf(w, "%s %s\n", prefix, palette.apply(style, line))
}

// truncateVerbose truncates long blocks to the configured max length.
func truncateVerbose(value string) string {
	return truncateVerboseWithLimit(value, verboseMaxBytes, verboseTruncationMarker)
}

// truncateVerboseInline truncates inline strings to the configured max length.
func truncateVerboseInline(value string) string {
	return truncateVerboseInlineWithLimit(value, verboseMaxBytes, verboseInlineTruncationMarker)
}

func truncateVerboseWithLimit(value string, maxBytes int, marker string) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	if maxBytes <= len(marker) {
		return marker[:maxBytes]
	}
	return value[:maxBytes-len(marker)] + marker
}

func truncateVerboseInlineWithLimit(value string, maxBytes int, marker string) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	if maxBytes <= len(marker) {
		return marker[:maxBytes]
	}
	return value[:maxBytes-len(marker)] + marker
}

// limitOutputLines trims multi-line strings to a maximum number of lines.
func limitOutputLines(value string, maxLines int) string {
	if maxLines <= 0 {
		return value
	}
	trimmed := strings.TrimRight(value, "\n")
	if strings.TrimSpace(trimmed) == "" {
		return ""
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) <= maxLines {
		return strings.Join(lines, "\n")
	}
	lines = lines[:maxLines]
	last := maxLines - 1
	if strings.TrimSpace(lines[last]) == "" {
		lines[last] = verboseInlineTruncationMarker
	} else {
		lines[last] = lines[last] + " " + verboseInlineTruncationMarker
	}
	return strings.Join(lines, "\n")
}
