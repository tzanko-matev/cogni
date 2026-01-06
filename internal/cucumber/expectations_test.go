package cucumber

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExpectationsAndValidate(t *testing.T) {
	dir := t.TempDir()
	data := `examples:
  cli_run_defaults:
    implemented: true
    notes: "baseline"
  cli_run_outputs: false
`
	path := filepath.Join(dir, "expectations.yml")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write expectations: %v", err)
	}

	expectations, err := LoadExpectations(dir)
	if err != nil {
		t.Fatalf("load expectations: %v", err)
	}
	if len(expectations) != 2 {
		t.Fatalf("expected 2 expectations, got %d", len(expectations))
	}

	examples := []Example{
		{ID: "cli_run_defaults"},
		{ID: "cli_run_outputs"},
	}
	if err := ValidateExpectations(expectations, examples); err != nil {
		t.Fatalf("validate expectations: %v", err)
	}
}

func TestValidateExpectationsMissing(t *testing.T) {
	expectations := map[string]Expectation{
		"cli_run_defaults": {ExampleID: "cli_run_defaults", Implemented: true},
	}
	examples := []Example{
		{ID: "cli_run_defaults"},
		{ID: "cli_run_outputs"},
	}
	if err := ValidateExpectations(expectations, examples); err == nil {
		t.Fatalf("expected missing expectations error")
	}
}
