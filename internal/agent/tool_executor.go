package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cogni/internal/tools"
)

type RunnerExecutor struct {
	Runner *tools.Runner
}

func (e RunnerExecutor) Execute(ctx context.Context, call ToolCall) tools.CallResult {
	if e.Runner == nil {
		return errorResult(call.Name, "tool runner is not configured")
	}
	switch call.Name {
	case "list_files":
		glob, _, err := getOptionalString(call.Args, "glob")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		return e.Runner.ListFiles(ctx, tools.ListFilesArgs{Glob: glob})
	case "search":
		query, err := getRequiredString(call.Args, "query")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		paths, err := getOptionalStringSlice(call.Args, "paths")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		return e.Runner.Search(ctx, tools.SearchArgs{Query: query, Paths: paths})
	case "read_file":
		path, err := getRequiredString(call.Args, "path")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		startLine, err := getOptionalInt(call.Args, "start_line")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		endLine, err := getOptionalInt(call.Args, "end_line")
		if err != nil {
			return errorResult(call.Name, err.Error())
		}
		return e.Runner.ReadFile(ctx, tools.ReadFileArgs{Path: path, StartLine: startLine, EndLine: endLine})
	default:
		return errorResult(call.Name, fmt.Sprintf("unknown tool %q", call.Name))
	}
}

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

func getRequiredString(args map[string]any, key string) (string, error) {
	value, ok, err := getOptionalString(args, key)
	if err != nil {
		return "", err
	}
	if err := checkOptionalError(ok, value, key); err != nil {
		return "", err
	}
	return value, nil
}

func getOptionalString(args map[string]any, key string) (string, bool, error) {
	if args == nil {
		return "", false, nil
	}
	raw, ok := args[key]
	if !ok {
		return "", false, nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", false, fmt.Errorf("%s must be a string", key)
	}
	return strings.TrimSpace(value), true, nil
}

func getOptionalStringSlice(args map[string]any, key string) ([]string, error) {
	if args == nil {
		return nil, nil
	}
	raw, ok := args[key]
	if !ok {
		return nil, nil
	}
	switch value := raw.(type) {
	case []string:
		return value, nil
	case []any:
		paths := make([]string, 0, len(value))
		for _, item := range value {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s entries must be strings", key)
			}
			paths = append(paths, text)
		}
		return paths, nil
	default:
		return nil, fmt.Errorf("%s must be a list of strings", key)
	}
}

func getOptionalInt(args map[string]any, key string) (*int, error) {
	if args == nil {
		return nil, nil
	}
	raw, ok := args[key]
	if !ok {
		return nil, nil
	}
	switch value := raw.(type) {
	case int:
		return &value, nil
	case int64:
		v := int(value)
		return &v, nil
	case float64:
		v := int(value)
		if float64(v) != value {
			return nil, fmt.Errorf("%s must be an integer", key)
		}
		return &v, nil
	default:
		return nil, fmt.Errorf("%s must be an integer", key)
	}
}

func checkOptionalError(ok bool, value string, key string) error {
	if !ok || strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", key)
	}
	return nil
}
