package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"cogni/internal/tools"
	"golang.org/x/term"
)

const (
	verbosePrefix                 = "[verbose]"
	verboseTruncationMarker       = "\n... [truncated]"
	verboseInlineTruncationMarker = "... [truncated]"
	verboseToolOutputMaxLines     = 5
)

var verboseMaxBytes = tools.DefaultLimits().MaxOutputBytes
var isTerminal = term.IsTerminal

const (
	ansiReset   = "\x1b[0m"
	ansiBold    = "\x1b[1m"
	ansiDim     = "\x1b[2m"
	ansiMagenta = "\x1b[35m"
	ansiRed     = "\x1b[31m"
	ansiYellow  = "\x1b[33m"
	ansiGreen   = "\x1b[32m"
	ansiCyan    = "\x1b[36m"
	ansiBlue    = "\x1b[34m"
	ansiGray    = "\x1b[90m"
)

type verboseStyle int

const (
	styleDefault verboseStyle = iota
	styleDim
	styleHeadingPrompt
	styleHeadingOutput
	styleHeadingToolCall
	styleHeadingToolResult
	styleHeadingMetrics
	styleHeadingTask
	styleHeadingError
)

type verbosePalette struct {
	enabled bool
}

// logVerbose emits a styled verbose line when verbosity is enabled.
func logVerbose(opts RunOptions, style verboseStyle, message string) {
	if !opts.Verbose || opts.VerboseWriter == nil {
		return
	}
	palette := paletteFor(opts.VerboseWriter, opts.NoColor)
	writeVerboseLine(opts.VerboseWriter, palette, style, message)
}

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

func logVerbosePrompt(opts RunOptions, prompt Prompt, step int) {
	if !opts.Verbose || opts.VerboseWriter == nil {
		return
	}
	header := fmt.Sprintf("LLM prompt (step %d)", step)
	logVerboseBlock(opts, header, formatPrompt(prompt), styleHeadingPrompt, styleDim)
}

func writeVerboseLine(w io.Writer, palette verbosePalette, style verboseStyle, line string) {
	prefix := verbosePrefix
	if palette.enabled {
		prefix = palette.apply(styleDim, prefix)
	}
	fmt.Fprintf(w, "%s %s\n", prefix, palette.apply(style, line))
}

func paletteFor(writer io.Writer, noColor bool) verbosePalette {
	if noColor {
		return verbosePalette{enabled: false}
	}
	return verbosePalette{enabled: shouldUseStyling(writer)}
}

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
		return isTerminal(int(file.Fd()))
	}
	if fder, ok := writer.(interface{ Fd() uintptr }); ok {
		return isTerminal(int(fder.Fd()))
	}
	return false
}

func (p verbosePalette) apply(style verboseStyle, text string) string {
	if !p.enabled {
		return text
	}
	switch style {
	case styleDim:
		return ansiDim + ansiGray + text + ansiReset
	case styleHeadingPrompt:
		return ansiBold + ansiCyan + text + ansiReset
	case styleHeadingOutput:
		return ansiBold + ansiMagenta + text + ansiReset
	case styleHeadingToolCall:
		return ansiBold + ansiYellow + text + ansiReset
	case styleHeadingToolResult:
		return ansiBold + ansiGreen + text + ansiReset
	case styleHeadingMetrics:
		return ansiBold + ansiBlue + text + ansiReset
	case styleHeadingTask:
		return ansiBold + ansiCyan + text + ansiReset
	case styleHeadingError:
		return ansiBold + ansiRed + text + ansiReset
	default:
		return text
	}
}

func formatPrompt(prompt Prompt) string {
	var builder strings.Builder
	if strings.TrimSpace(prompt.Instructions) != "" {
		builder.WriteString("instructions:\n")
		builder.WriteString(prompt.Instructions)
		builder.WriteString("\n")
	}
	if len(prompt.Tools) > 0 {
		toolNames := make([]string, 0, len(prompt.Tools))
		for _, tool := range prompt.Tools {
			toolNames = append(toolNames, tool.Name)
		}
		builder.WriteString("tools: ")
		builder.WriteString(strings.Join(toolNames, ", "))
		builder.WriteString("\n")
	}
	if strings.TrimSpace(prompt.OutputSchema) != "" {
		builder.WriteString("output_schema:\n")
		builder.WriteString(prompt.OutputSchema)
		builder.WriteString("\n")
	}
	if len(prompt.InputItems) > 0 {
		builder.WriteString("input_items:\n")
		for _, item := range prompt.InputItems {
			builder.WriteString(formatHistoryItem(item))
		}
	}
	return strings.TrimRight(builder.String(), "\n")
}

func formatHistoryItem(item HistoryItem) string {
	switch content := item.Content.(type) {
	case HistoryText:
		return fmt.Sprintf("- %s: %s\n", item.Role, content.Text)
	case ToolCall:
		return fmt.Sprintf("- %s: tool_call id=%s name=%s args=%s\n", item.Role, content.ID, content.Name, formatArgs(content.Args))
	case ToolOutput:
		output := strings.TrimRight(content.Result.Output, "\n")
		if output != "" {
			output = truncateVerboseInline(limitOutputLines(output, verboseToolOutputMaxLines))
			output = indentLines(output, "  ")
			return fmt.Sprintf("- %s: tool_output call_id=%s tool=%s bytes=%d truncated=%t error=%s\n%s\n", item.Role, content.ToolCallID, content.Result.Tool, content.Result.OutputBytes, content.Result.Truncated, content.Result.Error, output)
		}
		return fmt.Sprintf("- %s: tool_output call_id=%s tool=%s bytes=%d truncated=%t error=%s\n", item.Role, content.ToolCallID, content.Result.Tool, content.Result.OutputBytes, content.Result.Truncated, content.Result.Error)
	default:
		return fmt.Sprintf("- %s: %v\n", item.Role, content)
	}
}

func formatArgs(args ToolCallArgs) string {
	if len(args) == 0 {
		return "{}"
	}
	payload, err := json.Marshal(args)
	if err != nil {
		return "<invalid args>"
	}
	return string(payload)
}

func indentLines(value, prefix string) string {
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func truncateVerbose(value string) string {
	if verboseMaxBytes <= 0 || len(value) <= verboseMaxBytes {
		return value
	}
	if verboseMaxBytes <= len(verboseTruncationMarker) {
		return verboseTruncationMarker[:verboseMaxBytes]
	}
	return value[:verboseMaxBytes-len(verboseTruncationMarker)] + verboseTruncationMarker
}

func truncateVerboseInline(value string) string {
	if verboseMaxBytes <= 0 || len(value) <= verboseMaxBytes {
		return value
	}
	if verboseMaxBytes <= len(verboseInlineTruncationMarker) {
		return verboseInlineTruncationMarker[:verboseMaxBytes]
	}
	return value[:verboseMaxBytes-len(verboseInlineTruncationMarker)] + verboseInlineTruncationMarker
}

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
