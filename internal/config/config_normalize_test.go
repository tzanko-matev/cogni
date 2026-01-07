package config

import "testing"

// TestNormalizeDefaultAgent verifies default agent normalization.
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
