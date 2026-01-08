package agent

import (
	"context"
	"fmt"
	"time"

	"cogni/internal/tools"
)

// RunnerExecutor executes built-in tool calls against a tools.Runner.
type RunnerExecutor struct {
	Runner *tools.Runner
}

// Execute dispatches a tool call to the underlying runner.
func (e RunnerExecutor) Execute(ctx context.Context, call ToolCall) tools.CallResult {
	if e.Runner == nil {
		return errorResult(call.Name, "tool runner is not configured")
	}
	switch call.Name {
	case "list_files":
		glob, _, err := call.Args.OptionalString("glob")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		return e.Runner.ListFiles(ctx, tools.ListFilesArgs{Glob: glob})
	case "list_dir":
		path, err := call.Args.RequiredString("path")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		offset, err := call.Args.OptionalInt("offset")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		limit, err := call.Args.OptionalInt("limit")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		depth, err := call.Args.OptionalInt("depth")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		return e.Runner.ListDir(ctx, tools.ListDirArgs{
			Path:   path,
			Offset: offset,
			Limit:  limit,
			Depth:  depth,
		})
	case "search":
		query, err := call.Args.RequiredString("query")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		paths, err := call.Args.OptionalStringSlice("paths")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		return e.Runner.Search(ctx, tools.SearchArgs{Query: query, Paths: paths})
	case "read_file":
		path, err := call.Args.RequiredString("path")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		startLine, err := call.Args.OptionalInt("start_line")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		endLine, err := call.Args.OptionalInt("end_line")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		return e.Runner.ReadFile(ctx, tools.ReadFileArgs{Path: path, StartLine: startLine, EndLine: endLine})
	default:
		return errorResult(call.Name, fmt.Sprintf("unknown tool %q", call.Name))
	}
}

// errorResult constructs a tool result describing a tool execution error.
func errorResult(name, message string) tools.CallResult {
	now := time.Now()
	output := "error: " + message
	return tools.CallResult{
		Tool:        name,
		Output:      output,
		OutputBytes: len(output),
		Truncated:   false,
		StartedAt:   now,
		FinishedAt:  now,
		Duration:    0,
		Error:       message,
	}
}
