package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// ReadFile executes the read_file tool.
func (r *Runner) ReadFile(ctx context.Context, args ReadFileArgs) CallResult {
	start := r.clock()
	output, truncated, err := r.readFile(ctx, args)
	end := r.clock()
	return r.finalize("read_file", start, end, output, truncated, err)
}

// readFile returns a file slice with line numbers.
func (r *Runner) readFile(ctx context.Context, args ReadFileArgs) (string, bool, error) {
	_ = ctx
	if strings.TrimSpace(args.Path) == "" {
		return "", false, fmt.Errorf("path is required")
	}
	startLine, endLine, err := normalizeLineRange(args.StartLine, args.EndLine)
	if err != nil {
		return "", false, err
	}
	rel, abs, err := resolvePath(r.Root, args.Path)
	if err != nil {
		return "", false, err
	}

	file, err := os.Open(abs)
	if err != nil {
		return "", false, fmt.Errorf("open %s: %w", rel, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", false, fmt.Errorf("stat %s: %w", rel, err)
	}
	if info.IsDir() {
		return "", false, fmt.Errorf("%s is a directory", rel)
	}

	var builder strings.Builder
	builder.WriteString(rel)
	builder.WriteString("\n")

	reader := bufio.NewReader(file)
	lineNumber := 0
	truncated := false
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return "", false, fmt.Errorf("read %s: %w", rel, readErr)
		}
		if line == "" && readErr == io.EOF {
			break
		}
		lineNumber++
		if lineNumber < startLine {
			if readErr == io.EOF {
				break
			}
			continue
		}
		if endLine != 0 && lineNumber > endLine {
			break
		}
		line = strings.TrimRight(line, "\n")
		builder.WriteString(fmt.Sprintf("%d:%s\n", lineNumber, line))
		if r.Limits.MaxReadBytes > 0 && builder.Len() >= r.Limits.MaxReadBytes {
			truncated = true
			break
		}
		if readErr == io.EOF {
			break
		}
	}

	return builder.String(), truncated, nil
}

// normalizeLineRange validates and normalizes line range inputs.
func normalizeLineRange(start, end *int) (int, int, error) {
	startLine := 1
	if start != nil {
		if *start < 1 {
			return 0, 0, fmt.Errorf("start_line must be >= 1")
		}
		startLine = *start
	}
	endLine := 0
	if end != nil {
		if *end < 1 {
			return 0, 0, fmt.Errorf("end_line must be >= 1")
		}
		endLine = *end
	}
	if endLine != 0 && endLine < startLine {
		return 0, 0, fmt.Errorf("end_line must be >= start_line")
	}
	return startLine, endLine, nil
}
