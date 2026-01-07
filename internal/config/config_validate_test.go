package config

import (
	"errors"
	"strings"
	"testing"
)

// TestValidateDetectsDuplicateAgentIDs verifies duplicate agent IDs are flagged.
func TestValidateDetectsDuplicateAgentIDs(t *testing.T) {
	cfg := validConfig()
	cfg.Agents = append(cfg.Agents, cfg.Agents[0])

	err := Validate(&cfg, ".")
	if err == nil {
		t.Fatalf("expected validation error")
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if len(validationErr.Issues) == 0 {
		t.Fatalf("expected issues, got none")
	}
}

// TestValidateMissingOutputDir verifies missing output dir is flagged.
func TestValidateMissingOutputDir(t *testing.T) {
	cfg := validConfig()
	cfg.Repo.OutputDir = ""

	err := Validate(&cfg, ".")
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "repo.output_dir") {
		t.Fatalf("expected output_dir error, got %q", err.Error())
	}
}

// TestValidateMissingSchemaFile verifies missing schema files are flagged.
func TestValidateMissingSchemaFile(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks[0].Eval.JSONSchema = "schemas/missing.json"

	err := Validate(&cfg, t.TempDir())
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "json_schema") {
		t.Fatalf("expected schema error, got %q", err.Error())
	}
}

// TestValidateRejectsNegativeBudgets verifies negative budgets are rejected.
func TestValidateRejectsNegativeBudgets(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks[0].Budget.MaxTokens = -1
	cfg.Tasks[0].Budget.MaxSeconds = -5
	cfg.Tasks[0].Budget.MaxSteps = -2

	err := Validate(&cfg, ".")
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "budget") {
		t.Fatalf("expected budget error, got %q", err.Error())
	}
}
