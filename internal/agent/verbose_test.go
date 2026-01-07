package agent

import (
	"bytes"
	"strings"
	"testing"
)

// fakeTTY simulates a terminal writer for styling tests.
type fakeTTY struct {
	bytes.Buffer
}

// Fd returns a dummy file descriptor for fakeTTY.
func (t *fakeTTY) Fd() uintptr {
	return uintptr(1)
}

// TestVerboseNoColorDisablesStyling verifies NO_COLOR disables ANSI styling.
func TestVerboseNoColorDisablesStyling(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "1")

	origTerminal := isTerminal
	isTerminal = func(_ int) bool { return true }
	t.Cleanup(func() { isTerminal = origTerminal })

	tty := &fakeTTY{}
	logVerbose(RunOptions{Verbose: true, VerboseWriter: tty, NoColor: true}, styleHeadingPrompt, "hello")
	if strings.Contains(tty.String(), "\x1b[") {
		t.Fatalf("expected no ANSI codes when no-color is set, got %q", tty.String())
	}

	tty.Reset()
	logVerbose(RunOptions{Verbose: true, VerboseWriter: tty, NoColor: false}, styleHeadingPrompt, "hello")
	if !strings.Contains(tty.String(), "\x1b[") {
		t.Fatalf("expected ANSI codes when styling is enabled, got %q", tty.String())
	}
}
