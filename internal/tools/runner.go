package tools

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// truncationMarker marks truncated output.
const truncationMarker = "\n... [truncated]"

// Limits configure read and output size caps for tool execution.
type Limits struct {
	MaxReadBytes   int
	MaxOutputBytes int
}

// DefaultLimits returns the default read/output limits.
func DefaultLimits() Limits {
	return Limits{
		MaxReadBytes:   200 * 1024,
		MaxOutputBytes: 200 * 1024,
	}
}

// CallResult captures a tool execution outcome.
type CallResult struct {
	Tool        string
	Output      string
	OutputBytes int
	Truncated   bool
	StartedAt   time.Time
	FinishedAt  time.Time
	Duration    time.Duration
	Error       string
}

// ListFilesArgs configures list_files tool execution.
type ListFilesArgs struct {
	Glob string
}

// SearchArgs configures search tool execution.
type SearchArgs struct {
	Query string
	Paths []string
}

// ReadFileArgs configures read_file tool execution.
type ReadFileArgs struct {
	Path      string
	StartLine *int
	EndLine   *int
}

// Runner executes repository tools within a repo root.
type Runner struct {
	Root   string
	Limits Limits
	clock  func() time.Time
}

// NewRunner constructs a Runner for a repository root.
func NewRunner(root string) (*Runner, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("root is empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root is not a directory")
	}
	return &Runner{
		Root:   abs,
		Limits: DefaultLimits(),
		clock:  time.Now,
	}, nil
}

// ListFiles executes the list_files tool.
func (r *Runner) ListFiles(ctx context.Context, args ListFilesArgs) CallResult {
	start := r.clock()
	output, err := r.listFiles(ctx, args)
	end := r.clock()
	return r.finalize("list_files", start, end, output, false, err)
}

// Search executes the search tool.
func (r *Runner) Search(ctx context.Context, args SearchArgs) CallResult {
	start := r.clock()
	output, err := r.search(ctx, args)
	end := r.clock()
	return r.finalize("search", start, end, output, false, err)
}

// ReadFile executes the read_file tool.
func (r *Runner) ReadFile(ctx context.Context, args ReadFileArgs) CallResult {
	start := r.clock()
	output, truncated, err := r.readFile(ctx, args)
	end := r.clock()
	return r.finalize("read_file", start, end, output, truncated, err)
}

// finalize assembles a CallResult with timing and truncation metadata.
func (r *Runner) finalize(tool string, start, end time.Time, output string, truncated bool, err error) CallResult {
	if err != nil {
		output = fmt.Sprintf("error: %s", err.Error())
	}
	output, limited := applyOutputLimit(output, r.Limits.MaxOutputBytes, truncated)
	return CallResult{
		Tool:        tool,
		Output:      output,
		OutputBytes: len(output),
		Truncated:   limited,
		StartedAt:   start,
		FinishedAt:  end,
		Duration:    end.Sub(start),
		Error:       errorString(err),
	}
}

// listFiles returns file listings using ripgrep.
func (r *Runner) listFiles(ctx context.Context, args ListFilesArgs) (string, error) {
	rgArgs := []string{"--files"}
	if glob := strings.TrimSpace(args.Glob); glob != "" {
		rgArgs = append(rgArgs, "-g", glob)
	}
	return runRG(ctx, r.Root, rgArgs...)
}

// search runs ripgrep to find query matches.
func (r *Runner) search(ctx context.Context, args SearchArgs) (string, error) {
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	paths := make([]string, 0, len(args.Paths))
	for _, path := range args.Paths {
		rel, _, err := resolvePath(r.Root, path)
		if err != nil {
			return "", err
		}
		paths = append(paths, rel)
	}
	rgArgs := []string{"--no-heading", "--line-number", "--color", "never", "--", query}
	if len(paths) > 0 {
		rgArgs = append(rgArgs, paths...)
	}
	return runRG(ctx, r.Root, rgArgs...)
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

// resolvePath resolves a relative path within the repo root.
func resolvePath(root, path string) (string, string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", "", fmt.Errorf("path is empty")
	}
	cleaned := filepath.Clean(trimmed)
	var rel string
	if filepath.IsAbs(cleaned) {
		relative, err := filepath.Rel(root, cleaned)
		if err != nil {
			return "", "", fmt.Errorf("resolve path %q: %w", path, err)
		}
		rel = relative
	} else {
		rel = cleaned
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("path %q escapes root", path)
	}
	abs := filepath.Join(root, rel)
	return rel, abs, nil
}

// runRG executes ripgrep and returns stdout or an error.
func runRG(ctx context.Context, dir string, args ...string) (string, error) {
	if _, err := exec.LookPath("rg"); err != nil {
		return "", fmt.Errorf("rg not found")
	}
	cmd := exec.CommandContext(ctx, "rg", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return stdout.String(), nil
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = "no stderr"
		}
		return "", fmt.Errorf("rg %s: %w (%s)", strings.Join(args, " "), err, msg)
	}
	return stdout.String(), nil
}

// applyOutputLimit truncates output to a maximum size.
func applyOutputLimit(output string, max int, truncated bool) (string, bool) {
	if max <= 0 {
		return output, truncated
	}
	if len(output) > max {
		return truncateOutput(output, max)
	}
	if truncated {
		if len(output)+len(truncationMarker) <= max {
			return output + truncationMarker, true
		}
		return truncateOutput(output, max)
	}
	return output, false
}

// truncateOutput trims output and appends a truncation marker.
func truncateOutput(output string, max int) (string, bool) {
	if max <= 0 || len(output) <= max {
		return output, false
	}
	if max <= len(truncationMarker) {
		return truncationMarker[:max], true
	}
	return output[:max-len(truncationMarker)] + truncationMarker, true
}

// errorString formats errors for CallResult output.
func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
