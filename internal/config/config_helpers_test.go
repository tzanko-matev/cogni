package config

import (
	"os"
	"path/filepath"
	"testing"

	"cogni/internal/spec"
)

// validConfig returns a minimal config used by validation tests.
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

// writeCucumberFixture creates a feature and expectations directory for tests.
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
