package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"cogni/internal/tools"
)

const (
	verbosePrefix           = "[verbose]"
	verboseTruncationMarker = "\n... [truncated]"
)

var verboseMaxBytes = tools.DefaultLimits().MaxOutputBytes

func logVerbose(opts RunOptions, format string, args ...any) {
	if !opts.Verbose || opts.VerboseWriter == nil {
		return
	}
	writeVerboseLine(opts.VerboseWriter, fmt.Sprintf(format, args...))
}

func logVerboseBlock(opts RunOptions, header, body string) {
	if !opts.Verbose || opts.VerboseWriter == nil {
		return
	}
	writeVerboseLine(opts.VerboseWriter, header)
	trimmed := truncateVerbose(body)
	if strings.TrimSpace(trimmed) == "" {
		return
	}
	for _, line := range strings.Split(trimmed, "\n") {
		writeVerboseLine(opts.VerboseWriter, line)
	}
}

func logVerbosePrompt(opts RunOptions, prompt Prompt, step int) {
	if !opts.Verbose || opts.VerboseWriter == nil {
		return
	}
	header := fmt.Sprintf("LLM prompt (step %d)", step)
	logVerboseBlock(opts, header, formatPrompt(prompt))
}

func writeVerboseLine(w io.Writer, line string) {
	fmt.Fprintf(w, "%s %s\n", verbosePrefix, line)
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
	case string:
		return fmt.Sprintf("- %s: %s\n", item.Role, content)
	case ToolCall:
		return fmt.Sprintf("- %s: tool_call id=%s name=%s args=%s\n", item.Role, content.ID, content.Name, formatArgs(content.Args))
	case ToolOutput:
		output := strings.TrimRight(content.Result.Output, "\n")
		if output != "" {
			output = indentLines(output, "  ")
			return fmt.Sprintf("- %s: tool_output call_id=%s tool=%s bytes=%d truncated=%t error=%s\n%s\n", item.Role, content.ToolCallID, content.Result.Tool, content.Result.OutputBytes, content.Result.Truncated, content.Result.Error, output)
		}
		return fmt.Sprintf("- %s: tool_output call_id=%s tool=%s bytes=%d truncated=%t error=%s\n", item.Role, content.ToolCallID, content.Result.Tool, content.Result.OutputBytes, content.Result.Truncated, content.Result.Error)
	default:
		return fmt.Sprintf("- %s: %v\n", item.Role, content)
	}
}

func formatArgs(args map[string]any) string {
	if len(args) == 0 {
		return "{}"
	}
	payload, err := json.Marshal(args)
	if err != nil {
		return fmt.Sprintf("%v", args)
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
