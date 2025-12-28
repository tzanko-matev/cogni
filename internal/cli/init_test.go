package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommandCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "cogni.yml")
	schemaPath := filepath.Join(dir, "schemas", "auth_flow_summary.schema.json")

	var out, err bytes.Buffer
	code := Run([]string{"init", "--spec", specPath}, &out, &err)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d", ExitOK, code)
	}
	if err.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", err.String())
	}
	if !strings.Contains(out.String(), "Wrote") {
		t.Fatalf("expected output to include writes, got %q", out.String())
	}
	if _, statErr := os.Stat(specPath); statErr != nil {
		t.Fatalf("expected spec file to exist: %v", statErr)
	}
	if _, statErr := os.Stat(schemaPath); statErr != nil {
		t.Fatalf("expected schema file to exist: %v", statErr)
	}
}

func TestInitCommandRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "cogni.yml")
	if err := os.WriteFile(specPath, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	var out, err bytes.Buffer
	code := Run([]string{"init", "--spec", specPath}, &out, &err)
	if code != ExitError {
		t.Fatalf("expected exit %d, got %d", ExitError, code)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", out.String())
	}
	if !strings.Contains(err.String(), "already exists") {
		t.Fatalf("expected overwrite warning, got %q", err.String())
	}
}
