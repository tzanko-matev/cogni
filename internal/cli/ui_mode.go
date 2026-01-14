package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// uiModeDecision captures whether to use the live UI.
type uiModeDecision struct {
	useLive bool
	warning string
}

// isTerminal reports whether a writer is a TTY.
var isTerminal = defaultIsTerminal

// resolveUIMode determines whether to enable the live UI.
func resolveUIMode(mode string, verbose bool, stdout io.Writer) (uiModeDecision, error) {
	if verbose {
		return uiModeDecision{useLive: false}, nil
	}
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		normalized = "auto"
	}
	switch normalized {
	case "auto":
		return uiModeDecision{useLive: isTerminal(stdout)}, nil
	case "live":
		if isTerminal(stdout) {
			return uiModeDecision{useLive: true}, nil
		}
		return uiModeDecision{
			useLive: false,
			warning: "Live UI requested but stdout is not a TTY; falling back to plain output.",
		}, nil
	case "plain":
		return uiModeDecision{useLive: false}, nil
	default:
		return uiModeDecision{}, fmt.Errorf("invalid ui mode %q (expected auto|live|plain)", mode)
	}
}

// defaultIsTerminal inspects stdout for TTY support.
func defaultIsTerminal(stdout io.Writer) bool {
	if stdout == nil {
		return false
	}
	if file, ok := stdout.(*os.File); ok {
		return term.IsTerminal(int(file.Fd()))
	}
	if fder, ok := stdout.(interface{ Fd() uintptr }); ok {
		return term.IsTerminal(int(fder.Fd()))
	}
	return false
}
