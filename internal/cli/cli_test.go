package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootHelp(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"--help"}, &out, &err)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d", ExitOK, code)
	}
	if err.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", err.String())
	}
	output := out.String()
	if !strings.Contains(output, "Usage:") {
		t.Fatalf("expected usage header, got %q", output)
	}
	for _, cmd := range commands {
		if !strings.Contains(output, cmd.Name) {
			t.Fatalf("expected command %q in output", cmd.Name)
		}
	}
}

func TestNoArgsShowsUsage(t *testing.T) {
	var out, err bytes.Buffer
	code := Run(nil, &out, &err)
	if code != ExitUsage {
		t.Fatalf("expected exit %d, got %d", ExitUsage, code)
	}
	if err.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", err.String())
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("expected usage output, got %q", out.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"nope"}, &out, &err)
	if code != ExitUsage {
		t.Fatalf("expected exit %d, got %d", ExitUsage, code)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", out.String())
	}
	if !strings.Contains(err.String(), "Unknown command") {
		t.Fatalf("expected unknown command error, got %q", err.String())
	}
	if !strings.Contains(err.String(), "Usage:") {
		t.Fatalf("expected usage in stderr, got %q", err.String())
	}
}

func TestCommandHelp(t *testing.T) {
	for _, cmd := range commands {
		var out, err bytes.Buffer
		code := Run([]string{cmd.Name, "--help"}, &out, &err)
		if code != ExitOK {
			t.Fatalf("%s: expected exit %d, got %d", cmd.Name, ExitOK, code)
		}
		if err.Len() != 0 {
			t.Fatalf("%s: expected no stderr output, got %q", cmd.Name, err.String())
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Fatalf("%s: expected usage output, got %q", cmd.Name, out.String())
		}
		for _, line := range cmd.Usage {
			if !strings.Contains(out.String(), line) {
				t.Fatalf("%s: expected usage line %q", cmd.Name, line)
			}
		}
	}
}
