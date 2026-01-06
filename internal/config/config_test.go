package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogni/internal/spec"
)

func validConfig() spec.Config {
	return spec.Config{
		Version: 1,
		Repo: spec.RepoConfig{
			OutputDir: "./out",
		},
		Agents: []spec.AgentConfig{
			{
				ID:       "default",
				Type:     "builtin",
				Provider: "openrouter",
				Model:    "gpt-4.1-mini",
			},
		},
		DefaultAgent: "default",
		Tasks: []spec.TaskConfig{
			{
				ID:     "task1",
				Type:   "qa",
				Agent:  "default",
				Prompt: "hello",
			},
		},
	}
}

func TestNormalizeDefaultAgent(t *testing.T) {
	cfg := validConfig()
	cfg.DefaultAgent = ""
	cfg.Tasks[0].Agent = ""

	Normalize(&cfg)

	if cfg.DefaultAgent != "default" {
		t.Fatalf("expected default agent to be set, got %q", cfg.DefaultAgent)
	}
	if cfg.Tasks[0].Agent != "default" {
		t.Fatalf("expected task agent to inherit default, got %q", cfg.Tasks[0].Agent)
	}
}

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

func TestValidateCucumberEvalRequiresAdapter(t *testing.T) {
	baseDir, featurePath, _ := writeCucumberFixture(t)
	cfg := validConfig()
	cfg.Tasks = []spec.TaskConfig{{
		ID:             "cucumber",
		Type:           "cucumber_eval",
		Agent:          "default",
		PromptTemplate: "template",
		Features:       []string{featurePath},
	}}

	err := Validate(&cfg, baseDir)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "adapter") {
		t.Fatalf("expected adapter error, got %q", err.Error())
	}
}

func TestValidateCucumberEvalValidConfig(t *testing.T) {
	baseDir, featurePath, expectationsDir := writeCucumberFixture(t)
	cfg := validConfig()
	cfg.Adapters = []spec.AdapterConfig{{
		ID:              "manual",
		Type:            "cucumber_manual",
		FeatureRoots:    []string{"features"},
		ExpectationsDir: expectationsDir,
	}}
	cfg.Tasks = []spec.TaskConfig{{
		ID:             "cucumber",
		Type:           "cucumber_eval",
		Agent:          "default",
		Adapter:        "manual",
		PromptTemplate: "template",
		Features:       []string{featurePath},
	}}

	if err := Validate(&cfg, baseDir); err != nil {
		t.Fatalf("expected config to validate, got %v", err)
	}
}

func TestValidateCucumberAdapterRequiresExpectationsDir(t *testing.T) {
	baseDir, featurePath, _ := writeCucumberFixture(t)
	cfg := validConfig()
	cfg.Adapters = []spec.AdapterConfig{{
		ID:           "manual",
		Type:         "cucumber_manual",
		FeatureRoots: []string{"features"},
	}}
	cfg.Tasks = []spec.TaskConfig{{
		ID:             "cucumber",
		Type:           "cucumber_eval",
		Agent:          "default",
		Adapter:        "manual",
		PromptTemplate: "template",
		Features:       []string{featurePath},
	}}

	err := Validate(&cfg, baseDir)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "expectations_dir") {
		t.Fatalf("expected expectations_dir error, got %q", err.Error())
	}
}

func writeCucumberFixture(t *testing.T) (string, string, string) {
	t.Helper()
	baseDir := t.TempDir()
	featuresDir := filepath.Join(baseDir, "features")
	if err := os.MkdirAll(featuresDir, 0o755); err != nil {
		t.Fatalf("mkdir features: %v", err)
	}
	featurePath := filepath.Join(featuresDir, "sample.feature")
	if err := os.WriteFile(featurePath, []byte("Feature: Sample\n  Scenario: Example\n    Given a step\n"), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	expectationsDir := filepath.Join(baseDir, "expectations")
	if err := os.MkdirAll(expectationsDir, 0o755); err != nil {
		t.Fatalf("mkdir expectations: %v", err)
	}
	return baseDir, filepath.Join("features", "sample.feature"), filepath.Join("expectations")
}
