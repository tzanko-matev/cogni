package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCommandSuccess(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, ".cogni", "config.yml")
	config := []byte(`version: 1
repo:
  output_dir: "./out"
agents:
  - id: default
    type: builtin
    provider: openrouter
    model: gpt-4.1-mini
default_agent: default
tasks:
  - id: task1
    type: qa
    agent: default
    prompt: "hello"
`)
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(specPath, config, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var out, err bytes.Buffer
	code := Run([]string{"validate", "--spec", specPath}, &out, &err)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d", ExitOK, code)
	}
	if err.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", err.String())
	}
	if !strings.Contains(out.String(), "Config OK") {
		t.Fatalf("expected success message, got %q", out.String())
	}
}

func TestValidateCommandFailure(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, ".cogni", "config.yml")
	config := []byte(`version: 1
repo:
  output_dir: ""
`)
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(specPath, config, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var out, err bytes.Buffer
	code := Run([]string{"validate", "--spec", specPath}, &out, &err)
	if code != ExitError {
		t.Fatalf("expected exit %d, got %d", ExitError, code)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", out.String())
	}
	if !strings.Contains(err.String(), "Validation failed") {
		t.Fatalf("expected validation failure, got %q", err.String())
	}
}

func TestValidateFindsConfigInParent(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, ".cogni", "config.yml")
	config := []byte(`version: 1
repo:
  output_dir: "./out"
agents:
  - id: default
    type: builtin
    provider: openrouter
    model: gpt-4.1-mini
default_agent: default
tasks:
  - id: task1
    type: qa
    agent: default
    prompt: "hello"
`)
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(specPath, config, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	nested := filepath.Join(dir, "nested", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	var out, stderr bytes.Buffer
	code := Run([]string{"validate"}, &out, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(out.String(), "Config OK") {
		t.Fatalf("expected success message, got %q", out.String())
	}
}
