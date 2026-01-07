package tools

import "time"

// truncationMarker marks truncated output.
const truncationMarker = "\n... [truncated]"

// Limits configure read and output size caps for tool execution.
type Limits struct {
	MaxReadBytes   int
	MaxOutputBytes int
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
