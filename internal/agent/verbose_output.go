package agent

import (
	"fmt"
	"io"
	"strings"
)

// logVerbose emits a styled verbose line when verbosity is enabled.
func logVerbose(opts RunOptions, style verboseStyle, message string) {
	if !opts.Verbose || opts.VerboseWriter == nil {
		return
	}
	palette := paletteFor(opts.VerboseWriter, opts.NoColor)
	writeVerboseLine(opts.VerboseWriter, palette, style, message)
}

// logVerboseBlock writes a header and a multi-line body to the verbose stream.
func logVerboseBlock(opts RunOptions, header, body string, headerStyle, bodyStyle verboseStyle) {
	if !opts.Verbose || opts.VerboseWriter == nil {
		return
	}
	palette := paletteFor(opts.VerboseWriter, opts.NoColor)
	writeVerboseLine(opts.VerboseWriter, palette, headerStyle, header)
	trimmed := truncateVerbose(body)
	if strings.TrimSpace(trimmed) == "" {
		return
	}
	for _, line := range strings.Split(trimmed, "\n") {
		writeVerboseLine(opts.VerboseWriter, palette, bodyStyle, line)
	}
}

// logVerboseToolOutput writes a tool output block with truncation rules.
func logVerboseToolOutput(opts RunOptions, header, body string) {
	if !opts.Verbose || opts.VerboseWriter == nil {
		return
	}
	palette := paletteFor(opts.VerboseWriter, opts.NoColor)
	writeVerboseLine(opts.VerboseWriter, palette, styleHeadingToolResult, header)
	trimmed := truncateVerboseInline(limitOutputLines(body, verboseToolOutputMaxLines))
	if strings.TrimSpace(trimmed) == "" {
		return
	}
	for _, line := range strings.Split(trimmed, "\n") {
		writeVerboseLine(opts.VerboseWriter, palette, styleDefault, line)
	}
}

// logVerbosePrompt renders the prompt for verbose logging.
func logVerbosePrompt(opts RunOptions, prompt Prompt, step int) {
	if !opts.Verbose || opts.VerboseWriter == nil {
		return
	}
	header := fmt.Sprintf("LLM prompt (step %d)", step)
	logVerboseBlock(opts, header, formatPrompt(prompt), styleHeadingPrompt, styleDim)
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
	if verboseMaxBytes <= 0 || len(value) <= verboseMaxBytes {
		return value
	}
	if verboseMaxBytes <= len(verboseTruncationMarker) {
		return verboseTruncationMarker[:verboseMaxBytes]
	}
	return value[:verboseMaxBytes-len(verboseTruncationMarker)] + verboseTruncationMarker
}

// truncateVerboseInline truncates inline strings to the configured max length.
func truncateVerboseInline(value string) string {
	if verboseMaxBytes <= 0 || len(value) <= verboseMaxBytes {
		return value
	}
	if verboseMaxBytes <= len(verboseInlineTruncationMarker) {
		return verboseInlineTruncationMarker[:verboseMaxBytes]
	}
	return value[:verboseMaxBytes-len(verboseInlineTruncationMarker)] + verboseInlineTruncationMarker
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
