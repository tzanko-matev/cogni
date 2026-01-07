package config

import (
	"strings"
	"testing"

	"cogni/internal/spec"
)

// TestValidateCucumberEvalRequiresAdapter verifies cucumber eval requires an adapter.
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

// TestValidateCucumberEvalValidConfig verifies valid cucumber config passes.
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

// TestValidateCucumberAdapterRequiresExpectationsDir verifies expectations dir requirement.
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
