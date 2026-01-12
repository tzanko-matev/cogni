package call

import (
	"cogni/internal/tools"
	"golang.org/x/term"
)

// Verbose output formatting constants.
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
